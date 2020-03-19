package consensus

import (
	"container/list"
	"fmt"
	"sync"

	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/gobft/message"
)

type heightCache struct {
	height uint64
	m      map[common.BlockID]*message.Commit // prev-->Commit
}

type CommitCache struct {
	waterMark  uint64
	startH     uint64
	heightList *list.List
	m          map[uint64]*list.Element
	commitCnt  uint64
	sync.RWMutex
}

func NewCommitCache() *CommitCache {
	return &CommitCache{
		waterMark:  constants.MaxUncommittedBlockNum,
		startH:     0,
		commitCnt:  0,
		heightList: list.New(),
		m:          make(map[uint64]*list.Element),
	}
}

func (c *CommitCache) Add(commit *message.Commit) bool {
	c.Lock()
	defer c.Unlock()

	id := ConvertToBlockID(commit.Prev)
	height := id.BlockNum()
	if height > c.startH+c.waterMark || height < c.startH {
		return false
	}
	e, ok := c.m[height]
	if !ok {
		var i *list.Element
		for i = c.heightList.Front(); i != nil; i = i.Next() {
			if i.Value.(*heightCache).height > height {
				break
			}
		}
		hc := &heightCache{
			height: height,
			m:      make(map[common.BlockID]*message.Commit),
		}
		if i != nil {
			e = c.heightList.InsertBefore(hc, i)
		} else {
			e = c.heightList.PushBack(hc)
		}
		c.m[height] = e
	}
	hc := e.Value.(*heightCache)
	if _, exist := hc.m[id]; exist {
		return false
	}
	hc.m[id] = commit
	return true
}

func (c *CommitCache) String() string {
	str := ""
	for i := c.heightList.Front(); i != nil; i = i.Next() {
		str += fmt.Sprintf("\n--->height = %d, %v", i.Value.(*heightCache).height, i.Value.(*heightCache).m)
	}
	return str
}

func (c *CommitCache) Remove(id common.BlockID) {
	c.Lock()
	defer c.Unlock()

	height := id.BlockNum()
	if e, ok := c.m[height]; ok {
		hc := e.Value.(*heightCache)
		delete(hc.m, id)
	}
}

func (c *CommitCache) Commit(id common.BlockID) {
	c.Lock()
	defer c.Unlock()

	height := id.BlockNum()
	var i *list.Element
	for i = c.heightList.Front(); i != nil; i = i.Next() {
		hc := i.Value.(*heightCache)
		if hc.height < height {
			//delete(c.m, height)
			c.heightList.Remove(i)
			delete(c.m, height)
		} else {
			break
		}
	}
	c.commitCnt++
	c.startH = height
	if c.commitCnt%256 == 0 {
		// TODO: release map memory
	}
}

func (c *CommitCache) CommitOne() {

}

func (c *CommitCache) Has(id common.BlockID) bool {
	//c.RLock()
	//defer c.RUnlock()
	//height := id.BlockNum()
	//if height > c.startH+c.waterMark || height < c.startH {
	//	return false
	//}
	//e, ok := c.m[height]
	//if !ok {
	//	return false
	//}
	//hc := e.Value.(*heightCache)
	return true
}

func (c *CommitCache) Get(id common.BlockID) *message.Commit {
	c.RLock()
	defer c.RUnlock()

	height := id.BlockNum()
	if height > c.startH+c.waterMark || height < c.startH {
		return nil
	}
	e, ok := c.m[height]
	if !ok {
		return nil
	}
	hc := e.Value.(*heightCache)
	return hc.m[id]
}

func (c *CommitCache) HasDangling() bool {
	c.RLock()
	defer c.RUnlock()

	return c.heightList.Len() > 1
}

func (c *CommitCache) GetDanglingHeight() uint64 {
	c.RLock()
	defer c.RUnlock()

	f := c.heightList.Front().Next()
	return f.Value.(*heightCache).height
}
