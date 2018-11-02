package storage

// interface for insertion and updating
type DatabasePutter interface {
	// insert a new key-value pair, or update the value if the given key already exists
	Put(key []byte, value []byte) error
}

// interface for deletion
type DatabaseDeleter interface {
	// delete the given key and its value
	Delete(key []byte) error
}

// interface for key & value query
type DatabaseGetter interface {
	// check existence of the given key
	Has(key []byte) (bool, error)

	// query the value of the given key
	Get(key []byte) ([]byte, error)
}

// interface for key-space range scan
type DatabaseScanner interface {
	// create an iterator containing keys from [start, limit)
	// returned iterator points before the first key of given range
	// a nil start is the logical minimal key that is lesser than any existing keys
	// a nil limit is the logical maximum key that is greater than any existing keys
	NewIterator(start []byte, limit []byte) Iterator

	// release an iterator.
	DeleteIterator(it Iterator)
}

// interface for key iterator
type Iterator interface {
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

// interface for transactional execution of multiple writes
type DatabaseBatcher interface {
	// create a batch which can pack DatabasePutter & DatabaseDeleter operations and execute them atomically
	NewBatch() Batch

	// release a Batch
	DeleteBatch(b Batch)
}

// interface for transaction executor
type Batch interface {
	DatabasePutter
	DatabaseDeleter

	// execute all batched operations
	Write() error

	// reset the batch to empty
	Reset()
}

// interface for full functional database
type Database interface {
	DatabaseGetter
	DatabasePutter
	DatabaseDeleter
	DatabaseScanner
	DatabaseBatcher
	Close()
}

// interface for transaction feature
type Transactional interface {
	// start a new transaction session
	BeginTransaction()

	// end current transaction session, commit or discard changes
	EndTransaction(commit bool) error
}

// interface for databases that support transactions
type TrxDatabase interface {
	Transactional
	Database
}

// interface for revertible feature
type Revertible interface {
	// get current revision
	GetRevision() uint64

	// revert to the given revision
	// you can only revert to a revision that is less than or equal to current revision.
	// after reverted to revision r, r will be the current revision.
	RevertToRevision(r uint64) error

	// rebase to the given revision
	// after rebased to revision r, r will be the minimal revision you can revert to.
	RebaseToRevision(r uint64) error
}

// interface for databases that support reversion
type RevDatabase interface {
	Revertible
	Database
}

// interface for databases that support reversion and revision tagging
type TagRevertible interface {
	Revertible

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

// interface for databases that support reversion and revision tagging
type TagRevDatabase interface {
	TagRevertible
	Database
}
