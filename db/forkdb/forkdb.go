package forkdb

import (
	"fmt"
	"os"
	"sync"

	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/logging"
	"github.com/coschain/contentos-go/db/blocklog"
)

const defaultSize = 1024

// DB ...
type DB struct {
	//committed common.BlockID
	start         uint64
	head          common.BlockID
	lastCommitted common.BlockID

	list     [][]common.BlockID
	branches map[common.BlockID]common.ISignedBlock

	// previous BlockID ===> ISignedBlock
	detachedLink map[common.BlockID]common.ISignedBlock

	snapshot blocklog.BLog
	sync.RWMutex
}

// NewDB ...
func NewDB() *DB {
	// TODO: purge the detachedLink
	return &DB{
		list:         make([][]common.BlockID, defaultSize+1),
		branches:     make(map[common.BlockID]common.ISignedBlock),
		detachedLink: make(map[common.BlockID]common.ISignedBlock),
		//detached:     make(map[common.BlockID]common.ISignedBlock),
	}
}

// Snapshot...
func (db *DB) Snapshot(dir string) {
	db.RLock()
	defer db.RUnlock()

	if err := db.snapshot.Open(dir); err != nil {
		panic(err)
	}
	start := db.start
	end := db.head.BlockNum()
	for start <= end {
		for i := 0; i < len(db.list[start-db.start]); i++ {
			b, err := db.fetchBlock(db.list[start-db.start][i])
			if err != nil {
				panic(err)
			}
			if err = db.snapshot.Append(b); err != nil {
				panic(err)
			}
		}
		start++
	}
	logging.CLog().Debugf("[ForkDB][Snapshot] %d blocks stored.", end-db.start+1)
}

// LoadSnapshot...
func (db *DB) LoadSnapshot(avatar []common.ISignedBlock, dir string) {
	db.Lock()
	defer db.Unlock()

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return
	}
	if err := db.snapshot.Open(dir); err != nil {
		panic(err)
	}
	defer db.snapshot.Remove(dir)

	if db.snapshot.Empty() {
		return
	}

	// clear
	db.list = make([][]common.BlockID, defaultSize+1)
	db.branches = make(map[common.BlockID]common.ISignedBlock)
	db.detachedLink = make(map[common.BlockID]common.ISignedBlock)

	size := db.snapshot.Size()
	logging.CLog().Debugf("[ForkDB][LoadSnapshot] %d blocks detected.", size)
	var i int64
	for i = 0; i < size; i++ {
		if err := db.snapshot.ReadBlock(avatar[i], i); err != nil {
			panic(err)
		}
		if i == 0 {
			// TODO: it's gonna be a problem if the node never committed any block
			db.lastCommitted = avatar[i].Id()
		}
		logging.CLog().Debug("loading block #", avatar[i].Id().BlockNum())
		db.pushBlock(avatar[i])
		logging.CLog().Debugf("[ForkDB][LoadSnapshot] restore #%d, id %v, prev %d", avatar[i].Id().BlockNum(), avatar[i].Id(), avatar[i].Previous())
	}
	logging.CLog().Debugf("[ForkDB][LoadSnapshot] %d blocks loaded.", size)
}

// LastCommitted...
func (db *DB) LastCommitted() common.BlockID {
	db.RLock()
	defer db.RUnlock()
	return db.lastCommitted
}

// TotalBlockNum returns the total number of blocks contained in the DB
func (db *DB) TotalBlockNum() int {
	db.RLock()
	defer db.RUnlock()
	return len(db.branches)
}

// FetchBlock fetches a block corresponding to id
func (db *DB) FetchBlock(id common.BlockID) (common.ISignedBlock, error) {
	db.RLock()
	defer db.RUnlock()
	return db.fetchBlock(id)
}

func (db *DB) fetchBlock(id common.BlockID) (common.ISignedBlock, error) {
	b, ok := db.branches[id]
	if ok {
		return b, nil
	}
	return nil, fmt.Errorf("[ForkDB] No block has id of %v", id)
}

// FetchBlockByNum fetches a block corresponding to the block num
func (db *DB) FetchBlockByNum(num uint64) []common.ISignedBlock {
	db.RLock()
	defer db.RUnlock()
	if num < db.start || num > db.head.BlockNum() {
		return nil
	}
	list := db.list[num-db.start]
	ret := make([]common.ISignedBlock, len(list))
	for i := range list {
		b, _ := db.branches[list[i]]
		ret[i] = b
	}
	return ret
}

// PushBlock adds a block. If any of the forkchain has more than
// defaultSize blocks, purge will be triggered.
func (db *DB) PushBlock(b common.ISignedBlock) common.ISignedBlock {
	db.Lock()
	defer db.Unlock()

	return db.pushBlock(b)
}

func (db *DB) pushBlock(b common.ISignedBlock) common.ISignedBlock {
	id := b.Id()
	if db.Illegal(id) {
		return db.branches[db.head]
	}

	num := id.BlockNum()
	if len(db.branches) == 0 {
		db.head = id
		db.start = num
		db.list[0] = append(db.list[0], db.head)
		db.branches[id] = b
		return b
	}

	if _, ok := db.branches[id]; ok {
		return db.branches[db.head]
	}

	if num > db.head.BlockNum()+1 || num < db.start {
		return db.branches[db.head]
	}
	db.list[num-db.start] = append(db.list[num-db.start], id)
	prev := b.Previous()
	if _, ok := db.branches[prev]; !ok {
		db.detachedLink[prev] = b
		//db.detached[id] = b
	} else {
		db.branches[id] = b
		db.tryNewHead(id)
		db.pushDetached(id)
	}
	return db.branches[db.head]
}

func (db *DB) pushDetached(id common.BlockID) {
	ok := true
	var b common.ISignedBlock
	for ok {
		b, ok = db.detachedLink[id]
		if ok {
			delete(db.detachedLink, id)
			id = b.Id()
			db.branches[id] = b
			db.tryNewHead(id)
		}
	}
}

func (db *DB) tryNewHead(id common.BlockID) {
	curNum := id.BlockNum()
	if curNum == db.head.BlockNum()+1 {
		db.head = id
	}
}

// Head returns the head block of the longest chain, returns nil
// if the db is empty
func (db *DB) Head() common.ISignedBlock {
	db.RLock()
	defer db.RUnlock()
	if len(db.branches) == 0 {
		return nil
	}

	return db.branches[db.head]
}

// Empty returns true if DB contains no block
func (db *DB) Empty() bool {
	db.RLock()
	defer db.RUnlock()
	return db.head == common.EmptyBlockID
}

// Pop pops the head block
// NOTE: The only scenarios Pop should be called are when:
//  1.a fork switch occurs, hence the main branch and the fork
// 	  branch has a common ancestor that should NEVER be popped,
//    which also means the main branch cannot be popped empty
//  2.the newly appended block contains illegal transactions
// Popping an empty db results in undefined behaviour
func (db *DB) Pop() common.ISignedBlock {
	db.Lock()
	defer db.Unlock()
	ret := db.branches[db.head]
	db.head = ret.Previous()

	return ret
}

// FetchBranch finds the nearest ancestor of id1 and id2, then returns
// the 2 branches
func (db *DB) FetchBranch(id1, id2 common.BlockID) ([2][]common.BlockID, error) {
	db.RLock()
	defer db.RUnlock()
	num1 := id1.BlockNum()
	num2 := id2.BlockNum()
	tid1 := id1
	tid2 := id2
	var ret [2][]common.BlockID
	for num1 > num2 {
		ret[0] = append(ret[0], tid1)
		if b, err := db.getPrevID(tid1); err == nil {
			tid1 = b
			num1 = tid1.BlockNum()
		}
	}
	for num1 < num2 {
		ret[1] = append(ret[1], tid2)
		if b, err := db.getPrevID(tid2); err == nil {
			tid2 = b
			num2 = tid2.BlockNum()
		}
	}

	headNum := db.head.BlockNum()
	//for tid1 != tid2 && tid1.BlockNum() <= headNum-defaultSize {
	for tid1 != tid2 && tid1.BlockNum()+defaultSize > headNum {
		ret[0] = append(ret[0], tid1)
		ret[1] = append(ret[1], tid2)
		tmp, err := db.FetchBlock(tid1)
		if err != nil {
			return ret, err
		}
		tid1 = tmp.Previous()
		tmp, err = db.FetchBlock(tid2)
		if err != nil {
			return ret, err
		}
		tid2 = tmp.Previous()
	}
	if tid1 == tid2 {
		ret[0] = append(ret[0], tid1)
		ret[1] = append(ret[1], tid2)
	} else {
		// This can happen when multiple fork exist and grows simultaneously. To avoid
		// this, call Commit regularly
		errStr := fmt.Sprintf("[ForkDB] cannot find ancestor of %v and %v, unable to switch fork", id1, id2)
		panic(errStr)
	}

	return ret, nil
}

func (db *DB) getPrevID(id common.BlockID) (common.BlockID, error) {
	b, ok := db.branches[id]
	if !ok {
		return common.BlockID{}, fmt.Errorf("[ForkDB] absent key: %v", id)
	}
	return b.Previous(), nil

}

// FetchBlockFromMainBranch returns the num'th block on main branch
func (db *DB) FetchBlockFromMainBranch(num uint64) (common.ISignedBlock, error) {
	db.RLock()
	defer db.RUnlock()
	headNum := db.head.BlockNum()
	if num > headNum || num < db.start {
		return nil, fmt.Errorf("[ForkDB] num out of scope: %d [%d, %d]", num, db.start, headNum)
	}

	var ret common.ISignedBlock
	var err error
	cur := db.head
	for headNum >= num {
		ret, err = db.fetchBlock(cur)
		if err != nil {
			return nil, err
		}
		cur = ret.Previous()
		headNum = cur.BlockNum()
	}
	return ret, nil
}

// FetchBlocksSince fetches the main branch starting from id
func (db *DB) FetchBlocksSince(id common.BlockID) ([]common.ISignedBlock, []common.BlockID, error) {
	db.RLock()
	defer db.RUnlock()
	length := db.head.BlockNum() - id.BlockNum() + 1
	list := make([]common.ISignedBlock, length)
	list1 := make([]common.BlockID, length)
	cur := db.head
	var idx int
	for idx = int(length - 1); idx >= 0; idx-- {
		b, err := db.FetchBlock(cur)
		if err != nil {
			return nil, nil, err
		}
		list[idx] = b
		list1[idx] = cur
		cur = b.Previous()
	}
	if list1[0] != id {
		errStr := fmt.Sprintf("block %v is not on main branch", id)
		panic(errStr)
	}
	return list, list1, nil
}

// Commit sets the block pointed by id as irreversible. It peals off all
// other branches, sets id as the start block. It should be regularly
// called when a block is commited to save ram.
func (db *DB) Commit(id common.BlockID) {
	db.Lock()
	defer db.Unlock()
	if _, ok := db.branches[id]; !ok {
		panic("tried to commit a detached or non-exist block")
	}
	newList := make([][]common.BlockID, defaultSize+1)
	newBranches := make(map[common.BlockID]common.ISignedBlock)
	commitNum := id.BlockNum()
	startNum := commitNum + 1
	endNum := db.head.BlockNum()

	// copy all the valid block after the committed block
	newList[0] = append(newList[0], id)
	newBranches[id] = db.branches[id]
	for startNum <= endNum {
		for i := 0; i < len(db.list[startNum-db.start]); i++ {
			newId := db.list[startNum-db.start][i]
			b, err := db.fetchBlock(newId)
			if err != nil {
				continue
			}

			prev := b.Previous()
			detached := true
			for j := 0; j < len(newList[startNum-commitNum-1]); j++ {
				if newList[startNum-commitNum-1][j] == prev {
					detached = false
					break
				}
			}
			if !detached {
				newList[startNum-commitNum] = append(newList[startNum-commitNum], newId)
				newBranches[newId] = db.branches[newId]
			}
		}
		startNum++
	}

	// purge the branches
	db.list = newList
	db.branches = newBranches
	db.start = id.BlockNum()
	db.lastCommitted = id
}

// Illegal determines if the block has illegal transactions
func (db *DB) Illegal(id common.BlockID) bool {
	// TODO:
	return false
}

// MarkAsIllegal put the block in a blacklist to prevent DDoS attack
func (db *DB) MarkAsIllegal(id common.BlockID) {}
