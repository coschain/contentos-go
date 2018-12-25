package vm

import (
	"crypto/sha256"
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/prototype"
	"hash/crc32"
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

func (w *CosVMNative) PrintString(str string) {
	w.cosVM.ctx.Injector.Log(str)
}

func (w *CosVMNative) PrintUint32(value uint32) {
	w.cosVM.ctx.Injector.Log(fmt.Sprintf("%d", value))
}

func (w *CosVMNative) PrintUint64(value uint64) {
	w.cosVM.ctx.Injector.Log(fmt.Sprintf("%d", value))
}

func (w *CosVMNative) PrintBool(value bool) {
	if value {
		w.cosVM.ctx.Injector.Log("true")
	} else {
		w.cosVM.ctx.Injector.Log("false")
	}
}

func (w *CosVMNative) RequiredAuth(name string) {
	err := w.cosVM.ctx.Injector.RequireAuth(name)
	w.CosAssert(err == nil, "require auth error")
}

func (w *CosVMNative) GetBalanceByName(name string) uint64 {
	acc := table.NewSoAccountWrap(w.cosVM.db, &prototype.AccountName{Value: name})
	return acc.GetBalance().Value
}

func (w *CosVMNative) GetContractBalance(contract string, name string) uint64 {
	ctct := table.NewSoContractWrap(w.cosVM.db, &prototype.ContractId{Owner: &prototype.AccountName{Value: name}, Cname: contract})
	return ctct.GetBalance().Value
}

func (w *CosVMNative) SaveToStorage(key []byte, value []byte) {
	crc32q := crc32.MakeTable(0xD5828281)
	pos := int32(crc32.Checksum(key, crc32q))
	contractDB := table.NewSoContractDataWrap(w.cosVM.db, &prototype.ContractDataId{Owner: w.cosVM.ctx.Owner,
		Cname: w.cosVM.ctx.Contract, Pos: pos})
	err := contractDB.Create(func(tInfo *table.SoContractData) {
		tInfo.Key = key
		tInfo.Value = value
	})
	w.CosAssert(err == nil, fmt.Sprintf("save to storage error"))
}

func (w *CosVMNative) ReadFromStorage(key []byte) (value []byte) {
	crc32q := crc32.MakeTable(0xD5828281)
	pos := int32(crc32.Checksum(key, crc32q))
	contractDB := table.NewSoContractDataWrap(w.cosVM.db, &prototype.ContractDataId{Owner: w.cosVM.ctx.Owner,
		Cname: w.cosVM.ctx.Contract, Pos: pos})
	value = contractDB.GetValue()
	return
}

func (w *CosVMNative) CosAssert(condition bool, msg string) {
	if !condition {
		panic(msg)
	}
}

func (w *CosVMNative) ReadContractOpParams() string {
	return w.cosVM.ctx.Params
}

func (w *CosVMNative) ReadContractOpParamsLength() int {
	return len(w.cosVM.ctx.Params)
}

func (w *CosVMNative) ReadContractOwner() string {
	return w.cosVM.ctx.Owner.Value
}

func (w *CosVMNative) ReadContractCaller() string {
	return w.cosVM.ctx.Caller.Value
}

func (w *CosVMNative) ContractTransfer(to string, amount uint64) {
	err := w.cosVM.ctx.Injector.ContractTransfer(w.cosVM.ctx.Contract, w.cosVM.ctx.Owner.Value, to, amount)
	w.CosAssert(err == nil, fmt.Sprintf("transfer error: %v", err))
}

func (w *CosVMNative) GetSenderValue() uint64 {
	return w.cosVM.ctx.Amount.Value
}
