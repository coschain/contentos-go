package app

import (
	"fmt"
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/common/variables"
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

type Economist struct {
	db       iservices.IDatabaseService
	noticer  EventBus.Bus
	log *logrus.Logger
	dgp *DynamicGlobalPropsRW
}

//func mustNoError(err error, val string) {
//	if err != nil {
//		panic(val + " : " + err.Error())
//	}
//}

func ISqrt(n string) *big.Int {
	bigInt := new(big.Int)
	value, _ := bigInt.SetString(n, 10)
	sqrt := bigInt.Sqrt(value)
	return sqrt
}

func NewEconomist(db iservices.IDatabaseService, noticer EventBus.Bus, log *logrus.Logger) *Economist {
	return &Economist{db: db, noticer:noticer, log: log, dgp: &DynamicGlobalPropsRW{db: db}}
}

//func (e *Economist) GetProps() (*prototype.DynamicProperties, error) {
//	dgpWrap := DynamicGlobalPropsRW{db: e.db}
//	return dgpWrap.GetProps(), nil
//}

func (e *Economist) GetAccount(account *prototype.AccountName) (*table.SoAccountWrap, error) {
	accountWrap := table.NewSoAccountWrap(e.db, account)
	if !accountWrap.CheckExist() {
		return nil, errors.New(fmt.Sprintf("cannot find account %s", account.Value))
	}
	return accountWrap, nil
}

//func (e *Economist) modifyGlobalDynamicData(f func(props *prototype.DynamicProperties)) {
//	dgpWrap := DynamicGlobalPropsRW{db: e.db}
//	dgpWrap.ModifyProps(f)
//}

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
	//return annalBudget / (86400 / 3 * 365)
	return annalBudget / (86400 / constants.BlockInterval * 365)
}

func (e *Economist) Mint(trxObserver iservices.ITrxObserver) {
	//blockCurrent := constants.PerBlockCurrent
	//t0 := time.Now()
	globalProps := e.dgp.GetProps()
	if !globalProps.GetWitnessBootCompleted() {
		return
	}
	ith := globalProps.GetIthYear()
	annualBudget := e.CalculateBudget(ith)
	// new year arrived
	if globalProps.GetAnnualBudget().Value != annualBudget {
		e.dgp.ModifyProps(func(props *prototype.DynamicProperties) {
			props.AnnualBudget.Value = annualBudget
			props.AnnualMinted.Value = 0
		})
		// reload props
		globalProps = e.dgp.GetProps()
	}
	blockCurrent := e.CalculatePerBlockBudget(annualBudget)
	// prevent deficit
	if globalProps.GetAnnualBudget().Value > globalProps.GetAnnualMinted().Value &&
		globalProps.GetAnnualBudget().Value <= (globalProps.GetAnnualMinted().Value + blockCurrent) {
		blockCurrent = globalProps.GetAnnualBudget().Value - globalProps.GetAnnualMinted().Value
		// time to update year
		e.dgp.ModifyProps(func(props *prototype.DynamicProperties) {
			props.IthYear = props.IthYear + 1
		})
	}

	if globalProps.GetAnnualBudget().Value <= globalProps.GetAnnualMinted().Value {
		blockCurrent = 0
		e.dgp.ModifyProps(func(props *prototype.DynamicProperties) {
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
	bpRewardVesting := &prototype.Vest{Value: bpReward}
	// add ticket fee to the bp
	oldVest := bpWrap.GetVestingShares()
	//bpWrap.MdVestingShares(&prototype.Vest{Value: bpWrap.GetVestingShares().Value + bpReward})
	mustNoError(bpRewardVesting.Add(bpWrap.GetVestingShares()), "bpRewardVesting overflow")
	bpWrap.MdVestingShares(bpRewardVesting)
	updateWitnessVoteCount(e.db, globalProps.CurrentWitness, oldVest, bpRewardVesting)
	trxObserver.AddOpState(iservices.Add, "mint", globalProps.CurrentWitness.Value, bpReward)

	e.dgp.ModifyProps(func(props *prototype.DynamicProperties) {
		mustNoError(props.PostRewards.Add(&prototype.Vest{Value: postReward}), "PostRewards overflow")
		mustNoError(props.ReplyRewards.Add(&prototype.Vest{Value: replyReward}), "ReplyRewards overflow")
		mustNoError(props.PostDappRewards.Add(&prototype.Vest{Value: postDappRewards}), "PostDappRewards overflow")
		mustNoError(props.ReplyDappRewards.Add(&prototype.Vest{Value: replyDappRewards}), "ReplyDappRewards overflow")
		mustNoError(props.VoterRewards.Add(&prototype.Vest{Value: voterReward}), "VoterRewards overflow")
		mustNoError(props.AnnualMinted.Add(&prototype.Vest{Value: blockCurrent}), "AnnualMinted overflow")
		mustNoError(props.TotalVestingShares.Add(&prototype.Vest{Value: blockCurrent}), "TotalVestingShares overflow")
	})
}

// maybe slow
func (e *Economist) Distribute(trxObserver iservices.ITrxObserver) {
	globalProps := e.dgp.GetProps()
	if globalProps.GetCurrentEpochStartBlock() == uint64(0) {
		return
	}
	current := globalProps.HeadBlockNumber
	if globalProps.GetCurrentEpochStartBlock() + globalProps.GetEpochDuration() > current {
		return
	}
	iterator := table.NewAccountVestingSharesWrap(e.db)
	var accounts  []*prototype.AccountName
	var count uint32 = 0
	topN := globalProps.GetTopNAcquireFreeToken()
	err := iterator.ForEachByRevOrder(nil, nil, nil, nil, func(account *prototype.AccountName, sVal *prototype.Vest, idx uint32) bool {
		if count > topN {
			return false
		}
		accounts = append(accounts, account)
		count += 1
		return true
	})
	if err != nil {
		panic("economist distribute failed when iterator")
	}
	e.log.Info("economist epoch start block:", globalProps.GetCurrentEpochStartBlock())
	for _, account := range accounts {
		// type 0 free ticket
		key := &prototype.GiftTicketKeyType{Type: 0, From: []byte("contentos"), To: []byte(account.Value),
			CreateBlock: current}
		wrap := table.NewSoGiftTicketWrap(e.db, key)
		// impossible
		if wrap.CheckExist() {
			wrap.MdExpireBlock(current + globalProps.GetEpochDuration())
		} else {
			err = wrap.Create(func(tInfo *table.SoGiftTicket) {
				tInfo.Ticket = key
				tInfo.Denom = globalProps.PerTicketWeight
				tInfo.Count = 1
				tInfo.ExpireBlock = current + globalProps.GetEpochDuration()
			})
		}
	}
	e.dgp.ModifyProps(func(props *prototype.DynamicProperties) {
		props.CurrentEpochStartBlock = current
	})
}

// Should be claiming or direct modify the balance?
func (e *Economist) Do(trxObserver iservices.ITrxObserver) {
	globalProps := e.dgp.GetProps()
	if !globalProps.GetWitnessBootCompleted() {
		return
	}
	e.decayGlobalVotePower()
	iterator := table.NewPostCashoutBlockNumWrap(e.db)
	var pids []*uint64
	end := globalProps.HeadBlockNumber
	//postWeightedVps := globalProps.PostWeightedVps
	//replyWeightedVps := globalProps.ReplyWeightedVps
	t0 := time.Now()
	err := iterator.ForEachByOrder(nil, &end, nil, nil, func(mVal *uint64, sVal *uint64, idx uint32) bool {
		pids = append(pids, mVal)
		return true
	})
	e.log.Debugf("Do iterator spent: %v", time.Now().Sub(t0))
	if err != nil {
		panic("economist do failed when iterator")
	}
	var posts []*table.SoPostWrap
	var replies []*table.SoPostWrap

	//var postVpAccumulator uint64 = 0
	//var replyVpAccumulator uint64 = 0
	var postVpAccumulator, replyVpAccumulator big.Int
	// posts accumulate by linear, replies by sqrt
	for _, pid := range pids {
		post := table.NewSoPostWrap(e.db, pid)
		giftNum := new(big.Int).SetUint64(uint64(post.GetTicket()))
		giftVp := new(big.Int).Mul(giftNum, new(big.Int).SetUint64(globalProps.GetPerTicketWeight()))
		weightedVp := new(big.Int).Add(ISqrt(post.GetWeightedVp()), giftVp)
		if post.GetParentId() == 0 {
			posts = append(posts, post)
			postVpAccumulator.Add(&postVpAccumulator, weightedVp)
		} else {
			replies = append(replies, post)
			replyVpAccumulator.Add(&replyVpAccumulator, weightedVp)
		}
	}
	var globalPostWeightedVps, globalReplyWeightedVps, postWeightedVps, replyWeightedVps big.Int
	globalPostWeightedVps.SetString(globalProps.PostWeightedVps, 10)
	globalReplyWeightedVps.SetString(globalProps.ReplyWeightedVps, 10)
	postWeightedVps.Add(&globalPostWeightedVps, &postVpAccumulator)
	replyWeightedVps.Add(&globalReplyWeightedVps, &replyVpAccumulator)

	e.dgp.ModifyProps(func(props *prototype.DynamicProperties) {
		props.PostWeightedVps = postWeightedVps.String()
		props.ReplyWeightedVps = replyWeightedVps.String()
	})

	if postWeightedVps.Cmp(new(big.Int).SetInt64(0)) >= 0 {
		var rewards, dappRewards uint64
		//if postWeightedVps + postVpAccumulator == 0 {
		if postWeightedVps.Cmp(new(big.Int).SetInt64(0)) == 0 {
			rewards = 0
			dappRewards = 0
		}else {
			bigGlobalPostRewards := new(big.Int).SetUint64(globalProps.PostRewards.Value)
			bigVpMul := new(big.Int).Mul(&postVpAccumulator, bigGlobalPostRewards)
			rewards = new(big.Int).Div(bigVpMul, &postWeightedVps).Uint64()
			bigGlobalPostDappRewards := new(big.Int).SetUint64(globalProps.PostDappRewards.Value)
			bigDappVpMul := new(big.Int).Mul(&postVpAccumulator, bigGlobalPostDappRewards)
			dappRewards = new(big.Int).Div(bigDappVpMul, &postWeightedVps).Uint64()
		}

		e.log.Debugf("cashout posts length: %d", len(posts))
		if len(posts) > 0 {
			t := time.Now()
			e.postCashout(posts, rewards, dappRewards, trxObserver)
			e.log.Debugf("cashout posts spend: %v", time.Now().Sub(t))
		}
	}

	if replyWeightedVps.Cmp(new(big.Int).SetInt64(0)) >= 0 {
		var rewards, dappRewards uint64
		if replyWeightedVps.Cmp(new(big.Int).SetInt64(0)) == 0 {
			rewards = 0
			dappRewards = 0
		}else {
			bigGlobalReplyRewards := new(big.Int).SetUint64(globalProps.ReplyRewards.Value)
			bigVpMul := new(big.Int).Mul(&replyVpAccumulator, bigGlobalReplyRewards)
			rewards = new(big.Int).Div(bigVpMul, &replyWeightedVps).Uint64()
			//rewards = postVpAccumulator * globalProps.PostRewards.Value / (postWeightedVps + postVpAccumulator)
			bigGlobalReplyDappRewards := new(big.Int).SetUint64(globalProps.ReplyDappRewards.Value)
			bigDappVpMul := new(big.Int).Mul(&replyVpAccumulator, bigGlobalReplyDappRewards)
			dappRewards = new(big.Int).Div(bigDappVpMul, &replyWeightedVps).Uint64()
		}

		e.log.Debugf("cashout replies length: %d", len(replies))
		if len(replies) > 0 {
			t := time.Now()
			e.replyCashout(replies, rewards, dappRewards, trxObserver)
			e.log.Debugf("cashout reply spend: %v", time.Now().Sub(t))
		}
	}
}

func (e *Economist) decayGlobalVotePower() {
	e.dgp.ModifyProps(func(props *prototype.DynamicProperties) {
		var postWeightedVps, replyWeightedVps big.Int
		postWeightedVps.SetString(props.PostWeightedVps, 10)
		replyWeightedVps.SetString(props.ReplyWeightedVps, 10)
		var postWeightedDecay big.Int
		postWeightedDecay.Mul(&postWeightedVps, new(big.Int).SetUint64(constants.BlockInterval))
		postWeightedDecay.Div(&postWeightedDecay, new(big.Int).SetUint64(variables.VpDecayTime()))
		postWeightedVps.Sub(&postWeightedVps, &postWeightedDecay)
		//props.PostWeightedVps -= props.PostWeightedVps * constants.BlockInterval / variables.VpDecayTime()
		var replyWeightedDecay big.Int
		replyWeightedDecay.Mul(&replyWeightedVps, new(big.Int).SetUint64(constants.BlockInterval))
		replyWeightedDecay.Div(&replyWeightedDecay, new(big.Int).SetUint64(variables.VpDecayTime()))
		replyWeightedVps.Sub(&replyWeightedVps, &replyWeightedDecay)
		props.PostWeightedVps = postWeightedVps.String()
		props.ReplyWeightedVps = replyWeightedVps.String()
		//props.ReplyWeightedVps -= props.ReplyWeightedVps * constants.BlockInterval / variables.VpDecayTime()
	})
}

func (e *Economist) postCashout(posts []*table.SoPostWrap, blockReward uint64, blockDappReward uint64, trxObserver iservices.ITrxObserver) {
	globalProps := e.dgp.GetProps()

	//var vpAccumulator uint64 = 0
	t0 := time.Now()
	var vpAccumulator big.Int
	for _, post := range posts {
		//vp, _ := new(big.Int).SetString(post.GetWeightedVp(), 10)
		//vpAccumulator.Add(&vpAccumulator, vp)
		//vpAccumulator += post.GetWeightedVp()
		giftNum := new(big.Int).SetUint64(uint64(post.GetTicket()))
		giftVp := new(big.Int).Mul(giftNum, new(big.Int).SetUint64(globalProps.GetPerTicketWeight()))
		weightedVp := new(big.Int).Add(ISqrt(post.GetWeightedVp()), giftVp)
		vpAccumulator.Add(&vpAccumulator, weightedVp)
	}
	e.log.Debugf("cashout post weight cashout spend: %v", time.Now().Sub(t0))
	bigBlockRewards := new(big.Int).SetUint64(blockReward)
	bigBlockDappReward := new(big.Int).SetUint64(blockDappReward)
//	e.log.Debugf("current block post total vp:%d, global vp:%d", vpAccumulator, globalProps.PostWeightedVps)
	var spentPostReward uint64 = 0
	var spentDappReward uint64 = 0
	//var spentVoterReward uint64 = 0
	for _, post := range posts {
		tPost := time.Now()
		author := post.GetAuthor().Value
		var reward uint64 = 0
		var beneficiaryReward uint64 = 0
		// divide zero exception
		if vpAccumulator.Cmp(new(big.Int).SetInt64(0)) > 0 {
			//bigVpAccumulator := new(big.Int).SetUint64(vpAccumulator)
			//reward = post.GetWeightedVp() * blockReward / vpAccumulator
			//beneficiaryReward = post.GetWeightedVp() * blockDappReward / vpAccumulator
			//spentPostReward += reward
			//spentDappReward += beneficiaryReward
			//weightedVp := post.GetWeightedVp()
			//bigWeightedVp, _ := new(big.Int).SetString(weightedVp, 10)
			// perticketprice * num
			giftNum := new(big.Int).SetUint64(uint64(post.GetTicket()))
			giftVp := new(big.Int).Mul(giftNum, new(big.Int).SetUint64(globalProps.GetPerTicketWeight()))
			bigWeightedVp := new(big.Int).Add(ISqrt(post.GetWeightedVp()), giftVp)
			bigRewardMul := new(big.Int).Mul(bigWeightedVp,  bigBlockRewards)
			reward = new(big.Int).Div(bigRewardMul, &vpAccumulator).Uint64()
			bigDappRewardMul := new(big.Int).Mul(bigWeightedVp, bigBlockDappReward)
			beneficiaryReward = new(big.Int).Div(bigDappRewardMul, &vpAccumulator).Uint64()
			spentPostReward += reward
			spentDappReward += beneficiaryReward
		}
		tPostWeight := time.Now()
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
				oldVest := beneficiaryWrap.GetVestingShares()
				vestingRewards := &prototype.Vest{Value: r}
				mustNoError(vestingRewards.Add(beneficiaryWrap.GetVestingShares()), "Post Beneficiary VestingRewards Overflow")
				beneficiaryWrap.MdVestingShares(vestingRewards)
				updateWitnessVoteCount(e.db, &prototype.AccountName{Value: name}, oldVest, vestingRewards)
				spentBeneficiaryReward += r
				e.noticer.Publish(constants.NoticeCashout, name, post.GetPostId(), r, globalProps.GetHeadBlockNumber())
				trxObserver.AddOpState(iservices.Add, "cashout", name , r)
			}
		}
		tBeneficiary := time.Now()
		if beneficiaryReward - spentBeneficiaryReward > 0 {
			reward += beneficiaryReward - spentBeneficiaryReward
		}
		authorWrap, err := e.GetAccount(&prototype.AccountName{Value: author})
		if err != nil {
			e.log.Debugf("post cashout get account %s failed", author)
			continue
		} else {
			oldVest := authorWrap.GetVestingShares()
			vestingRewards := &prototype.Vest{Value: reward}
			mustNoError(vestingRewards.Add(authorWrap.GetVestingShares()), "Post VestingRewards Overflow")
			authorWrap.MdVestingShares(vestingRewards)
			t := time.Now()
			t1, t2 := updateWitnessVoteCount(e.db, &prototype.AccountName{Value: author}, oldVest, vestingRewards)
			e.log.Debugf("post cashout updateWitnessVoteCount: %v, foreach: %v, get: %v", time.Now().Sub(t), t1, t2)
		}
		post.MdCashoutBlockNum(math.MaxUint32)
		post.MdRewards(&prototype.Vest{Value: reward})
		post.MdDappRewards(&prototype.Vest{Value: beneficiaryReward})
		if reward > 0 {
			e.noticer.Publish(constants.NoticeCashout, author, post.GetPostId(), reward, globalProps.GetHeadBlockNumber())
			trxObserver.AddOpState(iservices.Add, "cashout", author, reward)
		}
		tCashout := time.Now()
		e.log.Debugf("cashout postWeight: %v, beneficiary: %v, postCashout: %v", tPostWeight.Sub(tPost),
			tBeneficiary.Sub(tPostWeight), tCashout.Sub(tBeneficiary))
	}
	e.dgp.ModifyProps(func(props *prototype.DynamicProperties) {
		//props.PostRewards.Value -= spentPostReward
		//props.PostDappRewards.Value -= spentDappReward
		mustNoError(props.PostRewards.Sub(&prototype.Vest{Value: spentPostReward}), "Sub SpentPostReward overflow")
		mustNoError(props.PostDappRewards.Sub(&prototype.Vest{Value: spentDappReward}), "Sub SpentDappReward overflow")
	})
}

// use same algorithm to simplify
func (e *Economist) replyCashout(replies []*table.SoPostWrap, blockReward uint64, blockDappReward uint64, trxObserver iservices.ITrxObserver) {
	globalProps := e.dgp.GetProps()
	//var vpAccumulator uint64 = 0
	var vpAccumulator big.Int
	for _, reply := range replies {
		//vpAccumulator += ISqrt(reply.GetWeightedVp())
		//vpAccumulator += reply.GetWeightedVp()
		//vp, _ := new(big.Int).SetString(reply.GetWeightedVp(), 10)
		//vpAccumulator.Add(&vpAccumulator, vp)
		giftNum := new(big.Int).SetUint64(uint64(reply.GetTicket()))
		giftVp := new(big.Int).Mul(giftNum, new(big.Int).SetUint64(globalProps.GetPerTicketWeight()))
		weightedVp := new(big.Int).Add(ISqrt(reply.GetWeightedVp()), giftVp)
		vpAccumulator.Add(&vpAccumulator, weightedVp)
	}
	bigBlockRewards := new(big.Int).SetUint64(blockReward)
	bigBlockDappReward := new(big.Int).SetUint64(blockDappReward)
//	e.log.Debugf("current block reply total vp:%d, global vp:%d", vpAccumulator, globalProps.ReplyWeightedVps)
	var spentReplyReward uint64 = 0
	var spentDappReward uint64 = 0
	//var spentVoterReward uint64 = 0
	for _, reply := range replies {
		author := reply.GetAuthor().Value
		var reward uint64 = 0
		var beneficiaryReward uint64 = 0
		//var voterReward uint64 = 0
		// divide zero exception
		if vpAccumulator.Cmp(new(big.Int).SetInt64(0)) > 0 {
			//bigVpAccumulator := new(big.Int).SetUint64(vpAccumulator)
			//weightedVp := ISqrt(reply.GetWeightedVp())
			//weightedVp := reply.GetWeightedVp()
			//bigWeightedVp, _ := new(big.Int).SetString(weightedVp, 10)
			giftNum := new(big.Int).SetUint64(uint64(reply.GetTicket()))
			giftVp := new(big.Int).Mul(giftNum, new(big.Int).SetUint64(globalProps.GetPerTicketWeight()))
			bigWeightedVp := new(big.Int).Add(ISqrt(reply.GetWeightedVp()), giftVp)
			bigRewardMul := new(big.Int).Mul(bigWeightedVp,  bigBlockRewards)
			reward = new(big.Int).Div(bigRewardMul, &vpAccumulator).Uint64()
			bigDappRewardMul := new(big.Int).Mul(bigWeightedVp, bigBlockDappReward)
			beneficiaryReward = new(big.Int).Div(bigDappRewardMul, &vpAccumulator).Uint64()
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
				//beneficiaryWrap.MdVestingShares(&prototype.Vest{ Value: r + beneficiaryWrap.GetVestingShares().Value})
				oldVest := beneficiaryWrap.GetVestingShares()
				vestingRewards := &prototype.Vest{Value: r}
				mustNoError(vestingRewards.Add(beneficiaryWrap.GetVestingShares()), "Reply Beneficiary VestingRewards Overflow")
				beneficiaryWrap.MdVestingShares(vestingRewards)
				updateWitnessVoteCount(e.db, &prototype.AccountName{Value: name}, oldVest, vestingRewards)
				spentBeneficiaryReward += r
				e.noticer.Publish(constants.NoticeCashout, name, reply.GetPostId(), r, globalProps.GetHeadBlockNumber())
				trxObserver.AddOpState(iservices.Add, "cashout", name, r)
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
			//authorWrap.MdVestingShares(&prototype.Vest{ Value: reward + authorWrap.GetVestingShares().Value })
			oldVest := authorWrap.GetVestingShares()
			vestingRewards := &prototype.Vest{Value: reward}
			mustNoError(vestingRewards.Add(authorWrap.GetVestingShares()), "Reply VestingRewards Overflow")
			authorWrap.MdVestingShares(vestingRewards)
			t := time.Now()
			t1, t2 := updateWitnessVoteCount(e.db, &prototype.AccountName{Value: author}, oldVest, vestingRewards)
			e.log.Debugf("reply cashout updateWitnessVoteCount: %v, foreach: %v, get: %v", time.Now().Sub(t), t1, t2)
		}
		reply.MdCashoutBlockNum(math.MaxUint32)
		reply.MdRewards(&prototype.Vest{Value: reward})
		reply.MdDappRewards(&prototype.Vest{Value: beneficiaryReward})
		if reward > 0 {
			e.noticer.Publish(constants.NoticeCashout, author, reply.GetPostId(), reward, globalProps.GetHeadBlockNumber())
			trxObserver.AddOpState(iservices.Add, "cashout", author, reward)
		}
	}
	e.log.Infof("cashout: [reply] blockRewards: %d, blockDappRewards: %d, spendPostReward: %d, spendDappReward: %d",
		blockReward, blockDappReward, spentReplyReward, spentDappReward)
	e.dgp.ModifyProps(func(props *prototype.DynamicProperties) {
		//props.ReplyRewards.Value -= spentReplyReward
		//props.ReplyDappRewards.Value -= spentDappReward
		mustNoError(props.ReplyRewards.Sub(&prototype.Vest{Value: spentReplyReward}), "Sub SpentReplyReward overflow")
		mustNoError(props.ReplyDappRewards.Sub(&prototype.Vest{Value: spentDappReward}), "Sub SpentDappReward overflow")
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
	globalProps := e.dgp.GetProps()
	if !globalProps.GetWitnessBootCompleted() {
		return
	}
	//timestamp := globalProps.Time.UtcSeconds
	//iterator := table.NewAccountNextPowerdownTimeWrap(e.db)
	iterator := table.NewAccountNextPowerdownBlockNumWrap(e.db)
	var accountNames []*prototype.AccountName

	timing := common.NewTiming()
	timing.Begin()

	current := globalProps.HeadBlockNumber
	err := iterator.ForEachByOrder(nil, &current, nil, nil, func(mVal *prototype.AccountName, sVal *uint64, idx uint32) bool {
		accountNames = append(accountNames, mVal)
		return true
	})
	if err != nil {
		panic("economist powerdown failed when iterator")
	}
	timing.Mark()

	var powerdownQuota uint64 = 0
	for _, accountName := range accountNames {
		accountWrap := table.NewSoAccountWrap(e.db, accountName)
		if accountWrap.GetToPowerdown().Value-accountWrap.GetHasPowerdown().Value < accountWrap.GetEachPowerdownRate().Value {
			powerdownQuota = Min(accountWrap.GetVestingShares().Value, accountWrap.GetToPowerdown().Value-accountWrap.GetHasPowerdown().Value)
		} else {
			powerdownQuota = Min(accountWrap.GetVestingShares().Value, accountWrap.GetEachPowerdownRate().Value)
		}
		oldVest := accountWrap.GetVestingShares()
		vesting := accountWrap.GetVestingShares().Value - powerdownQuota
		balance := accountWrap.GetBalance().Value + powerdownQuota
		hasPowerDown := accountWrap.GetHasPowerdown().Value + powerdownQuota
		accountWrap.MdVestingShares(&prototype.Vest{Value: vesting})
		newVest := accountWrap.GetVestingShares()
		updateWitnessVoteCount(e.db, accountName, oldVest, newVest)
		accountWrap.MdBalance(&prototype.Coin{Value: balance})
		accountWrap.MdHasPowerdown(&prototype.Vest{Value: hasPowerDown})
		// update total cos and total vesting shares
		e.dgp.ModifyProps(func(props *prototype.DynamicProperties) {
			mustNoError(props.TotalCos.Add(&prototype.Coin{Value: powerdownQuota}), "PowerDownQuota Cos Overflow")
			mustNoError(props.TotalVestingShares.Sub(&prototype.Vest{Value: powerdownQuota}), "PowerDownQuota Vest Overflow")
			//props.TotalCos.Value += powerdownQuota
			//props.TotalVestingShares.Value -= powerdownQuota
		})
		if accountWrap.GetHasPowerdown().Value >= accountWrap.GetToPowerdown().Value || accountWrap.GetVestingShares().Value == 0 {
			accountWrap.MdEachPowerdownRate(&prototype.Vest{Value: 0})
			accountWrap.MdNextPowerdownBlockNum(math.MaxUint32)
		} else {
			accountWrap.MdNextPowerdownBlockNum(current + constants.PowerDownBlockInterval)
		}
	}
	timing.End()
	e.log.Debugf("powerdown: %s", timing.String())
}
