package vm

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/go-interpreter/wagon/exec"
	"github.com/go-interpreter/wagon/wasm"
	"github.com/inconshreveable/log15"
	"reflect"
	"sync"
)

type CosVM struct {
	nativeFuncName []string
	nativeFuncSigs []wasm.FunctionSig
	nativeFuncs    []wasm.Function
	ctx            *Context
	lock           sync.RWMutex
	logger         log15.Logger
}

func NewCosVM(ctx *Context, logger log15.Logger) *CosVM {
	return &CosVM{nativeFuncName: []string{}, nativeFuncSigs: []wasm.FunctionSig{},
		nativeFuncs: []wasm.Function{}, ctx: ctx, logger: logger}
}

func (w *CosVM) initNativeFuncs() {
	w.Register("sha256", w.Sha256)
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

func (w *CosVM) sha256(in []byte) [32]byte {
	return sha256.Sum256(in)
}

func (w *CosVM) Sha256(proc *exec.Process, pSrc int64, lenSrc int32, pDst int64, lenDst int32) {
	srcBuf := make([]byte, lenSrc)
	_, _ = proc.ReadAt(srcBuf, pSrc)
	out := w.sha256(srcBuf)
	_, _ = proc.WriteAt(out[:lenDst], pDst)
}
