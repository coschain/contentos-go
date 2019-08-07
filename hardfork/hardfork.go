package hardfork

import "github.com/coschain/contentos-go/prototype"

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
	hardForks      []*ActionSet
	currentActions *ActionSet
	currentIdx  uint64
}

func NewHardFork() *HardFork {
	ret := &HardFork{
		hardForks: make([]*ActionSet, 0),
		currentActions: NewActionSet(0, false),
		currentIdx: 0,
	}
	ret.hardForks = append(ret.hardForks, NewActionSet(0, false))
	return ret
}

func (hf *HardFork) Apply(height uint64) {
	if height <= hf.hardForks[hf.currentIdx].hardForkHeight || hf.currentIdx == uint64(len(hf.hardForks)) {
		return
	}
	for {
		if hf.hardForks[hf.currentIdx+1].hardForkHeight > height {
			return
		}
		for k, v := range hf.hardForks[hf.currentIdx+1].actions {
			if k == "new_op" {
				v()
				continue
			} else {
				hf.currentActions.actions[k] = v
			}
		}
	}
}

func (hf *HardFork) CurrentAction(name ActionName) Action {
	if a, exist := hf.currentActions.actions[name]; !exist {
		return func(...interface{}) {}
	} else {
		return a
	}
}

func (hf *HardFork) new(height uint64, replay bool) *ActionSet {
	as := NewActionSet(height, replay)
	hf.hardForks = append(hf.hardForks, as)
	return as
}

func (hf *HardFork) init() {
	hf.new(10, false).AddAction("hello", hello_10)

	hf.new(20, false).AddAction("hello", hello_20).
									AddAction("byebye", byebye_20)

	hf.new(30, false).AddAction("new_op", func(args ...interface{}){
		prototype.RegisterNewOperation("report", (*prototype.Operation_Op15)(nil), (*prototype.ReportOperation)(nil))
	})
}
