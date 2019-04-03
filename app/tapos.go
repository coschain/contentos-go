package app

import (
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"sync"
)

type TaposChecker struct {
	db iservices.IDatabaseRW
	blockIds [common.TaposMaxBlockCount][]byte
	lastBlock uint64
	lock sync.RWMutex
}

func NewTaposChecker(db iservices.IDatabaseRW, lastBlock uint64) *TaposChecker {
	c := &TaposChecker{
		db: db,
		lastBlock: lastBlock,
	}
	_ = c.loadAllBlockIds()
	return c
}

func (c *TaposChecker) loadBlockId(fromBlockNum, toBlockNum uint64) error {
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

func (c *TaposChecker) loadAllBlockIds() error {
	return c.loadBlockId(1, common.TaposMaxBlockCount)
}


func (c *TaposChecker) Check(trx *prototype.Transaction) error {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if expected := common.TaposRefBlockPrefix(c.blockIds[common.TaposRefBlockNum(uint64(trx.RefBlockNum))]); expected != trx.RefBlockPrefix {
		return fmt.Errorf("prefix mismatch, expecting %08x, got %08x", expected, trx.RefBlockPrefix)
	}
	return nil
}

func (c *TaposChecker) BlockApplied(b *prototype.SignedBlock) {
	c.lock.Lock()
	defer c.lock.Unlock()

	blockNum := b.SignedHeader.Number()
	if blockNum > c.lastBlock {
		count := blockNum - c.lastBlock
		if count < common.TaposMaxBlockCount {
			_ = c.loadBlockId(c.lastBlock + 1, blockNum)
		} else {
			_ = c.loadAllBlockIds()
		}
		c.lastBlock = blockNum
	}
}

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
		c.lastBlock = blockNum - 1
	}
}

func (c *TaposChecker) BlockCommitted(blockNum uint64) {

}