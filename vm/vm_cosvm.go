package vm

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/vm/cache"
	"github.com/coschain/contentos-go/vm/context"
	"github.com/coschain/contentos-go/vm/validator"
	"github.com/go-interpreter/wagon/exec"
	"github.com/go-interpreter/wagon/wasm"
	"github.com/sirupsen/logrus"
	"reflect"
)

const (
	maxReadLength int = 100
)

type CosVM struct {
	nativeFuncName []string
	nativeFuncSigs []wasm.FunctionSig
	nativeFuncs    []wasm.Function
	ctx            *vmcontext.Context
	db             iservices.IDatabaseRW
	props          *prototype.DynamicProperties
	logger         *logrus.Logger
	spentGas       uint64
	Vm             *exec.VM
}

func NewCosVM(ctx *vmcontext.Context, db iservices.IDatabaseRW, props *prototype.DynamicProperties, logger *logrus.Logger) *CosVM {
	// spentGas should be 0 or maxint?
	cosVM := &CosVM{nativeFuncName: []string{}, nativeFuncSigs: []wasm.FunctionSig{},
		nativeFuncs: []wasm.Function{}, ctx: ctx, logger: logger, db: db, props: props, spentGas: 0}
	// can replace native func
	cosVM.initNativeFuncs()
	return cosVM
}

func (w *CosVM) initNativeFuncs() {
	w.Register("sha256", e_sha256, 500)
	w.Register("current_block_number", e_currentBlockNumber, 100)
	w.Register("current_timestamp", e_currentTimestamp, 100)
	w.Register("current_witness", e_currentWitness, 150)
	w.Register("print_str", e_printString, 100)
	w.Register("print_int", e_printInt64, 100)
	w.Register("print_uint", e_printUint64, 100)
	w.Register("require_auth", e_requiredAuth, 200)
	w.Register("get_user_balance", e_getUserBalance, 100)
	w.Register("get_contract_balance", e_getContractBalance, 100)
	w.Register("cos_assert", e_cosAssert, 100)
	w.Register("abort", e_cosAbort, 100)
	w.Register("read_contract_op_params", e_readContractOpParams, 100)
	w.Register("read_contract_name", e_readContractName, 100)
	w.Register("read_contract_method", e_readContractMethod, 100)
	w.Register("read_contract_owner", e_readContractOwner, 100)
	w.Register("read_contract_caller", e_readContractCaller, 100)
	w.Register("read_contract_sender_value", e_readContractSenderValue, 100)
	w.Register("contract_call", e_contractCall, 1000)
	w.Register("contract_called_by_user", e_contractCalledByUser, 100)
	w.Register("read_calling_contract_owner", e_readCallingContractOwner, 100)
	w.Register("read_calling_contract_name", e_readCallingContractName, 100)
	w.Register("transfer_to_user", e_contractTransferToUser, 800)
	w.Register("transfer_to_contract", e_contractTransferToContract, 800)
	w.Register("table_get_record", e_tableGetRecord, 800)
	w.Register("table_new_record", e_tableNewRecord, 1200)
	w.Register("table_update_record", e_tableUpdateRecord, 1200)
	w.Register("table_delete_record", e_tableDeleteRecord, 1000)
	w.Register("table_get_record_ex", e_tableGetRecordEx, 1500)

	w.Register("get_block_producers", e_getBlockProducers, 500)

	w.Register("set_reputation_admin", e_setReputationAdmin, 0)
	w.Register("get_reputation_admin", e_getReputationAdmin, 100)
	w.Register("set_reputation", e_setReputation, 0)

	w.Register("set_copyright_admin", e_setCopyrightAdmin, 0)
	w.Register("set_copyright", e_setCopyright, 0)
	w.Register("set_freeze",e_freeze,0)

	// for memeory
	w.Register("memcpy", e_memcpy, 100)
	w.Register("memset", e_memset, 100)
	w.Register("memmove", e_memmove, 100)
	w.Register("memcmp", e_memcmp, 100)
	// for io
	w.Register("copy", e_copy, 100)

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

func (w *CosVM) runEntry(entryName string) (ret uint32, err error) {
	defer func() {
		if e := recover(); e != nil {
			ret = 1
			err = errors.New(fmt.Sprintf("%v", e))
		}
	}()

	vc := vmcache.GetVmCache()
	vm := vc.Fetch(w.ctx.Owner.Value, w.ctx.Contract, w.ctx.CodeHash.Hash)
	if vm != nil {
		w.logger.Debugf("VMCACHE hit: %s.%s hash=%x", w.ctx.Owner.Value, w.ctx.Contract, w.ctx.CodeHash.Hash)
		vm.Reset()
	} else {
		w.logger.Debugf("VMCACHE missed: %s.%s hash=%x", w.ctx.Owner.Value, w.ctx.Contract, w.ctx.CodeHash.Hash)
		vmModule, errRead := w.readModule()
		if errRead != nil {
			ret = 1
			err = errRead
			return
		}
		vm, err = exec.NewVM(vmModule)
	}
	if err != nil {
		ret = 1
		return
	}
	defer vc.Put(w.ctx.Owner.Value, w.ctx.Contract, w.ctx.CodeHash.Hash, vm)
	w.Vm = vm

	nativeFuncs := NewCosVMNative(w)
	vm.SetTag( nativeFuncs )
	defer func() {
		w.spentGas = vm.CostGas
	}()

	vm.InitGasTable(w.ctx.Gas)
	var entryIndex = -1
	for name, entry := range vm.Module().Export.Entries {
		if name == entryName && entry.Kind == wasm.ExternalFunction {
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

func (w *CosVM) RunMain() (ret uint32, err error) {
	return w.runEntry("main")
}

func (w *CosVM) Run() (ret uint32, err error) {
	return w.runEntry("apply")
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
