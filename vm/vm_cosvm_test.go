package vm

import (
	"fmt"
	"github.com/inconshreveable/log15"
	"io/ioutil"
	"testing"
)

func TestCosVM_simpleAdd(t *testing.T) {
	wasmFile := "./testdata/add.wasm"
	data, _ := ioutil.ReadFile(wasmFile)
	context := Context{Code: data}
	vm := NewCosVM(&context, nil, nil, log15.New())
	vm.Register("add", add)
	ret, _ := vm.Run()
	fmt.Println(ret)
}
