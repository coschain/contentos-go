package app

import (
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/gogo/protobuf/proto"
	"sync"
	"sync/atomic"
)

type TrxCallback func(result *prototype.EstimateTrxResult)

type TrxEntry struct {
	result    *prototype.EstimateTrxResult
	sig       string
	size      int
	signer    string
	signerKey *prototype.PublicKeyType
	callback  TrxCallback
}

func NewTrxMgrEntry(trx *prototype.SignedTransaction, callback TrxCallback) *TrxEntry {
	return &TrxEntry{
		result: &prototype.EstimateTrxResult{
			SigTrx:  trx,
			Receipt: &prototype.TransactionReceiptWithInfo{Status: prototype.StatusSuccess},
		},
		callback: callback,
	}
}

func (e *TrxEntry) SetError(err error) error {
	e.result.Receipt.Status = prototype.StatusError
	e.result.Receipt.ErrorInfo = err.Error()
	return err
}

func (e *TrxEntry) Deliver() {
	if e.callback != nil {
		e.callback(e.result)
	}
}

func (e *TrxEntry) InitCheck() error {
	trx := e.result.SigTrx
	if err := trx.Validate(); err != nil {
		return e.SetError(err)
	}
	e.sig = string(trx.Signature.Sig)
	e.size = proto.Size(trx)
	if e.size > constants.MaxTransactionSize {
		return e.SetError(fmt.Errorf("trx too large, size = %d > %d", e.size, constants.MaxTransactionSize))
	}
	creator := ""
	if creators := trx.GetOpCreatorsMap(); len(creators) != 1 {
		return e.SetError(fmt.Errorf("non-unique trx creators, found %d", len(creators)))
	} else {
		for creator = range creators {
			break
		}
	}
	e.signer = creator
	if signKey, err := trx.ExportPubKeys(prototype.ChainId{Value: 0}); err != nil {
		return e.SetError(fmt.Errorf("cannot export signing key: %s", err.Error()))
	} else {
		e.signerKey = signKey
	}
	return nil
}

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

func (e *TrxEntry) CheckTapos(checker *TaposChecker) error {
	if err := checker.Check(e.result.SigTrx.Trx); err != nil {
		return e.SetError(fmt.Errorf("tapos failed: %s", err.Error()))
	}
	return nil
}

func (e *TrxEntry) CheckSignerKey(fetcher *AuthFetcher) error {
	if err := fetcher.CheckPublicKey(e.signer, e.signerKey); err != nil {
		return e.SetError(fmt.Errorf("signature failed: %s", err.Error()))
	}
	return nil
}

func (e *TrxEntry) CheckInBlockTrxs(checker *InBlockTrxChecker) error {
	if checker.Has(e.result.SigTrx) {
		return e.SetError(errors.New("found duplicate in-block trx"))
	}
	return nil
}

const (
	sMaxWaitingCount  = constants.TrxMaxExpirationTime * 20000
)

type ITrxMgrPlugin interface {
	BlockApplied(b *prototype.SignedBlock)
	BlockReverted(blockNum uint64)
	BlockCommitted(blockNum uint64)
}

type TrxMgr struct {
	db 				iservices.IDatabaseRW
	headTime		uint32
	waiting 		map[string]*TrxEntry
	waitingLock 	sync.RWMutex
	fetched 		map[string]*TrxEntry
	fetchedLock 	sync.RWMutex
	byUser			map[string]map[string]bool
	byUserLock  	sync.RWMutex
	auth 			*AuthFetcher
	tapos 			*TaposChecker
	history 		*InBlockTrxChecker
	plugins         []ITrxMgrPlugin
}

func NewTrxMgr(db iservices.IDatabaseRW, lastBlock, commitBlock uint64) *TrxMgr {
	auth := NewAuthFetcher(db, lastBlock, commitBlock)
	tapos := NewTaposChecker(db, lastBlock)
	history := NewInBlockTrxChecker(db, lastBlock)
	return &TrxMgr{
		db:       db,
		headTime: (&DynamicGlobalPropsRW{db:db}).GetProps().GetTime().GetUtcSeconds(),
		waiting:  make(map[string]*TrxEntry),
		fetched:  make(map[string]*TrxEntry),
		byUser:   make(map[string]map[string]bool),
		auth: auth,
		tapos: tapos,
		history: history,
		plugins: []ITrxMgrPlugin{ auth, tapos, history },
	}
}

func (m *TrxMgr) AddTrx(trx *prototype.SignedTransaction, callback TrxCallback) {
	entry := NewTrxMgrEntry(trx, callback)
	if trx == nil || trx.Signature == nil {
		_ = entry.SetError(errors.New("invalid trx"))
		m.deliverEntry(entry)
		return
	}
	if m.isProcessingTrx(trx) {
		_ = entry.SetError(errors.New("trx already in process"))
		m.deliverEntry(entry)
		return
	}
	go func() {
		if entry.InitCheck() != nil || !m.checkTrx(entry, atomic.LoadUint32(&m.headTime)) {
			m.deliverEntry(entry)
		} else {
			m.waitingLock.Lock()
			defer m.waitingLock.Unlock()
			m.fetchedLock.RLock()
			defer m.fetchedLock.RUnlock()
			m.addToWaiting(entry)
		}
	}()
}

func (m *TrxMgr) FetchTrx(blockTime uint32, maxCount, maxSize int) (results []*prototype.EstimateTrxResult) {
	m.waitingLock.Lock()
	defer m.waitingLock.Unlock()

	m.fetchedLock.Lock()
	defer m.fetchedLock.Unlock()

	counter, size := 0, 0
	for s, e := range m.waiting {
		if maxCount > 0 && counter >= maxCount {
			break
		}
		if maxSize > 0 && size >= maxSize {
			break
		}
		if m.checkTrx(e, blockTime) {
			m.deliverEntry(e)
			continue
		}
		results = append(results, e.result)
		m.fetched[s] = e
		delete(m.waiting, s)
		counter++
		size += e.size
	}
	return
}

func (m *TrxMgr) ReturnTrx(failed bool, results...*prototype.EstimateTrxResult) {
	m.fetchedLock.Lock()
	defer m.fetchedLock.Unlock()

	var waits []*TrxEntry
	for _, r := range results {
		s := string(r.SigTrx.Signature.Sig)
		e := m.fetched[s]
		if e != nil {
			if failed {
				m.deliverEntry(e)
			} else {
				waits = append(waits, e)
			}
			delete(m.fetched, s)
		}
	}
	if len(waits) > 0 {
		m.waitingLock.Lock()
		defer m.waitingLock.Unlock()
		m.addToWaiting(waits...)
	}
}

func (m *TrxMgr) BlockApplied(b *prototype.SignedBlock) {
	atomic.StoreUint32(&m.headTime, b.SignedHeader.Header.Timestamp.UtcSeconds)

	m.fetchedLock.Lock()
	m.waitingLock.Lock()
	for _, txw := range b.Transactions {
		s := string(txw.SigTrx.Signature.Sig)
		if e := m.fetched[s]; e != nil {
			m.deliverEntry(e)
			delete(m.fetched, s)
		}
		if e := m.waiting[s]; e != nil {
			m.deliverEntry(e)
			delete(m.waiting, s)
		}
	}
	m.waitingLock.Unlock()
	m.fetchedLock.Unlock()

	m.callPlugins(func(plugin ITrxMgrPlugin) {
		plugin.BlockApplied(b)
	})
}

func (m *TrxMgr) BlockCommitted(blockNum uint64) {
	m.callPlugins(func(plugin ITrxMgrPlugin) {
		plugin.BlockCommitted(blockNum)
	})
}

func (m *TrxMgr) BlockReverted(blockNum uint64) {
	m.callPlugins(func(plugin ITrxMgrPlugin) {
		plugin.BlockReverted(blockNum)
	})
}

func (m *TrxMgr) addToWaiting(entries...*TrxEntry) {
	for _, e := range entries {
		if len(m.waiting) > sMaxWaitingCount {
			_ = e.SetError(errors.New("too many waiting trxs"))
			m.deliverEntry(e)
			continue
		}
		if m.isProcessingNoLock(e.result.SigTrx) {
			_ = e.SetError(errors.New("trx already in process"))
			m.deliverEntry(e)
			continue
		}
		m.waiting[e.sig] = e
		m.addTrxByUser(e)
	}
}

func (m *TrxMgr) isProcessingTrx(trx *prototype.SignedTransaction) bool {
	m.waitingLock.RLock()
	defer m.waitingLock.RUnlock()
	m.fetchedLock.RLock()
	defer m.fetchedLock.RUnlock()
	return m.isProcessingNoLock(trx)
}

func (m *TrxMgr) isProcessingNoLock(trx *prototype.SignedTransaction) bool {
	s := string(trx.Signature.Sig)
	return m.waiting[s] != nil || m.fetched[s] != nil
}

func (m *TrxMgr) checkTrx(e *TrxEntry, blockTime uint32) bool {
	return e.CheckExpiration(blockTime) == nil &&
		e.CheckTapos(m.tapos) == nil &&
		e.CheckSignerKey(m.auth) == nil &&
		e.CheckInBlockTrxs(m.history) == nil
}

func (m *TrxMgr) addTrxByUser(e *TrxEntry) {
	if e != nil && len(e.sig) > 0 && len(e.signer) > 0 {
		m.byUserLock.Lock()
		defer m.byUserLock.Unlock()
		sigs := m.byUser[e.signer]
		if sigs == nil {
			sigs = make(map[string]bool)
			m.byUser[e.signer] = sigs
		}
		sigs[e.sig] = true
	}
}

func (m *TrxMgr) removeTrxByUser(e *TrxEntry) {
	if e != nil && len(e.sig) > 0 && len(e.signer) > 0 {
		m.byUserLock.Lock()
		defer m.byUserLock.Unlock()
		delete(m.byUser[e.signer], e.sig)
	}
}

func (m *TrxMgr) deliverEntry(e *TrxEntry) {
	go func() {
		m.removeTrxByUser(e)
		e.Deliver()
	}()
}

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
