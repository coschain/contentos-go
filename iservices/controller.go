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
	HeadBlockTime() *prototype.TimePointSec

	TransferToVest( value *prototype.Coin)
	TransferFromVest( value *prototype.Vest)

}
