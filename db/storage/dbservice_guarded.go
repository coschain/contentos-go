package storage

import (
	"errors"
	"github.com/coschain/contentos-go/node"
)

type GuardedDatabaseService struct {
	DatabaseService
}

func NewGuardedDatabaseService(ctx *node.ServiceContext, dbPath string) (*GuardedDatabaseService, error) {
	svc, err := NewDatabaseService(ctx, dbPath)
	if err != nil {
		return nil, err
	}
	return &GuardedDatabaseService{*svc}, nil
}

//
// Dangerous action: to revert a database with on-going transactions
// Neither reversion nor transactions will fail, but it's probably not your intention.
// This action is explicitly forbidden for safety.
//
func (s *GuardedDatabaseService) checkRevert() (err error) {
	if s.DatabaseService.TransactionHeight() > 0 {
		err = errors.New("Don't revert a database with on-going transactions.")
	}
	return err
}

func (s *GuardedDatabaseService) RevertToRevision(r uint64) error  {
	if err := s.checkRevert(); err != nil {
		return err
	}
	return s.DatabaseService.RevertToRevision(r)
}

func (s *GuardedDatabaseService) RevertToTag(tag string) error {
	if err := s.checkRevert(); err != nil {
		return err
	}
	return s.DatabaseService.RevertToTag(tag)
}
