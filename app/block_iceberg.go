package app

import (
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/iservices"
	"github.com/sirupsen/logrus"
	"strconv"
	"sync"
	"sync/atomic"
)

const (
	GENESIS_TAG = "after_init_genesis"
)

var keyLatestBlockApplyChecksum = []byte("__latest_block_apply_checksum__")

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
	defaultBlockIcebergLowWM  = 127
)

// the block iceberg
type BlockIceberg struct {
	lock          sync.RWMutex               // lock for internal state
	db            iservices.IDatabaseService // database service
	log           *logrus.Logger			 // the logger
	inProgress    bool                       // indicating if there's a on-going block
	next          uint64                     // next block number
	hasFinalized  bool                       // indicating if there exists any finalized blocks
	finalized     uint64                     // the last finalized block number
	seaLevel      uint64                     // the oldest in-memory block
	highWM, lowWM uint64                     // the high/low watermark of in-memory block count
	lastBlockApplyHash uint64
	enableBAH  bool
}

// NewBlockIceberg() returns an instance of block iceberg.
func NewBlockIceberg(db iservices.IDatabaseService, logger *logrus.Logger, enableBAH bool) *BlockIceberg {
	var (
		hasBlock, hasFinalized, latest, finalized = false, false, uint64(0), uint64(0)
		err error
	)
	current, base := db.GetRevisionAndBase()
	latest, err = blockNumberFromString(db.GetRevisionTag(current))
	hasBlock = err == nil
	if !hasBlock {
		latest = 0
	}
	finalized, err = blockNumberFromString(db.GetRevisionTag(base))
	hasFinalized = err == nil
	if !hasFinalized {
		finalized = 0
	}
	berg := &BlockIceberg{
		db:           db,
		log:          logger,
		inProgress:   false,
		next:         latest + 1,
		hasFinalized: hasFinalized,
		finalized:    finalized,
		seaLevel:     latest + 1,
		highWM:       defaultBlockIcebergHighWM,
		lowWM:        defaultBlockIcebergLowWM,
		enableBAH:    enableBAH,
	}
	berg.loadBlockApplyHash()
	return berg
}

func (b *BlockIceberg) BeginBlock(blockNum uint64) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.log.Debugf("ICEBERG: BeginBlock(%d) begin. finalized=%d, sealevel=%d, next=%d", blockNum, b.finalized, b.seaLevel, b.next)

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
	b.log.Debugf("ICEBERG: BeginBlock(%d) end. finalized=%d, sealevel=%d, next=%d", blockNum, b.finalized, b.seaLevel, b.next)
	return nil
}

func (b *BlockIceberg) EndBlock(commit bool) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.log.Debugf("ICEBERG: EndBlock(%v) begin. finalized=%d, sealevel=%d, next=%d", commit, b.finalized, b.seaLevel, b.next)

	if !b.inProgress {
		return fmt.Errorf("cannot end a block without begin it first")
	}
	if !commit {
		err := b.db.EndTransaction(false)
		if err != nil {
			b.log.Errorf("ICEBERG: EndBlock commit error: %s", err.Error())
		}
		b.next--
	} else {
		if b.enableBAH {
			b.saveBlockApplyHash(common.PackBlockApplyHash(b.db.HashOfTopTransaction()))
		}
	}
	b.inProgress = false
	b.log.Debugf("ICEBERG: EndBlock(%v) end. finalized=%d, sealevel=%d, next=%d", commit, b.finalized, b.seaLevel, b.next)
	return nil
}

func (b *BlockIceberg) RevertBlock(blockNum uint64) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.log.Debugf("ICEBERG: RevertBlock %d begin. finalized=%d, sealevel=%d, next=%d", blockNum, b.finalized, b.seaLevel, b.next)

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
		err := b.db.RollbackTag(blockNumberToString(blockNum))
		if err != nil {
			b.log.Errorf("ICEBERG: RevertBlock %d RollbackTag(%d) error: %s", blockNum, blockNum, err.Error())
		}
	} else {
		// we're reverting a block in reversible db.

		// all in-memory blocks should be erased since they are offspring of our target.
		if b.seaLevel < b.next {
			err := b.db.RollbackTag(blockNumberToString(b.seaLevel))
			if err != nil {
				b.log.Errorf("ICEBERG: RevertBlock %d RollbackTag(%d) error: %s", blockNum, b.seaLevel, err.Error())
			}
		}

		// now we rollback the db
		if blockNum > 1 {
			err := b.db.RevertToTag(blockNumberToString(blockNum - 1))
			if err != nil {
				b.log.Errorf("ICEBERG: RevertBlock %d RevertToTag(%d) error: %s", blockNum, blockNum - 1, err.Error())
			}
		} else {
			// we're reverting block #1, i.e. rollback to the state just after init_genesis().
			err := b.db.RevertToTag(GENESIS_TAG)
			if err != nil {
				b.log.Errorf("ICEBERG: RevertBlock %d RevertToTag(%s) error: %s", blockNum, GENESIS_TAG, err.Error())
			}
		}
		b.seaLevel = blockNum
	}
	b.loadBlockApplyHash()
	b.next = blockNum
	b.log.Debugf("ICEBERG: RevertBlock %d end. finalized=%d, sealevel=%d, next=%d", blockNum, b.finalized, b.seaLevel, b.next)
	return nil
}

func (b *BlockIceberg) FinalizeBlock(blockNum uint64) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.log.Debugf("ICEBERG: FinalizeBlock %d begin. finalized=%d, sealevel=%d, next=%d", blockNum, b.finalized, b.seaLevel, b.next)

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
		err := b.db.RebaseToTag(tag)
		if err != nil {
			b.log.Errorf("ICEBERG: FinalizeBlock %d RebaseToTag(%s) error: %s", blockNum, tag, err.Error())
		}
	} else {
		// we're finalizing a block in memory.

		// basically it needs 2 steps,
		// step 1, move every in-memory finalized block into reversible db
		// step 2, finalize everything in reversible db
		err := b.db.EnableReversion(false)
		if err != nil {
			b.log.Errorf("ICEBERG: FinalizeBlock %d EnableReversion(false) error: %s", blockNum, err.Error())
		}
		err = b.db.Squash(tag)
		if err != nil {
			b.log.Errorf("ICEBERG: FinalizeBlock %d Squash(%s) error: %s", blockNum, tag, err.Error())
		}
		err = b.db.EnableReversion(true)
		if err != nil {
			b.log.Errorf("ICEBERG: FinalizeBlock %d EnableReversion(true) error: %s", blockNum, err.Error())
		}
		b.seaLevel = blockNum + 1
	}

	b.hasFinalized, b.finalized = true, blockNum
	b.log.Debugf("ICEBERG: FinalizeBlock %d end. finalized=%d, sealevel=%d, next=%d", blockNum, b.finalized, b.seaLevel, b.next)
	return nil
}

func (b *BlockIceberg) sink(blocks uint64) {
	b.log.Debugf("ICEBERG: sink %d block(s) begin. finalized=%d, sealevel=%d, next=%d", blocks, b.finalized, b.seaLevel, b.next)
	num := b.seaLevel
	blocksBak := blocks
	for blocks > 0 {
		tag := blockNumberToString(num)
		err := b.db.Squash(tag)
		if err != nil {
			b.log.Errorf("ICEBERG: sink %d block(s), Squash(%s) error: %s", blocks, tag, err.Error())
		}
		b.seaLevel++
		num++
		blocks--
	}
	b.log.Debugf("ICEBERG: sink %d block(s) end. finalized=%d, sealevel=%d, next=%d", blocksBak, b.finalized, b.seaLevel, b.next)
}

func (b *BlockIceberg) LastFinalizedBlock() (blockNum uint64, err error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.hasFinalized {
		blockNum, err = b.finalized, nil
	} else {
		blockNum, err = 0, nil
	}
	return
}

func (b *BlockIceberg) LatestBlock() (blockNum uint64, inProgress bool, err error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.next > 1 {
		return b.next - 1, b.inProgress, nil
	}
	return 0, false, nil
}

func (b *BlockIceberg) loadBlockApplyHash() {
	if !b.enableBAH {
		return
	}
	bah, _ := b.db.Get(keyLatestBlockApplyChecksum)
	if h, err := strconv.ParseUint(string(bah), 16, 64); err == nil {
		atomic.StoreUint64(&b.lastBlockApplyHash, h)
		b.log.Debugf("BlockApplyHash load: %016x", h)
	} else {
		atomic.StoreUint64(&b.lastBlockApplyHash, 0)
		b.log.Debugf("BlockApplyHash load: %016x, err=%s, raw=%s", 0, err.Error(), string(bah))
	}
}

func (b *BlockIceberg) saveBlockApplyHash(hash uint64) {
	if !b.enableBAH {
		return
	}
	_ = b.db.Put(keyLatestBlockApplyChecksum, []byte(strconv.FormatUint(hash, 16)))
	atomic.StoreUint64(&b.lastBlockApplyHash, hash)
	b.log.Debugf("BlockApplyHash save: %016x", hash)
}

func (b *BlockIceberg) LatestBlockApplyHash() uint64 {
	return atomic.LoadUint64(&b.lastBlockApplyHash)
}

func (b *BlockIceberg) LatestBlockApplyHashUnpacked() (version, hash uint32) {
	return common.UnpackBlockApplyHash(b.LatestBlockApplyHash())
}
