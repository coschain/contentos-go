package consensus

import (
	"github.com/coschain/contentos-go/common"
	//"github.com/coschain/contentos-go/proto/type-proto"
)

type IConsensus interface {
	// CurrentProducer returns current producer
	CurrentProducer() IProducer
	// ActiveProducers returns a list of accounts that actively produce blocks
	ActiveProducers() []IProducer
	// InitProducer sets the prod as the default producer, further producers
	// should be maintained by the specific Consensus algorithm
	InitProducer(prod string)

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
