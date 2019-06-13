package consensus

import (
	"container/list"
	"github.com/coschain/gobft/message"
)

type Dynasty struct {
	Seq uint64
	validators    []*publicValidator
	priv          *privateValidator
}

func NewDynasty(n uint64, vs []*publicValidator, p *privateValidator) *Dynasty {
	return &Dynasty{
		Seq: n,
		validators: vs,
		priv: p,
	}
}

func (d *Dynasty) GetValidatorNum() int {
	return len(d.validators)
}

func (d *Dynasty) GetValidatorByPubKey(k message.PubKey) *publicValidator {
	for i := range d.validators {
		if d.validators[i].bftPubKey == k {
			return d.validators[i]
		}
	}
	return nil
}

type Dynasties struct {
	dynasties *list.List
}

func NewDynasties() *Dynasties {
	return &Dynasties{
		dynasties: list.New(),
	}
}

func (ds *Dynasties) Empty() bool {
	return ds.dynasties.Len() == 0
}

func (ds *Dynasties) Len() int {
	return ds.dynasties.Len()
}

func (ds *Dynasties) Front() *Dynasty {
	return ds.dynasties.Front().Value.(*Dynasty)
}

func (ds *Dynasties) Back() *Dynasty {
	return ds.dynasties.Back().Value.(*Dynasty)
}

func (ds *Dynasties) PushFront(d *Dynasty) {
	ds.dynasties.PushFront(d)
}

func (ds *Dynasties) PushBack(d *Dynasty) {
	ds.dynasties.PushBack(d)
}

func (ds *Dynasties) PopFront() {
	ds.dynasties.Remove(ds.dynasties.Front())
}

func (ds *Dynasties) PopBack() {
	ds.dynasties.Remove(ds.dynasties.Back())
}

func (ds *Dynasties) PopBefore(seq uint64) *Dynasty {
	var res *Dynasty = nil
	for front := ds.dynasties.Front(); front != nil; {
		d := front.Value.(*Dynasty)
		if d.Seq > seq {
			break
		}
		res = d
		ds.dynasties.Remove(front)
		front = ds.dynasties.Front()
	}
	return res
}

func (ds *Dynasties) Purge(seq uint64) {
	if last := ds.PopBefore(seq); last != nil {
		ds.PushFront(last)
	}
	//fmt.Printf("****first dynasty at %d", ds.Front().Seq)
	//for i := range ds.Front().validators {
	//	fmt.Print(" ", ds.Front().validators[i].accountName)
	//}
}

func (ds *Dynasties) PopAfter(seq uint64) *Dynasty {
	var res *Dynasty = nil
	for back := ds.dynasties.Back(); back != nil; {
		d := back.Value.(*Dynasty)
		if d.Seq < seq {
			break
		}
		res = d
		ds.dynasties.Remove(back)
		back = ds.dynasties.Back()
	}
	return res
}