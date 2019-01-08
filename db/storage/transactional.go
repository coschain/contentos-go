package storage

//
// This file implements transactional feature for any Database interface.
//

import (
	"bytes"
	"errors"
	"github.com/coschain/contentos-go/common"
	"sync"
)

//
// inTrxDB represents a temporary Database bound with an uncommitted transaction,
// where changes are stored in memory, but not committed to underlying database until commit() is called.
//
type inTrxDB struct {
	db       Database
	mem      *MemoryDatabase
	changes  []writeOp
	removals map[string]bool
	lock     sync.RWMutex // for internal struct data
	dblock   sync.RWMutex // for database operations
}

func (db *inTrxDB) Close() {

}

// commit all changes to underlying database
func (db *inTrxDB) commit() {
	db.lock.Lock()
	defer db.lock.Unlock()

	// create a write batch of underlying database
	// fill the batch with changes and execute it
	if len(db.changes) > 0 {
		b := db.db.NewBatch()
		for _, op := range db.changes {
			if op.Del {
				b.Delete(op.Key)
			} else {
				b.Put(op.Key, op.Value)
			}
		}
		b.Write()

		// clear changes
		db.changes = db.changes[:0]
		db.removals = make(map[string]bool)
	}
}

func (db *inTrxDB) Has(key []byte) (bool, error) {
	db.dblock.RLock()
	defer db.dblock.RUnlock()

	// memory db first, then underlying db
	found, err := db.mem.Has(key)
	if !found {
		db.lock.RLock()
		defer db.lock.RUnlock()

		// if the key was deleted, just return false
		if _, deleted := db.removals[string(key)]; deleted {
			return false, err
		}
		found, err = db.db.Has(key)
	}
	return found, err
}

func (db *inTrxDB) Get(key []byte) ([]byte, error) {
	db.dblock.RLock()
	defer db.dblock.RUnlock()

	// memory db first, then underlying db
	data, err := db.mem.Get(key)
	if data == nil {
		db.lock.RLock()
		defer db.lock.RUnlock()

		// if the key was deleted, just return a not-found error
		if _, deleted := db.removals[string(key)]; deleted {
			return nil, err
		}
		// try underlying db
		data, err = db.db.Get(key)
	}
	return data, err
}

func (db *inTrxDB) Put(key []byte, value []byte) error {
	db.dblock.Lock()
	defer db.dblock.Unlock()

	// write to mem db only
	err := db.mem.Put(key, value)
	if err == nil {
		db.lock.Lock()
		defer db.lock.Unlock()

		// remember this operation
		db.changes = append(db.changes, writeOp{
			Key:   common.CopyBytes(key),
			Value: common.CopyBytes(value),
			Del:   false,
		})
		delete(db.removals, string(key))
	}
	return err
}

func (db *inTrxDB) Delete(key []byte) error {
	db.dblock.Lock()
	defer db.dblock.Unlock()

	// write to mem db only
	err := db.mem.Delete(key)
	if err == nil {
		db.lock.Lock()
		defer db.lock.Unlock()

		// remember this operation
		db.changes = append(db.changes, writeOp{
			Key:   common.CopyBytes(key),
			Value: nil,
			Del:   true,
		})
		db.removals[string(key)] = true
	}
	return err
}

func (db *inTrxDB) makeIterator(start []byte, limit []byte, reversed bool) Iterator {
	db.lock.RLock()
	defer db.lock.RUnlock()

	// basically, we need to merge iterators of memory db and underlying db.
	// inTrxIterator is introduced for that job.

	// memory db never contains removed keys, but underlying db might.
	// so we use removed keys as the initial filter for underlying db iteration.
	removals := make(map[string]bool)
	for k, v := range db.removals {
		removals[k[:]] = v
	}
	var mIter, dIter Iterator
	if reversed {
		mIter = db.mem.NewReversedIterator(start, limit)
		dIter = db.db.NewReversedIterator(start, limit)
	} else {
		mIter = db.mem.NewIterator(start, limit)
		dIter = db.db.NewIterator(start, limit)
	}
	return &inTrxIterator{
		memIter: inTrxIteratorItem{
			it:     mIter,
			filter: make(map[string]bool),
		},
		dbIter: inTrxIteratorItem{
			it:     dIter,
			filter: removals,
		},
		reversed: reversed,
	}
}

func (db *inTrxDB) NewIterator(start []byte, limit []byte) Iterator {
	return db.makeIterator(start, limit, false)
}

func (db *inTrxDB) NewReversedIterator(start []byte, limit []byte) Iterator {
	return db.makeIterator(start, limit, true)
}

func (db *inTrxDB) DeleteIterator(it Iterator) {

}

// it's an iterator wrapper of either mem db or underlying db
type inTrxIteratorItem struct {
	it     Iterator        // the original Iterator
	filter map[string]bool // keys in the filter must be skipped
	k, v   []byte          // key & value of current position
	end    bool            // has reached the end
}

type inTrxIterator struct {
	memIter, dbIter inTrxIteratorItem  // the 2 iterators, memIter for mem db, dbIter for underlying db
	selected        *inTrxIteratorItem // where to read key & value
	reversed        bool
}

func (it *inTrxIterator) Valid() bool {
	return it.selected != nil
}

func (it *inTrxIterator) Key() ([]byte, error) {
	if it.Valid() {
		return it.selected.k, nil
	}
	return nil, errors.New("invalid iterator")
}

func (it *inTrxIterator) Value() ([]byte, error) {
	if it.Valid() {
		return it.selected.v, nil
	}
	return nil, errors.New("invalid iterator")
}

// move an iterator
func advanceIterItem(item *inTrxIteratorItem) (moved bool) {
	moved = false
	for !item.end {
		ok := item.it.Next()
		if !ok {
			// we can't move the iterator any more. it has reached the end.
			item.end = true
			item.k, item.v = nil, nil
		} else {
			// update the key & value of current position
			item.k, _ = item.it.Key()
			item.v, _ = item.it.Value()

			// filter the key
			if item.k != nil {
				if _, found := item.filter[string(item.k)]; !found {
					// the key is valid. job is done.
					moved = true
					break
				} else {
					// the key should be skipped
					item.k, item.v = nil, nil
				}
			}
		}
	}
	return moved
}

// move forward
func (it *inTrxIterator) Next() bool {
	// first, consume current key & value
	if it.selected != nil {
		it.selected.k, it.selected.v = nil, nil
		it.selected = nil
	}

	// we will advance any iterator (mem db and/or underlying db), whose key & value were consumed (i.e. == nil)

	// move the mem db iterator if necessary
	if it.memIter.k == nil {
		advanceIterItem(&it.memIter)
		if it.memIter.k != nil {
			// any key from mem db must override the underlying db
			it.dbIter.filter[string(it.memIter.k)] = true

			// if the key equals to current key of underlying db, discard the latter.
			if it.dbIter.k != nil {
				if bytes.Compare(it.memIter.k, it.dbIter.k) == 0 {
					it.dbIter.k, it.dbIter.v = nil, nil
				}
			}
		}
	}

	// move the underlying db iterator if necessary
	if it.dbIter.k == nil {
		advanceIterItem(&it.dbIter)
	}

	// select the smaller one from 2 iterators
	// if reversed, select the bigger one
	multiplier := 1
	if it.reversed {
		multiplier = -1
	}
	if it.memIter.k != nil && it.dbIter.k != nil {
		if bytes.Compare(it.memIter.k, it.dbIter.k)*multiplier <= 0 {
			it.selected = &it.memIter
		} else {
			it.selected = &it.dbIter
		}
	} else if it.memIter.k != nil && it.dbIter.k == nil {
		it.selected = &it.memIter
	} else if it.memIter.k == nil && it.dbIter.k != nil {
		it.selected = &it.dbIter
	}

	return it.selected != nil
}

func (db *inTrxDB) NewBatch() Batch {
	return &inTrxDBBatch{db: db}
}

func (db *inTrxDB) DeleteBatch(b Batch) {

}

// the batch
type inTrxDBBatch struct {
	db      *inTrxDB
	changes []writeOp
}

func (b *inTrxDBBatch) Write() error {
	for _, op := range b.changes {
		if op.Del {
			b.db.Delete(op.Key)
		} else {
			b.db.Put(op.Key, op.Value)
		}
	}
	return nil
}

func (b *inTrxDBBatch) Reset() {
	b.changes = b.changes[:0]
}

func (b *inTrxDBBatch) Put(key []byte, value []byte) error {
	b.changes = append(b.changes, writeOp{
		Key:   common.CopyBytes(key),
		Value: common.CopyBytes(value),
		Del:   false,
	})
	return nil
}

func (b *inTrxDBBatch) Delete(key []byte) error {
	b.changes = append(b.changes, writeOp{
		Key:   common.CopyBytes(key),
		Value: nil,
		Del:   true,
	})
	return nil
}

//
// TransactionalDatabase adds transactional feature on its underlying database
//
type TransactionalDatabase struct {
	db         Database      // underlying db
	dirtyRead  bool          // dirty-read, true: read from top-most trx, false: read from underlying db
	dirtyWrite bool          // dirty-write, true: write to top-most trx, false: write to underlying db
	trx        []*inTrxDB    // current transaction stack
	lock       *sync.RWMutex // the lock
}

func NewTransactionalDatabase(db Database, dirtyRead bool) *TransactionalDatabase {
	return &TransactionalDatabase{
		db:         db,
		dirtyRead:  dirtyRead,
		dirtyWrite: true,
		lock:       new(sync.RWMutex),
	}
}

// get the top-most transaction db
func (db *TransactionalDatabase) topTrx() *inTrxDB {
	if trxCount := len(db.trx); trxCount > 0 {
		return db.trx[trxCount-1]
	}
	return nil
}

func (db *TransactionalDatabase) dbSelect(preferTopTrx bool) Database {
	selected := db.db
	if preferTopTrx {
		if topTrx := db.topTrx(); topTrx != nil {
			selected = topTrx
		}
	}
	return selected
}

// the Database interface for reading
func (db *TransactionalDatabase) readerDB() Database {
	return db.dbSelect(db.dirtyRead)
}

// the Database interface for writing
func (db *TransactionalDatabase) writerDB() Database {
	return db.dbSelect(db.dirtyWrite)
}

// start a transaction session
func (db *TransactionalDatabase) BeginTransaction() {
	db.lock.Lock()
	defer db.lock.Unlock()

	// transactions are stacked.
	// bottom transaction is the underlying database of top transaction
	var trxDb Database
	if top := db.topTrx(); top != nil {
		trxDb = top
	} else {
		trxDb = db.cleanRead()
	}

	// push the new transaction
	newTrx := inTrxDB{
		db:       trxDb,
		mem:      NewMemoryDatabase(),
		removals: make(map[string]bool),
	}
	db.trx = append(db.trx, &newTrx)
}

// end a transaction session. commit or discard changes
func (db *TransactionalDatabase) EndTransaction(commit bool) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	if commit {
		if trx := db.topTrx(); trx != nil {
			trx.commit()
		}
	}
	if trxnum := len(db.trx); trxnum > 0 {
		db.trx = db.trx[:len(db.trx)-1]
	} else {
		return errors.New("unexpected EndTransaction")
	}
	return nil
}

func (db *TransactionalDatabase) TransactionHeight() uint {
	return uint(len(db.trx))
}

// return an Database Interface with no support for dirty-read.
func (db *TransactionalDatabase) cleanRead() *TransactionalDatabase {
	if !db.dirtyRead && !db.dirtyWrite {
		return db
	}
	return &TransactionalDatabase{
		db:         db.db,
		dirtyRead:  false,
		dirtyWrite: false,
		lock:       db.lock,
	}
}

func (db *TransactionalDatabase) Has(key []byte) (bool, error) {
	return db.readerDB().Has(key)
}

func (db *TransactionalDatabase) Get(key []byte) ([]byte, error) {
	return db.readerDB().Get(key)
}

func (db *TransactionalDatabase) Put(key []byte, value []byte) error {
	return db.writerDB().Put(key, value)
}

func (db *TransactionalDatabase) Delete(key []byte) error {
	return db.writerDB().Delete(key)
}

func (db *TransactionalDatabase) NewIterator(start []byte, limit []byte) Iterator {
	return db.readerDB().NewIterator(start, limit)
}

func (db *TransactionalDatabase) NewReversedIterator(start []byte, limit []byte) Iterator {
	return db.readerDB().NewReversedIterator(start, limit)
}

func (db *TransactionalDatabase) DeleteIterator(it Iterator) {
	db.readerDB().DeleteIterator(it)
}

func (db *TransactionalDatabase) NewBatch() Batch {
	return db.writerDB().NewBatch()
}

func (db *TransactionalDatabase) DeleteBatch(b Batch) {
	db.writerDB().DeleteBatch(b)
}

func (db *TransactionalDatabase) Close() {

}
