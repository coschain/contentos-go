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
	// SetProduce sets the node as a block producer if prod == true
	SetProduce(prod bool)
	// SetBootstrap determines if the current node starts a new block chain
	SetBootstrap(b bool)

	// Start starts the consensus process
	Start()
	// Stop stops the consensus process
	Stop()
	// GenerateBlock generates a new block, possible implementation: Producer.Produce()
	GenerateBlock() (common.ISignedBlock, error)
	// PushTransaction accepts the trx if and only if
	// 1. it's valid
	// 2. the current node is a producer
	// PushTransaction(trx common.ISignedTransaction)

	// PushBlock adds b to the block fork DB, called if ValidateBlock returns true
	PushBlock(b common.ISignedBlock) error
	// RemoveBlock removes a block and its successor from the block fork DB
	RemoveBlock(bh common.BlockID)
	// ForkRoot returns the common accesstor of two forks
	ForkRoot(fork1, fork2 common.BlockID) common.BlockID

	// apply the state change, called if b is the head block of the longest chain
	applyBlock(b common.ISignedBlock) error
	// undo state change
	revertBlock(height int) error
}

type IProducer interface {
	Produce() (common.ISignedBlock, error)
}
