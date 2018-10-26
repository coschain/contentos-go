package forkdb

import (
	"fmt"

	"github.com/coschain/contentos-go/common"
)

const maxSize = 1024

// DB ...
type DB struct {
	//committed common.BlockID
	start  uint64
	offset uint64
	head   common.BlockID

	list     [][]common.BlockID
	branches map[common.BlockID]common.SignedBlock

	// previous BlockID ===> SignedBlock
	detached map[common.BlockID]common.SignedBlock
}

// NewDB ...
func NewDB() *DB {
	return &DB{
		list:     make([][]common.BlockID, maxSize*2+1),
		branches: make(map[common.BlockID]common.SignedBlock),
		detached: make(map[common.BlockID]common.SignedBlock),
	}
}

// Remove removes a block
func (db *DB) Remove(id common.BlockID) {
	num := id.BlockNum()
	if num > db.head.BlockNum()+1 || num < db.start {
		return
	}
	delete(db.branches, id)
	delete(db.detached, id)
	idx := num - db.start + db.offset
	for i := range db.list[idx] {
		if db.list[idx][i] == id {
			l := len(db.list[idx])
			db.list[idx][i], db.list[idx][l-1] = db.list[idx][l-1], db.list[idx][i]
			db.list[idx] = db.list[idx][:l-1]
		}
	}
}

// FetchBlock fetches a block corresponding to id
func (db *DB) FetchBlock(id common.BlockID) (common.SignedBlock, error) {
	b, ok := db.branches[id]
	if ok {
		return b, nil
	}
	return nil, fmt.Errorf("No block has id of %v", id)
}

// FetchBlockByNum fetches a block corresponding to the block num
func (db *DB) FetchBlockByNum(num uint64) []common.SignedBlock {
	list := db.list[num-db.start+db.offset]
	ret := make([]common.SignedBlock, len(list))
	for i := range list {
		b, _ := db.branches[list[i]]
		ret[i] = b
	}
	return ret
}

// PushBlock adds a block. If any of the forkchain has more than
// maxSize blocks, purge will be triggered.
func (db *DB) PushBlock(b common.SignedBlock) common.SignedBlock {
	id := b.Id()
	num := id.BlockNum()
	if len(db.branches) == 0 {
		db.head = id
		db.start = num
		db.list[0] = append(db.list[0], db.head)
		return b
	}

	if _, ok := db.branches[id]; ok {
		return db.branches[db.head]
	}

	if num > db.head.BlockNum()+1 || num < db.start {
		return db.branches[db.head]
	}
	db.list[num-db.start+db.offset] = append(db.list[num-db.start+db.offset], id)
	db.branches[id] = b
	prev := b.Previous()
	if _, ok := db.branches[prev]; !ok {
		db.detached[prev] = b
	} else {
		db.pushDetached(id)
	}
	db.tryNewHead(id)
	return db.branches[db.head]
}

func (db *DB) pushDetached(id common.BlockID) {
	ok := true
	var b common.SignedBlock
	for ok {
		b, ok = db.detached[id]
		if ok {
			delete(db.detached, id)
			id = b.Id()
			db.tryNewHead(id)
		}
	}
}

func (db *DB) tryNewHead(id common.BlockID) {
	curNum := id.BlockNum()
	if curNum > db.head.BlockNum() {
		db.head = id
		if curNum-db.start >= maxSize {
			db.start++
			db.offset++
		}
		if db.offset >= maxSize {
			db.purge()
		}
	}
}

func (db *DB) purge() {
	var cnt uint64
	for cnt = 0; cnt < db.offset; cnt++ {
		for i := range db.list[cnt] {
			delete(db.branches, db.list[cnt][i])
			delete(db.detached, db.list[cnt][i])
		}
	}

	newList := make([][]common.BlockID, maxSize*2+1)
	copy(newList, db.list[db.offset:])
	db.list = newList
}

// Head returns the head block of the longest chain, returns nil
// if the db is empty
func (db *DB) Head() common.SignedBlock {
	if len(db.branches) == 0 {
		return nil
	}

	return db.branches[db.head]
}

// Pop pops the head block
// NOTE: The only senario Pop should be called is when a fork switch
// occurs, hence the main branch and the fork branch has a common ancestor
// that should NEVER be poped, which also means the main branch cannot be
// poped empty
func (db *DB) Pop() common.SignedBlock {
	ret := db.branches[db.head]
	db.head = ret.Previous()
	if _, ok := db.branches[db.head]; !ok {
		panic("The main branch was poped empty")
	}
	return ret
}

// FetchNewBranch finds the nearest ancestor of b1 and b2, then returns
// the list of the longer chain, starting from the ancestor block
func (db *DB) FetchNewBranch(b1, b2 common.BlockID) []common.BlockID {
	return nil
}

// FetchBlockFromMainBranch returns the num'th block on main branch
func (db *DB) FetchBlockFromMainBranch(num uint64) common.SignedBlock {
	return nil
}

// Commit sets the block pointed by id as irreversible. It peals off all
// other branches, sets id as the start block in list and branches. It
// should be regularly called when a block is commited to save ram.
func (db *DB) Commit(id common.BlockID) {

}
