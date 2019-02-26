package iservices

import (
	"time"
	"github.com/coschain/contentos-go/common"
)

var ConsensusServerName = "consensus"

type IConsensus interface {
	// NOTE: producers should be maintained by the specific Consensus algorithm
	// CurrentProducer returns current producer
	CurrentProducer() string
	// ActiveProducers returns a list of accounts that actively produce blocks
	ActiveProducers() []string

	// SetBootstrap determines if the current node starts a new block chain
	SetBootstrap(b bool)

	// PushTransaction accepts the trx
	PushTransaction(trx common.ISignedTransaction, wait bool, broadcast bool) common.ITransactionReceiptWithInfo

	//Add transaction to pending list,the transaction will be applied when generate a block
	PushTransactionToPending(trx common.ISignedTransaction, callBack func(err error))
	// PushBlock adds b to the block fork DB, called if ValidateBlock returns true
	PushBlock(b common.ISignedBlock)

	// Push sends a user defined msg to consensus
	Push(msg interface{})

	// GetLastBFTCommit get the last irreversible block info. @evidence
	// is the information that can prove the id is indeed the last irreversible one.
	// e.g. if user uses bft to achieve fast ack, @evidence can simply be the collection
	// of the vote message
	GetLastBFTCommit() (evidence interface{})

	GetNextBFTCheckPoint(blockNum uint64) (evidence interface{})

	// GetHeadBlockId returns the block id of the head block
	GetHeadBlockId() common.BlockID

	// GetIDs returns a list of block ids which remote peer may not have
	GetIDs(start, end common.BlockID) ([]common.BlockID, error)

	// FetchBlock returns the block whose id is the given param
	FetchBlock(id common.BlockID) (common.ISignedBlock, error)

	// HasBlock query the local blockchain whether it has the given block id
	HasBlock(id common.BlockID) bool

	// FetchBlocksSince returns blocks in the range of (id, max(headID, id+1024))
	FetchBlocksSince(id common.BlockID) ([]common.ISignedBlock, error)

	// FetchBlocks returns blocks in the range of [from, to]
	FetchBlocks(from, to uint64) ([]common.ISignedBlock, error)

	// NOTE: the following methods are testing methods and should only be called by multinodetester2
	// ResetProdTimer reset the prodTimer in dpos
	ResetProdTimer(t time.Duration)

	// ResetTicker reset the Ticker in DPoS
	ResetTicker(t time.Time)

	GetLIB() common.BlockID

	// MaybeProduceBlock check whether should produce a block
	MaybeProduceBlock()
}
