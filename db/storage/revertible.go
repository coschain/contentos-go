package storage

//
// This file implements TagRevDatabase interface to provide a rollback feature for any Database.
//

import (
	"sync"
	"bytes"
	"encoding/gob"
	"fmt"
	"errors"
	"github.com/coschain/contentos-go/common"
)

const (
	info_prefix = "__rev_info_"
	key_rev_num = info_prefix + "rev_num"
	rev_op_prefix = info_prefix + "op_"
	max_op_key = rev_op_prefix + "fffffffffffffffff"
	min_op_key = rev_op_prefix + "000000000000000"
	key_rev_tags = info_prefix + "rev_tags"
)

type RevertibleDatabase struct {
	db Database
	rev revNumber
	tag revTags
	lock sync.RWMutex
}

type revNumber struct {
	Current uint64		// current revision
	Base uint64			// minimal revision that can be reverted to
}

func encodeRevNumber(r revNumber) []byte {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	enc.Encode(r)
	return buf.Bytes()
}

func decodeRevNumber(data []byte) *revNumber {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var r revNumber
	if err := dec.Decode(&r); err == nil {
		return &r
	}
	return nil
}

func (db *RevertibleDatabase) loadRevNum() {
	data, err := db.db.Get([]byte(key_rev_num))
	if err == nil {
		if r := decodeRevNumber(data); r != nil {
			db.rev = *r
			return
		}
	}
	db.rev = revNumber{0, 0}
}

func NewRevertibleDatabase(db Database) *RevertibleDatabase {
	rdb := RevertibleDatabase{ db: db }
	rdb.lock.Lock()
	defer rdb.lock.Unlock()

	rdb.loadRevNum()
	rdb.loadRevTags()
	return &rdb
}

func keyOfReversionOp(rev uint64) []byte {
	return []byte(fmt.Sprintf("%s%016x", rev_op_prefix, ^rev))
}

func (db *RevertibleDatabase) GetRevision() uint64 {
	curr, _ := db.GetRevisionAndBase()
	return curr
}

func (db *RevertibleDatabase) GetRevisionAndBase() (current uint64, base uint64) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	return db.rev.Current, db.rev.Base
}

func (db *RevertibleDatabase) RevertToRevision(r uint64) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	if r > db.rev.Current {
		return errors.New(fmt.Sprintf("cannot revert to a future revision %d. current revision is %d",
			r, db.rev.Current))
	}
	if r < db.rev.Base {
		return errors.New(fmt.Sprintf("cannot revert to revision %d. %d is the minimal revertible revision",
			r, db.rev.Base))
	}
	if r == db.rev.Current {
		return nil
	}

	b := db.db.NewBatch()
	defer db.db.DeleteBatch(b)

	limit := []byte(max_op_key)
	if r > 0 {
		limit = keyOfReversionOp(r - 1)
	}
	it := db.db.NewIterator([]byte(min_op_key), limit)
	for it.Next() {
		k, _ := it.Key()
		v, _ := it.Value()
		if k != nil && v != nil {
			opSlice := decodeWriteOpSlice(v)
			if opSlice != nil {
				b.Delete(k)
				for _, op := range opSlice {
					if op.Del {
						b.Delete(op.Key)
					} else {
						b.Put(op.Key, op.Value)
					}
				}

			} else {
				return errors.New("invalid revision log")
			}
		} else {
			return errors.New("invalid revision log")
		}
	}
	b.Put([]byte(key_rev_num), encodeRevNumber(revNumber{ r, db.rev.Base }))

	tags := db.revTagsCopy()
	cleanRevTags(&tags, revNumber{ r, db.rev.Base })
	b.Put([]byte(key_rev_tags), encodeRevTags(tags))

	err := b.Write()
	if err == nil {
		db.rev.Current = r
		db.tag = tags
	}
	return err
}

func (db *RevertibleDatabase) RebaseToRevision(r uint64) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	if r > db.rev.Current {
		return errors.New(fmt.Sprintf("cannot rebase to a future revision %d. current revision is %d",
			r, db.rev.Current))
	}
	if r < db.rev.Base {
		return errors.New(fmt.Sprintf("cannot rebase to revision %d. %d is the minimal revertible revision",
			r, db.rev.Base))
	}
	if r == db.rev.Base {
		return nil
	}

	b := db.db.NewBatch()
	defer db.db.DeleteBatch(b)

	start := []byte(min_op_key)
	if r > 0 {
		start = keyOfReversionOp(r - 1)
	}
	it := db.db.NewIterator(start, []byte(max_op_key))
	for it.Next() {
		k, _ := it.Key()
		if k != nil {
			b.Delete(k)

		} else {
			return errors.New("invalid revision log")
		}
	}
	b.Put([]byte(key_rev_num), encodeRevNumber(revNumber{ db.rev.Current, r }))

	tags := db.revTagsCopy()
	cleanRevTags(&tags, revNumber{ db.rev.Current, r })
	b.Put([]byte(key_rev_tags), encodeRevTags(tags))

	err := b.Write()
	if err == nil {
		db.rev.Base = r
		db.tag = tags
	}
	return err
}

func (db *RevertibleDatabase) Close() {

}

func (db *RevertibleDatabase) Has(key []byte) (bool, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	return db.db.Has(key)
}

func (db *RevertibleDatabase) Get(key []byte) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	return db.db.Get(key)
}

func (db *RevertibleDatabase) put(key []byte, value []byte) error {
	b := db.db.NewBatch()
	b.Put(key, value)

	oldValue, err := db.db.Get(key)
	if err != nil {
		b.Put(keyOfReversionOp(db.rev.Current), encodeWriteOpSlice([]writeOp{{key, nil, true}}))
	} else {
		b.Put(keyOfReversionOp(db.rev.Current), encodeWriteOpSlice([]writeOp{{key, oldValue, false}}))
	}
	b.Put([]byte(key_rev_num), encodeRevNumber(revNumber{ db.rev.Current + 1, db.rev.Base }))

	if err = b.Write(); err == nil {
		db.rev.Current++
	}
	return err
}


func (db *RevertibleDatabase) Put(key []byte, value []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	return db.put(key, value)
}

func (db *RevertibleDatabase) delete(key []byte) error {
	oldValue, err := db.db.Get(key)
	if err == nil {
		b := db.db.NewBatch()
		b.Delete(key)
		b.Put(keyOfReversionOp(db.rev.Current), encodeWriteOpSlice([]writeOp{{key, oldValue, false}}))
		b.Put([]byte(key_rev_num), encodeRevNumber(revNumber{ db.rev.Current + 1, db.rev.Base }))

		if err = b.Write(); err == nil {
			db.rev.Current++
		} else {
			return err
		}
	}
	return nil
}

func (db *RevertibleDatabase) Delete(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	return db.delete(key)
}

func (db *RevertibleDatabase) NewIterator(start []byte, limit []byte) Iterator {
	return db.db.NewIterator(start, limit)
}

func (db *RevertibleDatabase) NewReversedIterator(start []byte, limit []byte) Iterator {
	return db.db.NewReversedIterator(start, limit)
}

func (db *RevertibleDatabase) DeleteIterator(it Iterator) {
	db.db.DeleteIterator(it)
}

func (db *RevertibleDatabase) NewBatch() Batch {
	return &revdbBatch{ db: db }
}

func (db *RevertibleDatabase) DeleteBatch(b Batch) {

}

type revdbBatch struct {
	db *RevertibleDatabase
	op []writeOp
}

func (b *revdbBatch) Write() error {
	b.db.lock.Lock()
	defer b.db.lock.Unlock()

	batch := b.db.db.NewBatch()
	var reverts []writeOp

	for _, op := range b.op {
		oldValue, err := b.db.db.Get(op.Key)
		if op.Del && err == nil {
			batch.Delete(op.Key)
			reverts = append([]writeOp{{
				Key: common.CopyBytes(op.Key),
				Value: common.CopyBytes(oldValue),
				Del: false,
			}}, reverts...)
		}
		if !op.Del {
			batch.Put(op.Key, op.Value)
			if err == nil {
				reverts = append([]writeOp{{
					Key: common.CopyBytes(op.Key),
					Value: common.CopyBytes(oldValue),
					Del: false,
				}}, reverts...)
			} else {
				reverts = append([]writeOp{{
					Key: common.CopyBytes(op.Key),
					Value: nil,
					Del: true,
				}}, reverts...)
			}
		}
	}
	batch.Put(keyOfReversionOp(b.db.rev.Current), encodeWriteOpSlice(reverts))
	batch.Put([]byte(key_rev_num), encodeRevNumber(revNumber{ b.db.rev.Current + 1, b.db.rev.Base }))

	err := batch.Write()
	if err == nil {
		b.db.rev.Current++
	}
	return err
}

func (b *revdbBatch) Reset() {
	b.op = b.op[:0]
}

func (b *revdbBatch) Put(key []byte, value []byte) error {
	b.op = append(b.op, writeOp{ common.CopyBytes(key), common.CopyBytes(value), false })
	return nil
}

func (b *revdbBatch) Delete(key []byte) error {
	b.op = append(b.op, writeOp{ common.CopyBytes(key), nil, true })
	return nil
}

//
// tagging
//

type revTags struct {
	Rev2Tag map[uint64]string
 	Tag2Rev map[string]uint64
}

func encodeRevTags(rt revTags) []byte {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	enc.Encode(rt)
	return buf.Bytes()
}

func decodeRevTags(data []byte) *revTags {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var rt revTags
	if err := dec.Decode(&rt); err == nil {
		return &rt
	}
	return nil
}

func (db *RevertibleDatabase) loadRevTags() {
	data, err := db.db.Get([]byte(key_rev_tags))
	if err == nil {
		if t := decodeRevTags(data); t != nil {
			db.tag = *t
			return
		}
	}
	db.tag = revTags{ map[uint64]string{}, map[string]uint64{} }
}

func (db *RevertibleDatabase) revTagsCopy() revTags {
	c := revTags{ map[uint64]string{}, map[string]uint64{} }
	for k, v := range db.tag.Rev2Tag {
		c.Rev2Tag[k] = v
	}
	for k, v := range db.tag.Tag2Rev {
		c.Tag2Rev[k] = v
	}
	return c
}

func cleanRevTags(rt *revTags, rn revNumber) {
	var deletes []uint64
	for r, t := range rt.Rev2Tag {
		if r < rn.Base || r > rn.Current {
			delete(rt.Tag2Rev, t)
			deletes = append(deletes, r)
		}
	}
	for _, r := range deletes {
		delete(rt.Rev2Tag, r)
	}
}

func (db *RevertibleDatabase) TagRevision(r uint64, tag string) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	if r > db.rev.Current || r < db.rev.Base {
		return errors.New(fmt.Sprintf("cannot tag a irreversible revision. %d not in [%d, %d]",
			r, db.rev.Base, db.rev.Current))
	}
	changed := false
	oldtag, _ := db.tag.Rev2Tag[r]
	backup := db.revTagsCopy()
	if len(tag) == 0 {
		if len(oldtag) > 0 {
			delete(db.tag.Rev2Tag, r)
			delete(db.tag.Tag2Rev, oldtag)
			changed = true
		}
	} else {
		if tag != oldtag {
			db.tag.Rev2Tag[r] = tag
			db.tag.Tag2Rev[tag] = r
			delete(db.tag.Tag2Rev, oldtag)
			changed = true
		}
	}
	var err error
	if changed {
		if err = db.db.Put([]byte(key_rev_tags), encodeRevTags(db.tag)); err != nil {
			db.tag = backup
		}
	}
	return err
}

func (db *RevertibleDatabase) GetTagRevision(tag string) (uint64, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	r, ok := db.tag.Tag2Rev[tag]
	if ok {
		return r, nil
	} else {
		return 0, errors.New("tag not found")
	}
}

func (db *RevertibleDatabase) GetRevisionTag(r uint64) string {
	db.lock.RLock()
	defer db.lock.RUnlock()
	return db.tag.Rev2Tag[r]
}

func (db *RevertibleDatabase) RevertToTag(tag string) error {
	if r, err := db.GetTagRevision(tag); err == nil {
		return db.RevertToRevision(r)
	} else {
		return err
	}
}

func (db *RevertibleDatabase) RebaseToTag(tag string) error {
	if r, err := db.GetTagRevision(tag); err == nil {
		return db.RebaseToRevision(r)
	} else {
		return err
	}
}
