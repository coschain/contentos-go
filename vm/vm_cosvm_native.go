package vm

import (
	"crypto/sha256"
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/vm/contract/abi"
	table2 "github.com/coschain/contentos-go/vm/contract/table"
	"github.com/hashicorp/golang-lru"
	"strings"
)

//type ICosVMNative interface {
//	Sha256(in []byte) [32]byte
//	CurrentBlockNumber() uint64
//	CurrentTimestamp() uint64
//	CurrentWitness() string
//	PrintString(str string)
//	PrintUint32(value uint32)
//	PrintUint64(value uint64)
//	PrintBool(value bool)
//	RequiredAuth(name string)
//	GetBalanceByName(name string) uint64
//	GetContractBalance(contract string, name string) uint64
//	SaveToStorage([]byte, []byte)
//	ReadFromStorage([]byte) (value []byte)
//
//}

type CosVMNative struct {
	cosVM *CosVM
	tablesCache *lru.Cache
}

const tablesCacheMaxSize = 64

func NewCosVMNative(vm *CosVM) *CosVMNative {
	tabCache, _ := lru.New(tablesCacheMaxSize)
	return &CosVMNative{
		cosVM: vm,
		tablesCache: tabCache,
	}
}

func (w *CosVMNative) Sha256(in []byte) [32]byte {
	return sha256.Sum256(in)
}

func (w *CosVMNative) CurrentBlockNumber() uint64 {
	return w.cosVM.props.HeadBlockNumber
}

func (w *CosVMNative) CurrentTimestamp() uint64 {
	return uint64(w.cosVM.props.Time.UtcSeconds)
}

func (w *CosVMNative) CurrentWitness() string {
	return w.cosVM.props.CurrentWitness.Value
}

func (w *CosVMNative) GetBlockProducers() string {
	return strings.Join(w.cosVM.ctx.Injector.GetBlockProducers(), " ")
}

func (w *CosVMNative) PrintString(str string) {
	w.cosVM.ctx.Injector.Log(str)
}

func (w *CosVMNative) PrintInt64(value int64) {
	w.cosVM.ctx.Injector.Log(fmt.Sprintf("%d", value))
}

func (w *CosVMNative) PrintUint64(value uint64) {
	w.cosVM.ctx.Injector.Log(fmt.Sprintf("%d", value))
}

func (w *CosVMNative) RequiredAuth(name string) {
	err := w.cosVM.ctx.Injector.RequireAuth(name)
	w.CosAssert(err == nil, "require auth error")
}

func (w *CosVMNative) GetUserBalance(name string) uint64 {
	acc := table.NewSoAccountWrap(w.cosVM.db, &prototype.AccountName{Value: name})
	return acc.GetBalance().Value
}

func (w *CosVMNative) GetContractBalance(contract string, name string) uint64 {
	ctct := table.NewSoContractWrap(w.cosVM.db, &prototype.ContractId{Owner: &prototype.AccountName{Value: name}, Cname: contract})
	value := ctct.GetBalance().Value
	return value
}

func (w *CosVMNative) CosAssert(condition bool, msg string) {
	if !condition {
		panic(msg)
	}
}

func (w *CosVMNative) CosAbort() {
	w.CosAssert(false, "abort() called.")
}

func (w *CosVMNative) ReadContractOpParams() string {
	return string(w.cosVM.ctx.ParamsData)
}

func (w *CosVMNative) ReadContractOwner() string {
	return w.cosVM.ctx.Owner.Value
}

func (w *CosVMNative) ReadContractCaller() string {
	return w.cosVM.ctx.Caller.Value
}

func (w *CosVMNative) ReadContractName() string {
	return w.cosVM.ctx.Contract
}

func (w *CosVMNative) ReadContractMethod() string {
	return w.cosVM.ctx.Method
}

func (w *CosVMNative) ReadContractSenderValue() uint64 {
	return w.cosVM.ctx.Amount.Value
}

func (w *CosVMNative) ContractCalledByUser() bool {
	return w.cosVM.ctx.CallingContractOwner == nil
}

func (w *CosVMNative) ReadCallingContractOwner() string {
	if !w.ContractCalledByUser() {
		return w.cosVM.ctx.CallingContractOwner.Value
	}
	return ""
}

func (w *CosVMNative) ReadCallingContractName() string {
	if !w.ContractCalledByUser() {
		return w.cosVM.ctx.CallingContractName
	}
	return ""
}

func (w *CosVMNative) ContractTransferToUser(to string, amount uint64) {
	w.cosVM.ctx.Injector.TransferFromContractToUser(w.cosVM.ctx.Contract, w.cosVM.ctx.Owner.Value, to, amount)
}

func (w *CosVMNative) ContractTransferToContract(owner, contract string, amount uint64) {
	w.cosVM.ctx.Injector.TransferFromContractToContract(w.cosVM.ctx.Contract, w.cosVM.ctx.Owner.Value, contract, owner, amount)
}

func (w *CosVMNative) ContractCall(owner, contract, method string, paramsData []byte, coins uint64) {
	spentGas := w.cosVM.SpentGas()
	w.CosAssert(w.cosVM.ctx.Gas > spentGas, "ContractCall(): out of gas.")
	w.cosVM.ctx.Injector.ContractCall(
		w.ReadContractCaller(),
		w.ReadContractOwner(),
		w.ReadContractName(),
		w.ReadContractMethod(),
		owner, contract, method, paramsData, coins, w.cosVM.ctx.Gas - spentGas)
}

func (w *CosVMNative) TableGetRecord(tableName string, primary []byte) []byte {
	tables := w.cosVM.ctx.Tables
	w.CosAssert(tables != nil, "TableGetRecord(): context tables not ready.")
	data, err := tables.Table(tableName).GetRecord(primary)
	//w.CosAssert(err == nil, fmt.Sprintf("TableGetRecord(): table.GetRecord() failed. %v", err))
	if err != nil {
		return nil
	}
	return data
}

func (w *CosVMNative) TableNewRecord(tableName string, record []byte) {
	tables := w.cosVM.ctx.Tables
	w.CosAssert(tables != nil, "TableNewRecord(): context tables not ready.")
	err := tables.Table(tableName).NewRecord(record)
	w.CosAssert(err == nil, fmt.Sprintf("TableNewRecord(): table.NewRecord() failed. %v", err))
}

func (w *CosVMNative) TableUpdateRecord(tableName string, primary []byte, record []byte) {
	tables := w.cosVM.ctx.Tables
	w.CosAssert(tables != nil, "TableUpdateRecord(): context tables not ready.")
	err := tables.Table(tableName).UpdateRecord(primary, record)
	w.CosAssert(err == nil, fmt.Sprintf("TableUpdateRecord(): table.UpdateRecord() failed. %v", err))
}

func (w *CosVMNative) TableDeleteRecord(tableName string, primary []byte) {
	tables := w.cosVM.ctx.Tables
	w.CosAssert(tables != nil, "TableDeleteRecord(): context tables not ready.")
	err := tables.Table(tableName).DeleteRecord(primary)
	w.CosAssert(err == nil, fmt.Sprintf("TableDeleteRecord(): table.DeleteRecord() failed. %v", err))
}

func (w *CosVMNative) TableGetRecordEx(ownerName, contractName, tableName string, primary []byte) []byte {
	var tables *table2.ContractTables

	contractKey := contractName + "@" + ownerName
	cached, ok := w.tablesCache.Get(contractKey)
	if ok {
		tables = cached.(*table2.ContractTables)
	} else {
		jsonAbi := w.cosVM.ctx.Injector.ContractABI(ownerName, contractName)
		w.CosAssert(len(jsonAbi) > 0, fmt.Sprintf("TableGetRecordEx(): no ABI for contract '%s' of account '%s'", contractName, ownerName))
		abiInterface, err := abi.UnmarshalABI([]byte(jsonAbi))
		if err != nil {
			w.CosAssert(false, fmt.Sprintf("TableGetRecordEx(): invalid ABI of contract '%s' of account '%s': %s", contractName, ownerName, err.Error()))
		}
		tables = table2.NewContractTables(ownerName, contractName, abiInterface, w.cosVM.db)
		w.CosAssert(tables != nil, "TableGetRecordEx(): tables creation failed")

		w.tablesCache.Add(contractKey, tables)
	}

	data, err := tables.Table(tableName).GetRecord(primary)
	if err != nil {
		return nil
	}
	return data
}

func (w *CosVMNative) SetReputationAdmin(name string) {
	singleId := int32(constants.SingletonId)
	props := *w.cosVM.props
	props.ReputationAdmin = prototype.NewAccountName(name)
	w.CosAssert(table.NewSoGlobalWrap(w.cosVM.db, &singleId).MdProps(&props), "failed to set reputation admin")
	w.cosVM.props.ReputationAdmin = props.ReputationAdmin
}

func (w *CosVMNative) GetReputationAdmin() (name string) {
	if w.cosVM.props.ReputationAdmin != nil {
		name = w.cosVM.props.ReputationAdmin.Value
	}
	return
}

func (w *CosVMNative) SetUserReputation(name string, value uint32, memo string) {
	account := table.NewSoAccountWrap(w.cosVM.db, prototype.NewAccountName(name))
	w.CosAssert(account.MdReputation(value), fmt.Sprintf("failed to modify reputation of %s", name))
	w.CosAssert(account.MdReputationMemo(memo), fmt.Sprintf("failed to modify reputation memo of %s", name))
}
