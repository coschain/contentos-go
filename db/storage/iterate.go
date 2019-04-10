package storage

import (
	"github.com/coschain/contentos-go/common"
	"sync/atomic"
)

type dbKeyValue struct {
	key, value []byte
}

type dbIterator struct {
	db Database
	start, limit []byte
	reverse bool
	started int32
	itemChan chan *dbKeyValue
	usrBrk chan struct{}
	item *dbKeyValue
	ignores map[string]bool
}

func (it *dbIterator) doIteration() {
	it.db.Iterate(it.start, it.limit, it.reverse, func(key, value []byte) bool {
		select {
		case <-it.usrBrk:
			return false
		case it.itemChan <- &dbKeyValue{ key:common.CopyBytes(key), value:common.CopyBytes(value) }:
			return true
		}
	})
	close(it.itemChan)
}

func (it *dbIterator) next() (success bool) {
	if atomic.CompareAndSwapInt32(&it.started, 0, 1) {
		go it.doIteration()
	}
	ok := false
	select {
	case <-it.usrBrk:
		it.item, success = nil, false
	case it.item, ok = <-it.itemChan:
		if !ok {
			it.item, success = nil, false
		} else if it.ignore(it.item.key) {
			return it.next()
		}
		success = true
	}
	return
}

func (it *dbIterator) ignore(key []byte) bool {
	return it.ignores[string(key)]
}



func NewMergedIterator(databases []Database) *dbIterator {
	return nil
}

func NewPatchedIterator(patch, base Database, patchDeletes map[string]bool) *dbIterator {
	return nil
}

func (it *dbIterator) Iterate(start, limit []byte, reverse bool, callback func(key, value []byte) bool) {

}
