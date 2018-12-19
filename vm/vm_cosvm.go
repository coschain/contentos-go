package vm

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/vm/context"
	"github.com/go-interpreter/wagon/exec"
	"github.com/go-interpreter/wagon/wasm"
	"github.com/inconshreveable/log15"
	"hash/crc32"
	"io"
	"reflect"
	"sync"
)

const (
	maxReadLength int = 100
)

type CosVM struct {
	nativeFuncName []string
	nativeFuncSigs []wasm.FunctionSig
	nativeFuncs    []wasm.Function
	ctx            *vmcontext.Context
	db             iservices.IDatabaseService
	props          *prototype.DynamicProperties
	lock           sync.RWMutex
	logger         log15.Logger
}

func NewCosVM(ctx *vmcontext.Context, db iservices.IDatabaseService, props *prototype.DynamicProperties, logger log15.Logger) *CosVM {
	return &CosVM{nativeFuncName: []string{}, nativeFuncSigs: []wasm.FunctionSig{},
		nativeFuncs: []wasm.Function{}, ctx: ctx, logger: logger, db: db, props: props}
}

func (w *CosVM) initNativeFuncs() {
	w.Register("sha256", w.sha256)
	w.Register("current_block_number", w.currentBlockNumber)
	w.Register("current_timestamp", w.currentTimestamp)
	w.Register("current_witness", w.currentWitness)
	w.Register("print_string", w.printString)
	w.Register("print_uint32", w.printUint32)
	w.Register("print_uint64", w.printUint64)
	w.Register("print_bool", w.printBool)
	w.Register("require_auth", w.requiredAuth)
	w.Register("get_balance_by_name", w.getBalanceByName)
	w.Register("get_contract_balance", w.getContractBalance)
	w.Register("save_to_storage", w.saveToStorage)
	w.Register("read_from_storage", w.readFromStorage)
	w.Register("cos_assert", w.cosAssert)
	w.Register("read_contract_owner", w.readContractOwner)
	w.Register("read_contract_caller", w.readContractCaller)
	w.Register("contract_transfer", w.contractTransfer)
	w.Register("get_sender_value", w.getSenderValue)

	// for test
	w.Register("readt1", w.readT1)
	w.Register("readt2", w.readT2)
	w.Register("writet1", w.writeT1)
}

func (w *CosVM) Run() (ret uint32, err error) {
	defer func() {
		if e := recover(); e != nil {
			ret = 1
			err = errors.New("cosvm exit by exception")
		}
	}()
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
		ret = 1
		return
	}
	vm, err := exec.NewVM(vmModule)
	if err != nil {
		ret = 1
		return
	}

	var entryIndex = -1
	for name, entry := range vmModule.Export.Entries {
		if name == "main" && entry.Kind == wasm.ExternalFunction {
			entryIndex = int(entry.Index)
		}
	}
	if entryIndex >= 0 {
		r, e := vm.ExecCode(int64(entryIndex))
		ret = r.(uint32)
		err = e
	}
	return
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
		w.logger.Error("exactFuncSig error:", funcName, err)
		return
	}
	f := wasm.Function{Sig: &funcSig, Host: reflect.ValueOf(function), Body: &wasm.FunctionBody{}}
	w.nativeFuncName = append(w.nativeFuncName, funcName)
	w.nativeFuncSigs = append(w.nativeFuncSigs, funcSig)
	w.nativeFuncs = append(w.nativeFuncs, f)
}

func (w *CosVM) readT1(proc *exec.Process, pointer int32, maxLength int32) int32 {
	var msg []byte
	length, err := w.readStrAt(proc, pointer, maxLength, &msg)
	if err != nil {
		fmt.Println("read error:", err)
	}
	fmt.Printf("length %d\n", length)
	fmt.Println(string(msg))
	return length
}

func (w *CosVM) readT2(proc *exec.Process, pointer int32) int32 {
	var msg []byte
	length, err := w.readStrAt(proc, pointer, int32(maxReadLength), &msg)
	if err != nil {
		fmt.Println("read error:", err)
	}
	fmt.Printf("length %d\n", length)
	fmt.Println(string(msg))
	return length
}

func (w *CosVM) writeT1(proc *exec.Process, spointer int32, dpointer int32) int32 {
	var msg []byte
	length, err := w.readStrAt(proc, spointer, int32(maxReadLength), &msg)
	if err != nil {
		fmt.Println("read error:", err)
	}
	length, err = w.writeStrAt(proc, msg, dpointer, length)
	if err != nil {
		fmt.Println("write error:", err)
	}
	return length
}

func (w *CosVM) readStrAt(proc *exec.Process, pointer int32, maxLength int32, buf *[]byte) (length int32, err error) {
	if pointer == 0 && maxLength == 0 {
		return w.strLen(proc, pointer)
	} else {
		return w.readStr(proc, pointer, maxLength, buf)
	}
}

func (w *CosVM) strLen(proc *exec.Process, pointer int32) (length int32, err error) {
	// for now, the max read length is 36
	var buf []byte
	length, err = w.readStr(proc, pointer, int32(maxReadLength), &buf)
	// never raise error
	return length, nil
}

func (w *CosVM) readStr(proc *exec.Process, pointer int32, maxLength int32, buf *[]byte) (int32, error) {
	length := int(maxLength)
	data := make([]byte, maxLength)
	length, err := proc.ReadAt(data, int64(pointer))
	if err == io.ErrShortBuffer {
		w.logger.Error(fmt.Sprintf("io.ErrShortBuffer: %v", w.ctx))
		err = nil
	}
	// if read \0 in middle break
	for i, c := range data {
		if c == 0 {
			length = int(i)
			break
		}
	}
	*buf = append(*buf, data[:length]...)
	return int32(length), err
}

func (w *CosVM) writeStrAt(proc *exec.Process, bytes []byte, pointer int32, maxLen int32) (length int32, err error) {
	buf := make([]byte, maxLen)
	if len(bytes) > int(maxLen) {
		copy(buf, bytes[:maxLen])
	} else {
		copy(buf, bytes[:])
	}
	return w.writeStr(proc, buf, pointer)
}

func (w *CosVM) writeStr(proc *exec.Process, bytes []byte, pointer int32) (int32, error) {
	length := len(bytes)
	// \00 in str, break it and return front part
	for i, c := range bytes {
		if c == 0 {
			length = i
			break
		}
	}
	if length == 0 {
		return 0, errors.New("write nil")
	}
	buf := make([]byte, length)
	copy(buf, bytes[:length])
	length, err := proc.WriteAt(buf, int64(pointer))
	if err == io.ErrShortWrite {
		w.logger.Error(fmt.Sprintf("io.ErrShortWrite: %v", w.ctx))
		err = nil
	}
	return int32(length), err
}

func (w *CosVM) exactFuncSig(p reflect.Type) (wasm.FunctionSig, error) {
	var paramTypes []wasm.ValueType
	var returnTypes []wasm.ValueType
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
	var srcBuf []byte
	_, err := w.readStrAt(proc, pSrc, lenSrc, &srcBuf)
	if err != nil {
		w.logger.Error("sha256 read error:", err)
		return
	}
	out := sha256.Sum256(srcBuf)
	_, err = w.writeStrAt(proc, out[:], pDst, lenDst)
	if err != nil {
		panic(errors.New("write sha256 error"))
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

func (w *CosVM) currentWitness(proc *exec.Process, pDst int32) (length int32) {
	witness := w.CurrentWitness()
	buf := []byte(witness)
	length, err := w.writeStrAt(proc, buf, pDst, int32(maxReadLength))
	w.CosAssert(err == nil, "get current witness error")
	return length
}

func (w *CosVM) PrintString(str string) {
	fmt.Println(str)
}

func (w *CosVM) printString(proc *exec.Process, pStr int32, lenStr int32) {
	var str []byte
	_, err := w.readStrAt(proc, pStr, lenStr, &str)
	if err != nil {
		panic(errors.New("read pre print str error"))
	}
	w.PrintString(string(str))
}

// need support indirect uint32
//func (w *CosVM) PrintIndirectUint32(pointer int32) {
//
//}
//
//func (w *CosVM) printIndirectUint32(proc *exec.Process, pointer int32) {
//	value := w.readUint32(proc, pointer)
//
//}

func (w *CosVM) PrintUint32(value uint32) {
	fmt.Printf("%d\n", value)
}

func (w *CosVM) printUint32(proc *exec.Process, value int32) {
	w.PrintUint32(uint32(value))
}

func (w *CosVM) PrintUint64(value uint64) {
	fmt.Printf("%d\n", value)
}

func (w *CosVM) printUint64(proc *exec.Process, value int64) {
	w.PrintUint64(uint64(value))
}

func (w *CosVM) PrintBool(value bool) {
	if value {
		fmt.Println("true")
	} else {
		fmt.Println("false")
	}
}

func (w *CosVM) printBool(proc *exec.Process, value int32) {
	w.PrintBool(value > 0)
}

func (w *CosVM) RequiredAuth(name string) {
	err := w.ctx.Injector.RequireAuth(name)
	w.CosAssert(err == nil, "require auth error")
}

func (w *CosVM) requiredAuth(proc *exec.Process, pStr int32, pLen int32) {
	var name []byte
	_, err := w.readStrAt(proc, pStr, pLen, &name)
	if err != nil {
		panic("read auth name error")
	}
	w.RequiredAuth(string(name))
}

func (w *CosVM) GetBalanceByName(name string) uint64 {
	acc := table.NewSoAccountWrap(w.db, &prototype.AccountName{Value: name})
	return acc.GetBalance().Value
}

func (w *CosVM) getBalanceByName(proc *exec.Process, ptr int32, len int32) int64 {
	var name []byte
	_, err := w.readStrAt(proc, ptr, len, &name)
	if err != nil {
		panic(err)
	}
	return int64(w.GetBalanceByName(string(name)))
}

func (w *CosVM) GetContractBalance(contract string, name string) uint64 {
	ctct := table.NewSoContractWrap(w.db, &prototype.ContractId{Owner: &prototype.AccountName{Value: name}, Cname: contract})
	return ctct.GetBalance().Value
}

func (w *CosVM) getContractBalance(proc *exec.Process, cPtr int32, cLen int32, nPtr int32, nLen int32) int64 {
	var contract []byte
	_, err := w.readStrAt(proc, cPtr, cLen, &contract)
	if err != nil {
		panic(err)
	}
	var name []byte
	_, err = w.readStrAt(proc, nPtr, nLen, &name)
	if err != nil {
		w.logger.Error("get contract balance error when read name:", err)
	}
	return int64(w.GetContractBalance(string(contract), string(name)))
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
	var key []byte
	_, err := w.readStrAt(proc, pKey, kLen, &key)
	w.CosAssert(err == nil, "read key failed when save to storage")
	var value []byte
	_, err = w.readStrAt(proc, pValue, vLen, &value)
	w.CosAssert(err == nil, "read value failed when save to storage")
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
	var key []byte
	_, err := w.readStrAt(proc, pKey, kLen, &key)
	w.CosAssert(err == nil, "read key failed when read from stroage")
	value := w.ReadFromStorage(key)
	if len(value) > int(vLen) {
		value = value[:vLen]
	}
	_, err = w.writeStrAt(proc, value, pValue, vLen)
	w.CosAssert(err == nil, "write value failed when read from storage")
}

func (w *CosVM) CosAssert(condition bool, msg string) {
	if !condition {
		panic(msg)
	}
}

func (w *CosVM) cosAssert(proc *exec.Process, condition int32, pStr int32, len int32) {
	var msg []byte
	_, err := w.readStrAt(proc, pStr, len, &msg)
	if err != nil {
		panic("read msg when assert failed")
	}
	w.CosAssert(condition > 0, string(msg))
}

func (w *CosVM) ReadContractOpParams() string {
	return w.ctx.Params
}

func (w *CosVM) readContractOpParams(proc *exec.Process, ptr, length int32) {
	params := w.ReadContractOpParams()
	b := []byte(params)
	if len(b) > int(length) {
		b = b[:length]
	}
	_, err := w.writeStrAt(proc, b, ptr, length)
	w.CosAssert(err == nil, "read contract params error")
}

func (w *CosVM) ReadContractOpParamsLength() int {
	return len(w.ctx.Params)
}

func (w *CosVM) readContractOpParamsLength(proc *exec.Process) int32 {
	length := int32(w.ReadContractOpParamsLength())
	return length
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
	_, err := w.writeStrAt(proc, byteOwner, pStr, length)
	w.CosAssert(err == nil, "write owner into memory err")
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
	_, err := w.writeStrAt(proc, byteCaller, pStr, length)
	w.CosAssert(err == nil, "read contract caller error")
}

func (w *CosVM) ContractTransfer(to string, amount uint64) {
	err := w.ctx.Injector.ContractTransfer(w.ctx.Contract, w.ctx.Owner.Value, to, amount)
	w.CosAssert(err == nil, fmt.Sprintf("transfer error: %v", err))
}

func (w *CosVM) contractTransfer(proc *exec.Process, pTo, pToLen int32, amount int64) {
	var to []byte
	_, err := w.readStrAt(proc, pTo, pToLen, &to)
	w.CosAssert(err == nil, "read to err when transfer")
	w.ContractTransfer(string(to), uint64(amount))
}

func (w *CosVM) GetSenderValue() uint64 {
	return w.ctx.Amount.Value
}

func (w *CosVM) getSenderValue(proc *exec.Process) int64 {
	return int64(w.GetSenderValue())
}
