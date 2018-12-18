package vm

import (
	"github.com/coschain/contentos-go/prototype"
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
	myassert.Equal(ret, uint32(6))
}

func TestCosVM_readBytes1(t *testing.T) {
	wasmFile := "./testdata/read.wasm"
	myassert := assert.New(t)
	data, _ := ioutil.ReadFile(wasmFile)
	context := Context{Code: data}
	vm := NewCosVM(&context, nil, nil, log15.New())
	ret, _ := vm.Run()
	myassert.Equal(ret, uint32(11))
}

func TestCosVM_readBytes2(t *testing.T) {
	wasmFile := "./testdata/read2.wasm"
	myassert := assert.New(t)
	data, _ := ioutil.ReadFile(wasmFile)
	context := Context{Code: data}
	vm := NewCosVM(&context, nil, nil, log15.New())
	ret, _ := vm.Run()
	myassert.Equal(ret, uint32(11))
}

func TestCosVM_readBytes3(t *testing.T) {
	wasmFile := "./testdata/read3.wasm"
	myassert := assert.New(t)
	data, _ := ioutil.ReadFile(wasmFile)
	context := Context{Code: data}
	vm := NewCosVM(&context, nil, nil, log15.New())
	ret, _ := vm.Run()
	myassert.Equal(ret, uint32(5))
}

func TestCosVm_writeByte1(t *testing.T) {
	wasmFile := "./testdata/write1.wasm"
	myassert := assert.New(t)
	data, _ := ioutil.ReadFile(wasmFile)
	context := Context{Code: data}
	vm := NewCosVM(&context, nil, nil, log15.New())
	ret, _ := vm.Run()
	myassert.Equal(ret, uint32(11))
}

func TestCosVm_writeByte2(t *testing.T) {
	wasmFile := "./testdata/write2.wasm"
	myassert := assert.New(t)
	data, _ := ioutil.ReadFile(wasmFile)
	context := Context{Code: data}
	vm := NewCosVM(&context, nil, nil, log15.New())
	ret, _ := vm.Run()
	myassert.Equal(ret, uint32(5))
}

func TestCosVM_Print(t *testing.T) {
	wasmFile := "./testdata/print.wasm"
	myassert := assert.New(t)
	data, _ := ioutil.ReadFile(wasmFile)
	context := Context{Code: data}
	vm := NewCosVM(&context, nil, nil, log15.New())
	ret, _ := vm.Run()
	// assert no error
	myassert.Equal(ret, uint32(0))
}

func TestCosVM_Sha256(t *testing.T) {
	wasmFile := "./testdata/sha256.wasm"
	//myassert := assert.New(t)
	data, _ := ioutil.ReadFile(wasmFile)
	context := Context{Code: data}
	vm := NewCosVM(&context, nil, nil, log15.New())
	_, _ = vm.Run()
	// assert no error
	//myassert.Equal(ret, uint32(0));
	//out := sha256.Sum256([]byte("hello world"))
	//o := base64.StdEncoding.EncodeToString(out[:])
	// uU0nuZNNPgilLlLX2n2r+sSE7+N6U4DukIj3rOLvzek=
	//fmt.Println(o)
}

func TestCosVM_Props(t *testing.T) {
	wasmFile := "./testdata/props.wasm"
	data, _ := ioutil.ReadFile(wasmFile)
	context := Context{Code: data}
	props := &prototype.DynamicProperties{CurrentWitness: &prototype.AccountName{Value: "initminer"}, HeadBlockNumber: 1,
		Time: &prototype.TimePointSec{UtcSeconds: 42}}
	vm := NewCosVM(&context, nil, props, log15.New())
	_, _ = vm.Run()
}
