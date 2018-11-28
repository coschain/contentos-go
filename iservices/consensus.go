package iservices

import (
	"github.com/coschain/contentos-go/common"
)

var CS_SERVER_NAME = "consensus"
type IConsensus interface {
	// NOTE: producers should be maintained by the specific Consensus algorithm
	// CurrentProducer returns current producer
	CurrentProducer() string
	// ActiveProducers returns a list of accounts that actively produce blocks
	ActiveProducers() []string

	// SetBootstrap determines if the current node starts a new block chain
	SetBootstrap(b bool)

	// GenerateBlock generates a new block, possible implementation: Producer.Produce()
	// GenerateBlock() (common.ISignedBlock, error)
	// PushTransaction accepts the trx if and only if
	// 1. it's valid
	// 2. the current node is a producer
	PushTransaction(trx common.ISignedTransaction) common.ITransactionInvoice

	// PushBlock adds b to the block fork DB, called if ValidateBlock returns true
	PushBlock(b common.ISignedBlock)



	GetHeadBlockId() (common.BlockID)
	GetIDs(start, end common.BlockID) ([]common.BlockID, error)
	FetchBlock(id common.BlockID) (common.ISignedBlock, error)
	HasBlock(id common.BlockID) bool
	FetchBlocksSince(id common.BlockID) ([]common.ISignedBlock, error)

}