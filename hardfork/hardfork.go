package hardfork

import (
	"sort"
)

var HF *HardFork

type Action func(...interface{})
type ActionName string

type ActionSet struct {
	hardForkHeight uint64
	actions        map[ActionName]Action
	replay         bool
}

func NewActionSet(height uint64, replay bool) *ActionSet {
	return &ActionSet{
		hardForkHeight: height,
		replay:         replay,
		actions:        make(map[ActionName]Action),
	}
}

func (as *ActionSet) AddAction(name ActionName, a Action) *ActionSet {
	as.actions[name] = a
	return as
}

type HardFork struct {
	hardForks      map[uint64]*ActionSet
	currentActions *ActionSet

	currentIdx  uint64
	checkpoints []uint64
}

func NewHardFork() *HardFork {
	ret := &HardFork{
		hardForks: make(map[uint64]*ActionSet),
		currentActions: NewActionSet(0, false),
		currentIdx: 0,
		checkpoints: make([]uint64, 0),
	}
	return ret
}

func (hf *HardFork) Apply(height uint64) {
	if len(hf.checkpoints) != len(hf.hardForks) {
		hf.checkpoints = append(hf.checkpoints, 0)
		for k := range hf.hardForks {
			hf.checkpoints = append(hf.checkpoints, k)
		}
		sort.Slice(hf.checkpoints, func(i, j int) bool {
			return hf.checkpoints[i] < hf.checkpoints[j]
		})
	}

	if height <= hf.checkpoints[hf.currentIdx] || hf.currentIdx == uint64(len(hf.hardForks)) {
		return
	}
	for {
		if hf.checkpoints[hf.currentIdx+1] > height {
			return
		}
		for k, v := range hf.hardForks[hf.checkpoints[hf.currentIdx+1]].actions {
			if k == NewOP {
				v()
				continue
			} else {
				hf.currentActions.actions[k] = v
			}
		}
		hf.currentIdx++
	}
}

func (hf *HardFork) RegisterAction(height uint64, name ActionName, action Action) {
	as, exist := hf.hardForks[height]
	if !exist {
		as = NewActionSet(height, false)
		hf.hardForks[height] = as
	}
	as.actions[name] = action
}

func (hf *HardFork) CurrentAction(name ActionName) Action {
	if a, exist := hf.currentActions.actions[name]; !exist {
		return func(...interface{}) {}
	} else {
		return a
	}
}

func (hf *HardFork) String() string {
	return ""
}


func init() {
	HF = NewHardFork()
}
