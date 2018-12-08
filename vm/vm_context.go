package vm

import (
	"github.com/coschain/contentos-go/prototype"
)

type ContractName string

type Context struct {
	From     *prototype.AccountName
	Owner    *prototype.AccountName
	Contract string
	Method   string
	Params   []string
	Amount   *prototype.Coin
	Gas      *prototype.Coin
	Construct bool
	Code	 []byte
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


func NewContextFromApplyOp(op *prototype.ContractApplyOperation, code []byte) *Context  {
	return &Context{ From:op.Owner,
		Owner:op.Owner,
		Contract:op.Contract,
		Method:op.Method,
		Params:op.Params,
		Amount:op.Amount,
		Gas:op.Gas,
		Construct:false,
		Code:code,
	}
}

func (c *Context) Run() error {
	return nil
}