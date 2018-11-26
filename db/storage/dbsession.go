package storage

import (
	"bytes"
	"errors"
	"github.com/coschain/contentos-go/common"
	"sync"
)

type dbSession struct {
	db Database
	mem *MemoryDatabase
	changes []writeOp
	removals map[string]bool
	lock sync.RWMutex				// for internal struct data
	dblock sync.RWMutex				// for database operations
}


func (db *dbSession) Close() {

}

// commit all changes to underlying database
func (db *dbSession) commit() (err error) {
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
		err = b.Write()

		if err == nil {
			// clear changes
			db.changes = db.changes[:0]
			db.removals = make(map[string]bool)
		}
	}
	return err
}

func (db *dbSession) Has(key []byte) (bool, error) {
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

func (db *dbSession) Get(key []byte) ([]byte, error) {
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

func (db *dbSession) put(key []byte, value []byte) error {
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

func (db *dbSession) delete(key []byte) error {
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


func (db *dbSession) Put(key []byte, value []byte) error {
	db.dblock.Lock()
	defer db.dblock.Unlock()

	return db.put(key, value)
}

func (db *dbSession) Delete(key []byte) error {
	db.dblock.Lock()
	defer db.dblock.Unlock()

	return db.delete(key)
}

func (db *dbSession) makeIterator(start []byte, limit []byte, reversed bool) Iterator {
	db.lock.RLock()
	defer db.lock.RUnlock()

	// basically, we need to merge iterators of memory db and underlying db.
	// dbSessionIterator is introduced for that job.

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
	return &dbSessionIterator{
		memIter: dbSessionIteratorItem{
			it: mIter,
			filter: make(map[string]bool),
		},
		dbIter: dbSessionIteratorItem{
			it: dIter,
			filter: removals,
		},
		reversed: reversed,
	}
}

func (db *dbSession) NewIterator(start []byte, limit []byte) Iterator {
	return db.makeIterator(start, limit, false)
}

func (db *dbSession) NewReversedIterator(start []byte, limit []byte) Iterator {
	return db.makeIterator(start, limit, true)
}

func (db *dbSession) DeleteIterator(it Iterator) {

}

// it's an iterator wrapper of either mem db or underlying db
type dbSessionIteratorItem struct {
	it Iterator							// the original Iterator
	filter map[string]bool				// keys in the filter must be skipped
	k, v []byte							// key & value of current position
	end bool							// has reached the end
}

type dbSessionIterator struct {
	memIter, dbIter dbSessionIteratorItem  // the 2 iterators, memIter for mem db, dbIter for underlying db
	selected        *dbSessionIteratorItem // where to read key & value
	reversed        bool
}

func (it *dbSessionIterator) Valid() bool {
	return it.selected != nil
}

func (it *dbSessionIterator) Key() ([]byte, error) {
	if it.Valid() {
		return it.selected.k, nil
	}
	return nil, errors.New("invalid iterator")
}

func (it *dbSessionIterator) Value() ([]byte, error) {
	if it.Valid() {
		return it.selected.v, nil
	}
	return nil, errors.New("invalid iterator")
}

// move an iterator
func advanceDbSessionIterItem(item *dbSessionIteratorItem) (moved bool) {
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
func (it *dbSessionIterator) Next() bool {
	// first, consume current key & value
	if it.selected != nil {
		it.selected.k, it.selected.v = nil, nil
		it.selected = nil
	}

	// we will advance any iterator (mem db and/or underlying db), whose key & value were consumed (i.e. == nil)

	// move the mem db iterator if necessary
	if it.memIter.k == nil {
		advanceDbSessionIterItem(&it.memIter)
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
		advanceDbSessionIterItem(&it.dbIter)
	}

	// select the smaller one from 2 iterators
	// if reversed, select the bigger one
	multiplier := 1
	if it.reversed {
		multiplier = -1
	}
	if it.memIter.k != nil && it.dbIter.k != nil {
		if bytes.Compare(it.memIter.k, it.dbIter.k) * multiplier <= 0 {
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

func (db *dbSession) NewBatch() Batch {
	return &dbSessionBatch{ db: db }
}

func (db *dbSession) DeleteBatch(b Batch) {

}

// the batch
type dbSessionBatch struct {
	db *dbSession
	changes []writeOp
}

func (b *dbSessionBatch) Write() error {
	b.db.dblock.Lock()
	defer b.db.dblock.Unlock()

	for _, op := range b.changes {
		if op.Del {
			b.db.delete(op.Key)
		} else {
			b.db.put(op.Key, op.Value)
		}
	}
	return nil
}

func (b *dbSessionBatch) Reset() {
	b.changes = b.changes[:0]
}

func (b *dbSessionBatch) Put(key []byte, value []byte) error {
	b.changes = append(b.changes, writeOp{
		Key:   common.CopyBytes(key),
		Value: common.CopyBytes(value),
		Del:   false,
	})
	return nil
}

func (b *dbSessionBatch) Delete(key []byte) error {
	b.changes = append(b.changes, writeOp{
		Key:   common.CopyBytes(key),
		Value: nil,
		Del:   true,
	})
	return nil
}
