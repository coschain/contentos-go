package storage

//
// This file implements Database interface based on map[string][]byte.
//

import (
	"errors"
	"github.com/coschain/contentos-go/common"
	"sync"
)

type MemoryDatabase struct {
	db   map[string][]byte
	lock sync.RWMutex
}

func NewMemoryDatabase() *MemoryDatabase {
	return &MemoryDatabase{db: make(map[string][]byte)}
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

func (db *MemoryDatabase) Iterate(start, limit []byte, reverse bool, callback func(key, value []byte) bool) {
	// todo: not implemented yet
}

//
// DatabaseBatcher implementation
//

// create a batch which can pack DatabasePutter & DatabaseDeleter operations and execute them atomically
func (db *MemoryDatabase) NewBatch() Batch {
	return &memoryDatabaseBatch{db: db}
}

// release a Batch
func (db *MemoryDatabase) DeleteBatch(b Batch) {

}

//
// Batch implementation
//

type memoryDatabaseBatch struct {
	db *MemoryDatabase
	op []writeOp
}

// execute all batched operations
func (b *memoryDatabaseBatch) Write() error {
	b.db.lock.Lock()
	defer b.db.lock.Unlock()

	for _, kv := range b.op {
		if kv.Del {
			delete(b.db.db, string(kv.Key))
		} else {
			b.db.db[string(kv.Key)] = kv.Value
		}
	}
	return nil
}

// reset the batch to empty
func (b *memoryDatabaseBatch) Reset() {
	b.op = b.op[:0]
}

func (b *memoryDatabaseBatch) Put(key []byte, value []byte) error {
	b.op = append(b.op, writeOp{common.CopyBytes(key), common.CopyBytes(value), false})
	return nil
}

func (b *memoryDatabaseBatch) Delete(key []byte) error {
	b.op = append(b.op, writeOp{common.CopyBytes(key), nil, true})
	return nil
}
