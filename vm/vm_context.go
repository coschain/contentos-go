package vm

import (
	"github.com/coschain/contentos-go/app"
	"github.com/coschain/contentos-go/prototype"
)

type ContractName string

type Context struct {
	Caller    *prototype.AccountName
	Owner     *prototype.AccountName
	Contract  string
	Method    string
	Params    []string
	Amount    *prototype.Coin
	Gas       *prototype.Coin
	Construct bool
	Code      []byte
	Injector  *app.Injector
}

//
//func NewContextFromDeployOp(op *prototype.ContractDeployOperation) *Context  {
//	return &Context{ From:op.Owner,
//				Owner:op.Owner,
//				Contract:op.Contract,
//				Method:constants.Contract_Construct,
//				Params:[]string{},
//				Amount:nil,
//				Gas:nil,
//				Construct:true,
//				Code:op.Code,
//			}
//}

func NewContextFromApplyOp(op *prototype.ContractApplyOperation, code []byte, injector *app.Injector) *Context {
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
