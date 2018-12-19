package vmcontext

import (
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/vm/injector"
)

type ContractName string

type Context struct {
	Caller    *prototype.AccountName
	Owner     *prototype.AccountName
	Contract  string
	Method    string
	Params    string
	Amount    *prototype.Coin
	Gas       *prototype.Coin
	Construct bool
	Code      []byte
	Injector  vminjector.Injector
}

func NewContextFromApplyOp(op *prototype.ContractApplyOperation, code []byte, injector vminjector.Injector) *Context {
	return &Context{
		Caller:    op.Caller,
		Owner:     op.Owner,
		Contract:  op.Contract,
		Method:    op.Method,
		Params:    op.Params,
		Amount:    op.Amount,
		Gas:       op.Gas,
		Construct: false,
		Code:      code,
		Injector:  injector,
	}
}

func (c *Context) Run() error {
	return nil
}
