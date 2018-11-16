package consensus

import (
	"github.com/coschain/contentos-go/common"
	//"github.com/coschain/contentos-go/proto/type-proto"
)

type IConsensus interface {
	// NOTE: producers should be maintained by the specific Consensus algorithm
	// CurrentProducer returns current producer
	CurrentProducer() IProducer
	// ActiveProducers returns a list of accounts that actively produce blocks
	ActiveProducers() []IProducer

	// SetBootstrap determines if the current node starts a new block chain
	SetBootstrap(b bool)

	// GenerateBlock generates a new block, possible implementation: Producer.Produce()
	// GenerateBlock() (common.ISignedBlock, error)
	// PushTransaction accepts the trx if and only if
	// 1. it's valid
	// 2. the current node is a producer
	PushTransaction(trx common.ISignedTransaction)

	// PushBlock adds b to the block fork DB, called if ValidateBlock returns true
	PushBlock(b common.ISignedBlock)



	GetHeadBlockId() (common.BlockID)

	GetHashes(remote_head_id, current_head_id common.BlockID) []common.BlockID

	GetBlockByHash(id common.BlockID) common.ISignedBlock

	ChainHasBlock(id common.BlockID) bool
}

type IProducer interface {
	Produce() (common.ISignedBlock, error)
}
