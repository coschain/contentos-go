package storage

//
// This file implements Database interface based on levelDB.
//
// Call NewLevelDatabase() to create a new LevelDatabase or open an existing one.
// Call Close() method when the LevelDatabase is no longer needed.
// After using an iterator returned by NewIterator(), you need to call DeleteIterator().
//

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type LevelDatabase struct {
	file string
	db   *leveldb.DB
}

// create a database
func NewLevelDatabase(file string) (*LevelDatabase, error) {
	db, err := leveldb.OpenFile(file, &opt.Options{
		Filter: filter.NewBloomFilter(10),
	})
	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		db, err = leveldb.RecoverFile(file, nil)
	}
	if err != nil {
		return nil, err
	}
	return &LevelDatabase{
		file: file,
		db:   db,
	}, nil
}

// close a database
func (db *LevelDatabase) Close() {
	_ = db.db.Close()
}

// get the disk file path
func (db *LevelDatabase) FileName() string {
	return db.file
}

//
// DatabaseGetter implementation
//

// check existence of the given key
func (db *LevelDatabase) Has(key []byte) (bool, error) {
	return db.db.Has(key, nil)
}

// query the value of the given key
func (db *LevelDatabase) Get(key []byte) ([]byte, error) {
	data, err := db.db.Get(key, nil)
	if err != nil {
		return nil, err
	}
	return data, err
}

//
// DatabasePutter implementation
//

// insert a new key-value pair, or update the value if the given key already exists
func (db *LevelDatabase) Put(key []byte, value []byte) error {
	return db.db.Put(key, value, nil)
}

//
// DatabaseDeleter implementation
//

// delete the given key and its value
func (db *LevelDatabase) Delete(key []byte) error {
	return db.db.Delete(key, nil)
}

//
// DatabaseScanner implementation
//

func (db *LevelDatabase) Iterate(start, limit []byte, reverse bool, callback func(key, value []byte) bool) {
	it := db.db.NewIterator(&util.Range{Start:start, Limit:limit}, nil)
	defer it.Release()

	moves := []func()bool{ it.First, it.Next }
	if reverse {
		moves = []func()bool{ it.Last, it.Prev }
	}
	x, ok := 0, true
	for ok {
		if ok = moves[x](); ok {
			if callback != nil {
				ok = callback(it.Key(), it.Value())
			}
		}
		if x == 0 {
			x++
		}
	}
}

//
// DatabaseBatcher implementation
//

// create a batch which can pack DatabasePutter & DatabaseDeleter operations and execute them atomically
func (db *LevelDatabase) NewBatch() Batch {
	return &LevelDatabaseBatch{db: db.db, b: new(leveldb.Batch)}
}

// release a Batch
func (db *LevelDatabase) DeleteBatch(b Batch) {

}

//
// Batch implementation
//

type LevelDatabaseBatch struct {
	db *leveldb.DB
	b  *leveldb.Batch
}

// execute all batched operations
func (b *LevelDatabaseBatch) Write() error {
	return b.db.Write(b.b, nil)
}

// reset the batch to empty
func (b *LevelDatabaseBatch) Reset() {
	b.b.Reset()
}

func (b *LevelDatabaseBatch) Put(key []byte, value []byte) error {
	b.b.Put(key, value)
	return nil
}

func (b *LevelDatabaseBatch) Delete(key []byte) error {
	b.b.Delete(key)
	return nil
}
