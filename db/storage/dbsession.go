package storage

import (
	"github.com/coschain/contentos-go/common"
	"hash/crc32"
	"sync"
)

type dbSession struct {
	sync.RWMutex
	base Database
	puts Database
	dels Database
}

var (
	sDataHashFunc = crc32.ChecksumIEEE
	sDeletedValue = []byte("<deleted>")
	sHashOfDeleted = sDataHashFunc(sDeletedValue)
)

func NewDbSession(base Database) *dbSession {
	return &dbSession{
		base: base,
		puts: NewMemoryDatabase(),
		dels: NewMemoryDatabase(),
	}
}

func (db *dbSession) Close() {

}

func (db *dbSession) commitToDbWriter(w DatabaseWriter) (err error) {
	db.RLock()
	defer db.RUnlock()

	db.puts.Iterate(nil, nil, false, func(key, value []byte) bool {
		err = w.Put(key, value)
		return err == nil
	})
	if err == nil {
		db.dels.Iterate(nil, nil, false, func(key, value []byte) bool {
			err = w.Delete(key)
			return err == nil
		})
	}
	return err
}

// commit all changes to underlying database
func (db *dbSession) commit() (err error) {
	b := db.base.NewBatch()
	if err = db.commitToDbWriter(b); err != nil {
		return err
	}
	return b.Write()
}

func (db *dbSession) Has(key []byte) (bool, error) {
	db.RLock()
	defer db.RUnlock()

	found, err := db.puts.Has(key)
	if !found {
		if found, err = db.dels.Has(key); found {
			return false, nil
		}
		found, err = db.base.Has(key)
	}
	return found, err
}

func (db *dbSession) Get(key []byte) ([]byte, error) {
	db.RLock()
	defer db.RUnlock()

	data, err := db.puts.Get(key)
	if data == nil {
		if deleted, _ := db.dels.Has(key); deleted {
			return nil, err
		}
		// try underlying db
		data, err = db.base.Get(key)
	}
	return data, err
}

func (db *dbSession) put(key []byte, value []byte) error {
	err := db.puts.Put(key, value)
	if err == nil {
		_ = db.dels.Delete(key)
	}
	return err
}

func (db *dbSession) delete(key []byte) error {
	err := db.puts.Delete(key)
	if err == nil {
		_ = db.dels.Put(key, sDeletedValue)
	}
	return err
}

func (db *dbSession) Put(key []byte, value []byte) error {
	db.Lock()
	defer db.Unlock()
	return db.put(key, value)
}

func (db *dbSession) Delete(key []byte) error {
	db.Lock()
	defer db.Unlock()
	return db.delete(key)
}

func (db *dbSession) Iterate(start, limit []byte, reverse bool, callback func(key, value []byte) bool) {
	db.RLock()
	defer db.RUnlock()
	it := NewPatchedIterator(db.puts, db.dels, db.base)
	if it != nil {
		it.Iterate(start, limit, reverse, callback)
	}
}

func (db *dbSession) Hash() (hash uint32) {
	db.RLock()
	defer db.RUnlock()

	db.puts.Iterate(nil, nil, false, func(key, value []byte) bool {
		hash += sDataHashFunc(key)
		hash += sDataHashFunc(value)
		return true
	})
	db.dels.Iterate(nil, nil, false, func(key, value []byte) bool {
		hash += sDataHashFunc(key)
		hash += sHashOfDeleted
		return true
	})
	return
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
	b.db.Lock()
	defer b.db.Unlock()
	for _, op := range b.changes {
		if op.Del {
			_ = b.db.delete(op.Key)
		} else {
			_ = b.db.put(op.Key, op.Value)
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