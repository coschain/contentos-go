package app

import (
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/gogo/protobuf/proto"
	"github.com/sirupsen/logrus"
	"sync"
	"sync/atomic"
	"time"
)

// TrxCallback is the type of callback function reporting transaction process results.
type TrxCallback func(result *prototype.TransactionWrapperWithInfo)

// TrxEntry is a wrapper of a transaction with extra information.
type TrxEntry struct {
	chainId   prototype.ChainId	                    // id of block chain to which the transaction is sent
	result    *prototype.TransactionWrapperWithInfo	// process result involving the transaction
	trxId     string								// transaction id
	size      int									// transaction size
	signer    string								// requested account to sign the transaction
	signerKey *prototype.PublicKeyType				// the actual public key which signed the transaction
	callback  TrxCallback							// callback function
}

// NewTrxMgrEntry creates an instance of TrxEntry.
func NewTrxMgrEntry(chainId prototype.ChainId, trx *prototype.SignedTransaction, callback TrxCallback) *TrxEntry {
	return &TrxEntry{
		chainId: chainId,
		result: &prototype.TransactionWrapperWithInfo{
			SigTrx:  trx,
			Receipt: &prototype.TransactionReceiptWithInfo{Status: prototype.StatusSuccess},
		},
		callback: callback,
	}
}

// SetError sets the entry's result as given error, and returns the error.
func (e *TrxEntry) SetError(err error) error {
	e.result.Receipt.Status = prototype.StatusError
	e.result.Receipt.ErrorInfo = err.Error()
	return err
}

// Deliver calls entry's callback function.
func (e *TrxEntry) Deliver() {
	if e.callback != nil {
		e.callback(e.result)
	}
}

// InitCheck fills extra information of the entry, and do a basic validation check.
// Note that InitCheck is independent from chain state. We should do it only once for each transaction.
func (e *TrxEntry) InitCheck() error {
	trx := e.result.SigTrx
	// basic check
	if err := trx.Validate(); err != nil {
		return e.SetError(err)
	}

	if trxId, err := trx.Id(); err != nil {
		return e.SetError(err)
	} else {
		e.trxId = string(trxId.Hash)
	}

	// transaction size limit check
	e.size = proto.Size(trx)
	if e.size > constants.MaxTransactionSize {
		return e.SetError(fmt.Errorf("trx too large, size = %d > %d", e.size, constants.MaxTransactionSize))
	}
	// get the signer account name
	creator := ""
	if creators := trx.GetOpCreatorsMap(); len(creators) != 1 {
		return e.SetError(fmt.Errorf("non-unique trx creators, found %d", len(creators)))
	} else {
		for creator = range creators {
			break
		}
	}
	e.signer = creator
	// recover the signing public key from signature
	if signKey, err := trx.ExportPubKeys(e.chainId); err != nil {
		return e.SetError(fmt.Errorf("cannot export signing key: %s", err.Error()))
	} else {
		e.signerKey = signKey
	}
	return nil
}

// CheckExpiration checks if the transaction is valid based on its expiration.
func (e *TrxEntry) CheckExpiration(blockTime uint32) error {
	expiration := e.result.SigTrx.GetTrx().GetExpiration().GetUtcSeconds()
	if expiration < blockTime {
		return e.SetError(fmt.Errorf("trx expired, %d < %d", expiration, blockTime))
	}
	if expiration > blockTime + constants.TrxMaxExpirationTime {
		return e.SetError(fmt.Errorf("trx expiration too long, %d > %d + %d", expiration, blockTime, constants.TrxMaxExpirationTime))
	}
	return nil
}

// CheckTapos checks if the transaction is valid based on its tapos information.
func (e *TrxEntry) CheckTapos(checker *TaposChecker) error {
	if err := checker.Check(e.result.SigTrx.Trx); err != nil {
		return e.SetError(fmt.Errorf("tapos failed: %s", err.Error()))
	}
	return nil
}

// CheckSignerKey checks if the transaction is signed by correct public key.
func (e *TrxEntry) CheckSignerKey(fetcher *AuthFetcher) error {
	if err := fetcher.CheckPublicKey(e.signer, e.signerKey); err != nil {
		return e.SetError(fmt.Errorf("signature failed: %s", err.Error()))
	}
	return nil
}

// CheckInBlockTrxs checks if the transaction is a duplicate of any old transaction.
func (e *TrxEntry) CheckInBlockTrxs(checker *InBlockTrxChecker) error {
	if checker.Has(e.result.SigTrx) {
		return e.SetError(errors.New("found duplicate in-block trx"))
	}
	return nil
}

func (e *TrxEntry) GetTrxResult() *prototype.TransactionWrapperWithInfo {
	return e.result
}
func (e *TrxEntry) GetTrxSize() int {
	return e.size
}
func (e *TrxEntry) GetTrxSigner() string {
	return e.signer
}
func (e *TrxEntry) GetTrxSigningKey() *prototype.PublicKeyType {
	return e.signerKey
}

const (
	// maximum count of transactions that are waiting to be packed to blocks.
	// if this limit is reached, any incoming transaction will be refused directly.
	sMaxWaitingCount  = constants.TrxMaxExpirationTime * 2000

	// threshold over which cleanings are necessary
	sWaitingCountWaterMark  = sMaxWaitingCount / 10

	// minimal interval between cleanings
	sMinCleanupInterval = 10 * time.Second

	// shrink the waiting/fetched pools every 100K transactions
	sShrinkCountWaterMark = 100000
)

// ITrxMgrPlugin is an interface of manager plugins.
type ITrxMgrPlugin interface {
	BlockApplied(b *prototype.SignedBlock)				// called once after a block is successfully applied.
	BlockReverted(blockNum uint64)						// called once after a block is successfully reverted.
	BlockCommitted(blockNum uint64)						// called once after a block is successfully committed.
}

// The transaction manager.
type TrxMgr struct {
	chainId         prototype.ChainId                   // the chain
	db 				iservices.IDatabaseRW				// the database
	log             *logrus.Logger						// the logger
	headTime		uint32								// timestamp of head block, in seconds
	waiting 		map[string]*TrxEntry				// transactions waiting to be packed to blocks, trxId -> entry
	waitingLock 	sync.RWMutex						// lock of waiting transactions
	fetched 		map[string]*TrxEntry				// transactions being packed to a block, trxId -> entry
	fetchedLock 	sync.RWMutex						// lock of fetched transactions
	auth 			*AuthFetcher						// checker of transaction signatures
	tapos 			*TaposChecker						// checker of transaction tapos
	history 		*InBlockTrxChecker					// checker of transaction duplication
	plugins         []ITrxMgrPlugin						// manager plugins, consisting of above checkers
	lastCleanTime	time.Time							// last time we clean up expired waiting transactions
	shrinkCounter   uint64								// a counter to determine when to shrink pools
}

// NewTrxMgr creates an instance of TrxMgr.
func NewTrxMgr(chainId prototype.ChainId, db iservices.IDatabaseRW, logger *logrus.Logger, lastBlock, commitBlock uint64) *TrxMgr {
	auth := NewAuthFetcher(db, logger, lastBlock, commitBlock)
	tapos := NewTaposChecker(db, logger, lastBlock)
	history := NewInBlockTrxChecker(db, logger, lastBlock)
	return &TrxMgr{
		chainId:  chainId,
		db:       db,
		log:      logger,
		headTime: (&DynamicGlobalPropsRW{db:db}).GetProps().GetTime().GetUtcSeconds(),
		waiting:  make(map[string]*TrxEntry),
		fetched:  make(map[string]*TrxEntry),
		auth: auth,
		tapos: tapos,
		history: history,
		plugins: []ITrxMgrPlugin{ auth, tapos, history },
		lastCleanTime: time.Now(),
	}
}

// AddTrx processes an incoming transaction.
// AddTrx returns nil if the incoming transaction is accepted, otherwise an error is returned.
// If a non-nil callback is given, it will be called once asynchronously with the final process result.
func (m *TrxMgr) AddTrx(trx *prototype.SignedTransaction, callback TrxCallback) error {
	entry := NewTrxMgrEntry(m.chainId, trx, callback)
	// very basic nil pointer check
	if trx == nil || trx.Signature == nil {
		err := entry.SetError(errors.New("invalid trx"))
		m.deliverEntry(entry)
		return err
	}
	// very basic duplication check
	if m.isProcessingTrx(trx) != nil {
		err := entry.SetError(errors.New("trx already in process"))
		m.deliverEntry(entry)
		return err
	}
	c := make(chan error)
	go func() {
		ok := false
		// check the transaction
		if entry.InitCheck() != nil || m.checkTrx(entry, atomic.LoadUint32(&m.headTime), false) != nil {
			// deliver if failed
			m.deliverEntry(entry)
		} else {
			// if passed, try adding it to the waiting pool
			m.waitingLock.Lock()
			m.fetchedLock.RLock()

			ok = m.addToWaiting(entry) > 0

			m.fetchedLock.RUnlock()
			m.waitingLock.Unlock()
		}
		if !ok {
			c <- errors.New(entry.result.Receipt.ErrorInfo)
		} else {
			c <- nil
		}
	}()
	return <-c
}

// WaitingCount returns number of transactions that are waiting to be packed to blocks.
func (m *TrxMgr) WaitingCount() int {
	m.waitingLock.RLock()
	defer m.waitingLock.RUnlock()
	return len(m.waiting)
}

// FetchTrx fetches a batch of transactions from waiting pool.
// Block producer should call FetchTrx to collect transactions of new blocks.
func (m *TrxMgr) FetchTrx(blockTime uint32, maxCount, maxSize int) (entries []*TrxEntry) {
	m.waitingLock.Lock()
	defer m.waitingLock.Unlock()

	m.fetchedLock.Lock()
	defer m.fetchedLock.Unlock()

	counter, size := 0, 0
	// traverse the waiting pool
	for s, e := range m.waiting {
		// check count limit
		if maxCount > 0 && counter >= maxCount {
			break
		}
		// check size limit
		if maxSize > 0 && size >= maxSize {
			break
		}
		// check the transaction again
		// although transactions in the waiting pool had passed checks when they entered,
		// but chain state is keep changing, we have to redo state-dependent checks.
		if err := m.checkTrx(e, blockTime, true); err != nil {
			// if failed, deliver the transaction.
			m.log.Debugf("TRXMGR: FetchTrx check failed: %v, trxId=%x", err, []byte(e.trxId))
			m.deliverEntry(e)
		} else {
			// if passed, pick it
			entries = append(entries, e)
			// add it to the fetched pool
			m.fetched[s] = e
			counter++
			size += e.size
		}
		// remove from waiting pool
		delete(m.waiting, s)
	}
	return
}

// ReturnTrx notifies that some previously fetched transactions can't be packed into a block due to errors.
// Block producer should call ReturnTrx for transactions that failed being applied.
func (m *TrxMgr) ReturnTrx(entries ...*TrxEntry) {
	m.log.Debug("TRXMGR: ReturnTrx begin")
	timing := common.NewTiming()
	timing.Begin()

	m.fetchedLock.Lock()
	defer m.fetchedLock.Unlock()

	timing.Mark()

	for _, e := range entries {
		// any returning transaction should be previously fetched
		f := m.fetched[e.trxId]
		if f != nil {
			m.deliverEntry(f)
			delete(m.fetched, e.trxId)
		}
	}
	timing.End()
	m.log.Debugf("TRXMGR: ReturnTrx end: #tx=%d, %s", len(entries), timing.String())
}

// CheckBlockTrxs checks if transactions of a block are valid.
// If everything is ok, CheckBlockTrxs returns a TrxEntry slice for transactions and nil error, otherwise, a nil slice
// and an error is returned.
func (m *TrxMgr) CheckBlockTrxs(b *prototype.SignedBlock) (entries []*TrxEntry, err error) {
	m.log.Debugf("TRXMGR: CheckBlockTrxs begin %d", b.SignedHeader.Number())
	t0 := common.EasyTimer()
	if count := len(b.Transactions); count > 0 {
		blockTime := b.SignedHeader.Header.Timestamp.UtcSeconds
		errs := make([]error, count)
		entries = make([]*TrxEntry, count)
		errIdx := int32(-1)
		var wg sync.WaitGroup
		wg.Add(count)
		// check transactions asynchronously
		for i := 0; i < count; i++ {
			go func(idx int) {
				defer wg.Done()
				var err error
				trx := b.Transactions[idx].SigTrx
				e := NewTrxMgrEntry(m.chainId, trx, nil)

				// do we need the initial check?
				// yes for transactions that we never met, otherwise no.
				needInitCheck := true

				// if we have met this transaction before, skip initial check and fill up extra information.
				// this voids doing the expensive public key recovery again.
				if ptrx := m.isProcessingTrx(trx); ptrx != nil {
					needInitCheck = false
					e.trxId = ptrx.trxId
					e.size = ptrx.size
					e.signer = ptrx.signer
					e.signerKey = ptrx.signerKey
				}
				// do initial check if necessary
				if needInitCheck {
					err = e.InitCheck()
				}
				// do state-dependent checks
				if err == nil {
					err = m.checkTrx(e, blockTime, true)
				}
				// finalization works
				if err != nil {
					errs[idx] = err
					// remember the first error we met
					atomic.CompareAndSwapInt32(&errIdx, -1, int32(idx))
				} else {
					entries[idx] = e
				}
			}(i)
		}
		wg.Wait()
		if errIdx >= 0 {
			entries = nil
			err = fmt.Errorf("block %d trxs[%d] check failed: %s", b.SignedHeader.Number(), errIdx, errs[errIdx].Error())
		}

		// check duplicate transactions inside the block.
		// it's a must to prevent malicious block producers from replay attacking.
		// m.history won't help here coz it updates in block level instead of transaction level.
		trxSigs, dupTrx := make(map[string]bool), -1
		for idx, e := range entries {
			if trxSigs[e.trxId] {
				dupTrx = idx
				break
			}
			trxSigs[e.trxId] = true
		}
		if dupTrx >= 0 {
			entries = nil
			err = fmt.Errorf("block %d trxs[%d] duplicates", b.SignedHeader.Number(), dupTrx)
		}
	}
	m.log.Debugf("TRXMGR: CheckBlockTrxs end %d: #tx=%d, %v", b.SignedHeader.Number(), len(b.Transactions), t0)
	return
}

// BlockApplied *MUST* be called *AFTER* a block was successfully applied.
func (m *TrxMgr) BlockApplied(b *prototype.SignedBlock) {
	m.log.Debugf("TRXMGR: BlockApplied begin %d", b.SignedHeader.Number())

	timing := common.NewTiming()
	timing.Begin()

	// update head block time
	atomic.StoreUint32(&m.headTime, b.SignedHeader.Header.Timestamp.UtcSeconds)

	// deliver transactions that are waiting final results
	m.waitingLock.Lock()
	m.fetchedLock.Lock()

	timing.Mark()
	for _, txw := range b.Transactions {
		trxId, _ := txw.SigTrx.Id()
		s := string(trxId.Hash)
		if e := m.fetched[s]; e != nil {
			m.deliverEntry(e)
			delete(m.fetched, s)
		}
		if e := m.waiting[s]; e != nil {
			m.deliverEntry(e)
			delete(m.waiting, s)
		}
	}

	// clean expired waiting trxs if necessary
	m.cleanExpiredWaiting()

	// shrink pool memory if necessary
	m.shrinkPoolMemories()

	timing.Mark()

	m.fetchedLock.Unlock()
	m.waitingLock.Unlock()

	// plugin notifications
	m.callPlugins(func(plugin ITrxMgrPlugin) {
		plugin.BlockApplied(b)
	})

	timing.End()
	m.log.Debugf("TRXMGR: BlockApplied end %d: #tx=%d, %s", b.SignedHeader.Number(), len(b.Transactions), timing.String())
	m.log.Debugf("TRXMGR: auth-hit=%v", m.auth.HitRate())
}

// BlockCommitted *MUST* be called *AFTER* a block was successfully committed.
func (m *TrxMgr) BlockCommitted(blockNum uint64) {
	m.log.Debugf("TRXMGR: BlockCommitted begin %d", blockNum)
	t0 := common.EasyTimer()
	// plugin notifications
	m.callPlugins(func(plugin ITrxMgrPlugin) {
		plugin.BlockCommitted(blockNum)
	})
	m.log.Debugf("TRXMGR: BlockCommitted end %d: %v", blockNum, t0)
}

// BlockReverted *MUST* be called *AFTER* a block was successfully reverted.
func (m *TrxMgr) BlockReverted(blockNum uint64) {
	m.log.Debugf("TRXMGR: BlockReverted begin %d", blockNum)
	t0 := common.EasyTimer()
	// plugin notifications
	m.callPlugins(func(plugin ITrxMgrPlugin) {
		plugin.BlockReverted(blockNum)
	})
	m.log.Debugf("TRXMGR: BlockReverted end %d: %v", blockNum, t0)
}

// addToWaiting adds given transaction entries to the waiting pool, and returns the actual number added.
func (m *TrxMgr) addToWaiting(entries...*TrxEntry) (count int) {
	// clean expired waiting trxs if necessary
	m.cleanExpiredWaiting()

	for _, e := range entries {
		// check the max waiting count limit
		if len(m.waiting) > sMaxWaitingCount {
			_ = e.SetError(errors.New("too many waiting trxs"))
			m.deliverEntry(e)
			continue
		}
		// check duplication
		if m.isProcessingNoLock(e.result.SigTrx) != nil {
			_ = e.SetError(errors.New("trx already in process"))
			m.deliverEntry(e)
			continue
		}
		m.waiting[e.trxId] = e
		count++
	}

	atomic.AddUint64(&m.shrinkCounter, uint64(count))
	return
}

// isProcessingTrx is a thread safe version of isProcessingNoLock.
func (m *TrxMgr) isProcessingTrx(trx *prototype.SignedTransaction) *TrxEntry {
	m.waitingLock.RLock()
	defer m.waitingLock.RUnlock()
	m.fetchedLock.RLock()
	defer m.fetchedLock.RUnlock()
	return m.isProcessingNoLock(trx)
}

// isProcessingNoLock checks if given transaction is being processed by TrxMgr.
// It returns the transaction entry if given transaction is in the waiting pool or the fetched pool,
// otherwise, nil is returned.
func (m *TrxMgr) isProcessingNoLock(trx *prototype.SignedTransaction) *TrxEntry {
	if trx == nil {
		return nil
	}
	if trxId, err := trx.Id(); err == nil {
		s := string(trxId.Hash)
		if e := m.waiting[s]; e != nil {
			return e
		}
		return m.fetched[s]
	} else {
		return nil
	}
}

// checkTrx does state-dependent checks on given transaction.
func (m *TrxMgr) checkTrx(e *TrxEntry, blockTime uint32, checkTapos bool) (err error) {
	if err = e.CheckExpiration(blockTime); err != nil {
		return err
	}
	if checkTapos {
		if err = e.CheckTapos(m.tapos); err != nil {
			return err
		}
	}
	if err = e.CheckSignerKey(m.auth); err != nil {
		return err
	}
	if err = e.CheckInBlockTrxs(m.history); err != nil {
		return err
	}
	return
}

// deliverEntry delivers given transaction asynchronously.
func (m *TrxMgr) deliverEntry(e *TrxEntry) {
	go func() {
		e.Deliver()
	}()
}

// callPlugins is a helper method that calls given functor with each plugin as its argument.
func (m *TrxMgr) callPlugins(f func(plugin ITrxMgrPlugin)) {
	var wg sync.WaitGroup
	wg.Add(len(m.plugins))
	for i := range m.plugins {
		go func(idx int) {
			defer wg.Done()
			f(m.plugins[idx])
		}(i)
	}
	wg.Wait()
}

func (m *TrxMgr) DiscardAccountCache(name string) {
	m.auth.Discard(name)
}

//
// clean expired transactions from waiting pool if waiting pool is large enough.
//
// We need a cleaning procedure, especially for non-producer nodes.
// A non-producer node checks each block it applied and removes in-block transactions from the waiting pool.
// Without waiting pool cleaning, erroneous transactions will remain in the pool forever because they will never
// be packed into blocks. This can eventually fill up the waiting pool, leading to huge memory consumption and
// DoS for new transactions.
//
func (m *TrxMgr) cleanExpiredWaiting() {
	// when the waiting pool is small, we don't need cleaning
	if len(m.waiting) < sWaitingCountWaterMark {
		return
	}
	// we avoid frequent cleaning
	if headBlockTime := atomic.LoadUint32(&m.headTime); headBlockTime > 0 && time.Since(m.lastCleanTime) > sMinCleanupInterval {
		m.lastCleanTime = time.Now()
		for k, e := range m.waiting {
			if err := e.CheckExpiration(headBlockTime); err != nil {
				m.deliverEntry(e)
			}
			delete(m.waiting, k)
		}
	}
}

// delete(map, key) won't release any memory occupied by a map.
// so we need to re-copy our pools from time to time, otherwise they're eating memory slowly but forever.
func (m *TrxMgr) shrinkPoolMemories() {
	if atomic.LoadUint64(&m.shrinkCounter) > sShrinkCountWaterMark {
		atomic.StoreUint64(&m.shrinkCounter, 0)

		waiting, fetched := make(map[string]*TrxEntry), make(map[string]*TrxEntry)
		for k, e := range m.waiting {
			waiting[k] = e
		}
		for k, e := range m.fetched {
			fetched[k] = e
		}
		m.waiting, m.fetched = waiting, fetched
	}
}
