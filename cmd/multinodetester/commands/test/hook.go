package test

import (
	"github.com/coschain/contentos-go/common"
	"sync"
	"time"
)

const displayRange = 20

type commit struct {
	blk common.ISignedBlock
	t time.Time
}
type CommitInfo struct {
	recentlyCommitted []*commit
	start int
	displayRange int

	sync.RWMutex
}

func NewCommitInfo() *CommitInfo {
	return &CommitInfo{
		recentlyCommitted: make([]*commit, 0, displayRange*2),
		displayRange: displayRange,
	}
}

func (ci *CommitInfo) Commit(b common.ISignedBlock) {
	ci.Lock()
	defer ci.Unlock()

	if len(ci.recentlyCommitted) == displayRange*2 {
		newC := make([]*commit, 0, displayRange*2)
		for i:= displayRange*2-ci.displayRange; i<displayRange*2; i++ {
			newC = append(newC, ci.recentlyCommitted[i])
		}
		ci.recentlyCommitted = newC
		ci.start = 0
	}
	ci.recentlyCommitted = append(ci.recentlyCommitted, &commit{
		blk: b,
		t: time.Now(),
	})
	l := len(ci.recentlyCommitted)
	if l > ci.displayRange {
		ci.start = l-ci.displayRange
	}
}

func (ci *CommitInfo) MarginStepInfo() []float64 {
	ci.RLock()
	defer ci.RUnlock()

	info := make([]float64, 0, ci.displayRange)
	for i := ci.start+1; i<len(ci.recentlyCommitted); i++ {
		info = append(info, float64(ci.recentlyCommitted[i].blk.Id().BlockNum()-ci.recentlyCommitted[i-1].blk.Id().BlockNum()))
	}
	if len(info) < 2 {
		return []float64{2.0, 2.0}
	}
	return info
}

func (ci *CommitInfo) ConfirmationTimeInfo() []float64 {
	ci.RLock()
	defer ci.RUnlock()
	
	info := make([]float64, ci.displayRange)
	for i := ci.start; i<len(ci.recentlyCommitted); i++ {
		elapsed := ci.recentlyCommitted[i].t.Sub(time.Unix(int64(ci.recentlyCommitted[i].blk.Timestamp()), 0))
		info = append(info, float64(elapsed/time.Millisecond))
	}

	index := displayRange-1
	for i:=len(ci.recentlyCommitted)-1; i>0; i-- {
		committedBlock := ci.recentlyCommitted[i].blk
		prevCommittedBlock := ci.recentlyCommitted[i-1].blk
		gap := int(committedBlock.Id().BlockNum()-prevCommittedBlock.Id().BlockNum())
		startIndex := index-gap
		elapsed := float64(ci.recentlyCommitted[i].t.Sub(time.Unix(int64(ci.recentlyCommitted[i].blk.Timestamp()), 0))/time.Millisecond)
		for j:=index; j>startIndex; j-- {
			info[j] = elapsed
			elapsed += 1000.0
			if j==0 {
				return info
			}
		}
		index = startIndex
	}
	
	return info
}

func (ci *CommitInfo) commitHook(args ...interface{}) {
	ci.Commit(args[0].(common.ISignedBlock))
}

func (ci *CommitInfo) generateBlockHook(args ...interface{}) {

}

func (ci *CommitInfo) branches(args ...interface{}) {

}

func (ci *CommitInfo) switchFork(args ...interface{}) {

}