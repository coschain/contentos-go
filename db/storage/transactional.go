package storage

//
// This file implements transactional feature for any Database interface.
//


//
// TransactionalDatabase adds transactional feature on its underlying database
//
type TransactionalDatabase struct {
	dbDeque
}

func NewTransactionalDatabase(db Database, dirtyRead bool) *TransactionalDatabase {
	return &TransactionalDatabase{
		dbDeque: dbDeque{ db: db, readFront: dirtyRead },
	}
}

// start a transaction session
func (db *TransactionalDatabase) BeginTransaction() {
	db.PushFront()
}

// end a transaction session. commit or discard changes
func (db *TransactionalDatabase) EndTransaction(commit bool) error {
	return db.popFront(commit)
}

func (db *TransactionalDatabase) TransactionHeight() uint {
	return db.Size() - 1
}

func (db *TransactionalDatabase) HashOfTopTransaction() uint32 {
	return db.HashOfTopSession()
}
