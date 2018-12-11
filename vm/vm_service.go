package vm

import (
	"bytes"
	"fmt"
	"github.com/coschain/contentos-go/node"
	"github.com/go-interpreter/wagon/exec"
	"github.com/go-interpreter/wagon/wasm"
	"github.com/kataras/go-errors"
	"reflect"
	"sync"
)

type WasmVmService struct {
	nativeFuncName []string
	nativeFuncSigs []wasm.FunctionSig
	nativeFuncs    []wasm.Function
	module         *wasm.Module
	lock           sync.RWMutex
	ctx            *node.ServiceContext
}

func New(ctx *node.ServiceContext) (*WasmVmService, error) {
	return &WasmVmService{nativeFuncName: []string{}, nativeFuncSigs: []wasm.FunctionSig{},
		nativeFuncs: []wasm.Function{}, ctx: ctx}, nil
}

func (w *WasmVmService) Run(ctx *Context) (uint32, error) {
	code := ctx.Code
	module, err := wasm.ReadModule(bytes.NewReader(code), func(name string) (module *wasm.Module, e error) {
		return w.module, nil
	})
	// why resolvePath?
	if err != nil {
		return 1, err
	}
	vm, err := exec.NewVM(module)
	if err != nil {
		return 1, err
	}

	var entryIndex = -1
	for name, entry := range module.Export.Entries {
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

func (w *WasmVmService) Start(node *node.Node) error {
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
	w.module = m
	return nil
}

func (w *WasmVmService) Stop() error {
	return nil
}

func (w *WasmVmService) Register(funcName string, nativeFunc interface{}) error {
	w.lock.RLock()
	defer w.lock.RUnlock()
	rfunc := reflect.TypeOf(nativeFunc)
	if rfunc.Kind() != reflect.Func {
		return errors.New(fmt.Sprintf("%s is not a function", funcName))
	}
	// func should be func(proc *exec.Process, ... interface{})
	if rfunc.NumIn() < 1 {
		return errors.New(fmt.Sprintf("function signature of %s is wrong", funcName))
	}
	funcSig, err := w.exactFuncSig(rfunc)
	if err != nil {
		return err
	}
	function := wasm.Function{Sig: &funcSig, Host: reflect.ValueOf(nativeFunc), Body: &wasm.FunctionBody{}}
	w.nativeFuncName = append(w.nativeFuncName, funcName)
	w.nativeFuncSigs = append(w.nativeFuncSigs, funcSig)
	w.nativeFuncs = append(w.nativeFuncs, function)
	return nil
}

func (w *WasmVmService) exactFuncSig(p reflect.Type) (wasm.FunctionSig, error) {
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
