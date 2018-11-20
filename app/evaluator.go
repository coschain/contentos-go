package app

import (
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/prototype"
)

func mustNoError(err error) {
	if err != nil {
		panic(err)
	}
}
func mustSuccess(b bool, val string) {
	if !b {
		panic(val)
	}
}

type ApplyContext struct {
	db      iservices.IDatabaseService
	control iservices.IController
}

type BaseEvaluator interface {
	Apply()
}

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

func (ev *AccountCreateEvaluator) Apply() {
	op := ev.op
	creatorWrap := table.NewSoAccountWrap(ev.ctx.db, op.Creator)

	opAssert(creatorWrap.CheckExist(), "creator not exist ")

	opAssert(creatorWrap.GetBalance().Value >= op.Fee.Value, "Insufficient balance to create account.")

	// check auth accounts
	for _, a := range op.Owner.AccountAuths {
		tmpAccountWrap := table.NewSoAccountWrap(ev.ctx.db, a.Name)
		opAssert(tmpAccountWrap.CheckExist(), "owner auth account not exist")
	}
	for _, a := range op.Active.AccountAuths {
		tmpAccountWrap := table.NewSoAccountWrap(ev.ctx.db, a.Name)
		opAssert(tmpAccountWrap.CheckExist(), "active auth account not exist")
	}
	for _, a := range op.Posting.AccountAuths {
		tmpAccountWrap := table.NewSoAccountWrap(ev.ctx.db, a.Name)
		opAssert(tmpAccountWrap.CheckExist(), "posting auth account not exist")
	}

	// sub creator's fee
	originBalance := creatorWrap.GetBalance()
	originBalance.Value -= op.Fee.Value
	creatorWrap.MdBalance(originBalance)

	// sub dynamic glaobal properties's total fee
	var i int32 = 0
	dgpWrap := table.NewSoDynamicGlobalPropertiesWrap(ev.ctx.db, &i)
	originTotal := dgpWrap.GetTotalCos()
	originTotal.Value -= op.Fee.Value
	dgpWrap.MdTotalCos(originTotal)

	// create account
	newAccountWrap := table.NewSoAccountWrap(ev.ctx.db, op.NewAccountName)
	opAssertE(newAccountWrap.Create(func(tInfo *table.SoAccount) {
		tInfo.Name = op.NewAccountName
		tInfo.Creator = op.Creator
		tInfo.CreatedTime = dgpWrap.GetTime()
		tInfo.PubKey = op.MemoKey
		tInfo.Balance = prototype.NewCoin(0)
		tInfo.VestingShares = prototype.NewVest(0)
	}), "duplicate create account object")

	// create account authority
	authorityWrap := table.NewSoAccountAuthorityObjectWrap(ev.ctx.db, op.NewAccountName)
	opAssertE(authorityWrap.Create(func(tInfo *table.SoAccountAuthorityObject) {
		tInfo.Account = op.NewAccountName
		tInfo.Posting = op.Posting
		tInfo.Active = op.Active
		tInfo.Owner = op.Owner
		tInfo.LastOwnerUpdate = prototype.NewTimePointSec(0)
	}), "duplicate create account authority object")

	// create vesting
	if op.Fee.Value > 0 {
		ev.ctx.control.CreateVesting(op.NewAccountName, op.Fee)
	}
}

func (ev *TransferEvaluator) Apply() {
	op := ev.op

	// @ active_challenged
	fromWrap := table.NewSoAccountWrap(ev.ctx.db, op.From)
	opAssert(fromWrap.GetBalance().Value >= op.Amount.Value, "Insufficient balance to transfer.")
	ev.ctx.control.SubBalance(op.From, op.Amount)
	ev.ctx.control.AddBalance(op.To, op.Amount)
}

func (ev *ReplyEvaluator) Apply() {
	op := ev.op
	cidWrap := table.NewSoPostWrap(ev.ctx.db, &op.Uuid)
	pidWrap := table.NewSoPostWrap(ev.ctx.db, &op.ParentUuid)

	opAssert(!cidWrap.CheckExist(), "post uuid exist")
	opAssert(pidWrap.CheckExist(), "parent uuid do not exist")

	opAssert(pidWrap.GetDepth()+1 < constants.POST_MAX_DEPTH, "reply depth error")

	opAssertE(cidWrap.Create(func(t *table.SoPost) {
		t.PostId = op.Uuid
		t.Tags = nil
		t.Title = ""
		t.Author = op.Owner
		t.Body = op.Content
		t.Created = ev.ctx.control.HeadBlockTime()
		t.LastPayout = prototype.NewTimePointSec(0) //TODO
		t.Depth = pidWrap.GetDepth() + 1
		t.Children = 0
		t.RootId = pidWrap.GetRootId()
		t.ParentId = constants.POST_INVALID_ID
		t.VoteCnt = 0

		// TODO  CreatedOrder / ReplyOrder sort
		// maybe should implement in plugin-services

	}), "create reply error")

	// Modify Parent Object
	opAssert(pidWrap.MdChildren(pidWrap.GetChildren()+1), "Modify Parent Children Error")
}

func (ev *PostEvaluator) Apply() {
	op := ev.op
	idWrap := table.NewSoPostWrap(ev.ctx.db, &op.Uuid)

	opAssert(!idWrap.CheckExist(), "post uuid exist")

	opAssertE(idWrap.Create(func(t *table.SoPost) {
		t.PostId = op.Uuid
		t.Tags = op.Tags
		t.Title = op.Title
		t.Author = op.Owner
		t.Body = op.Content
		t.Created = ev.ctx.control.HeadBlockTime()
		t.LastPayout = prototype.NewTimePointSec(0) //TODO
		t.Depth = 0
		t.Children = 0
		t.RootId = t.PostId
		t.ParentId = constants.POST_INVALID_ID
		t.VoteCnt = 0

		// TODO  CreatedOrder / ReplyOrder sort
		// maybe should implement in plugin-services

	}), "create post error")
}

func (ev *VoteEvaluator) Apply() {
	op := ev.op

	voterId := prototype.VoterId{Voter: op.Voter, PostId: op.Idx}
	vidWrap := table.NewSoVoteWrap(ev.ctx.db, &voterId)
	pidWrap := table.NewSoPostWrap(ev.ctx.db, &op.Idx)

	opAssert(!pidWrap.CheckExist(), "post invalid")
	opAssert(!vidWrap.CheckExist(), "vote info exist")

	opAssertE(vidWrap.Create(func(t *table.SoVote) {
		t.Voter = &voterId
		t.PostId = op.Idx
		t.VoteTime = ev.ctx.control.HeadBlockTime()
	}), "create voter object error")

	opAssert(pidWrap.MdVoteCnt(pidWrap.GetVoteCnt()+1), "set vote count error")
}

func (ev *BpRegisterEvaluator) Apply() {
	op := ev.op
	witnessWrap := table.NewSoWitnessWrap(ev.ctx.db, op.Owner)

	opAssert(witnessWrap.CheckExist(), "witness already exist")

	opAssertE(witnessWrap.Create(func(t *table.SoWitness) {
		t.Owner = op.Owner
		t.CreatedTime = ev.ctx.control.HeadBlockTime()
		t.Url = op.Url
		t.SigningKey = op.BlockSigningKey

		// TODO add others
	}), "add witness record error")
}

func (ev *BpUnregisterEvaluator) Apply() {
	panic("not yet implement")
}

func (ev *BpVoteEvaluator) Apply() {
	op := ev.op

	voterAccount := table.NewSoAccountWrap(ev.ctx.db, op.Voter)
	voteCnt := voterAccount.GetBpVoteCount()

	voterId := &prototype.BpVoterId{Voter: op.Voter, Witness: op.Witness}
	witnessId := &prototype.BpWitnessId{Voter: op.Voter, Witness: op.Witness}
	vidWrap := table.NewSoWitnessVoteWrap(ev.ctx.db, voterId)

	witnessWrap := table.NewSoWitnessWrap(ev.ctx.db, op.Witness)

	if op.Cancel {
		opAssert(voteCnt > 0, "vote count must not be 0")
		opAssert(vidWrap.CheckExist(), "vote record not exist")
		opAssert(vidWrap.RemoveWitnessVote(), "remove vote record error")
		opAssert(witnessWrap.GetVoteCount() > 0, "witness data error")
		opAssert(witnessWrap.MdVoteCount(witnessWrap.GetVoteCount()-1), "set witness data error")
		opAssert(voterAccount.MdBpVoteCount(voteCnt-1), "set voter data error")
	} else {
		opAssert(voteCnt < constants.MAX_BP_VOTE_COUNT, "vote count exceeding")

		opAssertE(vidWrap.Create(func(t *table.SoWitnessVote) {
			t.VoteTime = ev.ctx.control.HeadBlockTime()
			t.VoterId = voterId
			t.WitnessId = witnessId
		}), "add vote record error")

		opAssert(voterAccount.MdBpVoteCount(voteCnt+1), "set voter data error")
		opAssert(witnessWrap.MdVoteCount(witnessWrap.GetVoteCount()+1), "set witness data error")
	}

}
