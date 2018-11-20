package app

import (
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
)

func mustNoError( err error, val string )  {
	if ( err != nil ){
		panic( val + " : "+ err.Error() )
	}
}
func mustSuccess( b bool , val string)  {
	if ( !b ){
		panic(val)
	}
}

type ApplyContext struct {
	db iservices.IDatabaseService
	control iservices.IController
}

type BaseEvaluator interface {
	Apply()
}


type AccountCreateEvaluator struct{
	BaseEvaluator
	ctx *ApplyContext
	op *prototype.AccountCreateOperation
}

type TransferEvaluator struct{
	BaseEvaluator
	ctx *ApplyContext
	op *prototype.TransferOperation
}

type PostEvaluator struct{
	BaseEvaluator
	ctx *ApplyContext
	op *prototype.PostOperation
}
type ReplyEvaluator struct{
	BaseEvaluator
	ctx *ApplyContext
	op *prototype.ReplyOperation
}
type VoteEvaluator struct{
	BaseEvaluator
	ctx *ApplyContext
	op *prototype.VoteOperation
}

func (ev *AccountCreateEvaluator) Apply() {
	op := ev.op
	creatorWrap := table.NewSoAccountWrap(ev.ctx.db,op.Creator)

	mustSuccess( creatorWrap.CheckExist() , "creator not exist ")

	mustSuccess( creatorWrap.GetBalance().Value >= op.Fee.Value , "Insufficient balance to create account.")


	// check auth accounts
	for _,a := range op.Owner.AccountAuths {
		tmpAccountWrap := table.NewSoAccountWrap(ev.ctx.db,a.Name)
		mustSuccess( tmpAccountWrap.CheckExist(), "owner auth account not exist")
	}
	for _,a := range op.Active.AccountAuths {
		tmpAccountWrap := table.NewSoAccountWrap(ev.ctx.db,a.Name)
		mustSuccess( tmpAccountWrap.CheckExist(), "active auth account not exist")
	}
	for _,a := range op.Posting.AccountAuths {
		tmpAccountWrap := table.NewSoAccountWrap(ev.ctx.db,a.Name)
		mustSuccess( tmpAccountWrap.CheckExist(), "posting auth account not exist")
	}

	// sub creator's fee
	originBalance := creatorWrap.GetBalance()
	originBalance.Value -= op.Fee.Value
	creatorWrap.MdBalance(originBalance)

	// sub dynamic glaobal properties's total fee
	var i int32 = 0
	dgpWrap := table.NewSoDynamicGlobalPropertiesWrap(ev.ctx.db,&i)
	originTotal := dgpWrap.GetTotalCos()
	originTotal.Value -= op.Fee.Value
	dgpWrap.MdTotalCos(originTotal)

	// create account
	newAccountWrap := table.NewSoAccountWrap(ev.ctx.db,op.NewAccountName)
	mustNoError( newAccountWrap.Create(func(tInfo *table.SoAccount) {
		tInfo.Name             = op.NewAccountName
		tInfo.Creator          = op.Creator
		tInfo.CreatedTime      = dgpWrap.GetTime()
		tInfo.PubKey           = op.MemoKey
		tInfo.Balance          = prototype.NewCoin(0)
		tInfo.VestingShares    = prototype.NewVest(0)
	}) , "duplicate create account object")


	// create account authority
	authorityWrap := table.NewSoAccountAuthorityObjectWrap(ev.ctx.db,op.NewAccountName)
	mustNoError( authorityWrap.Create(func(tInfo *table.SoAccountAuthorityObject) {
		tInfo.Account            = op.NewAccountName
		tInfo.Posting            = op.Posting
		tInfo.Active             = op.Active
		tInfo.Owner              = op.Owner
		tInfo.LastOwnerUpdate    = prototype.NewTimePointSec(0)
	}) , "duplicate create account authority object")

	// create vesting
	if op.Fee.Value > 0 {
		ev.ctx.control.CreateVesting(op.NewAccountName,op.Fee)
	}
}

func (ev *TransferEvaluator) Apply() {
	op := ev.op

	// @ active_challenged
	fromWrap := table.NewSoAccountWrap(ev.ctx.db,op.From)
	mustSuccess( fromWrap.GetBalance().Value >= op.Amount.Value, "Insufficient balance to transfer.")
	ev.ctx.control.SubBalance(op.From,op.Amount)
	ev.ctx.control.AddBalance(op.To,op.Amount)
}

/*
PostId               uint64                  `protobuf:"varint,1,opt,name=post_id,json=postId,proto3" json:"post_id,omitempty"`
Category             string                  `protobuf:"bytes,2,opt,name=category,proto3" json:"category,omitempty"`
Author               *prototype.AccountName  `protobuf:"bytes,3,opt,name=author,proto3" json:"author,omitempty"`
Title                string                  `protobuf:"bytes,4,opt,name=title,proto3" json:"title,omitempty"`
Body                 string                  `protobuf:"bytes,5,opt,name=body,proto3" json:"body,omitempty"`
Tags                 []string                `protobuf:"bytes,6,rep,name=tags,proto3" json:"tags,omitempty"`
Created              *prototype.TimePointSec `protobuf:"bytes,7,opt,name=created,proto3" json:"created,omitempty"`
LastPayout           *prototype.TimePointSec `protobuf:"bytes,8,opt,name=last_payout,json=lastPayout,proto3" json:"last_payout,omitempty"`
Depth                uint32                  `protobuf:"varint,9,opt,name=depth,proto3" json:"depth,omitempty"`
Children             uint32                  `protobuf:"varint,10,opt,name=children,proto3" json:"children,omitempty"`
RootId               uint64                  `protobuf:"varint,11,opt,name=root_id,json=rootId,proto3" json:"root_id,omitempty"`
ParentId             uint64
CreatedOrder         *prototype.PostCreatedOrder `protobuf:"bytes,13,opt,name=created_order,json=createdOrder,proto3" json:"created_order,omitempty"`
ReplyOrder           *prototype.PostReplyOrder   `protobuf:"bytes,14,opt,name=reply_order,json=replyOrder,proto3" json:"reply_order,omitempty"`
*/
func (ev *ReplyEvaluator) Apply() {
	op 		:= ev.op
	cidWrap 	:= table.NewSoPostWrap( ev.ctx.db, &op.Uuid )
	pidWrap := table.NewSoPostWrap( ev.ctx.db, &op.ParentUuid )

	mustSuccess( !cidWrap.CheckExist(), "post uuid exist" )
	mustSuccess(  pidWrap.CheckExist(), "parent uuid do not exist" )

	mustSuccess( pidWrap.GetDepth() + 1 < constants.POST_MAX_DEPTH, "reply depth error")

	mustNoError( cidWrap.Create(func(t *table.SoPost) {
		t.PostId        = op.Uuid
		t.Tags          = nil
		t.Title         = ""
		t.Author        = op.Owner
		t.Body          = op.Content
		t.Created       = ev.ctx.control.HeadBlockTime()
		t.LastPayout    = prototype.NewTimePointSec(0)	//TODO
		t.Depth         = pidWrap.GetDepth() + 1
		t.Children      = 0
		t.RootId        = pidWrap.GetRootId()
		t.ParentId      = constants.POST_INVALID_ID
		t.VoteCnt		= 0

		// TODO  CreatedOrder / ReplyOrder sort
		// maybe should implement in plugin-services

	}), "create reply error")

	// Modify Parent Object
	mustSuccess( pidWrap.MdChildren( pidWrap.GetChildren() + 1 ), "Modify Parent Children Error" )
}

func (ev *PostEvaluator) Apply() {
	op 		:= ev.op
	idWrap 	:= table.NewSoPostWrap( ev.ctx.db, &op.Uuid )

	mustSuccess( !idWrap.CheckExist(), "post uuid exist" )

	mustNoError( idWrap.Create(func(t *table.SoPost) {
		t.PostId        = op.Uuid
		t.Tags          = op.Tags
		t.Title         = op.Title
		t.Author        = op.Owner
		t.Body          = op.Content
		t.Created       = ev.ctx.control.HeadBlockTime()
		t.LastPayout    = prototype.NewTimePointSec(0)	//TODO
		t.Depth         = 0
		t.Children      = 0
		t.RootId        = t.PostId
		t.ParentId      = constants.POST_INVALID_ID
		t.VoteCnt		= 0

		// TODO  CreatedOrder / ReplyOrder sort
		// maybe should implement in plugin-services

	}), "create post error" )
}


func (ev *VoteEvaluator) Apply() {
	op 		:= ev.op

	voterId := prototype.VoterId{ Voter:op.Voter, PostId:op.Idx }
	vidWrap := table.NewSoVoteWrap( ev.ctx.db, &voterId )
	pidWrap := table.NewSoPostWrap( ev.ctx.db, &op.Idx )

	mustSuccess( !pidWrap.CheckExist(), "post invalid" )
	mustSuccess( !vidWrap.CheckExist(), "vote info exist" )

	mustNoError( vidWrap.Create(func(t *table.SoVote) {
		t.Voter		= &voterId
		t.PostId	= op.Idx
		t.VoteTime	= ev.ctx.control.HeadBlockTime()
	}), "create voter object error")

	mustSuccess( pidWrap.MdVoteCnt( pidWrap.GetVoteCnt() + 1 ), "set vote count error" )
}