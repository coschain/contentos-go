package iservices

import (
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/prototype"
)

//
// This file defines interfaces of Database service.
//

var CTRL_SERVER_NAME = "ctrl"

type IController interface {
	PushTrx(trx *prototype.SignedTransaction) *prototype.TransactionInvoice
	PushBlock(blk *prototype.SignedBlock,skip prototype.SkipFlag)
	HeadBlockTime() *prototype.TimePointSec
	GenerateBlock(witness string, timestamp uint32, priKey *prototype.PrivateKeyType, skip prototype.SkipFlag) *prototype.SignedBlock
	GetWitnessTopN(n uint32) []string
	SetShuffledWitness(names []string)
	GetShuffledWitness() []string
	Pop(id *common.BlockID)

	TransferToVest( value *prototype.Coin)
	TransferFromVest( value *prototype.Vest)

}
