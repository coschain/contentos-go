package consensus

import (
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/gobft/message"
)

type CommitCache struct {

}

func NewCommitCache() *CommitCache {
	return &CommitCache{}
}

func (c *CommitCache) Add(commit *message.Commit) bool {
	return true
}

func (c *CommitCache) Remove(id common.BlockID) {

}

func (c *CommitCache) Commit(id common.BlockID) {

}

func (c *CommitCache) CommitOne() {

}

func (c *CommitCache) Has(id common.BlockID) bool {
	return true
}

func (c *CommitCache) Get(id common.BlockID) *message.Commit {
	return nil
}

func (c *CommitCache) HasDangling() bool {
	return false
}

func (c *CommitCache) GetDanglingHeight() uint64 {
	return 0
}