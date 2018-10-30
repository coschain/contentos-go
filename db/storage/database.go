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
