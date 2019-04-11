package storage

import (
	"bytes"
	"github.com/coschain/contentos-go/common"
	"strings"
	"sync"
	"sync/atomic"
)

// a wrapper of key-value pair
type dbKeyValue struct {
	key, value []byte
}

// dbIterator is a database iterator
type dbIterator struct {
	db           Database         // the database
	start, limit []byte           // query key range [start, limit)
	reverse      bool             // result order, true: descending, false: ascending
	started      int32            // has the iteration started?
	finished     int32            // has the iteration finished?
	itemChan     chan *dbKeyValue // channel of matched key-value pairs
	usrBrk       chan struct{}    // channel of user-break signal
	item         *dbKeyValue      // current matched key-value pair
	ignores      map[string]bool  // key set that should be skipped during iteration
	lock         sync.RWMutex     // for thread safety
}

// Next moves the iterator one step forward.
// It's thread safe, returns true if the move was successful, otherwise false.
// If the move was successful, Item() will return a non-nil key-value pair at current position.
func (it *dbIterator) Next() bool {
	it.lock.Lock()
	defer it.lock.Unlock()

	return it.nextNoLock()
}

// nextNoLock moves the iterator one step forward.
// It returns true if the move was successful, otherwise false.
func (it *dbIterator) nextNoLock() (success bool) {
	// if the iteration has finished, just return
	if it.Stopped() {
		return
	}
	// start the iteration if it has not started yet
	if atomic.CompareAndSwapInt32(&it.started, 0, 1) {
		go it.doIteration()
	}
	// wait until we received a matched key-value pair, or a user-break, or a end of iteration.
	stop := false
	for !stop {
		select {
		case <-it.usrBrk:
			// user break
			it.item, success, stop = nil, false, true
		case item, ok := <-it.itemChan:
			if !ok {
				// iteration finished
				it.item, success, stop = nil, false, true
			} else if it.ignores[string(it.item.key)] {
				// received a key-value that should be skipped
				it.item, success, stop = nil, false, false
			} else {
				// received a valid key-value pair
				it.item, success, stop = item, true, true
			}
		}
	}
	return
}

// doIteration iterates the database.
func (it *dbIterator) doIteration() {
	it.db.Iterate(it.start, it.limit, it.reverse, func(key, value []byte) bool {
		select {
		case <-it.usrBrk:
			// user break, stop iteration.
			return false
		case it.itemChan <- &dbKeyValue{key: common.CopyBytes(key), value: common.CopyBytes(value)}:
			// received a key-value pair, put it into channel.
			return true
		}
	})
	// iteration finished, close key-value channel and mark the finish-flag.
	close(it.itemChan)
	atomic.StoreInt32(&it.finished, 1)
}

// Stop sends user-break signal.
func (it *dbIterator) Stop() {
	close(it.usrBrk)
}

// Stopped returns if the iteration has finished.
func (it *dbIterator) Stopped() bool {
	return atomic.LoadInt32(&it.finished) != 0
}

// AddIgnore adds given key into the ignoring set.
// Note that if the newly added key matches current position, a move will be triggered.
func (it *dbIterator) AddIgnore(key string) {
	it.lock.Lock()
	defer it.lock.Unlock()

	// do nothing if iteration finished or the key already in ignoring set.
	if it.Stopped() || it.ignores[key] {
		return
	}
	// prepare for key comparison
	cmp, factor := -1, 1
	if it.reverse {
		factor = -1
	}
	// compare the new key and current position.
	if it.item != nil {
		cmp = strings.Compare(string(it.item.key), key) * factor
	}
	if cmp < 0 {
		// current position is ok, but we need to add the new key into ignoring set
		// for the detection of upcoming positions.
		it.ignores[key] = true
	} else if cmp == 0 {
		// current position got invalidated by the new key, so we have to move forward.
		// and we don't need to save the key, because upcoming positions will never match it for sure.
		it.nextNoLock()
	}
	// do nothing for cmp > 0, because the key has no effect on current and upcoming keys.
}

// Item returns the key-value pair of current position.
func (it *dbIterator) Item() *dbKeyValue {
	it.lock.RLock()
	defer it.lock.RUnlock()
	return it.item
}

// ClearItem nils key-value pair of current position.
func (it *dbIterator) ClearItem() {
	it.lock.Lock()
	defer it.lock.Unlock()
	it.item = nil
}

// dbIteratorGroup is a logical iterator consisting by a group of database iterators.
type dbIteratorGroup struct {
	iters       []*dbIterator				// iterators
	prioritized bool						// if true, iterators are sorted by descending priority order; if false, iterators have same priority.
	reverse     bool						// result order
	selected    int							// selected iterator (its Item() is the value of current position)
	finished    int32						// has the iteration finished?
}

// Next moves the iterator one step forward.
// It returns true if the move was successful, otherwise false.
func (g *dbIteratorGroup) Next() bool {
	if g.Stopped() {
		return false
	}
	// first, nils the item of selected iterator
	if g.selected >= 0 {
		g.iters[g.selected].ClearItem()
		g.selected = -1
	}
	// move all iterators whose item is nil, asynchronously
	iterCount := len(g.iters)
	g.asyncIters(0, iterCount-1, func(it *dbIterator, idx int) {
		// move the iterator if it's not finished and its item is nil.
		if it.Stopped() || it.Item() != nil || !it.Next() {
			return
		}
		// we moved to a new position and got new key-pair saved in it.item.
		// if prioritized, all iterators with lower priority should skip the key we just got.
		if g.prioritized {
			k := string(it.Item().key)
			// add the key to all iterators with lower priority, asynchronously
			g.asyncIters(idx+1, iterCount-1, func(it *dbIterator, idx int) {
				it.AddIgnore(k)
			})
		}
	})
	// select the minimal(maximum) item from all iterators
	factor := 1
	if g.reverse {
		factor = -1
	}
	for i := 0; i < iterCount; i++ {
		if kv := g.iters[i].Item(); kv == nil {
			continue
		} else if g.selected < 0 || bytes.Compare(kv.key, g.iters[g.selected].Item().key)*factor < 0 {
			g.selected = i
		}
	}
	return g.selected >= 0 && !g.Stopped()
}

// Stop stops the iteration.
func (g *dbIteratorGroup) Stop() {
	// mark as finished
	atomic.StoreInt32(&g.finished, 1)
	// stop each iterator
	g.asyncIters(0, len(g.iters)-1, func(it *dbIterator, idx int) {
		it.Stop()
	})
}

// Stopped returns if the iteration has finished.
func (g *dbIteratorGroup) Stopped() bool {
	return atomic.LoadInt32(&g.finished) != 0
}

// asyncIters is a helper method to apply given operation to a range of iterators asynchronously,
// and wait until all done.
func (g *dbIteratorGroup) asyncIters(from, to int, f func(it *dbIterator, idx int)) {
	if f == nil || from < 0 || from >= len(g.iters) || to < from || to >= len(g.iters) {
		return
	}
	count := to - from + 1
	var wg sync.WaitGroup
	wg.Add(count)
	for i := from; i <= to; i++ {
		go func(idx int) {
			defer wg.Done()
			f(g.iters[idx], idx)
		}(i)
	}
	wg.Wait()
}

// Item returns key-value pair of current position.
func (g *dbIteratorGroup) Item() *dbKeyValue {
	if g.selected >= 0 {
		return g.iters[g.selected].Item()
	}
	return nil
}


// dbGroupScanner is a DatabaseScanner for multiple databases
type dbGroupScanner struct {
	databases   []Database					// databases
	prioritized bool						// if true, databases are ordered by descending priority order, if false, all have same priority.
	initIgnores map[string]bool				// initial ignoring set for all databases
}

// newIteratorGroup creates an instance of dbIteratorGroup for given query parameters.
func (s *dbGroupScanner) newIteratorGroup(start, limit []byte, reverse bool) *dbIteratorGroup {
	g := &dbIteratorGroup{
		selected:    -1,
		prioritized: s.prioritized,
		reverse:     reverse,
	}
	for i := range s.databases {
		ignores := make(map[string]bool)
		for k, v := range s.initIgnores {
			ignores[k] = v
		}
		g.iters = append(g.iters, &dbIterator{
			db:       s.databases[i],
			start:    start,
			limit:    limit,
			reverse:  reverse,
			itemChan: make(chan *dbKeyValue),
			usrBrk:   make(chan struct{}),
			ignores:  ignores,
		})
	}
	return g
}

func (s *dbGroupScanner) Iterate(start, limit []byte, reverse bool, callback func(key, value []byte) bool) {
	if callback == nil {
		return
	}
	it := s.newIteratorGroup(start, limit, reverse)
	ok := true
	for ok && it.Next() {
		if kv := it.Item(); kv != nil {
			ok = callback(kv.key, kv.value)
		} else {
			ok = false
		}
		if !ok {
			it.Stop()
			break
		}
	}
}

// NewMergedIterator returns a DatabaseScanner for given group of databases.
// Group members must have zero-intersection of their key spaces. In other words, they must be shards.
func NewMergedIterator(databases []Database) DatabaseScanner {
	return &dbGroupScanner{
		databases:   databases,
		prioritized: false,
	}
}

// NewPatchedIterator returns a DatabaseScanner for given patch and base database.
func NewPatchedIterator(patch, base Database, patchDeletes map[string]bool) DatabaseScanner {
	return &dbGroupScanner{
		databases:   []Database{patch, base},
		prioritized: true,
		initIgnores: patchDeletes,
	}
}
