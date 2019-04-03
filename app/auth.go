package app

import (
	"bytes"
	"fmt"
	"github.com/coocood/freecache"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"sync"
)

const (
	sAuthCacheMaxSize = 16 * 1024 * 1024
)

type AuthFetcher struct {
	db iservices.IDatabaseRW
	cache *freecache.Cache
	changes map[uint64][]string
	last, commit uint64
	lock sync.RWMutex
}

func NewAuthFetcher(db iservices.IDatabaseRW, headBlockNum, lastCommitBlockNum uint64) *AuthFetcher {
	return &AuthFetcher{
		db: db,
		cache: freecache.NewCache(sAuthCacheMaxSize),
		changes: make(map[uint64][]string),
		last: headBlockNum,
		commit: lastCommitBlockNum,
	}
}

func (f *AuthFetcher) GetPublicKey(account string) (*prototype.PublicKeyType, error) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	data, err := f.cache.Get([]byte(account))
	if err != nil {
		auth := table.NewUniAccountAuthorityObjectAccountWrap(f.db).UniQueryAccount(prototype.NewAccountName(account))
		if auth == nil {
			return nil, fmt.Errorf("auth of %s not found", account)
		}
		key := auth.GetOwner().Key
		_ = f.cache.Set([]byte(account), key.Data, 0)
		return key, nil
	}
	return &prototype.PublicKeyType{ Data: data }, nil
}

func (f *AuthFetcher) CheckPublicKey(account string, key *prototype.PublicKeyType) error {
	if expected, err := f.GetPublicKey(account); err != nil {
		return err
	} else if bytes.Compare(expected.Data, key.Data) != 0 {
		return fmt.Errorf("key mismatch, expecting %x, given %x", expected.Data, key.Data)
	}
	return nil
}

func (f *AuthFetcher) BlockApplied(b *prototype.SignedBlock) {
	blockNum := b.GetSignedHeader().Number()

	f.lock.Lock()
	defer f.lock.Unlock()

	if blockNum > f.last {
		for _, w := range b.Transactions {
			if w.Invoice.Status == prototype.StatusSuccess {
				for _, op := range w.SigTrx.Trx.Operations {
					switch op.GetOp().(type) {
					case *prototype.Operation_Op1:
						createAccOp := op.GetOp1()
						f.newAccount(blockNum, createAccOp.GetNewAccountName().GetValue(), createAccOp.GetOwner().GetKey())
					}
				}
			}
		}
		f.last = blockNum
	}
}

func (f *AuthFetcher) BlockReverted(blockNum uint64) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if blockNum <= f.last && blockNum > f.commit {
		for i := blockNum; i <= f.last; i++ {
			for _, name := range f.changes[i] {
				f.cache.Del([]byte(name))
			}
			delete(f.changes, i)
		}
		f.last = blockNum - 1
	}
}

func (f *AuthFetcher) BlockCommitted(blockNum uint64) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if blockNum > f.commit && blockNum <= f.last {
		for i := f.commit + 1; i <= blockNum; i++ {
			delete(f.changes, i)
		}
	}
}

func (f *AuthFetcher) newAccount(blockNum uint64, name string, key *prototype.PublicKeyType) {
	_ = f.cache.Set([]byte(name), key.GetData(), 0)
	f.changes[blockNum] = append(f.changes[blockNum], name)
}
