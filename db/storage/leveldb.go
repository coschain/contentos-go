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
	"github.com/syndtr/goleveldb/leveldb/iterator"
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
	db.db.Close()
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

func (db *LevelDatabase) NewIterator(start []byte, limit []byte) Iterator {
	it := db.db.NewIterator(&util.Range{Start: start, Limit: limit}, nil)
	return &LevelDatabaseIterator{it: it}
}

func (db *LevelDatabase) NewReversedIterator(start []byte, limit []byte) Iterator {
	it := db.db.NewIterator(&util.Range{Start: start, Limit: limit}, nil)
	return &LevelDatabaseIterator{it: it, reversed: true, moved: false, moveLast: it.Last()}
}

func (db *LevelDatabase) DeleteIterator(it Iterator) {
	if levelIt, ok := it.(*LevelDatabaseIterator); ok {
		levelIt.it.Release()
	}
}

//
// Iterator implementation
//

type LevelDatabaseIterator struct {
	it       iterator.Iterator
	reversed bool
	moved    bool
	moveLast bool
}

// check if the iterator is a valid position, i.e. safe to call other methods
func (it *LevelDatabaseIterator) Valid() bool {
	if it.reversed && !it.moved {
		return false
	}
	return it.it.Valid()
}

// query the key of current position
func (it *LevelDatabaseIterator) Key() ([]byte, error) {
	if it.Valid() {
		return it.it.Key(), nil
	} else {
		return nil, errors.New("invalid iterator")
	}
}

// query the value of current position
func (it *LevelDatabaseIterator) Value() ([]byte, error) {
	if it.Valid() {
		return it.it.Value(), nil
	} else {
		return nil, errors.New("invalid iterator")
	}
}

// move to the next position
func (it *LevelDatabaseIterator) Next() bool {
	if it.reversed {
		if !it.moved {
			it.moved = true
			return it.moveLast
		}
		return it.it.Prev()
	}
	return it.it.Next()
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
