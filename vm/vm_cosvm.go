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
	"math"
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
	// can replace native func
	cosVM.initNativeFuncs()
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
	w.Register("get_user_balance", exports.getUserBalance, 100)
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
	w.Register("contract_call", exports.contractCall, 1000)
	w.Register("contract_called_by_user", exports.contractCalledByUser, 100)
	w.Register("read_calling_contract_owner", exports.readCallingContractOwner, 100)
	w.Register("read_calling_contract_name", exports.readCallingContractName, 100)
	w.Register("transfer_to_user", exports.contractTransferToUser, 800)
	w.Register("transfer_to_contract", exports.contractTransferToContract, 800)
	w.Register("table_get_record", exports.tableGetRecord, 800)
	w.Register("table_new_record", exports.tableNewRecord, 800)
	w.Register("table_update_record", exports.tableUpdateRecord, 800)
	w.Register("table_delete_record", exports.tableDeleteRecord, 800)

	// for memeory
	w.Register("memcpy", w.memcpy, 100)
	w.Register("memset", w.memset, 100)
	w.Register("memmove", w.memmove, 100)
	w.Register("memcmp", w.memcmp, 100)

	// for io
	w.Register("copy", w.copy, 100)

}

func (w *CosVM) readModule() (*wasm.Module, error) {
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
		if name == "apply" && entry.Kind == wasm.ExternalFunction {
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

func (w *CosVM) Estimate() (gas uint64, err error) {
	defer func() {
		if e := recover(); e != nil {
			gas = math.MaxUint64
			err = errors.New(fmt.Sprintf("estimate error: %v", e))
			return
		}
	}()
	vmModule, err := w.readModule()
	if err != nil {
		return math.MaxUint64, err
	}
	vm, err := exec.NewEstimator(vmModule)
	defer func() {
		w.spentGas = vm.CostGas
	}()
	var entryIndex = -1
	for name, entry := range vmModule.Export.Entries {
		if name == "main" && entry.Kind == wasm.ExternalFunction {
			entryIndex = int(entry.Index)
		}
	}
	if entryIndex >= 0 {
		_, err = vm.ExecCode(int64(entryIndex))
		if err != nil {
			return math.MaxUint64, err
		}
		gas = vm.CostGas
		return gas, nil
	} else {
		return 0, errors.New("unable to execute code")
	}
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

func (w *CosVM) read(proc *exec.Process, buffer int32, bufferSize int32, tag string) []byte {
	if bufferSize < 0 {
		panic(fmt.Sprintf("%s: negative reading size", tag))
	}
	buf := make([]byte, bufferSize)
	n, err := proc.ReadAt(buf, int64(buffer))
	if err != nil {
		panic(fmt.Sprintf("%s: reading failed. %v", tag, err))
	}
	return buf[:n]
}

func (w *CosVM) write(proc *exec.Process, data []byte, buffer int32, bufferSize int32, tag string) int32 {
	size := int32(len(data))
	if bufferSize <= 0 {
		return size
	}
	if size > bufferSize {
		size = bufferSize
	}
	n, err := proc.WriteAt(data[:size], int64(buffer))
	if err != nil {
		panic(fmt.Sprintf("%s: writing failed. %v", tag, err))
	}
	return int32(n)
}

func (w *CosVM) copy(proc *exec.Process, src int32, dst int32, length int32) int32 {
	data := w.read(proc, src, length, "copy->read")
	return w.write(proc, data, dst, length, "copy->write")
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
	w.write(proc, w.read(proc, src, size, "memcpy().src"), dst, size, "memcpy().dst")
	return dst
}

func (w *CosVM) memset(proc *exec.Process, dst, value, size int32) int32 {
	w.write(proc, bytes.Repeat([]byte{byte(value)}, int(size)), dst, size, "memset().dst")
	return dst
}

func (w *CosVM) memmove(proc *exec.Process, dst, src, size int32) int32 {
	return w.memcpy(proc, dst, src, size)
}

func (w *CosVM) memcmp(proc *exec.Process, lhs, rhs, size int32) int32 {
	return int32(bytes.Compare(
		w.read(proc, lhs, size, "memcmp().lhs"),
		w.read(proc, rhs, size, "memcmp().rhs")))
}
