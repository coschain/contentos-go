package app

import (
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
)

func mustNoError( err error )  {
	if ( err != nil ){
		panic(err)
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
	res := true
	err := newAccountWrap.Create(func(tInfo *table.SoAccount) {
		tInfo.Name = op.NewAccountName
		tInfo.Creator = op.Creator
		tInfo.CreatedTime = dgpWrap.GetTime()
		tInfo.PubKey = op.MemoKey
		cos := prototype.NewCoin(0)
		vest := prototype.NewVest(0)
		tInfo.Balance = cos
		tInfo.VestingShares = vest
	})
	if err != nil {
		res = false
	}
	mustSuccess(res , "duplicate create account object")

	// create account authority
	authorityWrap := table.NewSoAccountAuthorityObjectWrap(ev.ctx.db,op.NewAccountName)
	cRes := true
	err = authorityWrap.Create(func(tInfo *table.SoAccountAuthorityObject) {
		tInfo.Account = op.NewAccountName
		tInfo.Posting = op.Posting
		tInfo.Active = op.Active
		tInfo.Owner = op.Owner
		tInfo.LastOwnerUpdate = &prototype.TimePointSec{UtcSeconds:0}
	})
	if err != nil {
		cRes = false
	}
	mustSuccess(cRes , "duplicate create account authority object")

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
