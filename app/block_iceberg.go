package app

import (
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/iservices"
	"strconv"
	"sync"
)

// block number -> string
func blockNumberToString(blockNum uint64) string {
	return strconv.FormatUint(blockNum, 10)
}

// string -> block number
func blockNumberFromString(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}

const (
	defaultBlockIcebergHighWM = 128
	defaultBlockIcebergLowWM  = 32
)

// the block iceberg
type BlockIceberg struct {
	lock          sync.RWMutex               // lock for internal state
	db            iservices.IDatabaseService // database service
	inProgress    bool                       // indicating if there's a on-going block
	next          uint64                     // next block number
	hasFinalized  bool                       // indicating if there exists any finalized blocks
	finalized     uint64                     // the last finalized block number
	seaLevel      uint64                     // the oldest in-memory block
	highWM, lowWM uint64                     // the high/low watermark of in-memory block count
}

// NewBlockIceberg() returns an instance of block iceberg.
func NewBlockIceberg(db iservices.IDatabaseService) *BlockIceberg {
	current, base := db.GetRevisionAndBase()
	hasBlock, hasFinalized, latest, finalized := current > 0, base > 0, uint64(0), uint64(0)
	var err error
	if hasBlock {
		latest, err = blockNumberFromString(db.GetRevisionTag(current))
		if err != nil {
			return nil
		}
	}
	if hasFinalized {
		finalized, err = blockNumberFromString(db.GetRevisionTag(base))
		if err != nil {
			return nil
		}
	}
	return &BlockIceberg{
		db:           db,
		inProgress:   false,
		next:         latest + 1,
		hasFinalized: hasFinalized,
		finalized:    finalized,
		seaLevel:     latest + 1,
		highWM:       defaultBlockIcebergHighWM,
		lowWM:        defaultBlockIcebergLowWM,
	}
}

func (b *BlockIceberg) BeginBlock(blockNum uint64) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if blockNum == 0 {
		return errors.New("invalid block number 0")
	}
	if b.inProgress {
		return fmt.Errorf("cannot begin block %d without ending block %d first", blockNum, b.next-1)
	}
	if b.next != blockNum {
		return fmt.Errorf("cannot begin block %d. Block numbers must be consecutive and block %d is expected", blockNum, b.next)
	}
	b.inProgress = true
	b.next++

	// if we got too many non-finalized blocks in memory, move some into reversible db
	if b.next-b.seaLevel >= b.highWM {
		b.sink(b.next - b.seaLevel - b.lowWM)
	}
	b.db.BeginTransactionWithTag(blockNumberToString(blockNum))
	return nil
}

func (b *BlockIceberg) EndBlock(commit bool) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if !b.inProgress {
		return fmt.Errorf("cannot end a block without begin it first")
	}
	if !commit {
		b.db.EndTransaction(false)
		b.next--
	}
	b.inProgress = false
	return nil
}

func (b *BlockIceberg) RevertBlock(blockNum uint64) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if blockNum == 0 {
		return errors.New("invalid block number 0")
	}
	if b.inProgress {
		return fmt.Errorf("cannot revert block %d without ending block %d first", blockNum, b.next-1)
	}
	if b.hasFinalized && b.finalized >= blockNum {
		return fmt.Errorf("cannot revert block %d since minimal reversible block is %d", blockNum, b.finalized+1)
	}
	if b.next <= blockNum {
		return fmt.Errorf("cannot revert a future block %d since latest block is %d", blockNum, b.next-1)
	}
	if blockNum >= b.seaLevel {
		// we're reverting an in-memory block
		b.db.RollbackTag(blockNumberToString(blockNum))
	} else {
		// we're reverting a block in reversible db.

		// all in-memory blocks should be erased since they are offspring of our target.
		b.db.RollbackTag(blockNumberToString(b.seaLevel))

		// now we rollback the db
		if blockNum > 1 {
			b.db.RevertToTag(blockNumberToString(blockNum - 1))
		} else {
			b.db.RevertToRevision(0)
		}
		b.seaLevel = blockNum
	}
	b.next = blockNum
	return nil
}

func (b *BlockIceberg) FinalizeBlock(blockNum uint64) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if blockNum == 0 {
		return errors.New("invalid block number 0")
	}
	if b.hasFinalized && b.finalized >= blockNum {
		return nil
	}
	n := b.next
	if n > 0 {
		n--
	}
	if n > 0 && b.inProgress {
		n--
	}
	if n < blockNum {
		return fmt.Errorf("cannot finalize block %d since maximum finalizable block is %d", blockNum, n)
	}
	tag := blockNumberToString(blockNum)
	if blockNum < b.seaLevel {
		// we're finalizing a block in reversible db.
		b.db.RebaseToTag(tag)
	} else {
		// we're finalizing a block in memory.

		// basically it needs 2 steps,
		// step 1, move every in-memory finalized block into reversible db
		// step 2, finalize everything in reversible db
		b.db.EnableReversion(false)
		b.db.Squash(tag)
		b.db.TagRevision(b.db.GetRevision(), tag)
		b.db.EnableReversion(true)
	}

	b.hasFinalized, b.finalized = true, blockNum
	return nil
}

func (b *BlockIceberg) sink(blocks uint64) {
	num := b.seaLevel
	for blocks > 0 {
		tag := blockNumberToString(num)
		b.db.Squash(tag)
		b.db.TagRevision(b.db.GetRevision(), tag)
		b.seaLevel++
		num++
		blocks--
	}
}

func (b *BlockIceberg) LastFinalizedBlock() (blockNum uint64, err error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.hasFinalized {
		blockNum, err = b.finalized, nil
	} else {
		blockNum, err = 0, errors.New("no finalized blocks yet")
	}
	return
}

func (b *BlockIceberg) LatestBlock() (blockNum uint64, inProgress bool, err error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.next > 1 {
		return b.next - 1, b.inProgress, nil
	}
	return 0, false, errors.New("no block")
}
