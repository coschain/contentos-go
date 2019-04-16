package economist

import (
	"fmt"
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"math"
	"math/big"
	"time"
)

func Min(x, y uint64) uint64 {
	if x < y {
		return x
	} else {
		return y
	}
}

func ISqrt(n uint64) uint64 {
	var r1, r uint64 = n, n + 1
	for r1 < r {
		r, r1 = r1, (r1+n/r1)>>1
	}
	return r
}

type Economist struct {
	db       iservices.IDatabaseService
	noticer  EventBus.Bus
	singleId *int32
	log *logrus.Logger
}

func New(db iservices.IDatabaseService, noticer EventBus.Bus, singleId *int32, log *logrus.Logger) *Economist {
	return &Economist{db: db, noticer:noticer, singleId: singleId, log: log}
}

func (e *Economist) GetProps() (*prototype.DynamicProperties, error) {
	dgpWrap := table.NewSoGlobalWrap(e.db, e.singleId)
	if !dgpWrap.CheckExist() {
		return nil, errors.New("the mainkey is already exist")
	}
	return dgpWrap.GetProps(), nil
}

func (e *Economist) GetAccount(account *prototype.AccountName) (*table.SoAccountWrap, error) {
	accountWrap := table.NewSoAccountWrap(e.db, account)
	if !accountWrap.CheckExist() {
		return nil, errors.New(fmt.Sprintf("cannot find account %s", account.Value))
	}
	return accountWrap, nil
}

func (e *Economist) modifyGlobalDynamicData(f func(props *prototype.DynamicProperties)) {
	dgpWrap := table.NewSoGlobalWrap(e.db, e.singleId)
	props := dgpWrap.GetProps()

	f(props)

	success := dgpWrap.MdProps(props)
	if !success {
		panic("flush globalDynamic into db error")
	}
}

func (e *Economist) BaseBudget(ith uint32) uint64 {
	if ith > 12 {
		return 0
	}
	var remain uint64 = 0
	// 56 == 35000 - 448 * 13 * 12 / 2
	if ith == 12 {
		remain = uint64(constants.TotalCurrency) * uint64(56) / 1000 / 100 * constants.BaseRate
	}
	return uint64(ith) * uint64(constants.TotalCurrency) * uint64(448) / 1000 / 100 * constants.BaseRate + remain
}


// InitialBonus does not be managed by chain
func (e *Economist) CalculateBudget(ith uint32) uint64 {
	return e.BaseBudget(ith)
}

func (e *Economist) CalculatePerBlockBudget(annalBudget uint64) uint64 {
	return annalBudget / (86400 / 3 * 365)
}

func (e *Economist) Mint() {
	//blockCurrent := constants.PerBlockCurrent
	//t0 := time.Now()
	globalProps, err := e.GetProps()
	if err != nil {
		panic("Mint failed when getprops")
	}
	ith := globalProps.GetIthYear()
	annualBudget := e.CalculateBudget(ith)
	// new year arrived
	if globalProps.GetAnnualBudget().Value != annualBudget {
		e.modifyGlobalDynamicData(func(props *prototype.DynamicProperties) {
			props.AnnualBudget.Value = annualBudget
			props.AnnualMinted.Value = 0
		})
		// reload props
		globalProps, err = e.GetProps()
	}
	blockCurrent := e.CalculatePerBlockBudget(annualBudget)
	// prevent deficit
	if globalProps.GetAnnualBudget().Value > globalProps.GetAnnualMinted().Value &&
		globalProps.GetAnnualBudget().Value <= (globalProps.GetAnnualMinted().Value + blockCurrent) {
		blockCurrent = globalProps.GetAnnualBudget().Value - globalProps.GetAnnualMinted().Value
		// time to update year
		e.modifyGlobalDynamicData(func(props *prototype.DynamicProperties) {
			props.IthYear = props.IthYear + 1
		})
	}

	// it is impossible
	if globalProps.GetAnnualBudget().Value <= globalProps.GetAnnualMinted().Value {
		blockCurrent = 0
		e.modifyGlobalDynamicData(func(props *prototype.DynamicProperties) {
			props.IthYear = props.IthYear + 1
		})
	}

	creatorReward := blockCurrent * constants.RewardRateCreator / constants.PERCENT
	dappReward := blockCurrent * constants.RewardRateDapp / constants.PERCENT
	bpReward := blockCurrent - creatorReward - dappReward

	// merge author rewards and reply rewards
	postReward := creatorReward * constants.RewardRateAuthor / constants.PERCENT
	replyReward := creatorReward * constants.RewardRateReply / constants.PERCENT
	//voterReward := creatorReward * constants.RewardRateVoter / constants.PERCENT
	voterReward := creatorReward - postReward - replyReward
	//reportReward := creatorReward * constants.RewardRateReport / constants.PERCENT

	replyDappRewards := dappReward * constants.RewardRateReply / constants.PERCENT
	postDappRewards := dappReward - replyDappRewards


	bpWrap, err := e.GetAccount(globalProps.CurrentWitness)
	if err != nil {
		panic("Mint failed when get bp wrap")
	}
	// add rewards to bp
	bpWrap.MdVestingShares(&prototype.Vest{Value: bpWrap.GetVestingShares().Value + bpReward})

	e.modifyGlobalDynamicData(func(props *prototype.DynamicProperties) {
		props.PostRewards.Value += uint64(postReward)
		props.ReplyRewards.Value += uint64(replyReward)
		props.PostDappRewards.Value += uint64(postDappRewards)
		props.ReplyDappRewards.Value += uint64(replyDappRewards)
		props.VoterRewards.Value += uint64(voterReward)
		props.AnnualMinted.Value += blockCurrent
	})
}

// Should be claiming or direct modify the balance?
func (e *Economist) Do() {
	e.decayGlobalVotePower()
	globalProps, err := e.GetProps()
	if err != nil {
		panic("economist do failed when get props")
	}
	iterator := table.NewPostCashoutBlockNumWrap(e.db)
	var pids []*uint64
	end := globalProps.HeadBlockNumber
	postWeightedVps := globalProps.PostWeightedVps
	replyWeightedVps := globalProps.ReplyWeightedVps
	t0 := time.Now()
	err = iterator.ForEachByOrder(nil, &end, nil, nil, func(mVal *uint64, sVal *uint64, idx uint32) bool {
		pids = append(pids, mVal)
		return true
	})
	e.log.Debugf("Do iterator spent: %v", time.Now().Sub(t0))
	if err != nil {
		panic("economist do failed when iterator")
	}
	var posts []*table.SoPostWrap
	var replies []*table.SoPostWrap

	var postVpAccumulator uint64 = 0
	var replyVpAccumulator uint64 = 0
	// posts accumulate by linear, replies by sqrt
	for _, pid := range pids {
		post := table.NewSoPostWrap(e.db, pid)
		if post.GetParentId() == 0 {
			posts = append(posts, post)
			postVpAccumulator += post.GetWeightedVp()
		} else {
			replies = append(replies, post)
			//replyVpAccumulator += uint64(math.Ceil(math.Sqrt(float64(post.GetWeightedVp()))))
			replyVpAccumulator += ISqrt(post.GetWeightedVp())
		}
	}
	e.modifyGlobalDynamicData(func(props *prototype.DynamicProperties) {
		props.PostWeightedVps += postVpAccumulator
		props.ReplyWeightedVps += replyVpAccumulator
	})

	if postWeightedVps + postVpAccumulator >= 0 {
		var rewards, dappRewards uint64
		if postWeightedVps + postVpAccumulator == 0 {
			rewards = 0
			dappRewards = 0
		}else {
			// after big, it can not overflow
			bigVpSum := new(big.Int).SetUint64(postWeightedVps + postVpAccumulator)
			bigVpAccumulator := new(big.Int).SetUint64(postVpAccumulator)
			bigGlobalPostRewards := new(big.Int).SetUint64(globalProps.PostRewards.Value)
			bigVpMul := new(big.Int).Mul(bigVpAccumulator, bigGlobalPostRewards)
			rewards = new(big.Int).Div(bigVpMul, bigVpSum).Uint64()
			//rewards = postVpAccumulator * globalProps.PostRewards.Value / (postWeightedVps + postVpAccumulator)
			bigGlobalPostDappRewards := new(big.Int).SetUint64(globalProps.PostDappRewards.Value)
			bigDappVpMul := new(big.Int).Mul(bigVpAccumulator, bigGlobalPostDappRewards)
			dappRewards = new(big.Int).Div(bigDappVpMul, bigVpSum).Uint64()
			//dappRewards = postVpAccumulator * globalProps.PostDappRewards.Value / (postWeightedVps + postVpAccumulator)
		}

		e.log.Debugf("cashout posts length: %d", len(posts))
		if len(posts) > 0 {
			t := time.Now()
			e.postCashout(posts, rewards, dappRewards)
			e.log.Debugf("cashout posts spend: %v", time.Now().Sub(t))
		}
	}

	if replyWeightedVps + replyVpAccumulator >= 0 {
		var rewards, dappRewards uint64
		if replyWeightedVps + replyVpAccumulator == 0 {
			rewards = 0
			dappRewards = 0
		}else {
			//rewards = replyVpAccumulator * globalProps.ReplyRewards.Value / (replyWeightedVps + replyVpAccumulator)
			//dappRewards = replyVpAccumulator * globalProps.ReplyDappRewards.Value / (replyWeightedVps + replyVpAccumulator)
			bigVpSum := new(big.Int).SetUint64(replyWeightedVps + replyVpAccumulator)
			bigVpAccumulator := new(big.Int).SetUint64(replyVpAccumulator)
			bigGlobalReplyRewards := new(big.Int).SetUint64(globalProps.ReplyRewards.Value)
			bigVpMul := new(big.Int).Mul(bigVpAccumulator, bigGlobalReplyRewards)
			rewards = new(big.Int).Div(bigVpMul, bigVpSum).Uint64()
			//rewards = postVpAccumulator * globalProps.PostRewards.Value / (postWeightedVps + postVpAccumulator)
			bigGlobalReplyDappRewards := new(big.Int).SetUint64(globalProps.ReplyDappRewards.Value)
			bigDappVpMul := new(big.Int).Mul(bigVpAccumulator, bigGlobalReplyDappRewards)
			dappRewards = new(big.Int).Div(bigDappVpMul, bigVpSum).Uint64()
		}

		e.log.Debugf("cashout replies length: %d", len(replies))
		if len(replies) > 0 {
			t := time.Now()
			e.replyCashout(replies, rewards, dappRewards)
			e.log.Debugf("cashout reply spend: %v", time.Now().Sub(t))
		}
	}
}

func (e *Economist) decayGlobalVotePower() {
	e.modifyGlobalDynamicData(func(props *prototype.DynamicProperties) {
		props.PostWeightedVps -= props.PostWeightedVps * constants.BlockInterval / constants.VpDecayTime
		props.ReplyWeightedVps -= props.ReplyWeightedVps * constants.BlockInterval / constants.VpDecayTime
	})
}

func (e *Economist) postCashout(posts []*table.SoPostWrap, blockReward uint64, blockDappReward uint64) {
	globalProps, err := e.GetProps()
	if err != nil {
		panic("post cashout get props failed")
	}

	var vpAccumulator uint64 = 0
	for _, post := range posts {
		vpAccumulator += post.GetWeightedVp()
	}
	bigVpAccumulator := new(big.Int).SetUint64(vpAccumulator)
	bigBlockRewards := new(big.Int).SetUint64(blockReward)
	bigBlockDappReward := new(big.Int).SetUint64(blockDappReward)
	e.log.Debugf("current block post total vp:%d, global vp:%d", vpAccumulator, globalProps.PostWeightedVps)
	var spentPostReward uint64 = 0
	var spentDappReward uint64 = 0
	//var spentVoterReward uint64 = 0
	for _, post := range posts {
		author := post.GetAuthor().Value
		var reward uint64 = 0
		var beneficiaryReward uint64 = 0
		// divide zero exception
		if vpAccumulator > 0 {
			//reward = post.GetWeightedVp() * blockReward / vpAccumulator
			//beneficiaryReward = post.GetWeightedVp() * blockDappReward / vpAccumulator
			//spentPostReward += reward
			//spentDappReward += beneficiaryReward
			weightedVp := post.GetWeightedVp()
			bigWeightedVp := new(big.Int).SetUint64(weightedVp)
			bigRewardMul := new(big.Int).Mul(bigWeightedVp,  bigBlockRewards)
			reward = new(big.Int).Div(bigRewardMul, bigVpAccumulator).Uint64()
			bigDappRewardMul := new(big.Int).Mul(bigWeightedVp, bigBlockDappReward)
			beneficiaryReward = new(big.Int).Div(bigDappRewardMul, bigVpAccumulator).Uint64()
			spentPostReward += reward
			spentDappReward += beneficiaryReward
		}
		//e.voterCashout(post.GetPostId(), voterReward, post.GetWeightedVp(), innerRewards)
		beneficiaries := post.GetBeneficiaries()
		var spentBeneficiaryReward uint64 = 0
		var weightSum uint32 = 0
		for _, beneficiary := range beneficiaries {
			name := beneficiary.Name.Value
			weight := beneficiary.Weight
			weightSum += weight
			// malicious user, pass it
			if weightSum > constants.PERCENT {
				continue
			}
			// one of ten thousands
			//r := beneficiaryReward * uint64(weight) / constants.PERCENT
			r := new(big.Int).Div(new(big.Int).Mul(big.NewInt(int64(beneficiaryReward)), big.NewInt(int64(weight))), big.NewInt(constants.PERCENT)).Uint64()
			if r == 0 {
				continue
			}
			beneficiaryWrap, err := e.GetAccount(&prototype.AccountName{Value: name})
			if err != nil {
				e.log.Debugf("beneficiary get account %s failed", name)
				continue
			} else {
				beneficiaryWrap.MdVestingShares(&prototype.Vest{ Value: r + beneficiaryWrap.GetVestingShares().Value})
				spentBeneficiaryReward += r
				e.noticer.Publish(constants.NoticeCashout, name, post.GetPostId(), r, globalProps.GetHeadBlockNumber())
			}
		}
		if beneficiaryReward - spentBeneficiaryReward > 0 {
			reward += beneficiaryReward - spentBeneficiaryReward
		}
		authorWrap, err := e.GetAccount(&prototype.AccountName{Value: author})
		if err != nil {
			e.log.Debugf("post cashout get account %s failed", author)
			continue
		} else {
			authorWrap.MdVestingShares(&prototype.Vest{ Value: reward + authorWrap.GetVestingShares().Value })
		}
		post.MdCashoutBlockNum(math.MaxUint32)
		post.MdRewards(&prototype.Vest{Value: reward})
		post.MdDappRewards(&prototype.Vest{Value: beneficiaryReward})
		if reward > 0 {
			e.noticer.Publish(constants.NoticeCashout, author, post.GetPostId(), reward, globalProps.GetHeadBlockNumber())
		}
	}
	e.log.Infof("cashout: [post] blockRewards: %d, blockDappRewards: %d, spendPostReward: %d, spendDappReward: %d",
		blockReward, blockDappReward, spentPostReward, spentDappReward)
	e.modifyGlobalDynamicData(func(props *prototype.DynamicProperties) {
		props.PostRewards.Value -= spentPostReward
		props.PostDappRewards.Value -= spentDappReward
	})
}

// use same algorithm to simplify
func (e *Economist) replyCashout(replies []*table.SoPostWrap, blockReward uint64, blockDappReward uint64) {
	globalProps, err := e.GetProps()
	if err != nil {
		panic("reply cashout get props failed")
	}
	var vpAccumulator uint64 = 0
	for _, reply := range replies {
		vpAccumulator += ISqrt(reply.GetWeightedVp())
	}
	bigVpAccumulator := new(big.Int).SetUint64(vpAccumulator)
	bigBlockRewards := new(big.Int).SetUint64(blockReward)
	bigBlockDappReward := new(big.Int).SetUint64(blockDappReward)
	e.log.Debugf("current block reply total vp:%d, global vp:%d", vpAccumulator, globalProps.ReplyWeightedVps)
	var spentReplyReward uint64 = 0
	var spentDappReward uint64 = 0
	//var spentVoterReward uint64 = 0
	for _, reply := range replies {
		author := reply.GetAuthor().Value
		var reward uint64 = 0
		var beneficiaryReward uint64 = 0
		//var voterReward uint64 = 0
		// divide zero exception
		if vpAccumulator > 0 {
			weightedVp := ISqrt(reply.GetWeightedVp())
			bigWeightedVp := new(big.Int).SetUint64(weightedVp)
			bigRewardMul := new(big.Int).Mul(bigWeightedVp,  bigBlockRewards)
			reward = new(big.Int).Div(bigRewardMul, bigVpAccumulator).Uint64()
			bigDappRewardMul := new(big.Int).Mul(bigWeightedVp, bigBlockDappReward)
			beneficiaryReward = new(big.Int).Div(bigDappRewardMul, bigVpAccumulator).Uint64()
			spentReplyReward += reward
			spentDappReward += beneficiaryReward
		}
		//e.voterCashout(reply.GetPostId(), voterReward, reply.GetWeightedVp(), innerRewards)
		beneficiaries := reply.GetBeneficiaries()
		var spentBeneficiaryReward uint64 = 0
		var weightSum uint32 = 0
		for _, beneficiary := range beneficiaries {
			name := beneficiary.Name.Value
			weight := beneficiary.Weight
			weightSum += weight
			if weightSum > constants.PERCENT {
				continue
			}
			r := new(big.Int).Div(new(big.Int).Mul(big.NewInt(int64(beneficiaryReward)), big.NewInt(int64(weight))), big.NewInt(constants.PERCENT)).Uint64()
			//r := beneficiaryReward * uint64(weight) / constants.PERCENT
			if r == 0 {
				continue
			}
			beneficiaryWrap, err := e.GetAccount(&prototype.AccountName{Value: name})
			if err != nil {
				e.log.Debugf("beneficiary get account %s failed", name)
			} else {
				beneficiaryWrap.MdVestingShares(&prototype.Vest{ Value: r + beneficiaryWrap.GetVestingShares().Value})
				spentBeneficiaryReward += r
				e.noticer.Publish(constants.NoticeCashout, name, reply.GetPostId(), r, globalProps.GetHeadBlockNumber())
			}
		}
		if beneficiaryReward - spentBeneficiaryReward > 0 {
			reward += beneficiaryReward - spentBeneficiaryReward
		}
		authorWrap, err := e.GetAccount(&prototype.AccountName{Value: author})
		if err != nil {
			e.log.Debugf("reply cashout get account %s failed", author)
			continue
		} else {
			authorWrap.MdVestingShares(&prototype.Vest{ Value: reward + authorWrap.GetVestingShares().Value })
		}
		reply.MdCashoutBlockNum(math.MaxUint32)
		reply.MdRewards(&prototype.Vest{Value: reward})
		reply.MdDappRewards(&prototype.Vest{Value: beneficiaryReward})
		if reward > 0 {
			e.noticer.Publish(constants.NoticeCashout, author, reply.GetPostId(), reward, globalProps.GetHeadBlockNumber())
		}
	}
	e.log.Infof("cashout: [reply] blockRewards: %d, blockDappRewards: %d, spendPostReward: %d, spendDappReward: %d",
		blockReward, blockDappReward, spentReplyReward, spentDappReward)
	e.modifyGlobalDynamicData(func(props *prototype.DynamicProperties) {
		props.ReplyRewards.Value -= spentReplyReward
		props.ReplyDappRewards.Value -= spentDappReward
	})
}

//func (e *Economist) voterCashout(postId uint64, totalReward uint64, totalVp uint64, keeper map[string]*prototype.Vest) {
//	iterator := table.NewVotePostIdWrap(e.db)
//	start := postId
//	end := postId + 1
//	var voterIds []*prototype.VoterId
//	_ = iterator.ForEachByOrder(&start, &end, nil, nil, func(mVal *prototype.VoterId, sVal *uint64, idx uint32) bool {
//		voterIds = append(voterIds, mVal)
//		return true
//	})
//	for _, voterId := range voterIds {
//		wrap := table.NewSoVoteWrap(e.db, voterId)
//		vp := wrap.GetWeightedVp()
//		voter := voterId.Voter.Value
//		reward := totalReward * vp / totalVp
//		voterWrap, _ := e.GetAccount(&prototype.AccountName{Value: voter})
//		voterWrap.MdVestingShares(&prototype.Vest{Value: reward + voterWrap.GetVestingShares().Value})
//	}
//}

func (e *Economist) PowerDown() {
	globalProps, err := e.GetProps()
	if err != nil {
		panic("economist do failed when get props")
	}
	//timestamp := globalProps.Time.UtcSeconds
	//iterator := table.NewAccountNextPowerdownTimeWrap(e.db)
	iterator := table.NewAccountNextPowerdownBlockNumWrap(e.db)
	var accountNames []*prototype.AccountName
	t0 := time.Now()
	current := globalProps.HeadBlockNumber
	t0 = time.Now()
	err = iterator.ForEachByOrder(nil, &current, nil, nil, func(mVal *prototype.AccountName, sVal *uint64, idx uint32) bool {
		accountNames = append(accountNames, mVal)
		return true
	})
	e.log.Debugf("PowerDown iterator spent: %v", time.Now().Sub(t0))
	t1 := time.Now()
	var powerdownQuota uint64 = 0
	for _, accountName := range accountNames {
		accountWrap := table.NewSoAccountWrap(e.db, accountName)
		if accountWrap.GetToPowerdown().Value-accountWrap.GetHasPowerdown().Value < accountWrap.GetEachPowerdownRate().Value {
			powerdownQuota = Min(accountWrap.GetVestingShares().Value, accountWrap.GetToPowerdown().Value-accountWrap.GetHasPowerdown().Value)
		} else {
			powerdownQuota = Min(accountWrap.GetVestingShares().Value, accountWrap.GetEachPowerdownRate().Value)
		}
		vesting := accountWrap.GetVestingShares().Value - powerdownQuota
		balance := accountWrap.GetBalance().Value + powerdownQuota
		hasPowerDown := accountWrap.GetHasPowerdown().Value + powerdownQuota
		accountWrap.MdVestingShares(&prototype.Vest{Value: vesting})
		accountWrap.MdBalance(&prototype.Coin{Value: balance})
		accountWrap.MdHasPowerdown(&prototype.Vest{Value: hasPowerDown})
		// update total cos and total vesting shares
		e.modifyGlobalDynamicData(func(props *prototype.DynamicProperties) {
			props.TotalCos.Value += powerdownQuota
			props.TotalVestingShares.Value -= powerdownQuota
		})
		if accountWrap.GetHasPowerdown().Value >= accountWrap.GetToPowerdown().Value || accountWrap.GetVestingShares().Value == 0 {
			accountWrap.MdEachPowerdownRate(&prototype.Vest{Value: 0})
			accountWrap.MdNextPowerdownBlockNum(math.MaxUint32)
		} else {
			accountWrap.MdNextPowerdownBlockNum(current + constants.PowerDownBlockInterval)
		}
	}
	t2 := time.Now()
	e.log.Debugf("powerdown: %v|%v|%v", t2.Sub(t0), t1.Sub(t0), t2.Sub(t1))
}
