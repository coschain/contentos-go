package vmcontext

import (
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/vm/contract/abi"
	"github.com/coschain/contentos-go/vm/injector"
)

type ContractName string

type Context struct {
	Caller    *prototype.AccountName
	Owner     *prototype.AccountName
	Contract  string
	Method    string
	Params    string
	ParamsData []byte
	Amount    *prototype.Coin
	Gas       *prototype.Coin
	Construct bool
	Code      []byte
	Abi       string
	AbiInterface abi.IContractABI
	Injector  vminjector.Injector
}

func NewContextFromDeployOp(op *prototype.ContractDeployOperation, injector vminjector.Injector) *Context {
	return &Context{
		Owner:    op.Owner,
		Contract: op.Contract,
		Code:     op.Code,
		Abi:      op.Abi,
		Injector: injector,
	}
}

func NewContextFromApplyOp(op *prototype.ContractApplyOperation, params []byte, code []byte, abi abi.IContractABI, injector vminjector.Injector) *Context {
	return &Context{
		Caller:    op.Caller,
		Owner:     op.Owner,
		Contract:  op.Contract,
		Method:    op.Method,
		Params:    op.Params,
		ParamsData: params,
		Amount:    op.Amount,
		Gas:       op.Gas,
		Construct: false,
		Code:      code,
		AbiInterface: abi,
		Injector:  injector,
	}
}
