package vm

import (
	"github.com/inconshreveable/log15"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestCosVM_simpleAdd(t *testing.T) {
	wasmFile := "./testdata/add.wasm"
	myassert := assert.New(t)
	data, _ := ioutil.ReadFile(wasmFile)
	context := Context{Code: data}
	vm := NewCosVM(&context, nil, nil, log15.New())
	vm.Register("add", add)
	ret, _ := vm.Run()
	myassert.Equal(ret, 6)
}

func TestCosVM_readBytes1(t *testing.T) {
	wasmFile := "./testdata/read.wasm"
	myassert := assert.New(t)
	data, _ := ioutil.ReadFile(wasmFile)
	context := Context{Code: data}
	vm := NewCosVM(&context, nil, nil, log15.New())
	ret, _ := vm.Run()
	myassert.Equal(ret, 11)
}

func TestCosVM_readBytes2(t *testing.T) {
	wasmFile := "./testdata/read2.wasm"
	myassert := assert.New(t)
	data, _ := ioutil.ReadFile(wasmFile)
	context := Context{Code: data}
	vm := NewCosVM(&context, nil, nil, log15.New())
	ret, _ := vm.Run()
	myassert.Equal(ret, 11)
}

func TestCosVM_readBytes3(t *testing.T) {
	wasmFile := "./testdata/read3.wasm"
	myassert := assert.New(t)
	data, _ := ioutil.ReadFile(wasmFile)
	context := Context{Code: data}
	vm := NewCosVM(&context, nil, nil, log15.New())
	ret, _ := vm.Run()
	myassert.Equal(ret, 5)
}

func TestCosVm_writeByte1(t *testing.T) {
	wasmFile := "./testdata/write1.wasm"
	myassert := assert.New(t)
	data, _ := ioutil.ReadFile(wasmFile)
	context := Context{Code: data}
	vm := NewCosVM(&context, nil, nil, log15.New())
	ret, _ := vm.Run()
	myassert.Equal(ret, 12) // because add \0
}

func TestCosVm_writeByte2(t *testing.T) {
	wasmFile := "./testdata/write2.wasm"
	myassert := assert.New(t)
	data, _ := ioutil.ReadFile(wasmFile)
	context := Context{Code: data}
	vm := NewCosVM(&context, nil, nil, log15.New())
	ret, _ := vm.Run()
	myassert.Equal(ret, 6) // as above
}
