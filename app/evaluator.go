package app

import (
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/iservices"
	"fmt"
)

type BaseEvaluator interface {
	Apply(op *prototype.Operation)
}


type AccountCreateEvaluator struct{
	db iservices.IDatabaseService
	control iservices.IController
}

type TransferEvaluator struct{
	db iservices.IDatabaseService
	control iservices.IController
}

func (ev *AccountCreateEvaluator) SetDB(db iservices.IDatabaseService){
	ev.db = db
}

func  (ev *AccountCreateEvaluator) SetController(c iservices.IController){
	ev.control = c
}

func (ev *AccountCreateEvaluator) Apply(operation *prototype.Operation) {
	// write DB
	 o,ok := operation.Op.(*prototype.Operation_Op1)
	 if !ok {
		panic("type cast failed")
	}
	op := o.Op1
	creatorWrap := table.NewSoAccountWrap(ev.db,op.Creator)
	fmt.Println("1",creatorWrap)
	fmt.Println("2",op)
	if creatorWrap.GetBalance().Amount.Value < op.Fee.Amount.Value {
		panic("Insufficient balance to create account.")
	}

	// check auth accounts
	for _,a := range op.Owner.AccountAuths {
		tmpAccountWrap := table.NewSoAccountWrap(ev.db,a.Name)
		if !tmpAccountWrap.CheckExist() {
			panic("owner auth account not exist")
		}
	}
	for _,a := range op.Active.AccountAuths {
		tmpAccountWrap := table.NewSoAccountWrap(ev.db,a.Name)
		if !tmpAccountWrap.CheckExist() {
			panic("active auth account not exist")
		}
	}
	for _,a := range op.Posting.AccountAuths {
		tmpAccountWrap := table.NewSoAccountWrap(ev.db,a.Name)
		if !tmpAccountWrap.CheckExist() {
			panic("posting auth account not exist")
		}
	}

	// sub creator's fee
	originBalance := creatorWrap.GetBalance()
	originBalance.Amount.Value -= op.Fee.Amount.Value
	creatorWrap.MdBalance(*originBalance)

	// sub dynamic glaobal properties's total fee
	var i int32 = 0
	dgpWrap := table.NewSoDynamicGlobalPropertiesWrap(ev.db,&i)
	originTotal := dgpWrap.GetTotalCos()
	originTotal.Amount.Value -= op.Fee.Amount.Value
	dgpWrap.MdTotalCos(*originTotal)

	// create account
	newAccountWrap := table.NewSoAccountWrap(ev.db,op.NewAccountName)
	newAccount := &table.SoAccount{}
	newAccount.Name = op.NewAccountName
	newAccount.Creator = op.Creator
	newAccount.CreatedTime = dgpWrap.GetTime()
	newAccountWrap.CreateAccount(newAccount)

	// create account authority
	authorityWrap := table.NewSoAccountAuthorityObjectWrap(ev.db,op.NewAccountName)
	authority := &table.SoAccountAuthorityObject{}
	authority.Account = op.NewAccountName
	authority.Posting = op.Posting
	authority.Active = op.Active
	authority.Owner = op.Owner
	authority.LastOwnerUpdate = &prototype.TimePointSec{UtcSeconds:0}
	authorityWrap.CreateAccountAuthorityObject(authority)

	// create vesting
	if op.Fee.Amount.Value > 0 {
		ev.control.CreateVesting(op.NewAccountName,op.Fee)
	}
}

func (ev *TransferEvaluator) Apply(operation *prototype.Operation) {
	// write DB
	o,ok := operation.Op.(*prototype.Operation_Op2)
	if !ok {
		panic("type cast failed")
	}
	op := o.Op2

	// @ active_challenged

	fromWrap := table.NewSoAccountWrap(ev.db,op.From)
	if fromWrap.GetBalance().Amount.Value < op.Amount.Amount.Value {
		panic("Insufficient balance to transfer.")
	}

	ev.control.SubBalance(op.From,op.Amount)
	ev.control.AddBalance(op.To,op.Amount)
}
