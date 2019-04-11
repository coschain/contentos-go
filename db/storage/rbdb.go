package storage

import (
	"bytes"
	"errors"
	"github.com/coschain/contentos-go/common"
	"github.com/petar/GoLLRB/llrb"
	"sync"
)

type rbdbItem struct {
	key, value []byte
}

var (
	sMinItem, sMaxItem = llrb.Inf(-1), llrb.Inf(1)
)

func (item *rbdbItem) Less(than llrb.Item) bool {
	if than == sMinItem {
		return false
	} else if than == sMaxItem {
		return true
	} else {
		return bytes.Compare(item.key, than.(*rbdbItem).key) < 0
	}
}

type RedblackDatabase struct {
	rb *llrb.LLRB
	lock sync.RWMutex
}

func NewRedblackDatabase() *RedblackDatabase {
	return &RedblackDatabase{ rb: llrb.New() }
}

func (db *RedblackDatabase) Close() {

}

func (db *RedblackDatabase) Has(key []byte) (bool, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	return db.rb.Has(&rbdbItem{key:key}), nil
}

func (db *RedblackDatabase) Get(key []byte) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if item := db.rb.Get(&rbdbItem{key:key}); item != nil {
		return item.(*rbdbItem).value, nil
	} else {
		return nil, errors.New("not found")
	}
}

func (db *RedblackDatabase) Put(key []byte, value []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	return db.put(key, value)
}

func (db *RedblackDatabase) put(key []byte, value []byte) error {
	db.rb.ReplaceOrInsert(&rbdbItem{key:key, value:value})
	return nil
}

func (db *RedblackDatabase) Delete(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	return db.delete(key)
}

func (db *RedblackDatabase) delete(key []byte) error {
	db.rb.Delete(&rbdbItem{key:key})
	return nil
}

func (db *RedblackDatabase) Iterate(start, limit []byte, reverse bool, callback func(key, value []byte) bool) {
	// we require a read lock to block any writes
	db.lock.RLock()
	defer db.lock.RUnlock()

	if callback == nil {
		return
	}
	// convert start and limit to rbdbItem
	startItem, limitItem := sMinItem, sMaxItem
	if start != nil {
		startItem = &rbdbItem{key:start}
	}
	if limit != nil {
		limitItem = &rbdbItem{key:limit}
	}
	if !reverse {
		// LLRB ascending iteration over a range.
		db.rb.AscendRange(startItem, limitItem, func(item llrb.Item) bool {
			kv := item.(*rbdbItem)
			return callback(kv.key, kv.value)
		})
	} else {
		// LLRB doesn't provide an API for descending iteration over a range.
		// However, this can be done by combination of several other APIs.
		var skip, smallest *rbdbItem

		// first, we do an ascending iteration over [start, +infinity) to get the smallest item.
		db.rb.AscendGreaterOrEqual(startItem, func(item llrb.Item) bool {
			smallest = item.(*rbdbItem)
			return false
		})
		// the smallest item can always be found as long as there exists any item in [start, limit).
		// if not found, we're done since there's no item in [start, limit).
		if smallest == nil {
			return
		}
		// secondly, we get the limit item.
		if limit != nil {
			if item := db.rb.Get(limitItem); item != nil {
				skip = item.(*rbdbItem)
			}
		}

		// lastly, we do a descending iteration over (-infinity, limit]. The range is a superset of [start, limit).
		// we should stop as soon as the smallest item is seen.
		db.rb.DescendLessOrEqual(limitItem, func(item llrb.Item) bool {
			kv := item.(*rbdbItem)
			if kv == skip {
				// skip the limit item
				return true
			}
			return callback(kv.key, kv.value) && kv != smallest
		})
	}
}

func (db *RedblackDatabase) NewBatch() Batch {
	return &rbDatabaseBatch{db: db}
}

func (db *RedblackDatabase) DeleteBatch(b Batch) {

}

type rbDatabaseBatch struct {
	db *RedblackDatabase
	op []writeOp
	lock sync.RWMutex
}

func (b *rbDatabaseBatch) Write() error {
	b.lock.RLock()
	defer b.lock.RUnlock()

	b.db.lock.Lock()
	defer b.db.lock.Unlock()

	for _, kv := range b.op {
		if kv.Del {
			_ = b.db.delete(kv.Key)
		} else {
			_ = b.db.put(kv.Key, kv.Value)
		}
	}
	return nil
}

func (b *rbDatabaseBatch) Reset() {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.op = b.op[:0]
}

func (b *rbDatabaseBatch) Put(key []byte, value []byte) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.op = append(b.op, writeOp{common.CopyBytes(key), common.CopyBytes(value), false})
	return nil
}

func (b *rbDatabaseBatch) Delete(key []byte) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.op = append(b.op, writeOp{common.CopyBytes(key), nil, true})
	return nil
}
