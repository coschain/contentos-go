package vmcontext

import (
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/vm/contract/abi"
	"github.com/coschain/contentos-go/vm/contract/table"
	"github.com/coschain/contentos-go/vm/injector"
)

type ContractName string

type Context struct {
	Caller    *prototype.AccountName
	CallingContractOwner *prototype.AccountName
	CallingContractName string
	CallingContractMethod string
	Owner     *prototype.AccountName
	Contract  string
	Method    string
	Params    string
	ParamsData []byte
	Amount    *prototype.Coin
	Gas       uint64
	Construct bool
	Code      []byte
	CodeHash  *prototype.Sha256
	Abi       string
	AbiInterface abi.IContractABI
	Tables    *table.ContractTables
	Injector  vminjector.Injector
}

func NewContextFromDeployOp(op *prototype.ContractDeployOperation, code []byte, abiString string, injector vminjector.Injector) *Context {
	return &Context{
		Owner:    op.Owner,
		Contract: op.Contract,
		Code:     code,
		Abi:      abiString,
		Injector: injector,
	}
}

func NewContextFromApplyOp(op *prototype.ContractApplyOperation, params []byte, code []byte, codeHash *prototype.Sha256, abi abi.IContractABI, tables *table.ContractTables, injector vminjector.Injector) *Context {
	return &Context{
		Caller:    op.Caller,
		CallingContractOwner: nil,
		CallingContractName: "",
		CallingContractMethod: "",
		Owner:     op.Owner,
		Contract:  op.Contract,
		Method:    op.Method,
		Params:    op.Params,
		ParamsData: params,
		Amount:    op.Amount,
		Construct: false,
		Code:      code,
		CodeHash:  codeHash,
		AbiInterface: abi,
		Tables: tables,
		Injector:  injector,
	}
}

func NewContextFromInternalApplyOp(op *prototype.InternalContractApplyOperation, code []byte, codeHash *prototype.Sha256, abi abi.IContractABI, tables *table.ContractTables, injector vminjector.Injector) *Context {
	return &Context{
		Caller:    op.FromCaller,
		CallingContractOwner: op.FromOwner,
		CallingContractName: op.FromContract,
		CallingContractMethod: op.FromMethod,
		Owner:     op.ToOwner,
		Contract:  op.ToContract,
		Method:    op.ToMethod,
		Params:    "",
		ParamsData: op.Params,
		Amount:    op.Amount,
		Construct: false,
		Code:      code,
		CodeHash:  codeHash,
		AbiInterface: abi,
		Tables: tables,
		Injector:  injector,
	}
}
