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
	"time"
)

func Min(x, y uint64) uint64 {
	if x < y {
		return x
	} else {
		return y
	}
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

func (e *Economist) GetRewardsKeeper() (*prototype.InternalRewardsKeeper, error) {
	keeperWrap := table.NewSoRewardsKeeperWrap(e.db, e.singleId)
	if !keeperWrap.CheckExist() {
		return nil, errors.New("Economist access rewards keeper error")
	}
	return keeperWrap.GetKeeper(), nil
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

	authorReward := creatorReward * constants.RewardRateAuthor / constants.PERCENT
	replyReward := creatorReward * constants.RewardRateReply / constants.PERCENT
	//voterReward := creatorReward * constants.RewardRateVoter / constants.PERCENT
	voterReward := creatorReward - authorReward - replyReward
	//reportReward := creatorReward * constants.RewardRateReport / constants.PERCENT

	if err != nil {
		panic("Mint failed when getprops")
	}

	bpWrap, err := e.GetAccount(globalProps.CurrentWitness)
	if err != nil {
		panic("Mint failed when get bp wrap")
	}
	// add rewards to bp
	bpWrap.MdVestingShares(&prototype.Vest{Value: bpWrap.GetVestingShares().Value + bpReward})

	e.modifyGlobalDynamicData(func(props *prototype.DynamicProperties) {
		props.PostRewards.Value += uint64(authorReward)
		props.ReplyRewards.Value += uint64(replyReward)
		//props.ReportRewards.Value += uint64(reportReward)
		props.DappRewards.Value += uint64(dappReward)
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
	// for now, report reward does not calculate
	//if globalProps.HeadBlockNumber % constants.ReportCashout == 0 {
	//	reportRewards := globalProps.ReportRewards.Value
	//	postRewards := reportRewards / 2
	//	replyRewards := reportRewards - postRewards
	//	e.modifyGlobalDynamicData(func(props *prototype.DynamicProperties) {
	//		props.ReportRewards.Value = 0
	//		props.PostRewards.Value += postRewards
	//		props.ReplyRewards.Value += replyRewards
	//	})
	//}
	timestamp := globalProps.Time.UtcSeconds
	iterator := table.NewPostCashoutTimeWrap(e.db)
	var pids []*uint64
	err = iterator.ForEachByOrder(nil, &prototype.TimePointSec{UtcSeconds: timestamp}, nil, nil, func(mVal *uint64, sVal *prototype.TimePointSec, idx uint32) bool {
		pids = append(pids, mVal)
		return true
	})
	if err != nil {
		panic("economist do failed when iterator")
	}
	var posts []*table.SoPostWrap
	var replies []*table.SoPostWrap

	var vpAccumulator uint64 = 0
	// posts accumulate by linear, replies by sqrt
	for _, pid := range pids {
		post := table.NewSoPostWrap(e.db, pid)
		if post.GetParentId() == 0 {
			posts = append(posts, post)
			vpAccumulator += post.GetWeightedVp()
		} else {
			replies = append(replies, post)
			vpAccumulator += uint64(math.Ceil(math.Sqrt(float64(post.GetWeightedVp()))))
		}
	}
	e.modifyGlobalDynamicData(func(props *prototype.DynamicProperties) {
		props.WeightedVps += vpAccumulator
	})
	//keeper, err := e.GetRewardsKeeper()
	e.log.Debugf("cashout posts length: %d", len(posts))
	if len(posts) > 0 {
		t := time.Now()
		e.postCashout(posts)
		e.log.Debugf("cashout posts spend: %v", time.Now().Sub(t))
	}

	if err != nil {
		panic("economist do failed when get reward keeper")
	}

	e.log.Debugf("cashout replies length: %d", len(posts))
	if len(replies) > 0 {
		t := time.Now()
		e.replyCashout(replies)
		e.log.Debugf("cashout reply spend: %v", time.Now().Sub(t))
	}
	//e.updateRewardsKeeper(keeper)
}

func (e *Economist) decayGlobalVotePower() {
	e.modifyGlobalDynamicData(func(props *prototype.DynamicProperties) {
		props.WeightedVps -= props.WeightedVps * constants.BlockInterval / constants.VpDecayTime
	})
}

func (e *Economist) postCashout(posts []*table.SoPostWrap) {
	globalProps, err := e.GetProps()
	if err != nil {
		panic("post cashout get props failed")
	}

	var vpAccumulator uint64 = 0
	for _, post := range posts {
		vpAccumulator += post.GetWeightedVp()
	}
	var blockReward uint64 = 0
	var blockDappReward uint64 = 0
	//var blockVoterReward uint64 = 0
	if globalProps.WeightedVps > 0 {
		blockReward = vpAccumulator * globalProps.PostRewards.Value / globalProps.WeightedVps
		blockDappReward = vpAccumulator * globalProps.DappRewards.Value / globalProps.WeightedVps
		//blockVoterReward = vpAccumulator * globalProps.VoterRewards.Value / globalProps.WeightedVps
	}
	var spentPostReward uint64 = 0
	var spentDappReward uint64 = 0
	//var spentVoterReward uint64 = 0
	for _, post := range posts {
		author := post.GetAuthor().Value
		var reward uint64 = 0
		var beneficiaryReward uint64 = 0
		//var voterReward uint64 = 0
		// divide zero exception
		if vpAccumulator > 0 {
			reward = post.GetWeightedVp() * blockReward / vpAccumulator
			beneficiaryReward = post.GetWeightedVp() * blockDappReward / vpAccumulator
			//voterReward = post.GetWeightedVp() * blockVoterReward / vpAccumulator
			// discount from reward pool
			spentPostReward += reward
			spentDappReward += beneficiaryReward
			//spentVoterReward += voterReward
		}
		//e.voterCashout(post.GetPostId(), voterReward, post.GetWeightedVp(), innerRewards)
		beneficiaries := post.GetBeneficiaries()
		var spentBeneficiaryReward uint64 = 0
		for _, beneficiary := range beneficiaries {
			name := beneficiary.Name.Value
			weight := beneficiary.Weight
			// one of ten thousands
			r := beneficiaryReward * uint64(weight) / 10000
			beneficiaryWrap, err := e.GetAccount(&prototype.AccountName{Value: name})
			if err != nil {
				e.log.Debugf("beneficiary get account %s failed", name)
				continue
			} else {
				beneficiaryWrap.MdVestingShares(&prototype.Vest{ Value: r + beneficiaryWrap.GetVestingShares().Value})
				spentBeneficiaryReward += r
				e.noticer.Publish(constants.NoticeCashout, name, r, globalProps.GetHeadBlockNumber())
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
		e.noticer.Publish(constants.NoticeCashout, author, reward, globalProps.GetHeadBlockNumber())
		post.MdCashoutTime(&prototype.TimePointSec{UtcSeconds: math.MaxUint32})
		post.MdRewards(&prototype.Vest{Value: reward})
		post.MdDappRewards(&prototype.Vest{Value: beneficiaryReward})
	}
	e.modifyGlobalDynamicData(func(props *prototype.DynamicProperties) {
		props.PostRewards.Value -= spentPostReward
		//props.VoterRewards.Value -= spentVoterReward
		props.DappRewards.Value -= spentDappReward
	})
}

// use same algorithm to simplify
func (e *Economist) replyCashout(replies []*table.SoPostWrap) {
	globalProps, err := e.GetProps()
	if err != nil {
		panic("reply cashout get props failed")
	}
	var vpAccumulator uint64 = 0
	for _, reply := range replies {
		vpAccumulator += uint64(math.Ceil(math.Sqrt(float64(reply.GetWeightedVp()))))
	}
	var blockReward uint64 = 0
	var blockDappReward uint64 = 0
	//var blockVoterReward uint64 = 0
	if globalProps.WeightedVps > 0 {
		blockReward = vpAccumulator * globalProps.ReplyRewards.Value / globalProps.WeightedVps
		blockDappReward = vpAccumulator * globalProps.DappRewards.Value / globalProps.WeightedVps
		//blockVoterReward = vpAccumulator * globalProps.VoterRewards.Value / globalProps.WeightedVps
	}
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
			weightedVp := uint64(math.Ceil(math.Sqrt(float64(reply.GetWeightedVp()))))
			reward = weightedVp * blockReward / vpAccumulator
			beneficiaryReward = weightedVp * blockDappReward / vpAccumulator
			//voterReward = weightedVp * blockVoterReward / vpAccumulator
			spentReplyReward += reward
			spentDappReward += beneficiaryReward
			//spentVoterReward += voterReward
		}
		//e.voterCashout(reply.GetPostId(), voterReward, reply.GetWeightedVp(), innerRewards)
		beneficiaries := reply.GetBeneficiaries()
		var spentBeneficiaryReward uint64 = 0
		for _, beneficiary := range beneficiaries {
			name := beneficiary.Name.Value
			weight := beneficiary.Weight
			r := beneficiaryReward * uint64(weight) / 10000
			beneficiaryWrap, err := e.GetAccount(&prototype.AccountName{Value: name})
			if err != nil {
				e.log.Debugf("beneficiary get account %s failed", name)
			} else {
				beneficiaryWrap.MdVestingShares(&prototype.Vest{ Value: r + beneficiaryWrap.GetVestingShares().Value})
				spentBeneficiaryReward += r
				e.noticer.Publish(constants.NoticeCashout, name, r, globalProps.GetHeadBlockNumber())
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
		e.noticer.Publish(constants.NoticeCashout, author, reward, globalProps.GetHeadBlockNumber())
		reply.MdCashoutTime(&prototype.TimePointSec{UtcSeconds: math.MaxUint32})
		reply.MdRewards(&prototype.Vest{Value: reward})
		reply.MdDappRewards(&prototype.Vest{Value: beneficiaryReward})
	}
	e.modifyGlobalDynamicData(func(props *prototype.DynamicProperties) {
		props.ReplyRewards.Value -= spentReplyReward
		//props.VoterRewards.Value -= spentVoterReward
		props.DappRewards.Value -= spentDappReward
	})
}

func (e *Economist) voterCashout(postId uint64, totalReward uint64, totalVp uint64, keeper map[string]*prototype.Vest) {
	iterator := table.NewVotePostIdWrap(e.db)
	start := postId
	end := postId + 1
	var voterIds []*prototype.VoterId
	_ = iterator.ForEachByOrder(&start, &end, nil, nil, func(mVal *prototype.VoterId, sVal *uint64, idx uint32) bool {
		voterIds = append(voterIds, mVal)
		return true
	})
	for _, voterId := range voterIds {
		wrap := table.NewSoVoteWrap(e.db, voterId)
		vp := wrap.GetWeightedVp()
		voter := voterId.Voter.Value
		reward := totalReward * vp / totalVp
		voterWrap, _ := e.GetAccount(&prototype.AccountName{Value: voter})
		voterWrap.MdVestingShares(&prototype.Vest{Value: reward + voterWrap.GetVestingShares().Value})
	}
}

func (e *Economist) PowerDown() {
	globalProps, err := e.GetProps()
	if err != nil {
		panic("economist do failed when get props")
	}
	timestamp := globalProps.Time.UtcSeconds
	iterator := table.NewAccountNextPowerdownTimeWrap(e.db)
	var accountNames []*prototype.AccountName
	t0 := time.Now()
	err = iterator.ForEachByOrder(nil, &prototype.TimePointSec{UtcSeconds: timestamp}, nil, nil, func(mVal *prototype.AccountName, sVal *prototype.TimePointSec, idx uint32) bool {
		accountNames = append(accountNames, mVal)
		return true
	})
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
			accountWrap.MdNextPowerdownTime(&prototype.TimePointSec{UtcSeconds: math.MaxUint32})
		} else {
			accountWrap.MdNextPowerdownTime(&prototype.TimePointSec{UtcSeconds: timestamp + constants.POWER_DOWN_INTERVAL})
		}
	}
	t2 := time.Now()
	e.log.Debugf("powerdown: %v|%v|%v", t2.Sub(t0), t1.Sub(t0), t2.Sub(t1))
}
