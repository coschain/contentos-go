package iservices

//
// This file defines interfaces of Database service.
//

var DB_SERVER_NAME = "db"

//
// Interface for key iterator
// Iterator is *NOT* thread safe. you *cannot* share the same iterator among concurrent routines.
// but routines are safe to create and use their own iterators by calling NewIterator().
//
// An iterator represents the static view (snapshot) of the database at the time the iterator was created.
// Later changes to the database will not affect the iteration.
//
// Iterators is the same concept as "cursors" in DBMS docs. More specifically, our iterators are forward-only,
// read-only and static "cursors".
//
type IDatabaseIterator interface {
	// check if the iterator is a valid position, i.e. safe to call other methods
	Valid() bool

	// query the key of current position
	Key() ([]byte, error)

	// query the value of current position
	Value() ([]byte, error)

	// move to the next position
	// return true after success move, otherwise, false
	Next() bool
}

//
// interface for transaction executor
// methods must be thread safe
// write operations must be executed atomically
//
type IDatabaseBatch interface {
	// insert a new key-value pair, or update the value if the given key already exists
	Put(key []byte, value []byte) error

	// delete the given key and its value
	// if the given key does not exist, just return nil, indicating a successful deletion without doing anything.
	Delete(key []byte) error

	// execute all batched operations
	Write() error

	// reset the batch to empty
	Reset()
}

//
// Database Service
//
type IDatabaseService interface {
	//
	// basic database operations
	//

	// check existence of the given key
	Has(key []byte) (bool, error)

	// query the value of the given key
	Get(key []byte) ([]byte, error)

	// insert a new key-value pair, or update the value if the given key already exists
	Put(key []byte, value []byte) error

	// delete the given key and its value
	// if the given key does not exist, just return nil, indicating a successful deletion without doing anything.
	Delete(key []byte) error

	// create an iterator containing keys from [start, limit)
	// returned iterator points before the first key of given range
	// a nil start is the logical minimal key that is lesser than any existing keys
	// a nil limit is the logical maximum key that is greater than any existing keys
	NewIterator(start []byte, limit []byte) IDatabaseIterator

	// release an iterator.
	DeleteIterator(it IDatabaseIterator)

	// create a batch which can pack DatabasePutter & DatabaseDeleter operations and execute them atomically
	NewBatch() IDatabaseBatch

	// release a Batch
	DeleteBatch(b IDatabaseBatch)

	// close a database
	Close()

	//
	// nested transaction feature
	//

	// start a new transaction session
	BeginTransaction()

	// end current transaction session, commit or discard changes
	EndTransaction(commit bool) error


	//
	// data reversion feature
	//

	// get current revision
	GetRevision() uint64

	// revert to the given revision
	// you can only revert to a revision that is less than or equal to current revision.
	// after reverted to revision r, r will be the current revision.
	RevertToRevision(r uint64) error

	// rebase to the given revision
	// after rebased to revision r, r will be the minimal revision you can revert to.
	RebaseToRevision(r uint64) error


	//
	// revision tagging feature
	//

	// tag a revision
	TagRevision(r uint64, tag string) error

	// get revision of a tag
	GetTagRevision(tag string) (uint64, error)

	// get tag of a revision
	GetRevisionTag(r uint64) string

	// revert to a revision by its tag
	RevertToTag(tag string) error

	// rebase to a revision by its tag
	RebaseToTag(tag string) error
}
