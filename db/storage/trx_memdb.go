package storage

// This file implements TrxDatabase interface based on MemoryDatabase.

type TrxMemoryDatabase struct {
	db  *MemoryDatabase
	trx *TransactionalDatabase
}

func NewTrxMemoryDatabase(file string, dirtyRead bool) (*TrxMemoryDatabase, error) {
	db := NewMemoryDatabase()
	return &TrxMemoryDatabase{db: db, trx: NewTransactionalDatabase(db, dirtyRead)}, nil
}

func (db *TrxMemoryDatabase) Close() {
	db.trx.Close()
	db.db.Close()
}

func (db *TrxMemoryDatabase) Has(key []byte) (bool, error) {
	return db.trx.Has(key)
}

func (db *TrxMemoryDatabase) Get(key []byte) ([]byte, error) {
	return db.trx.Get(key)
}

func (db *TrxMemoryDatabase) Put(key []byte, value []byte) error {
	return db.trx.Put(key, value)
}

func (db *TrxMemoryDatabase) Delete(key []byte) error {
	return db.trx.Delete(key)
}

func (db *TrxMemoryDatabase) NewIterator(start []byte, limit []byte) Iterator {
	return db.trx.NewIterator(start, limit)
}

func (db *TrxMemoryDatabase) NewReversedIterator(start []byte, limit []byte) Iterator {
	return db.trx.NewReversedIterator(start, limit)
}

func (db *TrxMemoryDatabase) DeleteIterator(it Iterator) {
	db.trx.DeleteIterator(it)
}

func (db *TrxMemoryDatabase) NewBatch() Batch {
	return db.trx.NewBatch()
}

func (db *TrxMemoryDatabase) DeleteBatch(b Batch) {
	db.trx.DeleteBatch(b)
}

func (db *TrxMemoryDatabase) BeginTransaction() {
	db.trx.BeginTransaction()
}

func (db *TrxMemoryDatabase) EndTransaction(commit bool) error {
	return db.trx.EndTransaction(commit)
}

func (db *TrxMemoryDatabase) TransactionHeight() uint {
	return db.trx.TransactionHeight()
}
