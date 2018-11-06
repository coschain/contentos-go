package app

import "github.com/coschain/contentos-go/common/prototype"

type BaseEvaluator interface {
	Apply(op *prototype.Operation)
}


type AccountCreateEvaluator struct{

}

type TransferEvaluator struct{

}

func (a *AccountCreateEvaluator) Apply(op *prototype.Operation) {
	// write DB
	 o,ok := op.Op.(*prototype.Operation_Op1)
	 if !ok {
		panic("type cast failed")
	}
	accountCreateOp := o.Op1
	accountCreateOp.XXX_sizecache = 1
}

func (a *TransferEvaluator) Apply(op *prototype.Operation) {
	// write DB
	o,ok := op.Op.(*prototype.Operation_Op2)
	if !ok {
		panic("type cast failed")
	}
	transferOp := o.Op2
	transferOp.XXX_sizecache = 1
}
