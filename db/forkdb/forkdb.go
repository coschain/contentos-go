package forkdb

import (
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

}

// FetchBlock fetches a block corresponding to id
func (db *DB) FetchBlock(id common.BlockID) (common.SignedBlock, error) {
	return nil, nil
}

// FetchBlockByNum fetches a block corresponding to the block num
func (db *DB) FetchBlockByNum(num uint64) ([]common.SignedBlock, error) {
	return nil, nil
}

// PushBlock adds a block. If any of the forkchain has more than
// maxSize blocks, purge will be triggered.
func (db *DB) PushBlock(b common.SignedBlock) common.SignedBlock {
	id := b.Id()
	num := id.BlockNum()
	if len(db.list) == 0 {
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
	return nil
}

// Pop pops the head block of the longest chain
func (db *DB) Pop() {

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
