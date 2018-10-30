package storage

//
// This file implements Database interface based on map[string][]byte.
// Use MemoryDatabase for debugging only.
//

import (
	"sync"
	"errors"
	"github.com/coschain/contentos-go/common"
	"sort"
)

type MemoryDatabase struct {
	db map[string][]byte
	lock sync.RWMutex
}

func NewMemoryDatabase() *MemoryDatabase {
	return &MemoryDatabase{ db: make(map[string][]byte) }
}

func (db *MemoryDatabase) Close() {

}

//
// DatabaseGetter implementation
//

// check existence of the given key
func (db *MemoryDatabase) Has(key []byte) (bool, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	_, ok := db.db[string(key)]
	return ok, nil
}

// query the value of the given key
func (db *MemoryDatabase) Get(key []byte) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if value, ok := db.db[string(key)]; ok {
		return common.CopyBytes(value), nil
	}
	return nil, errors.New("not found")
}

//
// DatabasePutter implementation
//

// insert a new key-value pair, or update the value if the given key already exists
func (db *MemoryDatabase) Put(key []byte, value []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.db[string(key)] = common.CopyBytes(value)
	return nil
}

//
// DatabaseDeleter implementation
//

// delete the given key and its value
func (db *MemoryDatabase) Delete(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	delete(db.db, string(key))
	return nil
}

//
// DatabaseScanner implementation
//

func (db *MemoryDatabase) NewIterator(start []byte, limit []byte) Iterator {
	db.lock.RLock()
	defer db.lock.RUnlock()

	keys := []string{}
	var startStr, limitStr string
	if start != nil {
		startStr = string(start)
	}
	if (limit != nil) {
		limitStr = string(limit)
	}

	for k := range db.db {
		if start != nil {
			if k < startStr {
				continue
			}
		}
		if limit != nil {
			if k >= limitStr {
				continue
			}
		}
		keys = append(keys, k[:])
	}
	sort.Strings(keys)

	return &MemoryDatabaseIterator{ db:db, keys:keys, index:-1 }
}

func (db *MemoryDatabase) DeleteIterator(it Iterator) {

}

//
// Iterator implementation
//

type MemoryDatabaseIterator struct {
	db *MemoryDatabase
	keys []string
	index int
}

// check if the iterator is a valid position, i.e. safe to call other methods
func (it *MemoryDatabaseIterator) Valid() bool {
	return it.index >= 0 && it.index < len(it.keys)
}

// query the key of current position
func (it *MemoryDatabaseIterator) Key() ([]byte, error) {
	if !it.Valid() {
		return nil, errors.New("invalid iterator")
	}
	return []byte(it.keys[it.index]), nil
}

// query the value of current position
func (it *MemoryDatabaseIterator) Value() ([]byte, error) {
	if !it.Valid() {
		return nil, errors.New("invalid iterator")
	}
	it.db.lock.RLock()
	defer it.db.lock.RUnlock()

	if v, ok := it.db.db[it.keys[it.index]]; ok {
		return common.CopyBytes(v), nil
	}
	return nil, errors.New("not found")
}

// move to the next position
func (it *MemoryDatabaseIterator) Next() bool {
	nextIdx := it.index + 1
	if nextIdx >= 0 && nextIdx < len(it.keys) {
		it.index = nextIdx
		return true
	}
	return false
}


//
// DatabaseBatcher implementation
//

// create a batch which can pack DatabasePutter & DatabaseDeleter operations and execute them atomically
func (db *MemoryDatabase) NewBatch() Batch {
	return &MemoryDatabaseBatch { db:db }
}

// release a Batch
func (db *MemoryDatabase) DeleteBatch(b Batch) {

}

//
// Batch implementation
//

type kvpair struct {
	key, value []byte
	deleted bool
}

type MemoryDatabaseBatch struct {
	db *MemoryDatabase
	op []kvpair
}

// execute all batched operations
func (b *MemoryDatabaseBatch) Write() error {
	b.db.lock.Lock()
	defer b.db.lock.Unlock()

	for _, kv := range b.op {
		if kv.deleted {
			delete(b.db.db, string(kv.key))
		} else {
			b.db.db[string(kv.key)] = kv.value
		}
	}
	return nil
}

// reset the batch to empty
func (b *MemoryDatabaseBatch) Reset() {
	b.op = b.op[:0]
}

func (b *MemoryDatabaseBatch) Put(key []byte, value []byte) error {
	b.op = append(b.op, kvpair{ common.CopyBytes(key), common.CopyBytes(value), false })
	return nil
}

func (b *MemoryDatabaseBatch) Delete(key []byte) error {
	b.op = append(b.op, kvpair{ common.CopyBytes(key), nil, true })
	return nil
}
