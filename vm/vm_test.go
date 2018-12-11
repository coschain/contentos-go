package vm

import (
	"github.com/go-interpreter/wagon/exec"
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
	vm.Register("add", add)
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
	vm.Register("add", add)
	vm.Register("mul", mul)
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

func TestContext_Sha256(t *testing.T) {
	wasmFile := "./testdata/sha256.wasm"
	//myassert := assert.New(t)
	data, err := ioutil.ReadFile(wasmFile)
	if err != nil {
		t.Error(err)
	}
	vm, err := New(nil)
	if err != nil {
		t.Error(err)
	}
	err = vm.Start(nil)
	if err != nil {
		t.Error(err)
	}
	ctx := &Context{Code: data}
	_, err = vm.Run(ctx)
	if err != nil {
		t.Error(err)
	}
}
