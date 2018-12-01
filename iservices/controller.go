package iservices

import (
	"github.com/coschain/contentos-go/prototype"
)

//
// This file defines interfaces of Database service.
//

var ControlServerName = "ctrl"

type IController interface {
	PushTrx(trx *prototype.SignedTransaction) *prototype.TransactionInvoice
	PushBlock(blk *prototype.SignedBlock, skip prototype.SkipFlag) error
	HeadBlockTime() *prototype.TimePointSec
	GenerateBlock(witness string, pre *prototype.Sha256, timestamp uint32, priKey *prototype.PrivateKeyType, skip prototype.SkipFlag) *prototype.SignedBlock
	GetWitnessTopN(n uint32) []string
	SetShuffledWitness(names []string)
	GetShuffledWitness() []string
	// will set DB status to num
	PopBlockTo(num uint32)
	// will cut off DB status that before num
	Commit(num uint32)

	TransferToVest(value *prototype.Coin)
	TransferFromVest(value *prototype.Vest)

	AddWeightedVP(value uint64)
}
