package app

import (
	"crypto/sha256"
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/common/encoding/vme"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/vm"
	"github.com/coschain/contentos-go/vm/context"
	"github.com/coschain/contentos-go/vm/contract/abi"
	ct "github.com/coschain/contentos-go/vm/contract/table"
	"github.com/go-interpreter/wagon/exec"
	"math"
	"math/big"
	"sort"
	"strconv"
	"time"
)

func mustSuccess(b bool, val string) {
	if !b {
		panic(val)
	}
}

type AccountCreateEvaluator struct {
	BaseEvaluator
	BaseDelegate
	op  *prototype.AccountCreateOperation
}

type AccountUpdateEvaluator struct {
	BaseEvaluator
	BaseDelegate
	op  *prototype.AccountUpdateOperation
}

type TransferEvaluator struct {
	BaseEvaluator
	BaseDelegate
	op  *prototype.TransferOperation
}

type PostEvaluator struct {
	BaseEvaluator
	BaseDelegate
	op  *prototype.PostOperation
}
type ReplyEvaluator struct {
	BaseEvaluator
	BaseDelegate
	op  *prototype.ReplyOperation
}
type VoteEvaluator struct {
	BaseEvaluator
	BaseDelegate
	op  *prototype.VoteOperation
}
type BpRegisterEvaluator struct {
	BaseEvaluator
	BaseDelegate
	op  *prototype.BpRegisterOperation
}
type BpEnableEvaluator struct {
	BaseEvaluator
	BaseDelegate
	op  *prototype.BpEnableOperation
}

type BpUpdateEvaluator struct {
	BaseEvaluator
	BaseDelegate
	op  *prototype.BpUpdateOperation
}

type BpVoteEvaluator struct {
	BaseEvaluator
	BaseDelegate
	op  *prototype.BpVoteOperation
}

type FollowEvaluator struct {
	BaseEvaluator
	BaseDelegate
	op  *prototype.FollowOperation
}

type TransferToVestEvaluator struct {
	BaseEvaluator
	BaseDelegate
	op  *prototype.TransferToVestOperation
}

//type ClaimEvaluator struct {
//	BaseEvaluator
//	BaseDelegate
//	op  *prototype.ClaimOperation
//}

type ReportEvaluator struct {
	BaseEvaluator
	BaseDelegate
	op  *prototype.ReportOperation
}

type ConvertVestEvaluator struct {
	BaseEvaluator
	BaseDelegate
	op  *prototype.ConvertVestOperation
}

// I can cat out this awkward claimall operation until I can get value from rpc resp
//type ClaimAllEvaluator struct {
//	BaseEvaluator
//	BaseDelegate
//	op  *prototype.ClaimAllOperation
//}

type ContractDeployEvaluator struct {
	BaseEvaluator
	BaseDelegate
	op  *prototype.ContractDeployOperation
}

type ContractApplyEvaluator struct {
	BaseEvaluator
	BaseDelegate
	op  *prototype.ContractApplyOperation
}

type InternalContractApplyEvaluator struct {
	BaseEvaluator
	BaseDelegate
	op  *prototype.InternalContractApplyOperation
	remainGas uint64
	preVm *exec.VM
}

type StakeEvaluator struct {
	BaseEvaluator
	BaseDelegate
	op  *prototype.StakeOperation
}

type UnStakeEvaluator struct {
	BaseEvaluator
	BaseDelegate
	op  *prototype.UnStakeOperation
}

type AcquireTicketEvaluator struct {
	BaseEvaluator
	BaseDelegate
	op *prototype.AcquireTicketOperation
}

type VoteByTicketEvaluator struct {
	BaseEvaluator
	BaseDelegate
	op *prototype.VoteByTicketOperation
}

func init() {
	RegisterEvaluator((*prototype.AccountCreateOperation)(nil), func(delegate ApplyDelegate, op prototype.BaseOperation) BaseEvaluator {
		return &AccountCreateEvaluator {BaseDelegate: BaseDelegate{delegate:delegate}, op: op.(*prototype.AccountCreateOperation)}
	})
	RegisterEvaluator((*prototype.TransferOperation)(nil), func(delegate ApplyDelegate, op prototype.BaseOperation) BaseEvaluator {
		return &TransferEvaluator {BaseDelegate: BaseDelegate{delegate:delegate}, op: op.(*prototype.TransferOperation)}
	})
	RegisterEvaluator((*prototype.BpRegisterOperation)(nil), func(delegate ApplyDelegate, op prototype.BaseOperation) BaseEvaluator {
		return &BpRegisterEvaluator {BaseDelegate: BaseDelegate{delegate:delegate}, op: op.(*prototype.BpRegisterOperation)}
	})
	RegisterEvaluator((*prototype.BpEnableOperation)(nil), func(delegate ApplyDelegate, op prototype.BaseOperation) BaseEvaluator {
		return &BpEnableEvaluator {BaseDelegate: BaseDelegate{delegate:delegate}, op: op.(*prototype.BpEnableOperation)}
	})
	RegisterEvaluator((*prototype.BpVoteOperation)(nil), func(delegate ApplyDelegate, op prototype.BaseOperation) BaseEvaluator {
		return &BpVoteEvaluator {BaseDelegate: BaseDelegate{delegate:delegate}, op: op.(*prototype.BpVoteOperation)}
	})
	RegisterEvaluator((*prototype.PostOperation)(nil), func(delegate ApplyDelegate, op prototype.BaseOperation) BaseEvaluator {
		return &PostEvaluator {BaseDelegate: BaseDelegate{delegate:delegate}, op: op.(*prototype.PostOperation)}
	})
	RegisterEvaluator((*prototype.ReplyOperation)(nil), func(delegate ApplyDelegate, op prototype.BaseOperation) BaseEvaluator {
		return &ReplyEvaluator {BaseDelegate: BaseDelegate{delegate:delegate}, op: op.(*prototype.ReplyOperation)}
	})
	RegisterEvaluator((*prototype.FollowOperation)(nil), func(delegate ApplyDelegate, op prototype.BaseOperation) BaseEvaluator {
		return &FollowEvaluator {BaseDelegate: BaseDelegate{delegate:delegate}, op: op.(*prototype.FollowOperation)}
	})
	RegisterEvaluator((*prototype.VoteOperation)(nil), func(delegate ApplyDelegate, op prototype.BaseOperation) BaseEvaluator {
		return &VoteEvaluator {BaseDelegate: BaseDelegate{delegate:delegate}, op: op.(*prototype.VoteOperation)}
	})
	RegisterEvaluator((*prototype.TransferToVestOperation)(nil), func(delegate ApplyDelegate, op prototype.BaseOperation) BaseEvaluator {
		return &TransferToVestEvaluator {BaseDelegate: BaseDelegate{delegate:delegate}, op: op.(*prototype.TransferToVestOperation)}
	})
	RegisterEvaluator((*prototype.ContractDeployOperation)(nil), func(delegate ApplyDelegate, op prototype.BaseOperation) BaseEvaluator {
		return &ContractDeployEvaluator {BaseDelegate: BaseDelegate{delegate:delegate}, op: op.(*prototype.ContractDeployOperation)}
	})
	RegisterEvaluator((*prototype.ContractApplyOperation)(nil), func(delegate ApplyDelegate, op prototype.BaseOperation) BaseEvaluator {
		return &ContractApplyEvaluator {BaseDelegate: BaseDelegate{delegate:delegate}, op: op.(*prototype.ContractApplyOperation)}
	})
	RegisterEvaluator((*prototype.ReportOperation)(nil), func(delegate ApplyDelegate, op prototype.BaseOperation) BaseEvaluator {
		return &ReportEvaluator {BaseDelegate: BaseDelegate{delegate:delegate}, op: op.(*prototype.ReportOperation)}
	})
	RegisterEvaluator((*prototype.ConvertVestOperation)(nil), func(delegate ApplyDelegate, op prototype.BaseOperation) BaseEvaluator {
		return &ConvertVestEvaluator {BaseDelegate: BaseDelegate{delegate:delegate}, op: op.(*prototype.ConvertVestOperation)}
	})
	RegisterEvaluator((*prototype.StakeOperation)(nil), func(delegate ApplyDelegate, op prototype.BaseOperation) BaseEvaluator {
		return &StakeEvaluator {BaseDelegate: BaseDelegate{delegate:delegate}, op: op.(*prototype.StakeOperation)}
	})
	RegisterEvaluator((*prototype.UnStakeOperation)(nil), func(delegate ApplyDelegate, op prototype.BaseOperation) BaseEvaluator {
		return &UnStakeEvaluator {BaseDelegate: BaseDelegate{delegate:delegate}, op: op.(*prototype.UnStakeOperation)}
	})
	RegisterEvaluator((*prototype.BpUpdateOperation)(nil), func(delegate ApplyDelegate, op prototype.BaseOperation) BaseEvaluator {
		return &BpUpdateEvaluator {BaseDelegate: BaseDelegate{delegate:delegate}, op: op.(*prototype.BpUpdateOperation)}
	})
	RegisterEvaluator((*prototype.AccountUpdateOperation)(nil), func(delegate ApplyDelegate, op prototype.BaseOperation) BaseEvaluator {
		return &AccountUpdateEvaluator {BaseDelegate: BaseDelegate{delegate:delegate}, op: op.(*prototype.AccountUpdateOperation)}
	})
	RegisterEvaluator((*prototype.AcquireTicketOperation)(nil), func(delegate ApplyDelegate, op prototype.BaseOperation) BaseEvaluator {
		return &AcquireTicketEvaluator {BaseDelegate: BaseDelegate{delegate:delegate}, op: op.(*prototype.AcquireTicketOperation)}
	})
	RegisterEvaluator((*prototype.VoteByTicketOperation)(nil), func(delegate ApplyDelegate, op prototype.BaseOperation) BaseEvaluator {
		return &VoteByTicketEvaluator {BaseDelegate: BaseDelegate{delegate:delegate}, op: op.(*prototype.VoteByTicketOperation)}
	})
}

func (ev *AccountCreateEvaluator) Apply() {
	op := ev.op
	ev.VMInjector().RecordStaminaFee(op.Creator.Value, constants.CommonOpStamina)
	creatorWrap := table.NewSoAccountWrap(ev.Database(), op.Creator)

	creatorWrap.MustExist("creator not exist")

	dgpWrap := table.NewSoGlobalWrap(ev.Database(), &SingleId)
	globalFee := dgpWrap.GetProps().AccountCreateFee
	opAssert(op.Fee.Value >= globalFee.Value, fmt.Sprintf("Your fee is lower than global %d", globalFee.Value))

	accountCreateFee := op.Fee
	opAssert(creatorWrap.GetBalance().Value >= accountCreateFee.Value, "Insufficient balance to create account.")

	// sub creator's fee
	//originBalance := creatorWrap.GetBalance()
	//originBalance.Sub(accountCreateFee)
	//creatorWrap.SetBalance(originBalance)
	creatorWrap.Modify(func(tInfo *table.SoAccount) {
		tInfo.Balance.Sub(accountCreateFee)
	})

	// create account
	newAccountWrap := table.NewSoAccountWrap(ev.Database(), op.NewAccountName)
	newAccountWrap.Create(func(tInfo *table.SoAccount) {
		tInfo.Name = op.NewAccountName
		tInfo.Creator = op.Creator
		tInfo.CreatedTime = ev.GlobalProp().HeadBlockTime()
		tInfo.Balance = prototype.NewCoin(0)
		//tInfo.Vest = op.Fee.ToVest()
		tInfo.Vest = accountCreateFee.ToVest()
		tInfo.LastPostTime = ev.GlobalProp().HeadBlockTime()
		tInfo.LastVoteTime = ev.GlobalProp().HeadBlockTime()
		tInfo.NextPowerdownBlockNum = math.MaxUint64
		tInfo.EachPowerdownRate = &prototype.Vest{Value: 0}
		tInfo.ToPowerdown = &prototype.Vest{Value: 0}
		tInfo.HasPowerdown = &prototype.Vest{Value: 0}
		tInfo.PubKey = op.PubKey
		tInfo.StakeVest = prototype.NewVest(0)
		tInfo.Reputation = constants.DefaultReputation
		tInfo.ChargedTicket = 0
		tInfo.VotePower = 1000
	})

	// sub dynamic glaobal properties's total fee
	//ev.GlobalProp().TransferToVest(op.Fee)
	ev.GlobalProp().TransferToVest(accountCreateFee)
	ev.GlobalProp().ModifyProps(func(props *prototype.DynamicProperties) {
		props.TotalUserCnt++
	})
}

func (ev *AccountUpdateEvaluator) Apply() {
	op := ev.op
	ev.VMInjector().RecordStaminaFee(op.Owner.Value, constants.CommonOpStamina)

	updaterWrap := table.NewSoAccountWrap(ev.Database(), op.Owner)
	updaterWrap.MustExist("update account not exist ")

	updaterWrap.SetPubKey(op.PubKey)
}

func (ev *TransferEvaluator) Apply() {
	op := ev.op
	ev.VMInjector().RecordStaminaFee(op.From.Value, constants.CommonOpStamina)

	// @ active_challenged
	fromWrap := table.NewSoAccountWrap(ev.Database(), op.From)
	toWrap := table.NewSoAccountWrap(ev.Database(), op.To)

	toWrap.MustExist("To account do not exist ")

	opAssert(op.From.Value != op.To.Value, "Transfer must between two different accounts")

	//fBalance := fromWrap.GetBalance()
	//tBalance := toWrap.GetBalance()

	//fBalance.Sub(op.Amount)
	//fromWrap.SetBalance(fBalance)
	fromWrap.Modify(func(tInfo *table.SoAccount) {
		tInfo.Balance.Sub(op.Amount)
	})

	//tBalance.Add(op.Amount)
	//toWrap.SetBalance(tBalance)
	toWrap.Modify(func(tInfo *table.SoAccount) {
		tInfo.Balance.Add(op.Amount)
	})

	ev.TrxObserver().AddOpState(iservices.Replace, "balance", fromWrap.GetName().Value, fromWrap.GetBalance().Value)
	ev.TrxObserver().AddOpState(iservices.Replace, "balance", toWrap.GetName().Value, toWrap.GetBalance().Value)
}

func (ev *PostEvaluator) Apply() {
	op := ev.op
	ev.VMInjector().RecordStaminaFee(op.Owner.Value, constants.CommonOpStamina)

	idWrap := table.NewSoPostWrap(ev.Database(), &op.Uuid)
	idWrap.MustNotExist("post uuid exist")

	authorWrap := table.NewSoAccountWrap(ev.Database(), op.Owner)
	elapsedSeconds := ev.GlobalProp().HeadBlockTime().UtcSeconds - authorWrap.GetLastPostTime().UtcSeconds
	opAssert(elapsedSeconds > constants.MinPostInterval, "posting frequently")

	// default source is contentos
	idWrap.Create(func(t *table.SoPost) {
		t.PostId = op.Uuid
		t.Tags = op.Tags
		t.Title = op.Title
		t.Author = op.Owner
		t.Body = op.Content
		t.Created = ev.GlobalProp().HeadBlockTime()
		//t.CashoutTime = &prototype.TimePointSec{UtcSeconds: ev.GlobalProp().HeadBlockTime().UtcSeconds + uint32(constants.PostCashOutDelayTime)}
		//t.CashoutBlockNum = ev.GlobalProp().GetProps().HeadBlockNumber + constants.PostCashOutDelayBlock
		t.CashoutBlockNum = ev.GlobalProp().GetProps().HeadBlockNumber + constants.PostCashOutDelayBlock
		t.Depth = 0
		t.Children = 0
		t.ParentId = 0
		t.RootId = 0
		t.Beneficiaries = op.Beneficiaries
		t.WeightedVp = "0"
		t.VoteCnt = 0
		t.Rewards = &prototype.Vest{Value: 0}
		t.DappRewards = &prototype.Vest{Value: 0}
		t.Ticket = 0
		t.Copyright = constants.CopyrightUnkown
	})

	//authorWrap.SetLastPostTime(ev.GlobalProp().HeadBlockTime())

	authorWrap.Modify(func(tInfo *table.SoAccount) {
		tInfo.LastPostTime = ev.GlobalProp().HeadBlockTime()
	})

	ev.GlobalProp().ModifyProps(func(props *prototype.DynamicProperties) {
		props.TotalPostCnt++
	})

	//timestamp := ev.GlobalProp().HeadBlockTime().UtcSeconds + uint32(constants.PostCashOutDelayTime) - uint32(constants.GenesisTime)
	//key := fmt.Sprintf("cashout:%d_%d", common.GetBucket(timestamp), op.Uuid)
	//value := "post"
	//opAssertE(ev.Database().Put([]byte(key), []byte(value)), "put post key into db error")

}

func (ev *ReplyEvaluator) Apply() {
	op := ev.op
	ev.VMInjector().RecordStaminaFee(op.Owner.Value, constants.CommonOpStamina)

	cidWrap := table.NewSoPostWrap(ev.Database(), &op.Uuid)
	pidWrap := table.NewSoPostWrap(ev.Database(), &op.ParentUuid)

	cidWrap.MustNotExist("post uuid exist")
	pidWrap.MustExist("parent uuid do not exist")

	opAssert(pidWrap.GetDepth()+1 < constants.PostMaxDepth, "reply depth error")

	authorWrap := table.NewSoAccountWrap(ev.Database(), op.Owner)
	elapsedSeconds := ev.GlobalProp().HeadBlockTime().UtcSeconds - authorWrap.GetLastPostTime().UtcSeconds
	opAssert(elapsedSeconds > constants.MinPostInterval, "reply frequently")

	var rootId uint64
	if pidWrap.GetRootId() == 0 {
		rootId = pidWrap.GetPostId()
	} else {
		rootId = pidWrap.GetRootId()
	}

	cidWrap.Create(func(t *table.SoPost) {
		t.PostId = op.Uuid
		t.Tags = nil
		t.Title = ""
		t.Author = op.Owner
		t.Body = op.Content
		t.Created = ev.GlobalProp().HeadBlockTime()
		//t.CashoutTime = &prototype.TimePointSec{UtcSeconds: ev.GlobalProp().HeadBlockTime().UtcSeconds + uint32(constants.PostCashOutDelayTime)}
		t.CashoutBlockNum = ev.GlobalProp().GetProps().HeadBlockNumber + constants.PostCashOutDelayBlock
		t.Depth = pidWrap.GetDepth() + 1
		t.Children = 0
		t.RootId = rootId
		t.ParentId = op.ParentUuid
		t.WeightedVp = "0"
		t.VoteCnt = 0
		t.Beneficiaries = op.Beneficiaries
		t.Rewards = &prototype.Vest{Value: 0}
		t.DappRewards = &prototype.Vest{Value: 0}
		t.Ticket = 0
	})

	//authorWrap.SetLastPostTime(ev.GlobalProp().HeadBlockTime())

	authorWrap.Modify(func(tInfo *table.SoAccount) {
		tInfo.LastPostTime = ev.GlobalProp().HeadBlockTime()
	})

	// Modify Parent Object
	//opAssert(pidWrap.SetChildren(pidWrap.GetChildren()+1), "Modify Parent Children Error")

	pidWrap.Modify(func(tInfo *table.SoPost) {
		tInfo.Children ++
	})

	//timestamp := ev.GlobalProp().HeadBlockTime().UtcSeconds + uint32(constants.PostCashOutDelayTime) - uint32(constants.GenesisTime)
	//key := fmt.Sprintf("cashout:%d_%d", common.GetBucket(timestamp), op.Uuid)
	//value := "reply"
	//opAssertE(ev.Database().Put([]byte(key), []byte(value)), "put reply key into db error")
}

// upvote is true: upvote otherwise downvote
// no downvote has been supplied by command, so I ignore it
func (ev *VoteEvaluator) Apply() {
	op := ev.op
	ev.VMInjector().RecordStaminaFee(op.Voter.Value, constants.CommonOpStamina)

	voterWrap := table.NewSoAccountWrap(ev.Database(), op.Voter)
	elapsedSeconds := ev.GlobalProp().HeadBlockTime().UtcSeconds - voterWrap.GetLastVoteTime().UtcSeconds
	opAssert(elapsedSeconds > constants.MinVoteInterval, "voting frequently")

	voterId := prototype.VoterId{Voter: op.Voter, PostId: op.Idx}
	voteWrap := table.NewSoVoteWrap(ev.Database(), &voterId)
	postWrap := table.NewSoPostWrap(ev.Database(), &op.Idx)

	postWrap.MustExist("post invalid")
	voteWrap.MustNotExist("vote info exist")

	regeneratedPower := constants.FullVP * elapsedSeconds / constants.VoteRegenerateTime
	var currentVp uint32
	votePower := voterWrap.GetVotePower() + regeneratedPower
	if votePower > constants.FullVP {
		currentVp = constants.FullVP
	} else {
		currentVp = votePower
	}
	//usedVp := (currentVp + constants.VoteLimitDuringRegenerate - 1) / constants.VoteLimitDuringRegenerate
	var usedVp uint32
	usedVp = uint32(constants.FullVP / constants.VPMarks)
	if currentVp < usedVp {
		usedVp = 0
	}

	voterWrap.Modify(func(tInfo *table.SoAccount) {
		tInfo.VotePower = currentVp - usedVp
		tInfo.LastVoteTime = ev.GlobalProp().HeadBlockTime()
	})

	//voterWrap.SetVotePower(currentVp - usedVp)
	//voterWrap.SetLastVoteTime(ev.GlobalProp().HeadBlockTime())
	vest := voterWrap.GetVest().Value
	// after constants.PERCENT replaced by 1000, max value is 10000000000 * 1000000 * 1000 / 30
	// 10000000000 * 1000000 * 1000 < 18446744073709552046 but 10000000000 * 1000000 > 9223372036854775807
	// so can not using int64 here
	//weightedVp := vest * uint64(usedVp)
	weightedVp := new(big.Int).SetUint64(vest)
	weightedVp.Sqrt(weightedVp)
	weightedVp.Mul(weightedVp, new(big.Int).SetUint64(uint64(usedVp)))

	// if voter's reputation is 0, she has no voting power.
	if voterWrap.GetReputation() == constants.MinReputation {
		weightedVp.SetUint64(0)
	}

	if postWrap.GetCashoutBlockNum() > ev.GlobalProp().GetProps().HeadBlockNumber {
		lastVp := postWrap.GetWeightedVp()
		var lvp, tvp big.Int
		//wvp.SetUint64(weightedVp)
		lvp.SetString(lastVp, 10)
		tvp.Add(weightedVp, &lvp)

		postWrap.Modify(func(tInfo *table.SoPost) {
			tInfo.WeightedVp = tvp.String()
			tInfo.VoteCnt++
		})

		voteWrap.Create(func(t *table.SoVote) {
			t.Voter = &voterId
			t.PostId = op.Idx
			t.Upvote = true
			t.WeightedVp = weightedVp.String()
			t.VoteTime = ev.GlobalProp().HeadBlockTime()
		})

		// add vote into cashout table
		voteCashoutBlockHeight := ev.GlobalProp().GetProps().HeadBlockNumber + constants.VoteCashOutDelayBlock
		voteCashoutWrap := table.NewSoVoteCashoutWrap(ev.Database(), &voteCashoutBlockHeight)

		if voteCashoutWrap.CheckExist() {
			voterIds := voteCashoutWrap.GetVoterIds()
			voterIds = append(voterIds, &voterId)
			voteCashoutWrap.SetVoterIds(voterIds)
		} else {
			voteCashoutWrap.Create(func(tInfo *table.SoVoteCashout) {
				tInfo.CashoutBlock = voteCashoutBlockHeight
				tInfo.VoterIds = []*prototype.VoterId{&voterId}
			})
		}
	}
}

func (ev *BpRegisterEvaluator) Apply() {
	op := ev.op
	ev.VMInjector().RecordStaminaFee(op.Owner.Value, constants.CommonOpStamina)

	accountWrap := table.NewSoAccountWrap(ev.Database(), op.Owner)
	accountWrap.MustExist("block producer account not exist")

	accountBalance := accountWrap.GetVest()
	opAssert(accountBalance.Value >= constants.MinBpRegisterVest,
		fmt.Sprintf("vest balance should greater than %d", constants.MinBpRegisterVest / constants.COSTokenDecimals))

	bpWrap := table.NewSoBlockProducerWrap(ev.Database(), op.Owner)
	bpWrap.MustNotExist("you are already a block producer, do not register twice")

	opAssert(accountWrap.GetReputation() > constants.MinReputation,
		fmt.Sprintf("reputation too low"))

	staminaFree := op.Props.StaminaFree
	opAssert(staminaFree >= constants.MinStaminaFree,
		fmt.Sprintf("proposed stamina free too low min value %d", constants.MinStaminaFree))
	opAssert(staminaFree <= constants.MaxStaminaFree,
		fmt.Sprintf("proposed stamina free too high max value %d", constants.MaxStaminaFree))

	tpsExpected := op.Props.TpsExpected
	opAssert(tpsExpected >= constants.MinTPSExpected,
		fmt.Sprintf("expected tps too low min value %d", constants.MinTPSExpected))
	opAssert(tpsExpected <= constants.MaxTPSExpected,
		fmt.Sprintf("expected tps too high max value %d", constants.MaxTPSExpected))

	accountCreateFee := op.Props.AccountCreationFee
	opAssert(accountCreateFee.Value >= constants.MinAccountCreateFee,
		fmt.Sprintf("account create fee too low min value %d", constants.MinAccountCreateFee))
	opAssert(accountCreateFee.Value <= constants.MaxAccountCreateFee,
		fmt.Sprintf("account create fee too high max value %d", constants.MaxAccountCreateFee))

	topNAcquireFreeToken := op.Props.TopNAcquireFreeToken
	opAssert(topNAcquireFreeToken <= constants.MaxTopN, fmt.Sprintf("top N VEST holders, the N is too big, " +
		"which should lower than %d", constants.MaxTopN))

	epochDuration := op.Props.EpochDuration
	//opAssert(epochDuration >= constants.MinEpochDuration, fmt.Sprintf("the epoch duration should greater than %d",
	//	constants.MinEpochDuration))

	perTicketPrice := op.Props.PerTicketPrice
	opAssert(perTicketPrice.Value >= constants.MinTicketPrice, fmt.Sprintf("the ticket price should greater than %d",
		constants.MinTicketPrice))

	perTicketWeight := op.Props.PerTicketWeight
	bpVest := &prototype.BpVestId{Active:true, VoteVest:&prototype.Vest{Value: 0}}

	bpWrap.Create(func(t *table.SoBlockProducer) {
		t.Owner = op.Owner
		t.CreatedTime = ev.GlobalProp().HeadBlockTime()
		t.Url = op.Url
		t.SigningKey = op.BlockSigningKey
		t.BpVest = bpVest
		t.ProposedStaminaFree = staminaFree
		t.TpsExpected = tpsExpected
		t.AccountCreateFee = accountCreateFee
		t.TopNAcquireFreeToken = topNAcquireFreeToken
		t.EpochDuration = epochDuration
		t.PerTicketPrice = perTicketPrice
		t.PerTicketWeight = perTicketWeight
		t.VoterCount = 0
		// TODO add others
	})
}

func (ev *BpEnableEvaluator) Apply() {
	op := ev.op
	ev.VMInjector().RecordStaminaFee(op.Owner.Value, constants.CommonOpStamina)

	bpWrap := table.NewSoBlockProducerWrap(ev.Database(), op.Owner)
	bpWrap.MustExist("block producer do not exist")

	if op.Cancel {
		opAssert(bpWrap.GetBpVest().Active, "block producer has already been disabled")

		bpVoteVest := bpWrap.GetBpVest().VoteVest
		newBpVest := &prototype.BpVestId{Active:false, VoteVest:bpVoteVest}

		bpWrap.SetBpVest(newBpVest)
	} else {
		opAssert(!bpWrap.GetBpVest().Active, "block producer has already been enabled")

		bpVoteVest := bpWrap.GetBpVest().VoteVest
		newBpVest := &prototype.BpVestId{Active:true, VoteVest:bpVoteVest}

		bpWrap.SetBpVest(newBpVest)
	}
}

func (ev *BpVoteEvaluator) Apply() {
	op := ev.op
	ev.VMInjector().RecordStaminaFee(op.Voter.Value, constants.CommonOpStamina)

	voterAccount := table.NewSoAccountWrap(ev.Database(), op.Voter)
	voterAccount.MustExist("voter account not exist")
	voteCnt := voterAccount.GetBpVoteCount()
	voterVests := voterAccount.GetVest()

	bpAccountWrap := table.NewSoAccountWrap(ev.Database(), op.BlockProducer)
	bpAccountWrap.MustExist("block producer account not exist ")

	bpWrap := table.NewSoBlockProducerWrap(ev.Database(), op.BlockProducer)
	bpWrap.MustExist("the account you want to vote is not a block producer")
	bpVoteVestCnt := bpWrap.GetBpVest().VoteVest
	bpActive := bpWrap.GetBpVest().Active
	bpVoterCount := bpWrap.GetVoterCount()

	bpId := &prototype.BpBlockProducerId{BlockProducer: op.BlockProducer, Voter: op.Voter}
	vidWrap := table.NewSoBlockProducerVoteWrap(ev.Database(), bpId)

	if op.Cancel {
		// delete vote record
		vidWrap.MustExist("vote record not exist")
		vidWrap.RemoveBlockProducerVote()

		// modify block producer bp vest
		bpVoteVestCnt.Sub(voterVests)
		newBpVest := &prototype.BpVestId{Active:bpActive, VoteVest:bpVoteVestCnt}
		bpWrap.SetBpVest(newBpVest)

		// modify block producer voter count
		opAssert(bpVoterCount > 0, "block producer voter count should be greater than 0")
		bpWrap.SetVoterCount(bpVoterCount-1)

		// modify voter bp_vote_count
		opAssert(voteCnt > 0, "vote count must not be 0")
		voterAccount.SetBpVoteCount(voteCnt-1)

	} else {
		// block producer should be in active mode
		opAssert(bpActive, "block producer has been disabled")

		// check duplicate vote
		vidWrap.MustNotExist("already vote to this bp, do not vote twice")

		// check voter vote count, it should be less than the limit
		opAssert(voteCnt < constants.PerUserBpVoteLimit, "vote count exceeding")

		// add vote record
		vidWrap.Create(func(t *table.SoBlockProducerVote) {
			t.BlockProducerId = bpId
			t.VoterName = op.Voter
			t.VoteTime = ev.GlobalProp().HeadBlockTime()
		})

		// modify voter vote count
		voterAccount.SetBpVoteCount(voteCnt+1)

		// modify block producer bp vest and voter count
		bpVoteVestCnt.Add(voterVests)
		newBpVest := &prototype.BpVestId{Active:bpActive, VoteVest:bpVoteVestCnt}
		bpWrap.SetBpVest(newBpVest)
		bpWrap.SetVoterCount(bpVoterCount+1)
	}
}

func (ev *BpUpdateEvaluator) Apply() {
	op := ev.op
	ev.VMInjector().RecordStaminaFee(op.Owner.Value, constants.CommonOpStamina)

	staminaFree := op.Props.StaminaFree
	opAssert(staminaFree >= constants.MinStaminaFree,
		fmt.Sprintf("proposed stamina free too low min value %d", constants.MinStaminaFree))
	opAssert(staminaFree <= constants.MaxStaminaFree,
		fmt.Sprintf("proposed stamina free too high max value %d", constants.MaxStaminaFree))

	tpsExpected := op.Props.TpsExpected
	opAssert(tpsExpected >= constants.MinTPSExpected,
		fmt.Sprintf("expected tps too low min value %d", constants.MinTPSExpected))
	opAssert(tpsExpected <= constants.MaxTPSExpected,
		fmt.Sprintf("expected tps too high max value %d", constants.MaxTPSExpected))

	accountCreateFee := op.Props.AccountCreationFee
	opAssert(accountCreateFee.Value >= constants.MinAccountCreateFee,
		fmt.Sprintf("account create fee too low min value %d", constants.MinAccountCreateFee))
	opAssert(accountCreateFee.Value <= constants.MaxAccountCreateFee,
		fmt.Sprintf("account create fee too high max value %d", constants.MaxAccountCreateFee))

	topNAcquireFreeToken := op.Props.TopNAcquireFreeToken
	opAssert(topNAcquireFreeToken <= constants.MaxTopN, fmt.Sprintf("top N VEST holders, the N is too big, " +
		"which should lower than %d", constants.MaxTopN))

	//epochDuration := op.EpochDuration

	perTicketPrice := op.Props.PerTicketPrice
	opAssert(perTicketPrice.Value >= constants.MinTicketPrice, fmt.Sprintf("the ticket price should greater than %d",
		constants.MinTicketPrice))

	//perTicketWeight := op.PerTicketWeight

	bpWrap := table.NewSoBlockProducerWrap(ev.Database(), op.Owner)
	//opAssert(bpWrap.SetProposedStaminaFree(staminaFree), "update bp proposed stamina free error")
	//opAssert(bpWrap.SetTpsExpected(tpsExpected), "update bp tps expected error")
	//opAssert(bpWrap.SetAccountCreateFee(accountCreateFee), "update account create fee error")
	//opAssert(bpWrap.SetTopNAcquireFreeToken(topNAcquireFreeToken), "update topna error")
	//opAssert(bpWrap.SetEpochDuration(epochDuration), "update epoch duration error")
	//opAssert(bpWrap.SetPerTicketPrice(perTicketPrice), "update per ticket price error")
	//opAssert(bpWrap.SetPerTicketWeight(perTicketWeight), "update per ticket weight error")

	bpWrap.Modify(func(tInfo *table.SoBlockProducer) {
		tInfo.ProposedStaminaFree = op.Props.StaminaFree
		tInfo.TpsExpected = op.Props.TpsExpected
		tInfo.AccountCreateFee = op.Props.AccountCreationFee
		tInfo.TopNAcquireFreeToken = op.Props.TopNAcquireFreeToken
		tInfo.EpochDuration = op.Props.EpochDuration
		tInfo.PerTicketPrice = op.Props.PerTicketPrice
		tInfo.PerTicketWeight = op.Props.PerTicketWeight
	})

}

func (ev *FollowEvaluator) Apply() {
	op := ev.op
	ev.VMInjector().RecordStaminaFee(op.Account.Value, constants.CommonOpStamina)

	acctWrap := table.NewSoAccountWrap(ev.Database(), op.Account)
	acctWrap.MustExist("follow account do not exist ")

	acctWrap = table.NewSoAccountWrap(ev.Database(), op.FAccount)
	acctWrap.MustExist("follow f_account do not exist ")

	opAssert( op.Account.Value != op.FAccount.Value, "can't follow yourself")
}

func (ev *TransferToVestEvaluator) Apply() {
	op := ev.op
	ev.VMInjector().RecordStaminaFee(op.From.Value, constants.CommonOpStamina)

	fidWrap := table.NewSoAccountWrap(ev.Database(), op.From)
	tidWrap := table.NewSoAccountWrap(ev.Database(), op.To)

	tidWrap.MustExist("to account do not exist")

	//fBalance := fidWrap.GetBalance()
	tVests := tidWrap.GetVest()
	oldVest := prototype.NewVest(tVests.Value)
	addVests := prototype.NewVest(op.Amount.Value)

	//fBalance.Sub(op.Amount)
	//fidWrap.SetBalance(fBalance)
	fidWrap.Modify(func(tInfo *table.SoAccount) {
		tInfo.Balance.Sub(op.Amount)
	})

	//tVests.Add(addVests)
	//tidWrap.SetVest(tVests)
	tidWrap.Modify(func(tInfo *table.SoAccount) {
		tInfo.Vest.Add(addVests)
	})

	updateBpVoteValue(ev.Database(), op.To, oldVest, tVests)

	ev.GlobalProp().TransferToVest(op.Amount)
}

func updateBpVoteValue(dba iservices.IDatabaseRW, voter *prototype.AccountName, oldVest, newVest *prototype.Vest) (t1, t2 time.Duration){
	getBpNameStart := common.EasyTimer()
	uniqueVoterQueryWrap := table.UniBlockProducerVoteVoterNameWrap{Dba:dba}
	bpId := uniqueVoterQueryWrap.UniQueryVoterName(voter)
	if bpId == nil {
		t1 = getBpNameStart.Elapsed()
		return
	}
	bpName := bpId.GetBlockProducerId().BlockProducer
	t1 = getBpNameStart.Elapsed()


	startTime := common.EasyTimer()
	bpWrap := table.NewSoBlockProducerWrap(dba, bpName)
	if bpWrap != nil && bpWrap.CheckExist() {
		bpVoteVestCnt := bpWrap.GetBpVest().VoteVest
		bpActive := bpWrap.GetBpVest().Active
		bpVoteVestCnt.Sub(oldVest)
		bpVoteVestCnt.Add(newVest)

		newBpVest := &prototype.BpVestId{Active:bpActive, VoteVest:bpVoteVestCnt}
		bpWrap.SetBpVest(newBpVest)
	}
	t2 = startTime.Elapsed()

	return
}

func (ev *ConvertVestEvaluator) Apply() {
	op := ev.op
	ev.VMInjector().RecordStaminaFee(op.From.Value, constants.CommonOpStamina)

	accWrap := table.NewSoAccountWrap(ev.Database(), op.From)
	accWrap.MustExist("account do not exist")
	//opAssert(op.Amount.Value >= uint64(1e6), "At least 1 VEST should be converted")
	opAssert(accWrap.GetVest().Value - uint64(constants.MinAccountCreateFee) >= op.Amount.Value, "VEST balance not enough")
	globalProps := ev.GlobalProp().GetProps()
	//timestamp := globalProps.Time.UtcSeconds
	currentBlock := globalProps.HeadBlockNumber
	eachRate := op.Amount.Value / (constants.ConvertWeeks - 1)
	//accWrap.MdNextPowerdownTime(&prototype.TimePointSec{UtcSeconds: timestamp + constants.POWER_DOWN_INTERVAL})
	//accWrap.SetNextPowerdownBlockNum(currentBlock + constants.PowerDownBlockInterval)
	//accWrap.SetEachPowerdownRate(&prototype.Vest{Value: eachRate})
	//accWrap.SetHasPowerdown(&prototype.Vest{Value: 0})
	//accWrap.SetToPowerdown(op.Amount)

	accWrap.Modify(func(t *table.SoAccount) {
		t.NextPowerdownBlockNum = currentBlock + constants.PowerDownBlockInterval
		t.EachPowerdownRate = &prototype.Vest{Value: eachRate}
		t.HasPowerdown = &prototype.Vest{Value: 0}
		t.ToPowerdown = op.Amount
	})
}

type byTag []int32

func (c byTag) Len() int {
	return len(c)
}
func (c byTag) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
func (c byTag) Less(i, j int) bool {
	return c[i] < c[j]
}

func mergeTags(existed []int32, new []prototype.ReportOperationTag) []int32 {
	len1 := len(existed)
	len2 := len(new)
	tmp := make([]int32, 0, len2)
	for i := 0; i < len2; i++ {
		tmp[i] = int32(new[i])
	}
	sort.Sort(byTag(existed))
	sort.Sort(byTag(tmp))

	res := make([]int32, 0, len1+len2)
	i := 0
	j := 0
	for {
		if i == len1 || j == len2 {
			break
		}
		if existed[i] <= tmp[j] {
			res = append(res, existed[i])
			if existed[i] == tmp[j] {
				j++
			}
			i++
		} else if existed[i] > tmp[j] {
			res = append(res, tmp[j])
			j++
		}
	}
	if i < len1 {
		res = append(res, existed[i:]...)
	}
	if j < len2 {
		res = append(res, tmp[i:]...)
	}

	return res
}

func (ev *ReportEvaluator) Apply() {
	op := ev.op
	ev.VMInjector().RecordStaminaFee(op.Reporter.Value, constants.CommonOpStamina)
	post := table.NewSoPostWrap(ev.Database(), &op.Reported)
	post.MustExist("the reported post doesn't exist")
	report := table.NewSoReportListWrap(ev.Database(), &op.Reported)
	if op.IsArbitration {
		report.MustExist("cannot arbitrate a non-existed post")
		if op.IsApproved {
			post.RemovePost()
			report.RemoveReportList()
			return
		}

		report.SetIsArbitrated(true)
	} else {
		if report.CheckExist() {
			if report.GetIsArbitrated() {
				opAssert(false, "cannot report a legal post")
			}
			report.SetReportedTimes(report.GetReportedTimes() + 1)
			existedTags := report.GetTags()
			newTags := op.ReportTag
			report.SetTags(mergeTags(existedTags, newTags))
			return
		}

		report.Create(func(tInfo *table.SoReportList) {
			tInfo.Uuid = op.Reported
			tInfo.ReportedTimes = 1
			tags := make([]int32, len(op.ReportTag))
			for i := range op.ReportTag {
				tags[i] = int32(op.ReportTag[i])
			}
			tInfo.Tags = tags
			tInfo.IsArbitrated = false
		})
	}
}

func (ev *ContractDeployEvaluator) Apply() {
	op := ev.op

	// code and abi decompression
	var (
		contractCode, contractAbi []byte
		err error
	)
	if contractCode, err = common.Decompress(op.Code); err != nil {
		opAssertE(err, "contract code decompression failed");
	}
	if contractAbi, err = common.Decompress(op.Abi); err != nil {
		opAssertE(err, "contract abi decompression failed");
	}

	ev.VMInjector().RecordStaminaFee(op.Owner.Value, constants.CommonOpStamina)

	cid 		:= prototype.ContractId{Owner: op.Owner, Cname: op.Contract}
	scid 		:= table.NewSoContractWrap(ev.Database(), &cid)
	checkSum 	:= sha256.Sum256(contractCode)
	codeHash    := &prototype.Sha256{ Hash:checkSum[:] }
	if scid.CheckExist() {
		opAssert( scid.GetUpgradeable(), "contract can not upgrade")
		opAssert( !scid.GetHash().Equal( codeHash ), "code hash can not be equal")
	}

	_, err = abi.UnmarshalABI(contractAbi)
	if err != nil {
		opAssertE(err, "invalid contract abi")
	}
	abiString := string(contractAbi)

	vmCtx := vmcontext.NewContextFromDeployOp(op, contractCode, abiString, nil)

	cosVM := vm.NewCosVM(vmCtx, nil, nil, nil)

	opAssertE(cosVM.Validate(), "validate code failed")

	if scid.CheckExist() {
		//scid.SetAbi( abiString )
		//scid.SetCode( contractCode )
		//scid.SetUpgradeable( op.Upgradeable )
		//scid.SetHash( codeHash )

		scid.Modify(func(tInfo *table.SoContract) {
			tInfo.Abi = abiString
			tInfo.Code = contractCode
			tInfo.Upgradeable = op.Upgradeable
			tInfo.Hash = codeHash
		})

	} else {
		scid.Create(func(t *table.SoContract) {
			t.Code = contractCode
			t.Id = &cid
			t.CreatedTime = ev.GlobalProp().HeadBlockTime()
			t.Abi = abiString
			t.Upgradeable = op.Upgradeable
			t.Hash = codeHash
			t.Balance = prototype.NewCoin(0)
			t.Url = op.Url
			t.Describe = op.Describe
		})
	}
}

func (ev *ContractApplyEvaluator) Apply() {
	op := ev.op

	cid := prototype.ContractId{Owner: op.Owner, Cname: op.Contract}
	scid := table.NewSoContractWrap(ev.Database(), &cid)
	scid.MustExist("contract name doesn't exist")

	acc := table.NewSoAccountWrap(ev.Database(), op.Caller)
	acc.MustExist("caller account doesn't exist")

	balance := acc.GetBalance().Value

	// the amount is also minicos or cos ?
	// here I assert it is minicos
	// also, I think balance base on minicos is far more reliable.
	if op.Amount != nil {
		opAssert(balance >= op.Amount.Value, "balance does not have enough fund to transfer")
	}

	code := scid.GetCode()
	codeHash := scid.GetHash()

	var err error
	var abiInterface abi.IContractABI
	var paramsData []byte
	var tables *ct.ContractTables

	if abiInterface, err = abi.UnmarshalABI([]byte(scid.GetAbi())); err != nil {
		opAssertE(err, "invalid contract abi")
	}
	if m := abiInterface.MethodByName(op.Method); m != nil {
		paramsData, err = vme.EncodeFromJson([]byte(op.Params), m.Args().Type())
		if err != nil {
			opAssertE(err, "invalid contract parameters")
		}
	} else {
		opAssert(false, "unknown contract method: "+op.Method)
	}

	if abiInterface != nil {
		tables = ct.NewContractTables(op.Owner.Value, op.Contract, abiInterface, ev.Database())
	}

	vmCtx := vmcontext.NewContextFromApplyOp(op, paramsData, code, codeHash, abiInterface, tables, ev.VMInjector(),
		ev.TrxObserver())
	// set max gas
	remain := ev.VMInjector().GetVmRemainCpuStamina(op.Caller.Value)
	remainGas := remain * constants.CpuConsumePointDen
	if remainGas > constants.MaxGasPerCall {
		vmCtx.Gas = constants.MaxGasPerCall
	} else {
		vmCtx.Gas = remainGas
	}
	// turn off gas limit
//	if !ev.GlobalProp().ctx.Config().ResourceCheck  {
//		vmCtx.Gas = constants.OneDayStamina * constants.CpuConsumePointDen
//	}

	// should be active ?
	//defer func() {
	//	_ := recover()
	//}()
	if op.Amount != nil && op.Amount.Value > 0 {
		vmCtx.Injector.TransferFromUserToContract(op.Caller.Value, op.Contract, op.Owner.Value, op.Amount.Value)
	}

	cosVM := vm.NewCosVM(vmCtx, ev.Database(), ev.GlobalProp().GetProps(), ev.Logger())

	ret, err := cosVM.Run()
	spentGas := cosVM.SpentGas()
	// need extra query db, is it a good way or should I pass account object as parameter?
	// DeductStamina and usertranfer could be panic (rarely, I can't image how it happens)
	// the panic should catch then return or bubble it ?

	vmCtx.Injector.RecordStaminaFee(op.Caller.Value, spentGas)
	if err != nil {
		vmCtx.Injector.Error(ret, err.Error())
		opAssertE(err, "execute vm error")
	}
	applyCnt := scid.GetApplyCount()
	scid.SetApplyCount(applyCnt+1)

}

func (ev *InternalContractApplyEvaluator) Apply() {
	op := ev.op

	fromContract := table.NewSoContractWrap(ev.Database(), &prototype.ContractId{Owner: op.FromOwner, Cname: op.FromContract})
	fromContract.MustExist("fromContract contract doesn't exist")

	toContract := table.NewSoContractWrap(ev.Database(), &prototype.ContractId{Owner: op.ToOwner, Cname: op.ToContract})
	toContract.MustExist("toContract contract doesn't exist")

	caller := table.NewSoAccountWrap(ev.Database(), op.FromCaller)
	caller.MustExist("caller account doesn't exist")

	opAssert(fromContract.GetBalance().Value >= op.Amount.Value, "fromContract balance less than transfer amount")

	code := toContract.GetCode()
	codeHash := toContract.GetHash()

	var err error
	var abiInterface abi.IContractABI
	var tables *ct.ContractTables

	if abiInterface, err = abi.UnmarshalABI([]byte(toContract.GetAbi())); err != nil {
		opAssertE(err, "invalid toContract abi")
	}
	if m := abiInterface.MethodByName(op.ToMethod); m != nil {
		_, err = vme.DecodeToJson(op.Params, m.Args().Type(), false)
		if err != nil {
			opAssertE(err, "invalid contract parameters")
		}
	} else {
		opAssert(false, "unknown contract method: "+op.ToMethod)
	}

	if abiInterface != nil {
		tables = ct.NewContractTables(op.ToOwner.Value, op.ToContract, abiInterface, ev.Database())
	}

	vmCtx := vmcontext.NewContextFromInternalApplyOp(op, code, codeHash, abiInterface, tables, ev.VMInjector(), ev.TrxObserver())
	vmCtx.Gas = ev.remainGas

	if op.Amount != nil && op.Amount.Value > 0 {
		vmCtx.Injector.TransferFromContractToContract(op.FromContract, op.FromOwner.Value, op.ToContract, op.ToOwner.Value, op.Amount.Value)
	}

	cosVM := vm.NewCosVM(vmCtx, ev.Database(), ev.GlobalProp().GetProps(), ev.Logger())
	//ev.Database().BeginTransaction()
	ret, err := cosVM.Run()
	spentGas := cosVM.SpentGas()
	ev.preVm.CostGas += spentGas
	//vmCtx.Injector.RecordStaminaFee(op.FromCaller.Value, spentGas)

	if err != nil {
		vmCtx.Injector.Error(ret, err.Error())
		//ev.Database().EndTransaction(false)
		// throw a panic, this panic should recover by upper contract vm context
		opAssertE(err, "internal contract apply failed")
	}
}

func (ev *StakeEvaluator) Apply() {
	op := ev.op
	ev.VMInjector().RecordStaminaFee(op.From.Value, constants.CommonOpStamina)

	fidWrap := table.NewSoAccountWrap(ev.Database(), op.From)
	tidWrap := table.NewSoAccountWrap(ev.Database(), op.To)

	fidWrap.MustExist("from account do not exist")
	tidWrap.MustExist("to account do not exist")

	//fBalance := fidWrap.GetBalance()
	//tVests := tidWrap.GetStakeVest()
	addVests := prototype.NewVest(op.Amount.Value)

	//fBalance.Sub(op.Amount)
	//fidWrap.SetBalance(fBalance)
	fidWrap.Modify(func(tInfo *table.SoAccount) {
		tInfo.Balance.Sub(op.Amount)
	})

	//tVests.Add(addVests)
	//tidWrap.SetStakeVest(tVests)
	tidWrap.Modify(func(tInfo *table.SoAccount) {
		tInfo.StakeVest.Add(addVests)
	})

	// unique stake record
	recordWrap := table.NewSoStakeRecordWrap(ev.Database(), &prototype.StakeRecord{
		From:   op.From,
		To: op.To,
	})
	if !recordWrap.CheckExist() {
		recordWrap.Create(func(record *table.SoStakeRecord) {
			record.Record = &prototype.StakeRecord{
				From:   &prototype.AccountName{Value: op.From.Value},
				To: &prototype.AccountName{Value: op.To.Value},
			}
			record.RecordReverse = &prototype.StakeRecordReverse{
				To:&prototype.AccountName{Value: op.To.Value},
				From:   &prototype.AccountName{Value: op.From.Value},
			}
			record.StakeAmount = prototype.NewVest(addVests.Value)
		})
	} else {
		//oldVest := recordWrap.GetStakeAmount()
		//oldVest.Add(addVests)
		//recordWrap.SetStakeAmount(oldVest)
		recordWrap.Modify(func(tInfo *table.SoStakeRecord) {
			tInfo.StakeAmount.Add(addVests)
		})
	}
	headBlockTime := ev.GlobalProp().HeadBlockTime()
	recordWrap.SetLastStakeTime(headBlockTime)

	ev.GlobalProp().TransferToVest(op.Amount)
	ev.GlobalProp().TransferToStakeVest(op.Amount)
}

func (ev *UnStakeEvaluator) Apply() {
	op := ev.op
	ev.VMInjector().RecordStaminaFee(op.Creditor.Value, constants.CommonOpStamina)

	recordWrap := table.NewSoStakeRecordWrap(ev.Database(), &prototype.StakeRecord{
		From:   op.Creditor,
		To: op.Debtor,
	})

	recordWrap.MustExist("stake record not exist")

	stakeTime := recordWrap.GetLastStakeTime()
	headBlockTime := ev.GlobalProp().HeadBlockTime()
	opAssert(headBlockTime.UtcSeconds-stakeTime.UtcSeconds > constants.StakeFreezeTime, "can not unstake when freeze")

	debtorWrap := table.NewSoAccountWrap(ev.Database(), op.Debtor)
	creditorWrap := table.NewSoAccountWrap(ev.Database(), op.Creditor)

	value := op.Amount

	//vest := debtorWrap.GetStakeVest()
	//vest.Sub(value.ToVest())
	//debtorWrap.SetStakeVest(vest)
	debtorWrap.Modify(func(tInfo *table.SoAccount) {
		tInfo.StakeVest.Sub(value.ToVest())
	})

	//fBalance := creditorWrap.GetBalance()
	//fBalance.Add(value)
	//creditorWrap.SetBalance(fBalance)
	creditorWrap.Modify(func(tInfo *table.SoAccount) {
		tInfo.Balance.Add(value)
	})

	// update stake record
	//oldVest := recordWrap.GetStakeAmount()
	//oldVest.Sub(value.ToVest())
	recordWrap.Modify(func(tInfo *table.SoStakeRecord) {
		tInfo.StakeAmount.Sub(value.ToVest())
	})

	ev.GlobalProp().TransferFromVest(value.ToVest())
	ev.GlobalProp().TransferFromStakeVest(value.ToVest())
}

func (ev *AcquireTicketEvaluator) Apply() {
	op := ev.op
	ev.VMInjector().RecordStaminaFee(op.Account.Value, constants.CommonOpStamina)

	account := table.NewSoAccountWrap(ev.Database(), op.Account)
	count := op.Count
	// why need to buy so many tickets ?
	opAssert(count > 0, "at least 1 ticket per turn")
	opAssert(count <= constants.MaxTicketsPerTurn, fmt.Sprintf("at most %d ticket per turn", int(constants.MaxTicketsPerTurn)))

	ticketPrice := ev.GlobalProp().GetProps().PerTicketPrice
	//balance := account.GetBalance()
	//oldVest := account.GetVest()

	fee := &prototype.Coin{Value: ticketPrice.Value}
	fee.Mul(count)
	//balance.Sub(fee)
	//account.SetBalance(balance)
	account.Modify(func(tInfo *table.SoAccount) {
		tInfo.Balance.Sub(fee)
	})

	opAssert(account.GetChargedTicket() + uint32(count) > account.GetChargedTicket(), "ticket count overflow")

	account.SetChargedTicket(account.GetChargedTicket() + uint32(count))

	//updateBpVoteValue(ev.Database(), op.Account, oldVest, vest)

	// record
	ticketKey := &prototype.GiftTicketKeyType{
		Type: 1,
		From: "contentos",
		To: op.Account.Value,
		CreateBlock: ev.GlobalProp().GetProps().HeadBlockNumber+1,
	}

	ticketWrap := table.NewSoGiftTicketWrap(ev.Database(), ticketKey)
	ticketWrap.MustNotExist("ticket record existed")

	_ = ticketWrap.Create(func(tInfo *table.SoGiftTicket) {
		tInfo.Ticket = ticketKey
		tInfo.Count = count
		tInfo.Denom = ev.GlobalProp().GetProps().GetPerTicketWeight()
		tInfo.ExpireBlock = math.MaxUint64
	})

	props := ev.GlobalProp().GetProps()

	currentIncome := props.GetTicketsIncome()
	vestFee := fee.ToVest()
	currentIncome.Add(vestFee)

	chargedTicketsNum := props.GetChargedTicketsNum()
	currentTicketsNum := chargedTicketsNum + count
	ev.GlobalProp().UpdateTicketIncomeAndNum(currentIncome, currentTicketsNum)
	ev.GlobalProp().ModifyProps(func(dprop *prototype.DynamicProperties) {
		dprop.TotalCos = dprop.TotalCos.Sub(fee)
	})
}

func (ev *VoteByTicketEvaluator) Apply() {
	op := ev.op
	ev.VMInjector().RecordStaminaFee(op.Account.Value, constants.CommonOpStamina)

	account := table.NewSoAccountWrap(ev.Database(), op.Account)
	postId := op.Idx
	var freeTicket uint32 = 0
	count := op.Count

	postWrap := table.NewSoPostWrap(ev.Database(), &op.Idx)
	postWrap.MustExist("post does not exist")

	originTicketCount := postWrap.GetTicket()

	// free ticket ?
	freeTicketWrap := table.NewSoGiftTicketWrap(ev.Database(), &prototype.GiftTicketKeyType{
		Type: 0,
		From: "contentos",
		To: op.Account.Value,
		CreateBlock: ev.GlobalProp().GetProps().GetCurrentEpochStartBlock(),
	})
	if freeTicketWrap.CheckExist() {
		freeTicket = 1
	}
	opAssert(count > 0, "at least 1 ticket to vote per turn")
	opAssert(count <= constants.MaxTicketsPerTurn, fmt.Sprintf("at most %d ticket per turn", int(constants.MaxTicketsPerTurn)))

	// if voter's reputation is 0, her tickets are useless.
	factor := uint32(1)
	if account.GetReputation() == constants.MinReputation {
		factor = 0
	}
	if freeTicket > 0 {
		// spend free ticket first
		count = count - 1
		opAssert(account.GetChargedTicket() >= uint32(count), "insufficient ticket to vote")
		account.SetChargedTicket(account.GetChargedTicket() - uint32(count))
		freeTicketWrap.RemoveGiftTicket()
		postWrap.SetTicket(originTicketCount + uint32(count + 1) * factor)
	} else {
		opAssert(account.GetChargedTicket() >= uint32(count), "insufficient ticket to vote")
		account.SetChargedTicket(account.GetChargedTicket() - uint32(count))
		postWrap.SetTicket(originTicketCount + uint32(count) * factor)
	}

	// record
	ticketKey := &prototype.GiftTicketKeyType{
		Type: 1,
		From: op.Account.Value,
		To: strconv.FormatUint(postId, 10),
		CreateBlock: ev.GlobalProp().GetProps().HeadBlockNumber+1,
	}
	ticketWrap := table.NewSoGiftTicketWrap(ev.Database(), ticketKey)
	ticketWrap.MustNotExist("ticket record existed")

	// record ticket vote action
	_ = ticketWrap.Create(func(tInfo *table.SoGiftTicket) {
		tInfo.Ticket = ticketKey
		tInfo.Count = count
		tInfo.Denom = ev.GlobalProp().GetProps().GetPerTicketWeight()
		tInfo.ExpireBlock = math.MaxUint64
	})

	//ev.GlobalProp().VoteByTicket(op.Account, postId, count)
	props := ev.GlobalProp().GetProps()
	currentBp := props.CurrentBlockProducer
	bpWrap := table.NewSoAccountWrap(ev.Database(), currentBp)
	bpWrap.MustExist(fmt.Sprintf("cannot find bp %s", currentBp.Value))

	// the per ticket price may change,so replace the per ticket price by totalincome / ticketnum
	opAssert(props.GetChargedTicketsNum() >= count, "should acquire tickets first")
	var equalValue *prototype.Vest
	if props.GetChargedTicketsNum() == 0 {
		equalValue = &prototype.Vest{Value: 0}
	} else {
		equalValue = &prototype.Vest{Value: props.GetTicketsIncome().Value / props.GetChargedTicketsNum()}
		equalValue.Mul(count)
	}
	currentIncome := props.GetTicketsIncome()
	currentIncome.Sub(equalValue)
	//c.modifyGlobalDynamicData(func(props *prototype.DynamicProperties) {
	//	props.TicketsIncome = income
	//	props.ChargedTicketsNum -= count
	//})
	chargedTicketsNum := props.GetChargedTicketsNum()
	currentTicketsNum := chargedTicketsNum - count
	ev.GlobalProp().UpdateTicketIncomeAndNum(currentIncome, currentTicketsNum)

	if equalValue.Value > 0 {
		ev.GlobalProp().ModifyProps(func(prop *prototype.DynamicProperties) {
			prop.TicketsBpBonus = prop.TicketsBpBonus.Add(equalValue)
			prop.TotalVest = prop.TotalVest.Add(equalValue)
		})
	}
}