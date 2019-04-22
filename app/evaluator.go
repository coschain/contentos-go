package app

import (
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/common/encoding/vme"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/vm"
	"github.com/coschain/contentos-go/vm/context"
	"github.com/coschain/contentos-go/vm/contract/abi"
	ct "github.com/coschain/contentos-go/vm/contract/table"
	"github.com/sirupsen/logrus"
)

type AccountCreateEvaluator struct {
	BaseEvaluator
	ctx *ApplyContext
	op  *prototype.AccountCreateOperation
}

type TransferEvaluator struct {
	BaseEvaluator
	ctx *ApplyContext
	op  *prototype.TransferOperation
}

type PostEvaluator struct {
	BaseEvaluator
	ctx *ApplyContext
	op  *prototype.PostOperation
}
type ReplyEvaluator struct {
	BaseEvaluator
	ctx *ApplyContext
	op  *prototype.ReplyOperation
}
type VoteEvaluator struct {
	BaseEvaluator
	ctx *ApplyContext
	op  *prototype.VoteOperation
}
type BpRegisterEvaluator struct {
	BaseEvaluator
	ctx *ApplyContext
	op  *prototype.BpRegisterOperation
}
type BpUnregisterEvaluator struct {
	BaseEvaluator
	ctx *ApplyContext
	op  *prototype.BpUnregisterOperation
}

type BpVoteEvaluator struct {
	BaseEvaluator
	ctx *ApplyContext
	op  *prototype.BpVoteOperation
}

type FollowEvaluator struct {
	BaseEvaluator
	ctx *ApplyContext
	op  *prototype.FollowOperation
}

type TransferToVestingEvaluator struct {
	BaseEvaluator
	ctx *ApplyContext
	op  *prototype.TransferToVestingOperation
}

type ClaimEvaluator struct {
	BaseEvaluator
	ctx *ApplyContext
	op  *prototype.ClaimOperation
}

// I can cat out this awkward claimall operation until I can get value from rpc resp
type ClaimAllEvaluator struct {
	BaseEvaluator
	ctx *ApplyContext
	op  *prototype.ClaimAllOperation
}

type ContractDeployEvaluator struct {
	BaseEvaluator
	ctx *ApplyContext
	op  *prototype.ContractDeployOperation
}

type ContractApplyEvaluator struct {
	BaseEvaluator
	ctx *ApplyContext
	op  *prototype.ContractApplyOperation
}

type ContractEstimateApplyEvaluator struct {
	BaseEvaluator
	ctx *ApplyContext
	op  *prototype.ContractEstimateApplyOperation
}

type InternalContractApplyEvaluator struct {
	BaseEvaluator
	ctx       *ApplyContext
	op        *prototype.InternalContractApplyOperation
	remainGas uint64
}

type StakeEvaluator struct {
	BaseEvaluator
	ctx *ApplyContext
	op  *prototype.StakeOperation
}

type UnStakeEvaluator struct {
	BaseEvaluator
	ctx *ApplyContext
	op  *prototype.UnStakeOperation
}

func (ev *AccountCreateEvaluator) Apply() {
	op := ev.op
	ev.ctx.trxCtx.RecordGasFee(op.Creator.Value, constants.CommonOpGas)

	creatorWrap := table.NewSoAccountWrap(ev.ctx.db, op.Creator)

	mustSuccess(creatorWrap.CheckExist(), "creator not exist ", prototype.StatusErrorDbExist)

	mustSuccess(creatorWrap.GetBalance().Value >= op.Fee.Value, "Insufficient balance to create account.", prototype.StatusErrorTrxValueCompare)

	// check auth accounts
	for _, a := range op.Owner.AccountAuths {
		tmpAccountWrap := table.NewSoAccountWrap(ev.ctx.db, a.Name)
		mustSuccess(tmpAccountWrap.CheckExist(), "owner auth account not exist", prototype.StatusErrorDbExist)
	}

	// sub creator's fee
	originBalance := creatorWrap.GetBalance()
	mustNoError(originBalance.Sub(op.Fee), "creator balance overflow", prototype.StatusErrorTrxMath)
	mustSuccess(creatorWrap.MdBalance(originBalance), "modify balance failed", prototype.StatusErrorDbUpdate)

	// create account
	newAccountWrap := table.NewSoAccountWrap(ev.ctx.db, op.NewAccountName)
	mustNoError(newAccountWrap.Create(func(tInfo *table.SoAccount) {
		tInfo.Name = op.NewAccountName
		tInfo.Creator = op.Creator
		tInfo.CreatedTime = ev.ctx.control.HeadBlockTime()
		tInfo.Balance = prototype.NewCoin(0)
		tInfo.VestingShares = op.Fee.ToVest()
		tInfo.LastPostTime = ev.ctx.control.HeadBlockTime()
		tInfo.LastVoteTime = ev.ctx.control.HeadBlockTime()
		tInfo.StakeVesting = prototype.NewVest(0)
	}), "duplicate create account object", prototype.StatusErrorDbCreate)

	// create account authority
	authorityWrap := table.NewSoAccountAuthorityObjectWrap(ev.ctx.db, op.NewAccountName)
	mustNoError(authorityWrap.Create(func(tInfo *table.SoAccountAuthorityObject) {
		tInfo.Account = op.NewAccountName
		tInfo.Owner = op.Owner
		tInfo.LastOwnerUpdate = prototype.NewTimePointSec(0)
	}), "duplicate create account authority object", prototype.StatusErrorDbCreate)

	// sub dynamic glaobal properties's total fee
	ev.ctx.control.TransferToVest(op.Fee)
}

func (ev *TransferEvaluator) Apply() {
	op := ev.op
	ev.ctx.trxCtx.RecordGasFee(op.From.Value, constants.CommonOpGas)

	// @ active_challenged
	fromWrap := table.NewSoAccountWrap(ev.ctx.db, op.From)
	toWrap := table.NewSoAccountWrap(ev.ctx.db, op.To)

	mustSuccess(toWrap.CheckExist(), "To account do not exist ", prototype.StatusErrorDbExist)

	fBalance := fromWrap.GetBalance()
	tBalance := toWrap.GetBalance()

	mustNoError(fBalance.Sub(op.Amount), "Insufficient balance to transfer.", prototype.StatusErrorTrxMath)
	mustSuccess(fromWrap.MdBalance(fBalance), "modify balance failed", prototype.StatusErrorDbUpdate)

	mustNoError(tBalance.Add(op.Amount), "balance overflow", prototype.StatusErrorTrxMath)
	mustSuccess(toWrap.MdBalance(tBalance), "modify balance failed", prototype.StatusErrorDbUpdate)
}

func (ev *PostEvaluator) Apply() {
	op := ev.op
	ev.ctx.trxCtx.RecordGasFee(op.Owner.Value, constants.CommonOpGas)

	idWrap := table.NewSoPostWrap(ev.ctx.db, &op.Uuid)
	mustSuccess(!idWrap.CheckExist(), "post uuid exist", prototype.StatusErrorDbExist)

	authorWrap := table.NewSoAccountWrap(ev.ctx.db, op.Owner)
	elapsedSeconds := ev.ctx.control.HeadBlockTime().UtcSeconds - authorWrap.GetLastPostTime().UtcSeconds
	mustSuccess(elapsedSeconds > constants.MIN_POST_INTERVAL, "posting frequently", prototype.StatusErrorTrxValueCompare)

	mustNoError(idWrap.Create(func(t *table.SoPost) {
		t.PostId = op.Uuid
		t.Tags = op.Tags
		t.Title = op.Title
		t.Author = op.Owner
		t.Body = op.Content
		t.Created = ev.ctx.control.HeadBlockTime()
		t.CashoutTime = &prototype.TimePointSec{UtcSeconds: ev.ctx.control.HeadBlockTime().UtcSeconds + uint32(constants.POST_CASHPUT_DELAY_TIME)}
		t.Depth = 0
		t.Children = 0
		t.RootId = t.PostId
		t.ParentId = 0
		t.RootId = 0
		t.Beneficiaries = op.Beneficiaries
		t.WeightedVp = 0
		t.VoteCnt = 0
	}), "create post error", prototype.StatusErrorDbCreate)

	authorWrap.MdLastPostTime(ev.ctx.control.HeadBlockTime())

	//timestamp := ev.ctx.control.HeadBlockTime().UtcSeconds + uint32(constants.POST_CASHPUT_DELAY_TIME) - uint32(constants.GenesisTime)
	//key := fmt.Sprintf("cashout:%d_%d", common.GetBucket(timestamp), op.Uuid)
	//value := "post"
	//opAssertE(ev.ctx.db.Put([]byte(key), []byte(value)), "put post key into db error")

}

func (ev *ReplyEvaluator) Apply() {
	op := ev.op
	ev.ctx.trxCtx.RecordGasFee(op.Owner.Value, constants.CommonOpGas)
	cidWrap := table.NewSoPostWrap(ev.ctx.db, &op.Uuid)
	pidWrap := table.NewSoPostWrap(ev.ctx.db, &op.ParentUuid)

	mustSuccess(!cidWrap.CheckExist(), "post uuid exist", prototype.StatusErrorDbExist)
	mustSuccess(pidWrap.CheckExist(), "parent uuid do not exist", prototype.StatusErrorDbExist)

	mustSuccess(pidWrap.GetDepth()+1 < constants.POST_MAX_DEPTH, "reply depth error", prototype.StatusErrorTrxValueCompare)

	authorWrap := table.NewSoAccountWrap(ev.ctx.db, op.Owner)
	elapsedSeconds := ev.ctx.control.HeadBlockTime().UtcSeconds - authorWrap.GetLastPostTime().UtcSeconds
	mustSuccess(elapsedSeconds > constants.MIN_POST_INTERVAL, "reply frequently", prototype.StatusErrorTrxValueCompare)

	var rootId uint64
	if pidWrap.GetRootId() == 0 {
		rootId = pidWrap.GetPostId()
	} else {
		rootId = pidWrap.GetRootId()
	}

	mustNoError(cidWrap.Create(func(t *table.SoPost) {
		t.PostId = op.Uuid
		t.Tags = nil
		t.Title = ""
		t.Author = op.Owner
		t.Body = op.Content
		t.Created = ev.ctx.control.HeadBlockTime()
		t.CashoutTime = &prototype.TimePointSec{UtcSeconds: ev.ctx.control.HeadBlockTime().UtcSeconds + uint32(constants.POST_CASHPUT_DELAY_TIME)}
		t.Depth = pidWrap.GetDepth() + 1
		t.Children = 0
		t.RootId = rootId
		t.ParentId = op.ParentUuid
		t.VoteCnt = 0
		t.Beneficiaries = op.Beneficiaries
	}), "create reply error", prototype.StatusErrorDbCreate)

	authorWrap.MdLastPostTime(ev.ctx.control.HeadBlockTime())
	// Modify Parent Object
	mustSuccess(pidWrap.MdChildren(pidWrap.GetChildren()+1), "Modify Parent Children Error", prototype.StatusErrorDbUpdate)

	//timestamp := ev.ctx.control.HeadBlockTime().UtcSeconds + uint32(constants.POST_CASHPUT_DELAY_TIME) - uint32(constants.GenesisTime)
	//key := fmt.Sprintf("cashout:%d_%d", common.GetBucket(timestamp), op.Uuid)
	//value := "reply"
	//opAssertE(ev.ctx.db.Put([]byte(key), []byte(value)), "put reply key into db error")
}

// upvote is true: upvote otherwise downvote
// no downvote has been supplied by command, so I ignore it
func (ev *VoteEvaluator) Apply() {
	op := ev.op
	ev.ctx.trxCtx.RecordGasFee(op.Voter.Value, constants.CommonOpGas)
	voterWrap := table.NewSoAccountWrap(ev.ctx.db, op.Voter)
	elapsedSeconds := ev.ctx.control.HeadBlockTime().UtcSeconds - voterWrap.GetLastVoteTime().UtcSeconds
	mustSuccess(elapsedSeconds > constants.MIN_VOTE_INTERVAL, "voting frequently", prototype.StatusErrorTrxValueCompare)

	voterId := prototype.VoterId{Voter: op.Voter, PostId: op.Idx}
	voteWrap := table.NewSoVoteWrap(ev.ctx.db, &voterId)
	postWrap := table.NewSoPostWrap(ev.ctx.db, &op.Idx)

	mustSuccess(postWrap.CheckExist(), "post invalid", prototype.StatusErrorDbExist)
	mustSuccess(!voteWrap.CheckExist(), "vote info exist", prototype.StatusErrorDbExist)

	//votePostWrap := table.NewVotePostIdWrap(ev.ctx.db)

	//for voteIter := votePostWrap.QueryListByOrder(&op.Idx, nil); voteIter.Valid(); voteIter.Next() {
	//	voterId := votePostWrap.GetMainVal(voteIter)
	//	if voterId.Voter.Value == op.Voter.Value {
	//		opAssertE(errors.New("Vote Error"), "vote to a same post")
	//	}
	//}

	regeneratedPower := constants.PERCENT * elapsedSeconds / constants.VOTE_REGENERATE_TIME
	var currentVp uint32
	votePower := voterWrap.GetVotePower() + regeneratedPower
	if votePower > constants.PERCENT {
		currentVp = constants.PERCENT
	} else {
		currentVp = votePower
	}
	usedVp := (currentVp + constants.VOTE_LIMITE_DURING_REGENERATE - 1) / constants.VOTE_LIMITE_DURING_REGENERATE

	voterWrap.MdVotePower(currentVp - usedVp)
	voterWrap.MdLastVoteTime(ev.ctx.control.HeadBlockTime())
	vesting := voterWrap.GetVestingShares().Value
	// todo: uint128
	weightedVp := vesting * uint64(usedVp)
	if postWrap.GetCashoutTime().UtcSeconds > ev.ctx.control.HeadBlockTime().UtcSeconds {
		lastVp := postWrap.GetWeightedVp()
		votePower := lastVp + weightedVp
		// add new vp into global
		ev.ctx.control.AddWeightedVP(weightedVp)
		// update post's weighted vp
		postWrap.MdWeightedVp(votePower)

		mustNoError(voteWrap.Create(func(t *table.SoVote) {
			t.Voter = &voterId
			t.PostId = op.Idx
			t.Upvote = true
			t.WeightedVp = weightedVp
			t.VoteTime = ev.ctx.control.HeadBlockTime()
		}), "create voter object error", prototype.StatusErrorDbCreate)

		mustSuccess(postWrap.MdVoteCnt(postWrap.GetVoteCnt()+1), "set vote count error", prototype.StatusErrorDbUpdate)
	}
}

func (ev *BpRegisterEvaluator) Apply() {
	op := ev.op
	ev.ctx.trxCtx.RecordGasFee(op.Owner.Value, constants.CommonOpGas)
	witnessWrap := table.NewSoWitnessWrap(ev.ctx.db, op.Owner)

	mustSuccess(!witnessWrap.CheckExist(), "witness already exist", prototype.StatusErrorDbExist)

	mustNoError(witnessWrap.Create(func(t *table.SoWitness) {
		t.Owner = op.Owner
		t.CreatedTime = ev.ctx.control.HeadBlockTime()
		t.Url = op.Url
		t.SigningKey = op.BlockSigningKey

		// TODO add others
	}), "add witness record error", prototype.StatusErrorDbCreate)
}

func (ev *BpUnregisterEvaluator) Apply() {
	// unregister op cost too much cpu time
	panic("not yet implement")

}

func (ev *BpVoteEvaluator) Apply() {
	op := ev.op
	ev.ctx.trxCtx.RecordGasFee(op.Voter.Value, constants.CommonOpGas)
	voterAccount := table.NewSoAccountWrap(ev.ctx.db, op.Voter)
	voteCnt := voterAccount.GetBpVoteCount()

	voterId := &prototype.BpVoterId{Voter: op.Voter, Witness: op.Witness}
	witnessId := &prototype.BpWitnessId{Voter: op.Voter, Witness: op.Witness}
	vidWrap := table.NewSoWitnessVoteWrap(ev.ctx.db, voterId)

	witAccWrap := table.NewSoAccountWrap(ev.ctx.db, op.Voter)
	mustSuccess(witAccWrap.CheckExist(), "witness account do not exist ", prototype.StatusErrorDbExist)

	witnessWrap := table.NewSoWitnessWrap(ev.ctx.db, op.Witness)

	if op.Cancel {
		mustSuccess(voteCnt > 0, "vote count must not be 0", prototype.StatusErrorTrxValueCompare)
		mustSuccess(vidWrap.CheckExist(), "vote record not exist", prototype.StatusErrorDbExist)
		mustSuccess(vidWrap.RemoveWitnessVote(), "remove vote record error", prototype.StatusErrorDbDelete)
		mustSuccess(witnessWrap.GetVoteCount() > 0, "witness data error", prototype.StatusErrorTrxValueCompare)
		mustSuccess(witnessWrap.MdVoteCount(witnessWrap.GetVoteCount()-1), "set witness data error", prototype.StatusErrorDbUpdate)
		mustSuccess(voterAccount.MdBpVoteCount(voteCnt-1), "set voter data error", prototype.StatusErrorDbUpdate)
	} else {
		mustSuccess(voteCnt < constants.MAX_BP_VOTE_COUNT, "vote count exceeding", prototype.StatusErrorTrxValueCompare)

		mustNoError(vidWrap.Create(func(t *table.SoWitnessVote) {
			t.VoteTime = ev.ctx.control.HeadBlockTime()
			t.VoterId = voterId
			t.WitnessId = witnessId
		}), "add vote record error", prototype.StatusErrorDbCreate)

		mustSuccess(voterAccount.MdBpVoteCount(voteCnt+1), "set voter data error", prototype.StatusErrorDbUpdate)
		mustSuccess(witnessWrap.MdVoteCount(witnessWrap.GetVoteCount()+1), "set witness data error", prototype.StatusErrorDbUpdate)
	}

}

func (ev *FollowEvaluator) Apply() {
	op := ev.op
	ev.ctx.trxCtx.RecordGasFee(op.Account.Value, constants.CommonOpGas)
	acctWrap := table.NewSoAccountWrap(ev.ctx.db, op.Account)
	mustSuccess(acctWrap.CheckExist(), "follow account do not exist ", prototype.StatusErrorDbExist)

	acctWrap = table.NewSoAccountWrap(ev.ctx.db, op.FAccount)
	mustSuccess(acctWrap.CheckExist(), "follow f_account do not exist ", prototype.StatusErrorDbExist)
}

func (ev *TransferToVestingEvaluator) Apply() {
	op := ev.op
	ev.ctx.trxCtx.RecordGasFee(op.From.Value, constants.CommonOpGas)
	fidWrap := table.NewSoAccountWrap(ev.ctx.db, op.From)
	tidWrap := table.NewSoAccountWrap(ev.ctx.db, op.To)

	mustSuccess(tidWrap.CheckExist(), "to account do not exist", prototype.StatusErrorDbExist)

	fBalance := fidWrap.GetBalance()
	tVests := tidWrap.GetVestingShares()
	addVests := prototype.NewVest(op.Amount.Value)

	mustNoError(fBalance.Sub(op.Amount), "balance not enough", prototype.StatusErrorTrxMath)
	mustSuccess(fidWrap.MdBalance(fBalance), "set from new balance error", prototype.StatusErrorDbUpdate)

	mustNoError(tVests.Add(addVests), "vests error", prototype.StatusErrorTrxMath)
	mustSuccess(tidWrap.MdVestingShares(tVests), "set to new vests error", prototype.StatusErrorDbUpdate)

	ev.ctx.control.TransferToVest(op.Amount)
}

func (ev *ClaimEvaluator) Apply() {
	op := ev.op
	ev.ctx.trxCtx.RecordGasFee(op.Account.Value, constants.CommonOpGas)
	account := op.Account
	accWrap := table.NewSoAccountWrap(ev.ctx.db, account)

	mustSuccess(accWrap.CheckExist(), "claim account do not exist", prototype.StatusErrorDbExist)

	var i int32 = 1
	keeperWrap := table.NewSoRewardsKeeperWrap(ev.ctx.db, &i)
	mustSuccess(keeperWrap.CheckExist(), "reward keeper do not exist", prototype.StatusErrorDbExist)

	keeper := keeperWrap.GetKeeper()
	innerRewards := keeper.Rewards

	amount := op.Amount

	if val, ok := innerRewards[account.Value]; ok {
		rewardBalance := val.Value
		var reward uint64
		if rewardBalance >= amount && rewardBalance-amount <= rewardBalance {
			reward = amount
		} else {
			reward = rewardBalance
		}
		if reward > 0 {
			vestingBalance := accWrap.GetVestingShares()
			accWrap.MdVestingShares(&prototype.Vest{Value: vestingBalance.Value + reward})
			val.Value -= reward
			keeperWrap.MdKeeper(keeper)
		} else {
			// do nothing
		}
	} else {
		mustSuccess(ok, "No remains reward on chain", prototype.StatusError)
	}

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
	ev.ctx.trxCtx.RecordGasFee(op.Reporter.Value, constants.CommonOpGas)
	post := table.NewSoPostWrap(ev.ctx.db, &op.Reported)
	mustSuccess(post.CheckExist(), "the reported post doesn't exist", prototype.StatusErrorDbExist)
	report := table.NewSoReportListWrap(ev.ctx.db, &op.Reported)
	if op.IsArbitration {
		mustSuccess(report.CheckExist(), "cannot arbitrate a non-existed post", prototype.StatusErrorDbExist)
		if op.IsApproved {
			post.RemovePost()
			report.RemoveReportList()
			return
		}

		report.MdIsArbitrated(true)
	} else {
		if report.CheckExist() {
			if report.GetIsArbitrated() {
				mustSuccess(false, "cannot report a legal post", prototype.StatusError)
			}
			report.MdReportedTimes(report.GetReportedTimes() + 1)
			existedTags := report.GetTags()
			newTags := op.ReportTag
			report.MdTags(mergeTags(existedTags, newTags))
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

func (ev *ClaimAllEvaluator) Apply() {
	op := ev.op
	ev.ctx.trxCtx.RecordGasFee(op.Account.Value, constants.CommonOpGas)
	account := op.Account
	accWrap := table.NewSoAccountWrap(ev.ctx.db, account)

	mustSuccess(accWrap.CheckExist(), "claim account do not exist", prototype.StatusErrorDbExist)

	var i int32 = 1
	keeperWrap := table.NewSoRewardsKeeperWrap(ev.ctx.db, &i)
	mustSuccess(keeperWrap.CheckExist(), "reward keeper do not exist", prototype.StatusErrorDbExist)

	keeper := keeperWrap.GetKeeper()
	innerRewards := keeper.Rewards

	if val, ok := innerRewards[account.Value]; ok {
		reward := val.Value
		if reward > 0 {
			vestingBalance := accWrap.GetVestingShares()
			accWrap.MdVestingShares(&prototype.Vest{Value: vestingBalance.Value + reward})
			val.Value -= reward
			keeperWrap.MdKeeper(keeper)
		} else {
			// do nothing
		}
	} else {
		mustSuccess(ok, "No remains reward on chain", prototype.StatusError)
	}

}

func (ev *ContractDeployEvaluator) Apply() {
	op := ev.op

	ev.ctx.trxCtx.RecordGasFee(op.Owner.Value, constants.CommonOpGas)

	cid := prototype.ContractId{Owner: op.Owner, Cname: op.Contract}
	scid := table.NewSoContractWrap(ev.ctx.db, &cid)

	mustSuccess(!scid.CheckExist(), "contract name exist", prototype.StatusErrorDbExist)

	_, err := abi.UnmarshalABI([]byte(op.GetAbi()))
	if err != nil {
		mustNoError(err, "invalid contract abi", prototype.StatusErrorAbi)
	}

	vmCtx := vmcontext.NewContextFromDeployOp(op, nil)

	cosVM := vm.NewCosVM(vmCtx, nil, nil, nil)

	mustNoError(cosVM.Validate(), "validate code failed", prototype.StatusErrorWasm)

	mustNoError(scid.Create(func(t *table.SoContract) {
		t.Code = op.Code
		t.Id = &cid
		t.CreatedTime = ev.ctx.control.HeadBlockTime()
		t.Abi = op.Abi
		t.Balance = prototype.NewCoin(0)
	}), "create contract data error", prototype.StatusErrorDbCreate)
}

//func (ev *ContractEstimateApplyEvaluator) Apply() {
//	op := ev.op
//	cid := prototype.ContractId{Owner: op.Owner, Cname: op.Contract}
//	scid := table.NewSoContractWrap(ev.ctx.db, &cid)
//	opAssert(scid.CheckExist(), "contract name doesn't exist")
//	acc := table.NewSoAccountWrap(ev.ctx.db, op.Caller)
//	opAssert(acc.CheckExist(), "account doesn't exist")
//
//	code := scid.GetCode()
//	vmCtx := &vmcontext.Context{Code: code, Caller: op.Caller,
//		Owner: op.Owner, Gas: &prototype.Coin{Value: math.MaxUint64}, Contract: op.Contract, Injector: ev.ctx.trxCtx}
//	cosVM := vm.NewCosVM(vmCtx, ev.ctx.db, ev.ctx.control.GetProps(), logrus.New())
//	spent, err := cosVM.Estimate()
//	if err != nil {
//		vmCtx.Injector.Error(500, err.Error())
//	} else {
//		vmCtx.Injector.Log(fmt.Sprintf("Estimate the operation would spent %d gas, "+
//			"the result does not include storage cost, and some edge fee. "+
//			"Recommend you pay some extra tips to cover the charge", spent))
//	}
//}

func (ev *ContractEstimateApplyEvaluator) Apply() {
	//panic("not yet implement")
	ev.ctx.trxCtx.Error(500, "high risk as malicious contract, deprecated.")
}

func (ev *ContractApplyEvaluator) Apply() {
	op := ev.op

	cid := prototype.ContractId{Owner: op.Owner, Cname: op.Contract}
	scid := table.NewSoContractWrap(ev.ctx.db, &cid)
	mustSuccess(scid.CheckExist(), "contract name doesn't exist", prototype.StatusErrorDbExist)

	acc := table.NewSoAccountWrap(ev.ctx.db, op.Caller)
	mustSuccess(acc.CheckExist(), "account doesn't exist", prototype.StatusErrorDbExist)

	balance := acc.GetBalance().Value

	// the amount is also minicos or cos ?
	// here I assert it is minicos
	// also, I think balance base on minicos is far more reliable.
	if op.Amount != nil {
		mustSuccess(balance > op.Amount.Value, "balance does not have enough fund to transfer", prototype.StatusErrorTrxValueCompare)
	}
	code := scid.GetCode()

	var err error
	var abiInterface abi.IContractABI
	var paramsData []byte
	var tables *ct.ContractTables

	if abiInterface, err = abi.UnmarshalABI([]byte(scid.GetAbi())); err != nil {
		mustNoError(err, "invalid contract abi", prototype.StatusErrorAbi)
	}
	if m := abiInterface.MethodByName(op.Method); m != nil {
		paramsData, err = vme.EncodeFromJson([]byte(op.Params), m.Args().Type())
		if err != nil {
			mustNoError(err, "invalid contract parameters", prototype.StatusErrorAbi)
		}
	} else {
		mustSuccess(false, "unknown contract method: "+op.Method, prototype.StatusErrorMethod)
	}

	if abiInterface != nil {
		tables = ct.NewContractTables(op.Owner.Value, op.Contract, abiInterface, ev.ctx.db)
	}

	vmCtx := vmcontext.NewContextFromApplyOp(op, paramsData, code, abiInterface, tables, ev.ctx.trxCtx)
	// set max gas
	remain := ev.ctx.trxCtx.GetVmRemainCpuStamina(op.Caller.Value)
	remainGas := remain * constants.CpuConsumePointDen
	if remainGas > constants.MaxGasPerCall {
		vmCtx.Gas = constants.MaxGasPerCall
	} else {
		vmCtx.Gas = remainGas
	}
	// turn off gas limit
	if !ev.ctx.control.ctx.Config().ResourceCheck {
		vmCtx.Gas = constants.OneDayStamina * constants.CpuConsumePointDen
	}
	// should be active ?
	//defer func() {
	//	_ := recover()
	//}()

	cosVM := vm.NewCosVM(vmCtx, ev.ctx.db, ev.ctx.control.GetProps(), logrus.New())

	ret, err := cosVM.Run()
	spentGas := cosVM.SpentGas()
	// need extra query db, is it a good way or should I pass account object as parameter?
	// deductgasfee and usertranfer could be panic (rarely, I can't image how it happens)
	// the panic should catch then return or bubble it ?
	vmCtx.Injector.RecordGasFee(op.Caller.Value, spentGas)
	if err != nil {
		vmCtx.Injector.Error(ret, err.Error())
		mustNoError(err, "internal contract apply failed", prototype.StatusErrorWasm)
	} else {
		if op.Amount != nil && op.Amount.Value > 0 {
			vmCtx.Injector.TransferFromUserToContract(op.Caller.Value, op.Contract, op.Owner.Value, op.Amount.Value)
		}
	}
}

func (ev *InternalContractApplyEvaluator) Apply() {
	op := ev.op

	fromContract := table.NewSoContractWrap(ev.ctx.db, &prototype.ContractId{Owner: op.FromOwner, Cname: op.FromContract})
	mustSuccess(fromContract.CheckExist(), "fromContract contract doesn't exist", prototype.StatusErrorDbExist)

	toContract := table.NewSoContractWrap(ev.ctx.db, &prototype.ContractId{Owner: op.ToOwner, Cname: op.ToContract})
	mustSuccess(toContract.CheckExist(), "toContract contract doesn't exist", prototype.StatusErrorDbExist)

	caller := table.NewSoAccountWrap(ev.ctx.db, op.FromCaller)
	mustSuccess(caller.CheckExist(), "caller account doesn't exist", prototype.StatusErrorDbExist)

	mustSuccess(fromContract.GetBalance().Value >= op.Amount.Value, "fromContract balance less than transfer amount", prototype.StatusErrorTrxValueCompare)

	if op.Amount != nil {
		mustSuccess(fromContract.GetBalance().Value >= op.Amount.Value, "fromContract balance less than transfer amount", prototype.StatusErrorTrxValueCompare)
	}
	code := toContract.GetCode()

	var err error
	var abiInterface abi.IContractABI
	var tables *ct.ContractTables

	if abiInterface, err = abi.UnmarshalABI([]byte(toContract.GetAbi())); err != nil {
		mustNoError(err, "invalid toContract abi", prototype.StatusErrorAbi)
	}
	if m := abiInterface.MethodByName(op.ToMethod); m != nil {
		_, err = vme.DecodeToJson(op.Params, m.Args().Type(), false)
		if err != nil {
			mustNoError(err, "invalid contract parameters", prototype.StatusErrorAbi)
		}
	} else {
		mustSuccess(false, "unknown contract method: "+op.ToMethod, prototype.StatusErrorMethod)
	}

	if abiInterface != nil {
		tables = ct.NewContractTables(op.ToOwner.Value, op.ToContract, abiInterface, ev.ctx.db)
	}

	vmCtx := vmcontext.NewContextFromInternalApplyOp(op, code, abiInterface, tables, ev.ctx.trxCtx)
	// set remain gas
	vmCtx.Gas = ev.remainGas
	cosVM := vm.NewCosVM(vmCtx, ev.ctx.db, ev.ctx.control.GetProps(), logrus.New())

	ev.ctx.db.BeginTransaction()
	ret, err := cosVM.Run()
	spentGas := cosVM.SpentGas()
	vmCtx.Injector.RecordGasFee(op.FromCaller.Value, spentGas)
	if err != nil {
		vmCtx.Injector.Error(ret, err.Error())
		ev.ctx.db.EndTransaction(false)
		// throw a panic, this panic should recover by upper contract vm context
		mustNoError(err, "internal contract apply failed", prototype.StatusErrorWasm)
	} else {
		if op.Amount != nil && op.Amount.Value > 0 {
			vmCtx.Injector.TransferFromContractToContract(op.FromContract, op.FromOwner.Value, op.ToContract, op.ToOwner.Value, op.Amount.Value)
		}
		ev.ctx.db.EndTransaction(true)
	}
}

func (ev *StakeEvaluator) Apply() {
	op := ev.op
	ev.ctx.trxCtx.RecordGasFee(op.Account.Value, constants.CommonOpGas)
	accountWrap := table.NewSoAccountWrap(ev.ctx.db, op.Account)

	value := &prototype.Coin{Value: op.Amount}

	fBalance := accountWrap.GetBalance()
	mustNoError(fBalance.Sub(value), "Insufficient balance to transfer.", prototype.StatusErrorTrxMath)
	mustSuccess(accountWrap.MdBalance(fBalance), "modify balance failed", prototype.StatusErrorDbUpdate)

	vest := accountWrap.GetStakeVesting()
	mustNoError(vest.Add(value.ToVest()), "vesting over flow.", prototype.StatusErrorTrxMath)
	mustSuccess(accountWrap.MdStakeVesting(vest), "modify vesting failed", prototype.StatusErrorDbUpdate)

	headBlockTime := ev.ctx.control.headBlockTime()
	accountWrap.MdLastStakeTime(headBlockTime)

	ev.ctx.control.TransferToVest(value)
}

func (ev *UnStakeEvaluator) Apply() {
	op := ev.op
	ev.ctx.trxCtx.RecordGasFee(op.Account.Value, constants.CommonOpGas)

	accountWrap := table.NewSoAccountWrap(ev.ctx.db, op.Account)

	headBlockTime := ev.ctx.control.headBlockTime()
	stakeTime := accountWrap.GetLastStakeTime()
	mustSuccess(headBlockTime.UtcSeconds-stakeTime.UtcSeconds > constants.StakeFreezeTime, "can not unstake when freeze", prototype.StatusError)

	value := &prototype.Coin{Value: op.Amount}

	vest := accountWrap.GetStakeVesting()
	mustNoError(vest.Sub(value.ToVest()), "vesting over flow.", prototype.StatusErrorTrxMath)
	mustSuccess(accountWrap.MdStakeVesting(vest), "modify vesting failed", prototype.StatusErrorDbUpdate)

	fBalance := accountWrap.GetBalance()
	mustNoError(fBalance.Add(value), "Insufficient balance to transfer.", prototype.StatusErrorTrxMath)
	mustSuccess(accountWrap.MdBalance(fBalance), "modify balance failed", prototype.StatusErrorDbUpdate)

	ev.ctx.control.TransferFromVest(value.ToVest())
}
