package vm

import (
	"github.com/go-interpreter/wagon/exec"
	"github.com/go-interpreter/wagon/wasm"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func add(proc *exec.Process, a, b int32) int32 {
	return a + b
}

func mul(proc *exec.Process, a, b int32) int32 {
	return a * b
}

func TestContext_Register(t *testing.T) {
	myassert := assert.New(t)
	vm, err := New(nil)
	if err != nil {
		t.Error(err)
	}
	err = vm.Register("add", add)
	if err != nil {
		t.Error(err)
	}
	myassert.Equal(vm.nativeFuncName[0], "add")
	myassert.Equal(vm.nativeFuncSigs[0].ParamTypes, []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32})
}

func TestContext_Run(t *testing.T) {
	wasmFile := "./testdata/add.wasm"
	myassert := assert.New(t)
	data, err := ioutil.ReadFile(wasmFile)
	if err != nil {
		t.Error(err)
	}
	vm, err := New(nil)
	if err != nil {
		t.Error(err)
	}
	err = vm.Register("add", add)
	if err != nil {
		t.Error(err)
	}
	err = vm.Start(nil)
	if err != nil {
		t.Error(err)
	}
	ctx := &Context{Code: data}
	ret, err := vm.Run(ctx)
	if err != nil {
		t.Error(err)
	}
	myassert.Equal(ret, uint32(6))
}

func TestContext_Run2(t *testing.T) {
	wasmFile := "./testdata/add_mul.wasm"
	myassert := assert.New(t)
	data, err := ioutil.ReadFile(wasmFile)
	if err != nil {
		t.Error(err)
	}
	vm, err := New(nil)
	if err != nil {
		t.Error(err)
	}
	_ = vm.Register("add", add)
	_ = vm.Register("mul", mul)
	err = vm.Start(nil)
	if err != nil {
		t.Error(err)
	}
	ctx := &Context{Code: data}
	ret, err := vm.Run(ctx)
	if err != nil {
		t.Error(err)
	}
	myassert.Equal(ret, uint32(12))
}
