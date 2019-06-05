package app

import (
	"bytes"
	"fmt"
	"github.com/coocood/freecache"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/sirupsen/logrus"
	"math/big"
	"sync"
	"sync/atomic"
)

const (
	// maximum cache size (in bytes) for {accountName, publicKey} pairs.
	// assuming that average length of account names is 10 and 33-byte compressed ECC public keys are used,
	// a 16MB cache can hold 384,000 most recently used keys.
	sAuthCacheMaxSize = 16 * 1024 * 1024
)

// AuthFetcher queries the public key of specified account.
// It's designed for best performance by using a memory cache.
type AuthFetcher struct {
	db                     iservices.IDatabaseRW 		// the database
	log                    *logrus.Logger				// the logger
	cache                  *freecache.Cache				// accountName -> publicKey cache
	changes                map[uint64][]string			// block -> accounts changed by this block
	last, commit           uint64						// latest and last committed block
	lock                   sync.RWMutex					// for thread safety
	totalQueries, totalHit int64						// for hit rate stats
}

// NewAuthFetcher creates an instance of AuthFetcher
func NewAuthFetcher(db iservices.IDatabaseRW, logger *logrus.Logger, headBlockNum, lastCommitBlockNum uint64) *AuthFetcher {
	return &AuthFetcher{
		db:      db,
		log:     logger,
		cache:   freecache.NewCache(sAuthCacheMaxSize),
		changes: make(map[uint64][]string),
		last:    headBlockNum,
		commit:  lastCommitBlockNum,
	}
}

// GetPublicKey returns the public key of given account.
// It returns a nil-key and an error if given account not found.
func (f *AuthFetcher) GetPublicKey(account string) (*prototype.PublicKeyType, error) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	// count the query
	atomic.AddInt64(&f.totalQueries, 1)
	// query the cache first
	data, err := f.cache.Get([]byte(account))
	// if cache missed, query the database
	if err != nil {
		auth := table.NewUniAccountNameWrap(f.db).UniQueryName(prototype.NewAccountName(account))
		if auth == nil {
			return nil, fmt.Errorf("auth of %s not found", account)
		}
		key := auth.GetOwner()
		// update cache
		_ = f.cache.Set([]byte(account), key.Data, 0)
		return key, nil
	}
	// count the cache hit
	atomic.AddInt64(&f.totalHit, 1)
	return &prototype.PublicKeyType{Data: data}, nil
}

// HitRate returns cache hit rate, in range [0, 1].
// Hit rate is the most important factor of AuthFetcher's performance, which is roughly proportional to 1/(1-hit_rate).
// If the rate is constantly below 0.99, we should consider increasing sAuthCacheMaxSize.
func (f *AuthFetcher) HitRate() (rate float64) {
	a, b := atomic.LoadInt64(&f.totalHit), atomic.LoadInt64(&f.totalQueries)
	if a > 0 && b > 0 && a <= b {
		rate, _ = big.NewRat(a, b).Float64()
	}
	return
}

// CacheCount returns number of cached {accountName, publicKey} pairs.
func (f *AuthFetcher) CacheCount() int64 {
	return f.cache.EntryCount()
}

// CheckPublicKey checks if given account and its public key are matched.
func (f *AuthFetcher) CheckPublicKey(account string, key *prototype.PublicKeyType) error {
	if expected, err := f.GetPublicKey(account); err != nil {
		return err
	} else if bytes.Compare(expected.Data, key.Data) != 0 {
		return fmt.Errorf("key mismatch, expecting %x, given %x", expected.Data, key.Data)
	}
	return nil
}

// BlockApplied *MUST* be called *AFTER* a block was successfully applied.
func (f *AuthFetcher) BlockApplied(b *prototype.SignedBlock) {
	blockNum := b.GetSignedHeader().Number()

	f.lock.Lock()
	defer f.lock.Unlock()

	if blockNum > f.last {
		// search the block for interested operations
		for _, w := range b.Transactions {
			if w.Receipt.Status == prototype.StatusSuccess {
				for _, op := range w.SigTrx.Trx.Operations {
					switch op.GetOp().(type) {
					// account creation
					case *prototype.Operation_Op1:
						createAccOp := op.GetOp1()
						f.newAccount(blockNum, createAccOp.GetNewAccountName().GetValue(), createAccOp.GetOwner())
					// account update
					case *prototype.Operation_Op20:
						accUpdateOp := op.GetOp20()
						f.newAccount(blockNum, accUpdateOp.GetOwner().GetValue(), accUpdateOp.GetPubkey())
					}
				}
			}
		}
		f.last = blockNum
	}
}

// BlockReverted *MUST* be called *AFTER* a block was successfully reverted.
func (f *AuthFetcher) BlockReverted(blockNum uint64) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if blockNum <= f.last && blockNum > f.commit {
		// for each reverted block
		for i := blockNum; i <= f.last; i++ {
			// remove cache of accounts who were changed by this block
			for _, name := range f.changes[i] {
				f.cache.Del([]byte(name))
			}
			// remove this block from change log
			delete(f.changes, i)
		}
		// fix latest block number
		f.last = blockNum - 1
	}
}

// BlockCommitted *SHOULD* be called *AFTER* a block was successfully committed.
func (f *AuthFetcher) BlockCommitted(blockNum uint64) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if blockNum > f.commit && blockNum <= f.last {
		// remove committed blocks from change log
		// change log is necessary for block reversion, since committed blocks are irreversible, we can safely
		// remove their entries from change log for optimized storage.
		for i := f.commit + 1; i <= blockNum; i++ {
			delete(f.changes, i)
		}
		// fix committed block number
		f.commit = blockNum
	}
}

// newAccount deals with AccountCreateOperation in a block.
func (f *AuthFetcher) newAccount(blockNum uint64, name string, key *prototype.PublicKeyType) {
	// cache the key of newly created account
	_ = f.cache.Set([]byte(name), key.GetData(), 0)
	// remember the change
	f.changes[blockNum] = append(f.changes[blockNum], name)
}
