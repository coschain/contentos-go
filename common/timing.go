package common

import (
	"fmt"
	"strings"
	"time"
)

type timingNode struct {
	started, finished bool
	marks []time.Time
	children []*timingNode
	parent *timingNode
}

func newTimingNode() *timingNode {
	return &timingNode{
		marks: make([]time.Time, 0, 100),
		children: make([]*timingNode, 0, 100),
	}
}

func (n *timingNode) begin() *timingNode {
	ret := n;
	if !n.finished {
		if !n.started {
			n.started = true
			n.mark()
		} else {
			child := newTimingNode()
			child.parent = n
			n.children[len(n.children) - 1] = child
			ret = child.begin()
		}
	}
	return ret;
}

func (n *timingNode) end() *timingNode {
	ret := n;
	if !n.finished && n.started {
		n.mark()
		ret = n.parent
		n.finished = true
	}
	return ret;
}

func (n *timingNode) mark() {
	if !n.finished {
		n.children = append(n.children, nil)
		n.marks = append(n.marks, time.Now())
	}
}

func (n *timingNode) duration() time.Duration {
	if n.finished && len(n.marks) >= 2 {
		return n.marks[len(n.marks) - 1].Sub(n.marks[0])
	}
	return time.Duration(0)
}

func (n *timingNode) setPartial(partial time.Duration) {
	if !n.finished && n.started {
		child := &timingNode{
			started: true,
			finished: true,
			marks: []time.Time{ n.marks[0], n.marks[0].Add(partial) },
			children: []*timingNode{ nil, nil },
			parent: n,
		}
		n.children[len(n.children) - 1] = child
	}
}

func (n *timingNode) String() (str string) {
	if count := len(n.marks); count >= 2 {
		parts := make([]string, count - 1)
		for i := 0; i < count - 1; i++ {
			s := fmt.Sprintf("%v", n.marks[i + 1].Sub(n.marks[i]))
			if n.children[i] != nil {
				s = fmt.Sprintf("%s(%s)", s, n.children[i].String())
			}
			parts[i] = s
		}
		str = strings.Join(parts, "|")
	}
	return
}

type Timing struct {
	root, current *timingNode
}

func NewTiming() *Timing {
	root := newTimingNode()
	root.parent = root
	return &Timing{ root:root, current:root }
}

func (t *Timing) Begin() {
	t.current = t.current.begin()
}

func (t *Timing) End() {
	t.current = t.current.end()
}

func (t *Timing) Mark() {
	t.current.mark()
}

func (t *Timing) SetPartial(partial time.Duration) {
	t.current.setPartial(partial)
}

func (t *Timing) String() string {
	return fmt.Sprintf("%v(%s)", t.root.duration(), t.root.String())
}
