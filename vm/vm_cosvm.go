package vm

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/go-interpreter/wagon/exec"
	"github.com/go-interpreter/wagon/wasm"
	"github.com/inconshreveable/log15"
	"hash/crc32"
	"reflect"
	"sync"
)

type CosVM struct {
	nativeFuncName []string
	nativeFuncSigs []wasm.FunctionSig
	nativeFuncs    []wasm.Function
	ctx            *Context
	db             iservices.IDatabaseService
	props          *prototype.DynamicProperties
	lock           sync.RWMutex
	logger         log15.Logger
}

func NewCosVM(ctx *Context, db iservices.IDatabaseService, props *prototype.DynamicProperties, logger log15.Logger) *CosVM {
	return &CosVM{nativeFuncName: []string{}, nativeFuncSigs: []wasm.FunctionSig{},
		nativeFuncs: []wasm.Function{}, ctx: ctx, logger: logger, db: db, props: props}
}

func (w *CosVM) initNativeFuncs() {
	w.Register("sha256", w.sha256)
	w.Register("current_block_number", w.currentBlockNumber)
	w.Register("current_timestamp", w.currentTimestamp)
	w.Register("current_witness", w.currentWitness)
}

func (w *CosVM) Run() (uint32, error) {
	w.initNativeFuncs()
	m := wasm.NewModule()
	m.Types = &wasm.SectionTypes{Entries: w.nativeFuncSigs}
	m.FunctionIndexSpace = w.nativeFuncs
	entries := make(map[string]wasm.ExportEntry)
	for idx, name := range w.nativeFuncName {
		entries[name] = wasm.ExportEntry{
			FieldStr: name,
			Kind:     wasm.ExternalFunction,
			Index:    uint32(idx),
		}
	}
	m.Export = &wasm.SectionExports{
		Entries: entries,
	}
	code := w.ctx.Code
	vmModule, err := wasm.ReadModule(bytes.NewReader(code), func(name string) (module *wasm.Module, e error) {
		return m, nil
	})
	if err != nil {
		return 1, err
	}
	vm, err := exec.NewVM(vmModule)
	if err != nil {
		return 1, err
	}

	var entryIndex = -1
	for name, entry := range vmModule.Export.Entries {
		if name == "main" && entry.Kind == wasm.ExternalFunction {
			entryIndex = int(entry.Index)
		}
	}
	if entryIndex >= 0 {
		r, err := vm.ExecCode(int64(entryIndex))
		if err != nil {
			if err.Error() != "exec: return" && err.Error() != "exec: revert" && err.Error() != "exec: suicide" {
				return 1, fmt.Errorf("Error excuting function %d: %v", 0, err)
			}
		}
		return r.(uint32), err
	}
	return 0, nil
}

func (w *CosVM) Register(funcName string, function interface{}) {
	w.lock.RLock()
	defer w.lock.RUnlock()
	rfunc := reflect.TypeOf(function)
	if rfunc.Kind() != reflect.Func {
		w.logger.Error(fmt.Sprintf("%s is not a function", funcName))
		return
	}
	// func should be func(proc *exec.Process, ... interface{})
	if rfunc.NumIn() < 1 {
		w.logger.Error(fmt.Sprintf("function signature of %s is wrong", funcName))
		return
	}
	funcSig, err := w.exactFuncSig(rfunc)
	if err != nil {
		w.logger.Error("exactFuncSig error:", err)
		return
	}
	f := wasm.Function{Sig: &funcSig, Host: reflect.ValueOf(function), Body: &wasm.FunctionBody{}}
	w.nativeFuncName = append(w.nativeFuncName, funcName)
	w.nativeFuncSigs = append(w.nativeFuncSigs, funcSig)
	w.nativeFuncs = append(w.nativeFuncs, f)
}

func (w *CosVM) readAt(proc *exec.Process, pointer int32, length int32) (buf []byte, err error) {
	buf = make([]byte, length)
	_, err = proc.ReadAt(buf, int64(pointer))
	return
}

func (w *CosVM) exactFuncSig(p reflect.Type) (wasm.FunctionSig, error) {
	paramTypes := []wasm.ValueType{}
	returnTypes := []wasm.ValueType{}
	argsLens := p.NumIn()
	returnLens := p.NumOut()
	// step over first params, it is proc
	for i := 1; i < argsLens; i++ {
		arg := p.In(i)
		switch arg.Kind() {
		case reflect.Int32:
			paramTypes = append(paramTypes, wasm.ValueTypeI32)
		case reflect.Int64:
			paramTypes = append(paramTypes, wasm.ValueTypeI64)
		case reflect.Float32:
			paramTypes = append(paramTypes, wasm.ValueTypeF32)
		case reflect.Float64:
			paramTypes = append(paramTypes, wasm.ValueTypeF64)
		default:
			return wasm.FunctionSig{ParamTypes: paramTypes, ReturnTypes: returnTypes}, errors.New("nativeFunc's type of arguments should in i32, i64, f32, f64")
		}
	}
	for i := 0; i < returnLens; i++ {
		arg := p.Out(i)
		switch arg.Kind() {
		case reflect.Int32:
			returnTypes = append(returnTypes, wasm.ValueTypeI32)
		case reflect.Int64:
			returnTypes = append(returnTypes, wasm.ValueTypeI64)
		case reflect.Float32:
			returnTypes = append(returnTypes, wasm.ValueTypeF32)
		case reflect.Float64:
			returnTypes = append(returnTypes, wasm.ValueTypeF64)
		default:
			return wasm.FunctionSig{ParamTypes: paramTypes, ReturnTypes: returnTypes}, errors.New("nativeFunc's type of arguments should in i32, i64, f32, f64")
		}
	}
	return wasm.FunctionSig{ParamTypes: paramTypes, ReturnTypes: returnTypes}, nil
}

// private version methods as interface and public version as implement
// isn't it strange ?
func (w *CosVM) Sha256(in []byte) [32]byte {
	return sha256.Sum256(in)
}

func (w *CosVM) sha256(proc *exec.Process, pSrc int32, lenSrc int32, pDst int32, lenDst int32) {
	srcBuf, err := w.readAt(proc, pSrc, lenSrc)
	if err != nil {
		w.logger.Error("sha256 read error:", err)
		return
	}
	out := w.Sha256(srcBuf)
	_, err = proc.WriteAt(out[:lenDst], int64(pDst))
	if err != nil {
		w.logger.Error("sha256 write error:", err)
	}
}

func (w *CosVM) CurrentBlockNumber() uint64 {
	return w.props.HeadBlockNumber
}

func (w *CosVM) currentBlockNumber(proc *exec.Process) int64 {
	return int64(w.CurrentBlockNumber())
}

func (w *CosVM) CurrentTimestamp() uint64 {
	return uint64(w.props.Time.UtcSeconds)
}

func (w *CosVM) currentTimestamp(proc *exec.Process) int64 {
	return int64(w.CurrentTimestamp())
}

func (w *CosVM) CurrentWitness() string {
	return w.props.CurrentWitness.Value
}

func (w *CosVM) currentWitness(proc *exec.Process, pDst int32, lenDst int32) {
	witness := w.CurrentWitness()
	buf := make([]byte, lenDst)
	copy(buf[:], witness)
	_, err := proc.WriteAt(buf, int64(pDst))
	if err != nil {
		w.logger.Error("current witness write error:", err)
	}
}

func (w *CosVM) PrintString(str string) {
	fmt.Printf(str)
}

func (w *CosVM) printString(proc *exec.Process, pStr int32, lenStr int32) {
	buf, err := w.readAt(proc, pStr, lenStr)
	if err != nil {
		w.logger.Error("print string error:", err)
	}
	w.PrintString(string(buf))
}

func (w *CosVM) PrintUint32(value uint32) {
	fmt.Printf("%d", value)
}

func (w *CosVM) printUint32(proc *exec.Process, value uint32) {
	w.PrintUint32(value)
}

func (w *CosVM) PrintUint64(value uint64) {
	fmt.Printf("%d", value)
}

func (w *CosVM) printUint64(proc *exec.Process, value uint64) {
	w.PrintUint64(value)
}

func (w *CosVM) PrintBool(value bool) {
	if value {
		fmt.Printf("true")
	} else {
		fmt.Printf("false")
	}
}

func (w *CosVM) printBool(proc *exec.Process, value bool) {
	w.PrintBool(value)
}

func (w *CosVM) RequiredAuth(name string) {
	w.ctx.Injector.RequireAuth(name)
}

func (w *CosVM) requiredAuth(proc *exec.Process, pStr int32, pLen int32) {
	name, err := w.readAt(proc, pStr, pLen)
	if err != nil {
		w.logger.Error("required auth read name error:", err)
	}
	w.RequiredAuth(string(name))
}

func (w *CosVM) GetBalanceByName(name string) uint64 {
	acc := table.NewSoAccountWrap(w.db, &prototype.AccountName{Value: name})
	return acc.GetBalance().Value
}

func (w *CosVM) getBalanceByName(proc *exec.Process, ptr int32, len int32) int64 {
	buf, err := w.readAt(proc, ptr, len)
	if err != nil {
		w.logger.Error("get balance by name error when read name:", err)
	}
	return int64(w.GetBalanceByName(string(buf)))
}

func (w *CosVM) GetContractBalance(contract string, name string) uint64 {
	ctct := table.NewSoContractWrap(w.db, &prototype.ContractId{Owner: &prototype.AccountName{Value: name}, Cname: contract})
	return ctct.GetBalance().Value
}

func (w *CosVM) getContractBalance(proc *exec.Process, cPtr int32, cLen int32, nPtr int32, nLen int32) int64 {
	cBuf, err := w.readAt(proc, cPtr, cLen)
	if err != nil {
		w.logger.Error("get contract balance error when read contract name:", err)
	}
	nBuf, err := w.readAt(proc, nPtr, nLen)
	if err != nil {
		w.logger.Error("get contract balance error when read name:", err)
	}
	return int64(w.GetContractBalance(string(cBuf), string(nBuf)))
}

func (w *CosVM) SaveToStorage(key []byte, value []byte) {
	crc32q := crc32.MakeTable(0xD5828281)
	pos := int32(crc32.Checksum(key, crc32q))
	contractDB := table.NewSoContractDataWrap(w.db, &prototype.ContractDataId{Owner: w.ctx.Owner, Cname: w.ctx.Contract, Pos: pos})
	err := contractDB.Create(func(tInfo *table.SoContractData) {
		tInfo.Key = key
		tInfo.Value = value
	})
	if err != nil {
		w.logger.Error("save to storage error, contract: %s, owner: %s", w.ctx.Contract, w.ctx.Owner.Value)
	}
}

func (w *CosVM) saveToStorage(proc *exec.Process, pKey int32, kLen int32, pValue int32, vLen int32) {
	key, err := w.readAt(proc, pKey, kLen)
	if err != nil {
		w.logger.Error("get contract balance error when read contract name:", err)
	}
	value, err := w.readAt(proc, pValue, vLen)
	if err != nil {
		w.logger.Error("get contract balance error when read name:", err)
	}
	w.SaveToStorage(key, value)
}

func (w *CosVM) ReadFromStorage(key []byte) (value []byte) {
	crc32q := crc32.MakeTable(0xD5828281)
	pos := int32(crc32.Checksum(key, crc32q))
	contractDB := table.NewSoContractDataWrap(w.db, &prototype.ContractDataId{Owner: w.ctx.Owner, Cname: w.ctx.Contract, Pos: pos})
	value = contractDB.GetValue()
	return
}

func (w *CosVM) readFromStorage(proc *exec.Process, pKey int32, kLen int32, pValue int32, vLen int32) {
	//key := make([]byte, kLen)
	//_, err := proc.ReadAt(key, int64(pKey))
	key, err := w.readAt(proc, pKey, kLen)
	if err != nil {
		w.logger.Error("get contract balance error when read contract name:", err)
	}
	value := w.ReadFromStorage(key)
	if len(value) > int(vLen) {
		value = value[:vLen]
	}
	_, err = proc.WriteAt(value, int64(pValue))
	if err != nil {
		w.logger.Error("get contract balance error when read name:", err)
	}
}

//func (w *CosVM) logSort()  {
//
//}

func (w *CosVM) CosAssert(condition bool, msg string) {
	if !condition {
		panic(msg)
	}
}

func (w *CosVM) cosAssert(proc *exec.Process, condition bool, pStr int32, len int32) {
	//msg := make([]byte, len)
	//_, err := proc.ReadAt(msg, int64(pStr))
	msg, err := w.readAt(proc, pStr, len)
	if err != nil {
		w.logger.Error("read cos assert msg error:", err)
	}
	w.CosAssert(condition, string(msg))
}

func (w *CosVM) ReadContractOwner() string {
	return w.ctx.Owner.Value
}

func (w *CosVM) readContractOwner(proc *exec.Process, pStr int32, length int32) {
	owner := w.ReadContractOwner()
	byteOwner := []byte(owner)
	if len(byteOwner) > int(length) {
		byteOwner = byteOwner[:length]
	}
	_, err := proc.WriteAt(byteOwner, int64(pStr))
	if err != nil {
		w.logger.Error("write owner into memory err:", err)
	}
}

func (w *CosVM) ReadContractCaller() string {
	return w.ctx.Caller.Value
}

func (w *CosVM) readContractCaller(proc *exec.Process, pStr int32, length int32) {
	caller := w.ReadContractCaller()
	byteCaller := []byte(caller)
	if len(byteCaller) > int(length) {
		byteCaller = byteCaller[:length]
	}
	_, err := proc.WriteAt(byteCaller, int64(pStr))
	if err != nil {
		w.logger.Error("write caller into memory err:", err)
	}
}

func (w *CosVM) Transfer(from string, to string, amount uint64, memo string) {
	w.ctx.Injector.Transfer(from, to, amount, memo)
}

func (w *CosVM) transfer(proc *exec.Process, pFrom, pFromLen, pTo, pToLen int32, amount int64, pMemo, pMemoLen int32) {
	from, _ := w.readAt(proc, pFrom, pFromLen)
	to, _ := w.readAt(proc, pTo, pToLen)
	memo, _ := w.readAt(proc, pMemo, pMemoLen)
	w.Transfer(string(from), string(to), uint64(amount), string(memo))
}

func (w *CosVM) GetSenderValue() uint64 {
	return w.ctx.Amount.Value
}

func (w *CosVM) getSenderValue(proc *exec.Process) int64 {
	return int64(w.GetSenderValue())
}
