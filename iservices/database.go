package iservices

//
// This file defines interfaces of Database service.
//

var DbServerName = "db"

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

type IDatabaseServiceId interface {
	ServiceId() uint32
}

type IDatabaseRW interface {
	IDatabaseServiceId

	// check existence of the given key
	Has(key []byte) (bool, error)

	// query the value of the given key
	Get(key []byte) ([]byte, error)

	// insert a new key-value pair, or update the value if the given key already exists
	Put(key []byte, value []byte) error

	// delete the given key and its value
	// if the given key does not exist, just return nil, indicating a successful deletion without doing anything.
	Delete(key []byte) error

	// Iterate enumerates keys in range [start, limit) and calls callback with each enumerated key and its value.
	// If callback returns false, the enumeration stops immediately, otherwise all matched keys will be enumerated.
	// a nil start is the logical minimal key that is lesser than any existing keys
	// a nil limit is the logical maximum key that is greater than any existing keys
	Iterate(start, limit []byte, reverse bool, callback func(key, value []byte) bool)

	// create a batch which can pack DatabasePutter & DatabaseDeleter operations and execute them atomically
	NewBatch() IDatabaseBatch

	// release a Batch
	DeleteBatch(b IDatabaseBatch)
}

type IDatabasePatch interface {
	IDatabaseRW

	// apply the patch
	Apply() error

	// patch on patch
	NewPatch() IDatabasePatch
}

//
// Database Service
//
type IDatabaseService interface {
	IDatabaseRW

	// close a database
	Close()

	//
	// nested transaction feature
	//

	// start a new transaction session
	BeginTransaction()

	// end current transaction session, commit or discard changes
	EndTransaction(commit bool) error

	HashOfTopTransaction() uint32

	// current transaction height
	TransactionHeight() uint

	BeginTransactionWithTag(tag string)

	Squash(tag string) error

	RollbackTag(tag string) error

	//
	// data reversion feature
	//

	// get current revision
	GetRevision() uint64

	// get current revision and base revision
	GetRevisionAndBase() (current uint64, base uint64)

	// revert to the given revision
	// you can only revert to a revision that is less than or equal to current revision.
	// after reverted to revision r, r will be the current revision.
	RevertToRevision(r uint64) error

	// rebase to the given revision
	// after rebased to revision r, r will be the minimal revision you can revert to.
	RebaseToRevision(r uint64) error

	EnableReversion(b bool) error
	ReversionEnabled() bool

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

	//
	// delete-all feature
	//

	// delete everything. make all data gone and unrecoverable.
	// service will stop and restart if already started.
	//
	// this method is *NOT* thread safe. caller *MUST* guarantee that,
	// - all iterators released by DeleteIterator() before calling DeleteAll()
	// - no service calls before successful return of DeleteAll()
	DeleteAll() error

	// R/W locking
	Lock()
	Unlock()
	RLock()
	RUnlock()

	NewPatch() IDatabasePatch
}
