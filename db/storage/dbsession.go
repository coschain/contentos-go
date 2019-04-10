package storage

import (
	"github.com/coschain/contentos-go/common"
	"sync"
)

type dbSession struct {
	db Database
	mem Database
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

func (db *dbSession) Iterate(start, limit []byte, reverse bool, callback func(key, value []byte) bool) {
	db.lock.RLock()
	defer db.lock.RUnlock()
	if it := NewPatchedIterator(db.mem, db.db, db.removals); it != nil {
		it.Iterate(start, limit, reverse, callback)
	}
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
