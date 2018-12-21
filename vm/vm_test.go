package vm

import (
	"fmt"
	"github.com/coschain/contentos-go/vm/context"
	"github.com/go-interpreter/wagon/exec"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"reflect"
	"runtime"
	"testing"
)

func add(proc *exec.Process, a, b int32) int32 {
	return a + b
}

func mul(proc *exec.Process, a, b int32) int32 {
	return a * b
}

func iadd(a, b int32) int32 {
	return a + b
}

func imul(a, b int32) int32 {
	return a * b
}

// I don't like the way to import runtime package only for fetch the function's name
func TestCosVM_Register(t *testing.T) {
	funcname := runtime.FuncForPC(reflect.ValueOf(add).Pointer()).Name()
	fmt.Println(funcname)
}

func TestCosVM_Register2(t *testing.T) {
	a := make([]byte, 3)
	b := "abcd"
	copy(a[:], b)
	fmt.Println(len(a))
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
	vm.Register("add", add, 100)
	err = vm.Start(nil)
	if err != nil {
		t.Error(err)
	}
	ctx := &vmcontext.Context{Code: data}
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
	vm.Register("add", add, 100)
	vm.Register("mul", mul, 100)
	err = vm.Start(nil)
	if err != nil {
		t.Error(err)
	}
	ctx := &vmcontext.Context{Code: data}
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
	//err = vm.Start(nil)
	//if err != nil {
	//	t.Error(err)
	//}
	ctx := &vmcontext.Context{Code: data}
	_, err = vm.Run(ctx)
	if err != nil {
		t.Error(err)
	}
}

type M struct {
	a string
}

func NewM() *M {
	return &M{a: "hello"}
}

func (m *M) Hello(name string) string {
	return m.a + name
}

type HelloImp func(name string) string

type Inj struct {
	HelloImp
}

func NewInj(b HelloImp) *Inj {
	return &Inj{b}
}

func (i *Inj) Run() {
	fmt.Println(i.HelloImp("world"))
}

func TestContext_FuncToInterface(t *testing.T) {
	var a interface{}
	a = iadd
	value := reflect.ValueOf(a)
	b := value.Call([]reflect.Value{reflect.ValueOf(int32(2)), reflect.ValueOf(int32(3))})
	c := b[0].Interface().(int32)
	fmt.Println(c)
	fmt.Println(imul(c, int32(2)))
}

func TestContext_MethodAsFunc(t *testing.T) {
	m := NewM()
	inj := NewInj(m.Hello)
	inj.Run()
}
