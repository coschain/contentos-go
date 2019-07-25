package test

import "github.com/coschain/contentos-go/common"

const bufLen = 40

type CommitInfo struct {
	recentlyCommitted []common.ISignedBlock
	start int
	displayRange int
}

func NewCommitInfo() *CommitInfo {
	return &CommitInfo{
		recentlyCommitted: make([]common.ISignedBlock, 0, bufLen),
		displayRange: 20,
	}
}

func (ci *CommitInfo) Commit(b common.ISignedBlock) {
	if len(ci.recentlyCommitted) == bufLen {
		newC := make([]common.ISignedBlock, 0, bufLen)
		for i:= bufLen-ci.displayRange; i<bufLen; i++ {
			newC = append(newC, ci.recentlyCommitted[i])
		}
		ci.recentlyCommitted = newC
		ci.start = 0
	}
	ci.recentlyCommitted = append(ci.recentlyCommitted, b)
	l := len(ci.recentlyCommitted)
	if l > ci.displayRange {
		ci.start = l-ci.displayRange
	}
}

func (ci *CommitInfo) MarginStepInfo() []float64 {
	info := make([]float64, 0, ci.displayRange)
	for i := ci.start+1; i<len(ci.recentlyCommitted); i++ {
		info = append(info, float64(ci.recentlyCommitted[i].Id().BlockNum()-ci.recentlyCommitted[i-1].Id().BlockNum()))
	}
	if len(info) < 2 {
		return []float64{2.0, 2.0}
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