package test

import (
	"github.com/coschain/contentos-go/common"
	"time"
)

const bufLen = 40

type commit struct {
	blk common.ISignedBlock
	t time.Time
}
type CommitInfo struct {
	recentlyCommitted []*commit
	start int
	displayRange int
}

func NewCommitInfo() *CommitInfo {
	return &CommitInfo{
		recentlyCommitted: make([]*commit, 0, bufLen),
		displayRange: 20,
	}
}

func (ci *CommitInfo) Commit(b common.ISignedBlock) {
	if len(ci.recentlyCommitted) == bufLen {
		newC := make([]*commit, 0, bufLen)
		for i:= bufLen-ci.displayRange; i<bufLen; i++ {
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
	info := make([]float64, 0, ci.displayRange)
	for i := ci.start; i<len(ci.recentlyCommitted); i++ {
		elapsed := ci.recentlyCommitted[i].t.Sub(time.Unix(int64(ci.recentlyCommitted[i].blk.Timestamp()), 0))
		info = append(info, float64(elapsed/time.Millisecond))
	}
	if len(info) < 2 {
		return []float64{2000.0, 2000.0}
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