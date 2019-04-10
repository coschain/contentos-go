package app

// This file implements a solution against the transaction replay attack.
//
// Transaction replay attack is an attacking way by storing a valid transaction and resending it later.
// For example, Alice signed a transaction transferring some coins to Bob. Bob can resend the transaction
// over and over again util Alice's balance exhausts.
//
// Steemit and EOS save and maintain all unexpired in-block transactions in state database. So that when
// a duplicate transaction is received and it's not expired, system can find it in database and refuse it.
// It's a working solution but maintaining unexpired in-block transactions is expensive, and much more so when
// we use on-disk database instead of a memory mapping.
//
// We have a similar but different idea. Suppose that maximum expiration of a transaction is 60s and a new block
// is born every second, we simply refuse any transaction which can be found in latest 60 blocks. What we're
// maintaining is the transactions in latest 60 blocks, which are essentially a superset of all unexpired in-block
// transactions. The benefit is that we can load and save these transactions in block level instead of transaction
// level, eliminating database I/O by hundreds of times.
//

import (
	"bytes"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/sirupsen/logrus"
	"sync"
)

const (
	// maximum number of recent blocks we need to maintain
	sMaxHistoryBlocks = constants.TrxMaxExpirationTime/constants.BlockInterval + 1

	// fixed size of transaction signatures
	sTrxSignatureSize = 65
)

// InBlockTrxEntry represents all transactions in a block
type InBlockTrxEntry struct {
	data []byte          // a blob by concatenating transaction signatures
	trxs map[string]bool // transaction set: transaction signature -> true
}

// NewInBlockEntry creates an instance of InBlockTrxEntry
func NewInBlockEntry(data []byte) *InBlockTrxEntry {
	e := &InBlockTrxEntry{data: data}
	// extract every transaction signature from the blob, and save them in transaction set
	if count := len(e.data) / sTrxSignatureSize; count > 0 {
		e.trxs = make(map[string]bool)
		offset := 0
		for i := 0; i < count; i++ {
			e.trxs[string(e.data[offset:offset+sTrxSignatureSize])] = true
			offset += sTrxSignatureSize
		}
	}
	return e
}

// InBlockTrxChecker checks if a given transaction can be found in latest blocks.
type InBlockTrxChecker struct {
	db   iservices.IDatabaseRW				// the database
	trxs map[uint64]*InBlockTrxEntry		// transactions of latest blocks: block number -> transactions in the block
	last uint64								// latest block number
	lock sync.RWMutex						// for thread safety
	log  *logrus.Logger						// the logger
}

// NewInBlockTrxChecker creates an instance of InBlockTrxChecker.
func NewInBlockTrxChecker(db iservices.IDatabaseRW, logger *logrus.Logger, last uint64) *InBlockTrxChecker {
	c := &InBlockTrxChecker{
		db:   db,
		log:  logger,
		trxs: make(map[uint64]*InBlockTrxEntry),
		last: last,
	}
	// load initial data from database
	c.loadAll()
	return c
}

// Has checks if a given transaction can be found in latest blocks.
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

// BlockApplied *MUST* be called *AFTER* a block was successfully applied.
func (c *InBlockTrxChecker) BlockApplied(b *prototype.SignedBlock) {
	blockNum := b.SignedHeader.Number()

	c.lock.Lock()
	defer c.lock.Unlock()
	if blockNum > c.last {
		// get all transactions from the applied block
		sigs := make([][]byte, len(b.Transactions))
		for i, w := range b.Transactions {
			sigs[i] = w.SigTrx.Signature.Sig
		}
		// save them into in-memory map
		c.trxs[blockNum] = NewInBlockEntry(bytes.Join(sigs, nil))
		// save them into database
		c.save(blockNum, blockNum)
		// forget too old blocks
		if blockNum > sMaxHistoryBlocks {
			oldBlock := blockNum - sMaxHistoryBlocks
			c.remove(oldBlock, oldBlock)
			delete(c.trxs, oldBlock)
		}
		// fix latest block number
		c.last = blockNum
	}
}

// BlockReverted *MUST* be called *AFTER* a block was successfully reverted.
func (c *InBlockTrxChecker) BlockReverted(blockNum uint64) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if blockNum <= c.last {
		// @start is the oldest block we're maintaining
		start := uint64(1)
		if c.last >= sMaxHistoryBlocks {
			start = c.last - sMaxHistoryBlocks + 1
		}
		// remove reverted blocks from both database and in-memory map
		c.remove(blockNum, c.last)
		for i := blockNum; i <= c.last; i++ {
			delete(c.trxs, i)
		}
		// fix latest block number
		c.last = blockNum - 1
		// @newStart is oldest block we should maintain after the reversion
		newStart := uint64(1)
		if c.last >= sMaxHistoryBlocks {
			newStart = c.last - sMaxHistoryBlocks + 1
		}
		// load old blocks from database
		if newStart < start {
			c.load(newStart, start-1)
		}
	}
}

// BlockReverted *SHOULD* be called *AFTER* a block was successfully committed.
func (c *InBlockTrxChecker) BlockCommitted(blockNum uint64) {
	// we care about changes of transaction set,
	// so block commitments can be ignored since they don't affect transaction set.
}

// load reads data of given block range from database and updates in-memory map.
func (c *InBlockTrxChecker) load(fromBlock, toBlock uint64) {
	for i := fromBlock; i <= toBlock; i++ {
		blockTrxs := table.NewUniBlocktrxsBlockWrap(c.db).UniQueryBlock(&i)
		if blockTrxs != nil {
			c.trxs[i] = NewInBlockEntry(blockTrxs.GetTrxs())
		}
	}
}

// loadAll reads data of all necessary blocks from database and updates in-memory map.
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

// save writes data of given block range to database.
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

// remove deletes given block range from database.
func (c *InBlockTrxChecker) remove(fromBlock, toBlock uint64) {
	for i := fromBlock; i <= toBlock; i++ {
		_ = table.NewSoBlocktrxsWrap(c.db, &i).RemoveBlocktrxs()
	}
}
