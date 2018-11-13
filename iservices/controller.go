package iservices

import (
	"github.com/coschain/contentos-go/prototype"
)

//
// This file defines interfaces of Database service.
//

var CTRL_SERVER_NAME = "ctrl"

type IController interface {
	PushTrx(trx *prototype.SignedTransaction) *prototype.TransactionInvoice
	PushBlock(blk *prototype.SignedBlock)
	CreateVesting(accountName *prototype.AccountName, cos *prototype.Coin) *prototype.Vest
	AddBalance(accountName *prototype.AccountName, cos *prototype.Coin)
	SubBalance(accountName *prototype.AccountName, cos *prototype.Coin)
}
