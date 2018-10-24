package forkdb

import (
	"contentos-go/common"
)

const maxSize = 1024

// DB ...
type DB struct {
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
// maxSize blocks, Purge will be triggered.
func (db *DB) PushBlock(block common.SignedBlock) {

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
