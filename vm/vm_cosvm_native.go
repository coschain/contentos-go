package vm

import (
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/iservices/itype"
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
//	CurrentBlockProducer() string
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

//func (w *CosVMNative) Sha256(in []byte) [32]byte {
//	return sha256.Sum256(in)
//}

func (w *CosVMNative) CurrentBlockNumber() uint64 {
	return w.cosVM.props.HeadBlockNumber
}

func (w *CosVMNative) CurrentTimestamp() uint64 {
	return uint64(w.cosVM.props.Time.UtcSeconds)
}

func (w *CosVMNative) CurrentBlockProducer() string {
	return w.cosVM.props.CurrentBlockProducer.Value
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

func (w *CosVMNative) UserExist(name string) bool {
	acc := table.NewSoAccountWrap(w.cosVM.db, &prototype.AccountName{Value: name})
	return acc.CheckExist()
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
	spentGas := w.cosVM.Vm.CostGas
	w.CosAssert(w.cosVM.ctx.Gas > spentGas, "ContractCall(): out of gas.")
	w.cosVM.ctx.Injector.ContractCall(
		w.ReadContractCaller(),
		w.ReadContractOwner(),
		w.ReadContractName(),
		w.ReadContractMethod(),
		owner, contract, method, paramsData, coins, w.cosVM.ctx.Gas - spentGas,w.cosVM.Vm)
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
	currentTable := tables.Table(tableName)
	currentTable.SetRecordCallback(func(what string, key, before, after interface{}) {
		w.addTableRecordChange(tableName, what, key, before, after)
	})
	err := currentTable.NewRecord(record)
	currentTable.SetRecordCallback(nil)
	w.CosAssert(err == nil, fmt.Sprintf("TableNewRecord(): table.NewRecord() failed. %v", err))

	decodeRecord, _ := currentTable.DecodeRecordToJson(record)
	var contractData itype.ContractData
	contractData.Contract = w.ReadContractName()
	contractData.ContractOwner = w.ReadContractOwner()
	contractData.Record = decodeRecord
	w.cosVM.ctx.TrxObserver.AddOpState(iservices.Insert, "contract", tableName, contractData)
}

func (w *CosVMNative) TableUpdateRecord(tableName string, primary []byte, record []byte) {
	tables := w.cosVM.ctx.Tables
	w.CosAssert(tables != nil, "TableUpdateRecord(): context tables not ready.")
	currentTable := tables.Table(tableName)
	currentTable.SetRecordCallback(func(what string, key, before, after interface{}) {
		w.addTableRecordChange(tableName, what, key, before, after)
	})
	err := currentTable.UpdateRecord(primary, record)
	currentTable.SetRecordCallback(nil)
	w.CosAssert(err == nil, fmt.Sprintf("TableUpdateRecord(): table.UpdateRecord() failed. %v", err))

	decodeRecord, _ := currentTable.DecodeRecordToJson(record)
	var contractData itype.ContractData
	contractData.Contract = w.ReadContractName()
	contractData.ContractOwner = w.ReadContractOwner()
	contractData.Record = decodeRecord
	w.cosVM.ctx.TrxObserver.AddOpState(iservices.Update, "contract", tableName, contractData)
}

func (w *CosVMNative) TableDeleteRecord(tableName string, primary []byte) {
	tables := w.cosVM.ctx.Tables
	w.CosAssert(tables != nil, "TableDeleteRecord(): context tables not ready.")
	currentTable := tables.Table(tableName)
	currentTable.SetRecordCallback(func(what string, key, before, after interface{}) {
		w.addTableRecordChange(tableName, what, key, before, after)
	})
	err := currentTable.DeleteRecord(primary)
	currentTable.SetRecordCallback(nil)
	w.CosAssert(err == nil, fmt.Sprintf("TableDeleteRecord(): table.DeleteRecord() failed. %v", err))
	// delete should be observer ? Yes and No
	// For yes, every modify should be record of course
	// For No, delete record using primary key which only used in delete
	//m := map[string]string{"contract": w.ReadCallingContractName(), "contract_owner": w.ReadCallingContractOwner(), "record": decodeRecord}
	//w.cosVM.ctx.TrxObserver.AddOpState(iservices.Delete, "contract", tableName, string(primary))
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
	table.NewSoGlobalWrap(w.cosVM.db, &singleId).SetProps(&props, "failed to set reputation admin")
	w.cosVM.props.ReputationAdmin = props.ReputationAdmin
}

func (w *CosVMNative) SetCopyrightAdmin(name string) {
	singleId := int32(constants.SingletonId)
	props := *w.cosVM.props
	props.CopyrightAdmin = prototype.NewAccountName(name)
	table.NewSoGlobalWrap(w.cosVM.db, &singleId).SetProps(&props, "failed to set copyright admin")
	w.cosVM.props.CopyrightAdmin = props.CopyrightAdmin
}

func (w *CosVMNative) GetCopyrightAdmin() (name string) {
	if w.cosVM.props.CopyrightAdmin != nil {
		name = w.cosVM.props.CopyrightAdmin.Value
	}
	return
}

func (w *CosVMNative) SetUserCopyright(postId uint64, value uint32, memo string) {
	post := table.NewSoPostWrap(w.cosVM.db, &postId)
	post.SetCopyright(value, fmt.Sprintf("failed to modify copyright of %d", postId)).
		SetCopyrightMemo(memo, fmt.Sprintf("failed to modify copyright memo of %d", postId))
}

func (w *CosVMNative) GetReputationAdmin() (name string) {
	if w.cosVM.props.ReputationAdmin != nil {
		name = w.cosVM.props.ReputationAdmin.Value
	}
	return
}

func (w *CosVMNative) SetUserReputation(name string, value uint32, memo string) {
	account := table.NewSoAccountWrap(w.cosVM.db, prototype.NewAccountName(name))
	account.SetReputation(value, fmt.Sprintf("failed to modify reputation of %s", name)).
		SetReputationMemo(memo, fmt.Sprintf("failed to modify reputation memo of %s", name))

	// if this account is bp and reputation come to constants.MinReputation, disable it
	if value == constants.MinReputation {
		bp := table.NewSoBlockProducerWrap(w.cosVM.db, prototype.NewAccountName(name))
		if bp != nil && bp.CheckExist() && bp.GetBpVest().Active {
			bpVest := bp.GetBpVest().VoteVest
			newBpVest := &prototype.BpVestId{Active:false, VoteVest:bpVest}
			bp.SetBpVest(newBpVest)
		}
	}
}

func (w *CosVMNative) SetUserFreeze(name string, value uint32, memo string) {
	account := table.NewSoAccountWrap(w.cosVM.db, prototype.NewAccountName(name))
	account.SetFreeze(value, fmt.Sprintf("failed to modify freeze of %s", name)).
		SetFreezeMemo(memo, fmt.Sprintf("failed to modify freeze memo of %s", name))
	if value != 0 {
		w.cosVM.ctx.Injector.DiscardAccountCache(name)
	}
}

type ContractTableRecordChange struct {
	Owner string			`json:"owner"`
	Contract string			`json:"contract"`
	Table string			`json:"table"`
	Key interface{}			`json:"key"`
	Before interface{}		`json:"before"`
	After interface{}		`json:"after"`
}

func (w *CosVMNative) addTableRecordChange(tableName, what string, key, before, after interface{}) {
	w.cosVM.ctx.Injector.StateChangeContext().AddChange(what, &ContractTableRecordChange{
		Owner: w.ReadContractOwner(),
		Contract: w.ReadContractName(),
		Table: tableName,
		Key: key,
		Before: before,
		After: after,
	})
}
