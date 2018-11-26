package storage

//
// This file implements transactional feature for any Database interface.
//

import (
	"sync"
	"errors"
)


//
// TransactionalDatabase adds transactional feature on its underlying database
//
type TransactionalDatabase struct {
	db Database					// underlying db
	dirtyRead bool				// dirty-read, true: read from top-most trx, false: read from underlying db
	dirtyWrite bool				// dirty-write, true: write to top-most trx, false: write to underlying db
	trx []*dbSession			// current transaction stack
	lock *sync.RWMutex			// the lock
}


func NewTransactionalDatabase(db Database, dirtyRead bool) *TransactionalDatabase {
	return &TransactionalDatabase{
		db: db,
		dirtyRead: dirtyRead,
		dirtyWrite: true,
		lock: new(sync.RWMutex),
	}
}

// get the top-most transaction db
func (db *TransactionalDatabase) topTrx() *dbSession {
	if trxCount := len(db.trx); trxCount > 0 {
		return db.trx[trxCount - 1]
	}
	return nil
}

func (db *TransactionalDatabase) dbSelect(preferTopTrx bool) Database {
	selected := db.db
	if preferTopTrx {
		if topTrx := db.topTrx(); topTrx != nil {
			selected = topTrx
		}
	}
	return selected
}

// the Database interface for reading
func (db *TransactionalDatabase) readerDB() Database {
	return db.dbSelect(db.dirtyRead)
}

// the Database interface for writing
func (db *TransactionalDatabase) writerDB() Database {
	return db.dbSelect(db.dirtyWrite)
}

// start a transaction session
func (db *TransactionalDatabase) BeginTransaction() {
	db.lock.Lock()
	defer db.lock.Unlock()

	// transactions are stacked.
	// bottom transaction is the underlying database of top transaction
	var trxDb Database
	if top := db.topTrx(); top != nil {
		trxDb = top
	} else {
		trxDb = db.cleanRead()
	}

	// push the new transaction
	newTrx := dbSession{
		db: trxDb,
		mem: NewMemoryDatabase(),
		removals: make(map[string]bool),
	}
	db.trx = append(db.trx, &newTrx)
}

// end a transaction session. commit or discard changes
func (db *TransactionalDatabase) EndTransaction(commit bool) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	var commitErr error
	if commit {
		if trx := db.topTrx(); trx != nil {
			commitErr = trx.commit()
		}
	}
	if trxnum := len(db.trx); trxnum > 0 {
		db.trx = db.trx[:len(db.trx) - 1]
	} else {
		return errors.New("unexpected EndTransaction")
	}
	return commitErr
}

func (db *TransactionalDatabase) TransactionHeight() uint {
	return uint(len(db.trx))
}

// return an Database Interface with no support for dirty-read.
func (db *TransactionalDatabase) cleanRead() *TransactionalDatabase {
	if !db.dirtyRead && !db.dirtyWrite {
		return db
	}
	return &TransactionalDatabase{
		db: db.db,
		dirtyRead: false,
		dirtyWrite: false,
		lock: db.lock,
	}
}

func (db *TransactionalDatabase) Has(key []byte) (bool, error) {
	return db.readerDB().Has(key)
}

func (db *TransactionalDatabase) Get(key []byte) ([]byte, error) {
	return db.readerDB().Get(key)
}

func (db *TransactionalDatabase) Put(key []byte, value []byte) error {
	return db.writerDB().Put(key, value)
}

func (db *TransactionalDatabase) Delete(key []byte) error {
	return db.writerDB().Delete(key)
}

func (db *TransactionalDatabase) NewIterator(start []byte, limit []byte) Iterator {
	return db.readerDB().NewIterator(start, limit)
}

func (db *TransactionalDatabase) NewReversedIterator(start []byte, limit []byte) Iterator {
	return db.readerDB().NewReversedIterator(start, limit)
}

func (db *TransactionalDatabase) DeleteIterator(it Iterator) {
	db.readerDB().DeleteIterator(it)
}

func (db *TransactionalDatabase) NewBatch() Batch {
	return db.writerDB().NewBatch()
}

func (db *TransactionalDatabase) DeleteBatch(b Batch) {
	db.writerDB().DeleteBatch(b)
}

func (db *TransactionalDatabase) Close() {

}
