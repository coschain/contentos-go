package storage

//
// This file implements a logical "Namespace" in a Database.
// "Namespace" simply prefixes its name to keys. For example,
// Given a namespace ns named "alice", ns.Get("age") queries "alice::age" from underlying Database.
//

import "github.com/coschain/contentos-go/common"

type namespace struct {
	db     Database
	name   string
	prefix []byte
	bound  []byte
}

func NewNamespace(db Database, name string) Database {
	return &namespace{
		db:     db,
		name:   name,
		prefix: append([]byte(name), 0),
		bound:  append([]byte(name), 1),
	}
}

func (ns *namespace) Close() {

}

func (ns *namespace) Name() string {
	return ns.name
}

func (ns *namespace) compositeKey(key []byte) []byte {
	ck := common.CopyBytes(ns.prefix)
	if key != nil {
		ck = append(ck, key...)
	}
	return ck
}

func (ns *namespace) decomposeKey(key []byte) []byte {
	return key[len(ns.prefix):]
}

//
// DatabaseGetter implementation
//

// check existence of the given key
func (ns *namespace) Has(key []byte) (bool, error) {
	return ns.db.Has(ns.compositeKey(key))
}

// query the value of the given key
func (ns *namespace) Get(key []byte) ([]byte, error) {
	return ns.db.Get(ns.compositeKey(key))
}

//
// DatabasePutter implementation
//

// insert a new key-value pair, or update the value if the given key already exists
func (ns *namespace) Put(key []byte, value []byte) error {
	return ns.db.Put(ns.compositeKey(key), value)
}

//
// DatabaseDeleter implementation
//

// delete the given key and its value
func (ns *namespace) Delete(key []byte) error {
	return ns.db.Delete(ns.compositeKey(key))
}

//
// DatabaseScanner implementation
//

func (ns *namespace) NewIterator(start []byte, limit []byte) Iterator {
	var newLimit []byte
	if limit == nil {
		newLimit = ns.bound
	} else {
		newLimit = ns.compositeKey(limit)
	}
	return &nsIterator{
		ns: ns,
		it: ns.db.NewIterator(ns.compositeKey(start), newLimit),
	}
}

func (ns *namespace) DeleteIterator(it Iterator) {
	ns.db.DeleteIterator(it)
}

//
// Iterator implementation
//
type nsIterator struct {
	ns *namespace
	it Iterator
}

// check if the iterator is a valid position, i.e. safe to call other methods
func (it *nsIterator) Valid() bool {
	return it.it.Valid()
}

// query the key of current position
func (it *nsIterator) Key() ([]byte, error) {
	k, err := it.it.Key()
	if err == nil {
		k = it.ns.decomposeKey(k)
	}
	return k, err
}

// query the value of current position
func (it *nsIterator) Value() ([]byte, error) {
	return it.it.Value()
}

// move to the next position
func (it *nsIterator) Next() bool {
	return it.it.Next()
}

//
// DatabaseBatcher implementation
//

// create a batch which can pack DatabasePutter & DatabaseDeleter operations and execute them atomically
func (ns *namespace) NewBatch() Batch {
	return &nsBatch{t: ns, b: ns.db.NewBatch()}
}

// release a Batch
func (ns *namespace) DeleteBatch(b Batch) {
	ns.db.DeleteBatch(b)
}

//
// Batch implementation
//

type nsBatch struct {
	t *namespace
	b Batch
}

// execute all batched operations
func (b *nsBatch) Write() error {
	return b.b.Write()
}

// reset the batch to empty
func (b *nsBatch) Reset() {
	b.b.Reset()
}

func (b *nsBatch) Put(key []byte, value []byte) error {
	return b.b.Put(b.t.compositeKey(key), value)
}

func (b *nsBatch) Delete(key []byte) error {
	return b.b.Delete(b.t.compositeKey(key))
}
