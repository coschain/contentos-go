package storage

// This file implements TrxDatabase interface based on LevelDB.

type TrxLevelDatabase struct {
	db  *LevelDatabase
	trx *TransactionalDatabase
}

func NewTrxLevelDatabase(file string, dirtyRead bool) (*TrxLevelDatabase, error) {
	db, err := NewLevelDatabase(file)
	if err != nil {
		return nil, err
	}
	return &TrxLevelDatabase{db: db, trx: NewTransactionalDatabase(db, dirtyRead)}, nil
}

func (db *TrxLevelDatabase) Close() {
	db.trx.Close()
	db.db.Close()
}

func (db *TrxLevelDatabase) Has(key []byte) (bool, error) {
	return db.trx.Has(key)
}

func (db *TrxLevelDatabase) Get(key []byte) ([]byte, error) {
	return db.trx.Get(key)
}

func (db *TrxLevelDatabase) Put(key []byte, value []byte) error {
	return db.trx.Put(key, value)
}

func (db *TrxLevelDatabase) Delete(key []byte) error {
	return db.trx.Delete(key)
}

func (db *TrxLevelDatabase) Iterate(start, limit []byte, reverse bool, callback func(key, value []byte) bool) {
	db.trx.Iterate(start, limit, reverse, callback)
}

func (db *TrxLevelDatabase) NewBatch() Batch {
	return db.trx.NewBatch()
}

func (db *TrxLevelDatabase) DeleteBatch(b Batch) {
	db.trx.DeleteBatch(b)
}

func (db *TrxLevelDatabase) BeginTransaction() {
	db.trx.BeginTransaction()
}

func (db *TrxLevelDatabase) EndTransaction(commit bool) error {
	return db.trx.EndTransaction(commit)
}

func (db *TrxLevelDatabase) TransactionHeight() uint {
	return db.trx.TransactionHeight()
}
