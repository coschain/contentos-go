package consensus

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

type Dynasties struct {

}

func NewDynasties() *Dynasties {
	// TODO: load dynasties snapshot
	return &Dynasties{}
}

func (ds *Dynasties) Empty() bool {
	return true
}

func (ds *Dynasties) Front() *Dynasty {
	return nil
}

func (ds *Dynasties) Back() *Dynasty {
	return nil
}

func (ds *Dynasties) PushFront(d *Dynasty) {

}

func (ds *Dynasties) PushBack(d *Dynasty) {

}

func (ds *Dynasties) PopFront() *Dynasty {
	return nil
}

func (ds *Dynasties) PopBack() *Dynasty {
	return nil
}