package vm

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/vm/context"
	"github.com/coschain/contentos-go/vm/validator"
	"github.com/go-interpreter/wagon/exec"
	"github.com/go-interpreter/wagon/wasm"
	"github.com/sirupsen/logrus"
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
	logger         *logrus.Logger
	spentGas       uint64
}

func NewCosVM(ctx *vmcontext.Context, db iservices.IDatabaseService, props *prototype.DynamicProperties, logger *logrus.Logger) *CosVM {
	// spentGas should be 0 or maxint?
	cosVM := &CosVM{nativeFuncName: []string{}, nativeFuncSigs: []wasm.FunctionSig{},
		nativeFuncs: []wasm.Function{}, ctx: ctx, logger: logger, db: db, props: props, spentGas: 0}
	return cosVM
}

func (w *CosVM) initNativeFuncs() {
	exports := &CosVMExport{&CosVMNative{cosVM: w}}
	w.Register("sha256", exports.sha256, 500)
	w.Register("current_block_number", exports.currentBlockNumber, 100)
	w.Register("current_timestamp", exports.currentTimestamp, 100)
	w.Register("current_witness", exports.currentWitness, 150)
	w.Register("print_str", exports.printString, 100)
	w.Register("print_int", exports.printInt64, 100)
	w.Register("print_uint", exports.printUint64, 100)
	w.Register("require_auth", exports.requiredAuth, 200)
	w.Register("get_balance_by_name", exports.getBalanceByName, 100)
	w.Register("get_contract_balance", exports.getContractBalance, 100)
	w.Register("save_to_storage", exports.saveToStorage, 1000)
	w.Register("read_from_storage", exports.readFromStorage, 300)
	w.Register("cos_assert", exports.cosAssert, 100)
	w.Register("abort", exports.cosAbort, 100)
	w.Register("read_contract_op_params", exports.readContractOpParams, 100)
	w.Register("read_contract_name", exports.readContractName, 100)
	w.Register("read_contract_method", exports.readContractMethod, 100)
	w.Register("read_contract_owner", exports.readContractOwner, 100)
	w.Register("read_contract_caller", exports.readContractCaller, 100)
	w.Register("read_contract_sender_value", exports.readContractSenderValue, 100)
	w.Register("transfer", exports.contractTransfer, 800)
	w.Register("table_get_record", exports.tableGetRecord, 800)
	w.Register("table_new_record", exports.tableNewRecord, 800)
	w.Register("table_update_record", exports.tableUpdateRecord, 800)
	w.Register("table_delete_record", exports.tableDeleteRecord, 800)

	// for memeory
	w.Register("memcpy", w.memcpy, 100)
	w.Register("memset", w.memset, 100)

	// for test
	w.Register("readt1", w.readT1, 10)
	w.Register("readt2", w.readT2, 10)
	w.Register("readt3", w.readT3, 10)
	w.Register("writet1", w.writeT1, 30)
}

func (w *CosVM) readModule() (*wasm.Module, error) {
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
	return vmModule, err
}

func (w *CosVM) Run() (ret uint32, err error) {
	defer func() {
		if e := recover(); e != nil {
			ret = 1
			err = errors.New(fmt.Sprintf("%v", e))
		}
	}()
	vmModule, err := w.readModule()
	if err != nil {
		ret = 1
		return
	}
	vm, err := exec.NewVM(vmModule)
	defer func() {
		w.spentGas = vm.CostGas
	}()
	if err != nil {
		ret = 1
		return
	}
	vm.InitGasTable(w.ctx.Gas.Value)
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

func (w *CosVM) SpentGas() uint64 {
	return w.spentGas
}

func (w *CosVM) Validate() error {
	vmModule, err := w.readModule()
	if err != nil {
		return err
	}
	err = vmvalidator.VerifyModule(vmModule)
	return err
}

func (w *CosVM) Register(funcName string, function interface{}, gas uint64) {
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
	f := wasm.Function{Sig: &funcSig, Host: reflect.ValueOf(function), Body: &wasm.FunctionBody{}, Gas: gas}
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
	fmt.Println(fmt.Sprintf("length %d", length))
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

func (w *CosVM) readT3(proc *exec.Process, pointer, pos int32) int32 {
	data := make([]byte, 1)
	_, _ = proc.ReadAt(data, int64(pointer+pos))
	return int32(data[0])
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
	if maxLength == 0 {
		return w.strLen(proc, pointer)
	} else {
		return w.readStr(proc, pointer, maxLength, buf)
	}
}

func (w *CosVM) strLen(proc *exec.Process, pointer int32) (length int32, err error) {
	// for now, the max read length is 36
	var buf []byte
	for {
		perLength, _ := w.readStr(proc, pointer, int32(maxReadLength), &buf)
		length += perLength
		pointer += perLength
		if perLength < int32(maxReadLength) {
			break
		}
	}
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

func (w *CosVM) memcpy(proc *exec.Process, dst, src, size int32) int32 {
	data := make([]byte, size)
	// ErrShortBuffer should be ignored ?
	length, _ := proc.ReadAt(data, int64(src))
	// as so on ErrShortWrite ?
	_, _ = proc.WriteAt(data[:length], int64(dst))
	return dst
}

func (w *CosVM) memset(proc *exec.Process, ptr, value, size int32) int32 {
	data := make([]byte, size)
	if value < 0 || value > 255 {
		panic("value should between 0 and 255")
	}
	for i := range data {
		data[i] = byte(value)
	}
	_, _ = proc.WriteAt(data, int64(ptr))
	return ptr
}
