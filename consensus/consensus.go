package consensus

import (
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/proto/type-proto"
)

type Consensus interface {
	// CurrentProducer returns current producer
	CurrentProducer() prototype.AccountName
	// ActiveProducer returns a list of accounts that actively produce blocks
	ActiveProducer() []prototype.AccountName

	// GenerateBlock generates a new block, possible implementation: Producer.Produce()
	GenerateBlock() error
	// PushTransaction accepts the trx if and only if
	// 1. it's valid
	// 2. the current node is a producer
	PushTransaction()
	// ValidateBlock returns true if b is direct successor of any fork chain
	ValidateBlock(b *common.SignedBlockIF) bool
	// AddBlock adds b to the block fork DB, called if ValidateBlock returns true
	AddBlock(b *common.SignedBlockIF) error
	// RemoveBlock removes a block and its successor from the block fork DB
	RemoveBlock(bh common.BlockID)
	// ForkRoot returns the common accesstor of two forks
	ForkRoot(fork1, fork2 common.BlockID) common.BlockID

	// apply the state change, called if b is the head block of the longest chain
	applyBlock(b *common.SignedBlockIF) error
	// undo state change
	revertBlock(height int) error
}

type Producer interface {
	Produce() (*common.SignedBlockIF, error)
}
