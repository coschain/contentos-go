package storage

//
// This file implements transactional feature for any Database interface.
//

import (
	"sync"
	"github.com/coschain/contentos-go/common"
	"bytes"
	"errors"
)

// defines a database writing operation (put or delete)
type writeOp struct {
	key, value []byte
	del bool
}

//
// inTrxDB represents a temporary Database bound with an uncommitted transaction,
// where changes are stored in memory, but not committed to underlying database until commit() is called.
//
type inTrxDB struct {
	db Database
	mem *MemoryDatabase
	changes []writeOp
	removals map[string]bool
	lock sync.RWMutex
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
			if op.del {
				b.Delete(op.key)
			} else {
				b.Put(op.key, op.value)
			}
		}
		b.Write()

		// clear changes
		db.changes = db.changes[:0]
		db.removals = make(map[string]bool)
	}
}

func (db *inTrxDB) Has(key []byte) (bool, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	// memory db first, then underlying db
	found, err := db.mem.Has(key)
	if !found {
		found, err = db.db.Has(key)
	}
	return found, err
}

func (db *inTrxDB) Get(key []byte) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	// memory db first, then underlying db
	data, err := db.mem.Get(key)
	if data == nil {
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
	db.lock.Lock()
	defer db.lock.Unlock()

	// write to mem db only
	err := db.mem.Put(key, value)
	if err == nil {
		// remember this operation
		db.changes = append(db.changes, writeOp{
			key:   common.CopyBytes(key),
			value: common.CopyBytes(value),
			del:   false,
		})
		delete(db.removals, string(key))
	}
	return err
}

func (db *inTrxDB) Delete(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	// write to mem db only
	err := db.mem.Delete(key)
	if err == nil {
		// remember this operation
		db.changes = append(db.changes, writeOp{
			key:   common.CopyBytes(key),
			value: nil,
			del:   true,
		})
		db.removals[string(key)] = true
	}
	return err
}

func (db *inTrxDB) NewIterator(start []byte, limit []byte) Iterator {
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
	return &inTrxIterator{
		memIter: inTrxIteratorItem{
			it: db.mem.NewIterator(start, limit),
			filter: make(map[string]bool),
		},
		dbIter: inTrxIteratorItem{
			it: db.db.NewIterator(start, limit),
			filter: removals,
		},
	}
}

func (db *inTrxDB) DeleteIterator(it Iterator) {

}

// it's an iterator wrapper of either mem db or underlying db
type inTrxIteratorItem struct {
	it Iterator							// the original Iterator
	filter map[string]bool				// keys in the filter must be skipped
	k, v []byte							// key & value of current position
	end bool							// has reached the end
}

type inTrxIterator struct {
	memIter, dbIter inTrxIteratorItem		// the 2 iterators, memIter for mem db, dbIter for underlying db
	selected *inTrxIteratorItem				// where to read key & value
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
		err := item.it.Next()
		if err {
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
	if it.memIter.k != nil && it.dbIter.k != nil {
		if bytes.Compare(it.memIter.k, it.dbIter.k) <= 0 {
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
	return &inTrxDBBatch{ db: db }
}

func (db *inTrxDB) DeleteBatch(b Batch) {

}

// the batch
type inTrxDBBatch struct {
	db *inTrxDB
	changes []writeOp
}

func (b *inTrxDBBatch) Write() error {
	b.db.lock.Lock()
	defer b.db.lock.Unlock()

	for _, op := range b.changes {
		if op.del {
			b.db.Delete(op.key)
		} else {
			b.db.Put(op.key, op.value)
		}
	}
	return nil
}

func (b *inTrxDBBatch) Reset() {
	b.changes = b.changes[:0]
}

func (b *inTrxDBBatch) Put(key []byte, value []byte) error {
	b.changes = append(b.changes, writeOp{
		key:   common.CopyBytes(key),
		value: common.CopyBytes(value),
		del:   false,
	})
	return nil
}

func (b *inTrxDBBatch) Delete(key []byte) error {
	b.changes = append(b.changes, writeOp{
		key:   common.CopyBytes(key),
		value: nil,
		del:   true,
	})
	return nil
}


//
// TransactionalDatabase adds transactional feature on its underlying database
//
type TransactionalDatabase struct {
	db Database					// underlying db
	dirtyRead bool				// dirty-read
	trx []*inTrxDB				// current transaction stack
	lock *sync.RWMutex			// the lock
}


func NewTransactionalDatabase(db Database, dirtyRead bool) *TransactionalDatabase {
	return &TransactionalDatabase{
		db: db,
		dirtyRead: dirtyRead,
		lock: new(sync.RWMutex),
	}
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
		db: trxDb,
		mem: NewMemoryDatabase(),
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
		db.trx = db.trx[:len(db.trx) - 1]
	} else {
		return errors.New("unexpected EndTransaction")
	}
	return nil
}

// get the top-most transaction db
func (db *TransactionalDatabase) topTrx() *inTrxDB {
	if trxCount := len(db.trx); trxCount > 0 {
		return db.trx[trxCount - 1]
	}
	return nil
}

// the Database interface for reading
func (db *TransactionalDatabase) readerDB() Database {
	if db.dirtyRead {
		// when dirty read enabled, read from the top-most transaction db
		if topTrx := db.topTrx(); topTrx != nil {
			return topTrx
		}
	}
	return db.db
}

// the Database interface for writing
func (db *TransactionalDatabase) writerDB() Database {
	// write to the top-most transaction db, if there's one
	if topTrx := db.topTrx(); topTrx != nil {
		return topTrx
	}
	return db.db
}

// return an Database Interface with no support for dirty-read.
func (db *TransactionalDatabase) cleanRead() *TransactionalDatabase {
	if !db.dirtyRead {
		return db
	}
	return &TransactionalDatabase{
		db: db.db,
		dirtyRead: false,
		lock: db.lock,
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
