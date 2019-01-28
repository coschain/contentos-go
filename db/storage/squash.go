package storage

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/pkg/errors"
	"strconv"
	"sync"
)

const SQUASH_COMMIT_NUM  = "commit_num"

type SquashableDatabase struct {
	TransactionalDatabase

	tags map[string]uint
	tagsByIdx map[uint]string
	lock sync.RWMutex
}

func NewSquashableDatabase(db Database, dirtyRead bool) *SquashableDatabase {
	return &SquashableDatabase{
		TransactionalDatabase: TransactionalDatabase{
			dbDeque: dbDeque{
				db:        db,
				readFront: dirtyRead,
			},
		},
		tags: make(map[string]uint),
		tagsByIdx: make(map[uint]string),
	}
}

func (db *SquashableDatabase) BeginTransaction() {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.TransactionalDatabase.BeginTransaction()
}

func (db *SquashableDatabase) EndTransaction(commit bool) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	err := db.TransactionalDatabase.EndTransaction(commit)

	frontIdx := db.Size() - 1
	if poppedTag, ok := db.tagsByIdx[frontIdx]; ok {
		delete(db.tagsByIdx, frontIdx)
		delete(db.tags, poppedTag)
	}

	return err
}

func (db *SquashableDatabase) BeginTransactionWithTag(tag string) {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.TransactionalDatabase.BeginTransaction()
	frontIdx := db.Size() - 2
	db.tags[tag] = frontIdx
	db.tagsByIdx[frontIdx] = tag
}

func (db *SquashableDatabase) Squash(tag string) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	if idx, ok := db.tags[tag]; ok {
		count := int(idx) + 1
		for i := 0; i < count; i++ {
			if err := db.PopBack(true); err != nil {
				fmt.Printf("pop fail,the error is %s",err)
				return err
			}
		}
		newTags := make(map[string]uint)
		newTagsByIdx := make(map[uint]string)
		for i, t := range db.tagsByIdx {
			if i > idx {
				newTagsByIdx[i - idx - 1] = t
				newTags[t] = i - idx - 1
			}
		}
		db.tags, db.tagsByIdx = newTags, newTagsByIdx
		//save the current commit number
		buf,err := encodeCommitNum(tag)
		if err != nil {
			return err
		}
		err = db.db.Put([]byte(SQUASH_COMMIT_NUM),buf)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("unknown tag: " + tag)
}

func (db *SquashableDatabase) RollBackToTag(tag string) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	if idx, ok := db.tags[tag]; ok {
		count := int(db.Size()-1) - int(idx)
		for i := 0; i < count; i++ {
			if err := db.PopFront(false); err != nil {
				return err
			}
		}
		newTags := make(map[string]uint)
		newTagsByIdx := make(map[uint]string)
		for i, t := range db.tagsByIdx {
			if i < idx {
				newTagsByIdx[i] = t
				newTags[t] = i
			}
		}
		db.tags, db.tagsByIdx = newTags, newTagsByIdx
		return nil
	}
	return errors.New("unknown tag: " + tag)
}

func (db *SquashableDatabase) GetCommitNum() (uint64,error) {
	key := []byte(SQUASH_COMMIT_NUM)
	var num uint64 = 0
	exi,err := db.db.Has(key)
	if err != nil {
		return 0, err
	}
	if exi == false {
		//has not commit any block
		return 0, nil
	}
	if buf,err := db.db.Get(key); err == nil {
		 val,dErr := decodeCommitNum(buf)
		 if dErr != nil {
		 	return 0,dErr
		 }
		 num,err = strconv.ParseUint(val,10,64)
	}
	return num,err
}

func encodeCommitNum(num string) ([]byte,error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(num)
	if err != nil {
		return nil,err
	}
	return buf.Bytes(),nil
}

func decodeCommitNum(data []byte) (string,error) {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var num  = "0"
	if err := dec.Decode(&num); err != nil {
		return "0",err
	}
	return num,nil
}
