package app

import (
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/sirupsen/logrus"
	"sync"
)

type TaposChecker struct {
	db        iservices.IDatabaseRW						// the database
	log       *logrus.Logger							// the logger
	blockIds  [common.TaposMaxBlockCount][]byte			// recent block ids
	lastBlock uint64									// latest block number
	lock      sync.RWMutex								// for thread safety

}

// NewTaposChecker creates an instance of TaposChecker
func NewTaposChecker(db iservices.IDatabaseRW, logger *logrus.Logger, lastBlock uint64) *TaposChecker {
	c := &TaposChecker{
		db:        db,
		log:       logger,
		lastBlock: lastBlock,
	}
	// load all recent block ids from database
	_ = c.loadAllBlockIds()
	return c
}

// loadBlockId loads ids of given block range.
func (c *TaposChecker) loadBlockId(fromBlockNum, toBlockNum uint64) error {
	c.log.Debugf("TAPOS: load block refs [%d, %d]", common.TaposRefBlockNum(fromBlockNum), common.TaposRefBlockNum(toBlockNum))
	for i := fromBlockNum; i <= toBlockNum; i++ {
		n := common.TaposRefBlockNum(i)
		if b := table.NewUniBlockSummaryObjectIdWrap(c.db).UniQueryId(&n); b == nil {
			return fmt.Errorf("cannot loadBlockId summary info of block refNumber %d", n)
		} else {
			c.blockIds[n] = common.CopyBytes(b.GetBlockId().Hash)
		}
	}
	return nil
}

// loadAllBlockIds loads all recent block ids.
func (c *TaposChecker) loadAllBlockIds() error {
	return c.loadBlockId(1, common.TaposMaxBlockCount)
}

// Check checks if given transaction's tapos data is valid.
func (c *TaposChecker) Check(trx *prototype.Transaction) error {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if expected := common.TaposRefBlockPrefix(c.blockIds[common.TaposRefBlockNum(uint64(trx.RefBlockNum))]); expected != trx.RefBlockPrefix {
		return fmt.Errorf("prefix mismatch, expecting %08x, got %08x", expected, trx.RefBlockPrefix)
	}
	return nil
}

// BlockApplied *MUST* be called *AFTER* a block was successfully applied.
func (c *TaposChecker) BlockApplied(b *prototype.SignedBlock) {
	c.lock.Lock()
	defer c.lock.Unlock()

	blockNum := b.SignedHeader.Number()
	if blockNum > c.lastBlock {
		count := blockNum - c.lastBlock
		if count < common.TaposMaxBlockCount {
			_ = c.loadBlockId(c.lastBlock+1, blockNum)
		} else {
			_ = c.loadAllBlockIds()
		}
		// fix latest block number
		c.lastBlock = blockNum
	}
}

// BlockReverted *MUST* be called *AFTER* a block was successfully reverted.
func (c *TaposChecker) BlockReverted(blockNum uint64) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if blockNum <= c.lastBlock {
		count := c.lastBlock - blockNum + 1
		if count < common.TaposMaxBlockCount {
			_ = c.loadBlockId(blockNum, c.lastBlock)
		} else {
			_ = c.loadAllBlockIds()
		}
		// fix latest block number
		c.lastBlock = blockNum - 1
	}
}

// BlockCommitted *MUST* be called *AFTER* a block was successfully committed.
func (c *TaposChecker) BlockCommitted(blockNum uint64) {

}
