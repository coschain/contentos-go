package storage

//
// This file implements the database service.
//
// the service uses levelDB as the underlying kv-store solution with additional supports for
// nested transactions and data reversion.
//
// NewDatabaseService() creates a service instance of type DatabaseService.
// DatabaseService implements both node.Service and iservices.IDatabaseService interfaces.
// the former is for service management, and the latter is for real function uses.
//

import (
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/sasha-s/go-deadlock"
	"os"
)

// the service type
type DatabaseService struct {
	path string
	db   *LevelDatabase
	rdb  *RevertibleDatabase
	tdb  *SquashableDatabase
	lock deadlock.RWMutex
}

// service constructor
func NewDatabaseService(ctx *node.ServiceContext, dbPath string) (*DatabaseService, error) {
	if ctx == nil || len(dbPath) == 0 {
		return nil, errors.New("invalid parameter")
	}
	path := ctx.ResolvePath(dbPath)
	if len(path) == 0 {
		return nil, errors.New("cannot resolve database path")
	}
	return &DatabaseService{path: path}, nil
}

func NewDatabase(dbPath string) (*DatabaseService, error) {
	if len(dbPath) == 0 {
		return nil, errors.New("invalid parameter")
	}
	return &DatabaseService{path: dbPath}, nil
}

//
// implementation of Service interface
//

func (s *DatabaseService) Start(node *node.Node) error {
	db, err := NewLevelDatabase(s.path)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to open or create leveldb at %s", s.path))
	}
	rdb := NewRevertibleDatabase(db)
	if rdb == nil {
		db.Close()
		return errors.New("failed to create reversible database")
	}
	tdb := NewSquashableDatabase(rdb, true)
	if tdb == nil {
		rdb.Close()
		db.Close()
		return errors.New("failed to create transactional database")
	}
	s.db, s.rdb, s.tdb = db, rdb, tdb
	return nil
}

func (s *DatabaseService) Stop() error {
	s.Close()
	return nil
}

//
// implementation of TagRevertible interface
//

func (s *DatabaseService) GetRevision() uint64 {
	return s.rdb.GetRevision()
}

func (s *DatabaseService) GetRevisionAndBase() (current uint64, base uint64) {
	return s.rdb.GetRevisionAndBase()
}

func (s *DatabaseService) RevertToRevision(r uint64) error {
	return s.rdb.RevertToRevision(r)
}

func (s *DatabaseService) RebaseToRevision(r uint64) error {
	return s.rdb.RebaseToRevision(r)
}

func (s *DatabaseService) TagRevision(r uint64, tag string) error {
	return s.rdb.TagRevision(r, tag)
}

func (s *DatabaseService) GetTagRevision(tag string) (uint64, error) {
	return s.rdb.GetTagRevision(tag)
}

func (s *DatabaseService) GetRevisionTag(r uint64) string {
	return s.rdb.GetRevisionTag(r)
}

func (s *DatabaseService) RevertToTag(tag string) error {
	return s.rdb.RevertToTag(tag)
}

func (s *DatabaseService) RebaseToTag(tag string) error {
	return s.rdb.RebaseToTag(tag)
}

func (s *DatabaseService) EnableReversion(b bool) error {
	return s.rdb.EnableReversion(b)
}

func (s *DatabaseService) ReversionEnabled() bool {
	return s.rdb.ReversionEnabled()
}

//
// implementation of Squashable interface
//

func (s *DatabaseService) BeginTransaction() {
	s.tdb.BeginTransaction()
}

func (s *DatabaseService) EndTransaction(commit bool) error {
	return s.tdb.EndTransaction(commit)
}

func (s *DatabaseService) TransactionHeight() uint {
	return s.tdb.TransactionHeight()
}

func (s *DatabaseService) BeginTransactionWithTag(tag string) {
	s.tdb.BeginTransactionWithTag(tag)
}

func (s *DatabaseService) Squash(tag string) error {
	return s.tdb.Squash(tag)
}

func (s *DatabaseService) RollbackTag(tag string) error {
	return s.tdb.RollbackTag(tag)
}

//
// implementation of Database interface
//
func (s *DatabaseService) Has(key []byte) (bool, error) {
	return s.tdb.Has(key)
}

func (s *DatabaseService) Get(key []byte) ([]byte, error) {
	return s.tdb.Get(key)
}

func (s *DatabaseService) Put(key []byte, value []byte) error {
	return s.tdb.Put(key, value)
}

func (s *DatabaseService) Delete(key []byte) error {
	return s.tdb.Delete(key)
}

func (s *DatabaseService) Iterate(start, limit []byte, reverse bool, callback func(key, value []byte) bool) {
	s.tdb.Iterate(start, limit, reverse, callback)
}

func (s *DatabaseService) NewBatch() iservices.IDatabaseBatch {
	return s.tdb.NewBatch()
}

func (s *DatabaseService) DeleteBatch(b iservices.IDatabaseBatch) {
	s.tdb.DeleteBatch(b)
}

func (s *DatabaseService) Close() {
	s.tdb.Close()
	s.rdb.Close()
	s.db.Close()
	s.db, s.rdb, s.tdb = nil, nil, nil
}

func (s *DatabaseService) DeleteAll() error {
	var err error
	restart := false
	if s.db != nil {
		err = s.Stop()
		restart = true
	}
	if err != nil {
		return err
	}
	err = os.RemoveAll(s.path)
	if err == nil && restart {
		err = s.Start(nil)
	}
	return err
}

func (s *DatabaseService) Lock() {
	s.lock.Lock()
}

func (s *DatabaseService) Unlock() {
	s.lock.Unlock()
}

func (s *DatabaseService) RLock() {
	s.lock.RLock()
}

func (s *DatabaseService) RUnlock() {
	s.lock.RUnlock()
}

func (s *DatabaseService) NewPatch() iservices.IDatabasePatch {
	return NewDatabasePatch(s.tdb)
}
