package app

import (
	"bytes"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"sync"
)

const (
	sMaxHistoryBlocks = constants.TrxMaxExpirationTime / constants.BlockInterval + 1
	sTrxSignatureSize = 65
)

type InBlockTrxEntry struct {
	data []byte
	trxs map[string]bool
}

func NewInBlockEntry(data []byte) *InBlockTrxEntry {
	e := &InBlockTrxEntry{ data:data }
	if count := len(e.data) / sTrxSignatureSize; count > 0 {
		e.trxs = make(map[string]bool)
		offset := 0
		for i := 0; i < count; i++ {
			e.trxs[string(e.data[offset:offset + sTrxSignatureSize])] = true
		}
	}
	return e
}

type InBlockTrxChecker struct {
	db iservices.IDatabaseRW
	trxs map[uint64]*InBlockTrxEntry
	last uint64
	lock sync.RWMutex
}

func NewInBlockTrxChecker(db iservices.IDatabaseRW, last uint64) *InBlockTrxChecker {
	c := &InBlockTrxChecker{
		db: db,
		trxs: make(map[uint64]*InBlockTrxEntry),
		last: last,
	}
	c.loadAll()
	return c
}

func (c *InBlockTrxChecker) Has(trx *prototype.SignedTransaction) bool {
	s := string(trx.Signature.Sig)

	c.lock.RLock()
	defer c.lock.RUnlock()
	for _, e := range c.trxs {
		if e.trxs[s] {
			return true
		}
	}
	return false
}

func (c *InBlockTrxChecker) BlockApplied(b *prototype.SignedBlock) {
	blockNum := b.SignedHeader.Number()

	c.lock.Lock()
	defer c.lock.Unlock()
	if blockNum > c.last {
		sigs := make([][]byte, len(b.Transactions))
		for i, w := range b.Transactions {
			sigs[i] = w.SigTrx.Signature.Sig
		}
		c.trxs[blockNum] = NewInBlockEntry(bytes.Join(sigs, nil))
		c.save(blockNum, blockNum)
		if blockNum > sMaxHistoryBlocks {
			oldBlock := blockNum - sMaxHistoryBlocks
			c.remove(oldBlock, oldBlock)
			delete(c.trxs, oldBlock)
		}
		c.last = blockNum
	}
}

func (c *InBlockTrxChecker) BlockReverted(blockNum uint64) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if blockNum <= c.last {
		start := uint64(1)
		if c.last >= sMaxHistoryBlocks {
			start = c.last - sMaxHistoryBlocks + 1
		}
		c.remove(blockNum, c.last)
		for i := blockNum; i <= c.last; i++ {
			delete(c.trxs, i)
		}
		c.last = blockNum - 1
		newStart := uint64(1)
		if c.last >= sMaxHistoryBlocks {
			newStart = c.last - sMaxHistoryBlocks + 1
		}
		if newStart < start {
			c.load(newStart, start - 1)
		}
	}
}

func (c *InBlockTrxChecker) BlockCommitted(blockNum uint64) {

}

func (c *InBlockTrxChecker) load(fromBlock, toBlock uint64) {
	for i := fromBlock; i <= toBlock; i++ {
		blockTrxs := table.NewUniBlocktrxsBlockWrap(c.db).UniQueryBlock(&i)
		if blockTrxs != nil {
			c.trxs[i] = NewInBlockEntry(blockTrxs.GetTrxs())
		}
	}
}

func (c *InBlockTrxChecker) loadAll() {
	if c.last < 1 {
		return
	}
	start := uint64(1)
	if c.last >= sMaxHistoryBlocks {
		start = c.last - sMaxHistoryBlocks + 1
	}
	c.load(start, c.last)
}

func (c *InBlockTrxChecker) save(fromBlock, toBlock uint64) {
	for i := fromBlock; i <= toBlock; i++ {
		if e := c.trxs[i]; e != nil {
			_ = table.NewSoBlocktrxsWrap(c.db, &i).Create(func(tInfo *table.SoBlocktrxs) {
				tInfo.Block = i
				tInfo.Trxs = e.data
			})
		}
	}
}

func (c *InBlockTrxChecker) remove(fromBlock, toBlock uint64) {
	for i := fromBlock; i <= toBlock; i++ {
		_ = table.NewSoBlocktrxsWrap(c.db, &i).RemoveBlocktrxs()
	}
}
