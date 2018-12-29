package vm

import (
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/vm/context"
	"github.com/go-interpreter/wagon/exec"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"math"
	"os"
	"testing"
)

const (
	dbPath = "/tmp/cos.db"
)

type FakeInjector struct {
	db iservices.IDatabaseService
}

func (i *FakeInjector) Error(code uint32, msg string) {
	fmt.Println("error:", msg)
}

func (i *FakeInjector) Log(msg string) {
	fmt.Println("info:", msg)
}

func (i *FakeInjector) RequireAuth(name string) error {
	fmt.Println("require auth:", name)
	return nil
}

func (i *FakeInjector) DeductGasFee(caller string, spent uint64) {
	fmt.Println("deduct gas fee", caller, spent)
}

func (i *FakeInjector) ContractTransfer(contract, owner, to string, amount uint64) {
	c := table.NewSoContractWrap(i.db, &prototype.ContractId{Owner: &prototype.AccountName{Value: owner}, Cname: contract})
	balance := c.GetBalance().Value
	if balance < amount {
		panic(fmt.Sprintf("Endanger Transfer Operation: %s, %s, %s, %d", contract, owner, to, amount))
	}
	acc := table.NewSoAccountWrap(i.db, &prototype.AccountName{Value: to})
	// need atomic ?
	c.MdBalance(&prototype.Coin{Value: balance - amount})
	acc.MdBalance(&prototype.Coin{Value: acc.GetBalance().Value + amount})
}

func (i *FakeInjector) UserTransfer(from, contract, owner string, amount uint64) {
	fmt.Println("user transfer", from, contract, owner, amount)
}

func fadd(proc *exec.Process, a, b float32) float32 {
	return a + b
}

func TestCosVM_simpleAdd(t *testing.T) {
	wasmFile := "./testdata/add.wasm"
	myassert := assert.New(t)
	data, _ := ioutil.ReadFile(wasmFile)
	context := vmcontext.Context{Code: data, Gas: &prototype.Coin{Value: math.MaxUint64}}
	vm := NewCosVM(&context, nil, nil, logrus.New())
	vm.Register("add", add, 3000)
	ret, _ := vm.Run()
	myassert.Equal(ret, uint32(6))
}

func TestCosVM_copy(t *testing.T) {
	wasmFile := "./testdata/copy.wasm"
	myassert := assert.New(t)
	data, _ := ioutil.ReadFile(wasmFile)
	context := vmcontext.Context{Code: data, Gas: &prototype.Coin{Value: math.MaxUint64}, Injector: &FakeInjector{nil}}
	vm := NewCosVM(&context, nil, nil, logrus.New())
	ret, err := vm.Run()
	myassert.NoError(err)
	myassert.Equal(ret, uint32(0))
}

func TestCosVM_Memset(t *testing.T) {
	wasmFile := "./testdata/memset.wasm"
	myassert := assert.New(t)
	data, _ := ioutil.ReadFile(wasmFile)
	context := vmcontext.Context{Code: data, Gas: &prototype.Coin{Value: math.MaxUint64}, Injector: &FakeInjector{nil}}
	vm := NewCosVM(&context, nil, nil, logrus.New())
	ret, err := vm.Run()
	myassert.NoError(err)
	myassert.Equal(ret, uint32(0))
}

func TestCosVM_ValidateFloat(t *testing.T) {
	wasmFile := "./testdata/float.wasm"
	data, _ := ioutil.ReadFile(wasmFile)
	myassert := assert.New(t)
	context := vmcontext.Context{Code: data, Gas: &prototype.Coin{Value: math.MaxUint64}}
	vm := NewCosVM(&context, nil, nil, logrus.New())
	vm.Register("add", fadd, 3000)
	err := vm.Validate()
	myassert.Error(err)
}

func TestCosVM_ValidateFloatGlobal(t *testing.T) {
	wasmFile := "./testdata/float_global.wasm"
	data, _ := ioutil.ReadFile(wasmFile)
	myassert := assert.New(t)
	context := vmcontext.Context{Code: data, Gas: &prototype.Coin{Value: math.MaxUint64}}
	vm := NewCosVM(&context, nil, nil, logrus.New())
	err := vm.Validate()
	myassert.Error(err)
}

func TestCosVm_NotExportFunc(t *testing.T) {
	wasmFile := "./testdata/notexport.wasm"
	data, _ := ioutil.ReadFile(wasmFile)
	myassert := assert.New(t)
	context := vmcontext.Context{Code: data, Gas: &prototype.Coin{Value: math.MaxUint64}}
	vm := NewCosVM(&context, nil, nil, logrus.New())
	err := vm.Validate()
	myassert.Error(err)
}

func TestCosVM_Print(t *testing.T) {
	wasmFile := "./testdata/print.wasm"
	myassert := assert.New(t)
	data, _ := ioutil.ReadFile(wasmFile)
	context := vmcontext.Context{Code: data, Gas: &prototype.Coin{Value: math.MaxUint64}, Injector: &FakeInjector{nil}}
	vm := NewCosVM(&context, nil, nil, logrus.New())
	ret, err := vm.Run()
	myassert.NoError(err)
	myassert.Equal(ret, uint32(0))
}

func TestCosVM_SpentGas(t *testing.T) {
	wasmFile := "./testdata/add.wasm"
	myassert := assert.New(t)
	data, _ := ioutil.ReadFile(wasmFile)
	context := vmcontext.Context{Code: data, Gas: &prototype.Coin{Value: math.MaxUint64}, Injector: &FakeInjector{nil}}
	vm := NewCosVM(&context, nil, nil, logrus.New())
	vm.Register("add", add, 200)
	_, _ = vm.Run()
	gas := vm.SpentGas()
	myassert.True(gas > 200)
}

func TestCosVM_Sha256(t *testing.T) {
	wasmFile := "./testdata/sha256.wasm"
	myassert := assert.New(t)
	data, _ := ioutil.ReadFile(wasmFile)
	context := vmcontext.Context{Code: data, Gas: &prototype.Coin{Value: math.MaxUint64}, Injector: &FakeInjector{nil}}
	vm := NewCosVM(&context, nil, nil, logrus.New())
	ret, err := vm.Run()
	myassert.NoError(err)
	myassert.Equal(ret, uint32(0))
}

func TestCosVM_Props(t *testing.T) {
	wasmFile := "./testdata/props.wasm"
	myassert := assert.New(t)
	data, _ := ioutil.ReadFile(wasmFile)
	context := vmcontext.Context{Code: data, Gas: &prototype.Coin{Value: math.MaxUint64}, Injector: &FakeInjector{nil}}
	props := &prototype.DynamicProperties{CurrentWitness: &prototype.AccountName{Value: "initminer"}, HeadBlockNumber: 1,
		Time: &prototype.TimePointSec{UtcSeconds: 42}}
	vm := NewCosVM(&context, nil, props, logrus.New())
	ret, err := vm.Run()
	myassert.NoError(err)
	myassert.Equal(ret, uint32(0))
}

func TestCosVM_CosAssert(t *testing.T) {
	wasmFile := "./testdata/props.wasm"
	myassert := assert.New(t)
	data, _ := ioutil.ReadFile(wasmFile)
	context := vmcontext.Context{Code: data, Gas: &prototype.Coin{Value: math.MaxUint64}}
	vm := NewCosVM(&context, nil, nil, logrus.New())
	ret, _ := vm.Run()
	myassert.Equal(ret, uint32(1))
}

func TestCosVM_RWStorage(t *testing.T) {
	db, err := storage.NewDatabase(dbPath)
	defer func() {
		_ = db.Stop()
		_ = os.RemoveAll(dbPath)
	}()
	if err != nil {
		t.Error(err)
	}
	err = db.Start(nil)
	if err != nil {
		t.Error(err)
	}
	wasmFile := "./testdata/rwstorage.wasm"
	myassert := assert.New(t)
	data, _ := ioutil.ReadFile(wasmFile)
	context := vmcontext.Context{Code: data, Gas: &prototype.Coin{Value: math.MaxUint64}, Injector: &FakeInjector{nil}}
	vm := NewCosVM(&context, db, nil, logrus.New())
	ret, err := vm.Run()
	myassert.NoError(err)
	myassert.Equal(ret, uint32(0))
}

func TestCosVM_Contract(t *testing.T) {
	db, err := storage.NewDatabase(dbPath)
	defer func() {
		_ = db.Stop()
		_ = os.RemoveAll(dbPath)
	}()
	if err != nil {
		t.Error(err)
	}
	err = db.Start(nil)
	if err != nil {
		t.Error(err)
	}
	name := &prototype.AccountName{Value: "initminer"}
	newAccountWrap := table.NewSoAccountWrap(db, name)
	err = newAccountWrap.Create(func(tInfo *table.SoAccount) {
		tInfo.Name = name
		tInfo.CreatedTime = &prototype.TimePointSec{UtcSeconds: 0}
		tInfo.Balance = prototype.NewCoin(5000)
		tInfo.VestingShares = prototype.NewVest(1000)
		tInfo.LastPostTime = &prototype.TimePointSec{UtcSeconds: 0}
		tInfo.LastVoteTime = &prototype.TimePointSec{UtcSeconds: 0}
	})

	contract := table.NewSoContractWrap(db, &prototype.ContractId{Owner: &prototype.AccountName{Value: "initminer"}, Cname: "hello"})
	err = contract.Create(func(tInfo *table.SoContract) {
		tInfo.CreatedTime = &prototype.TimePointSec{UtcSeconds: 0}
		tInfo.Balance = &prototype.Coin{Value: 2000}
	})
	if err != nil {
		t.Error(err)
	}
	wasmFile := "./testdata/contract_data.wasm"
	myassert := assert.New(t)
	data, _ := ioutil.ReadFile(wasmFile)
	context := vmcontext.Context{Code: data, Caller: &prototype.AccountName{Value: "alice"}, Contract: "hello",
		Owner: &prototype.AccountName{Value: "initminer"}, Amount: &prototype.Coin{Value: 100}, Gas: &prototype.Coin{Value: math.MaxUint64},
		Injector: &FakeInjector{nil}}
	vm := NewCosVM(&context, db, nil, logrus.New())
	ret, err := vm.Run()
	myassert.NoError(err)
	myassert.Equal(ret, uint32(0))
}

func TestCosVM_ContractTransfer(t *testing.T) {
	db, err := storage.NewDatabase(dbPath)
	defer func() {
		_ = db.Stop()
		_ = os.RemoveAll(dbPath)
	}()
	if err != nil {
		t.Error(err)
	}
	err = db.Start(nil)
	if err != nil {
		t.Error(err)
	}
	name := &prototype.AccountName{Value: "initminer"}
	newAccountWrap := table.NewSoAccountWrap(db, name)
	err = newAccountWrap.Create(func(tInfo *table.SoAccount) {
		tInfo.Name = name
		tInfo.CreatedTime = &prototype.TimePointSec{UtcSeconds: 0}
		tInfo.Balance = prototype.NewCoin(5000)
		tInfo.VestingShares = prototype.NewVest(1000)
		tInfo.LastPostTime = &prototype.TimePointSec{UtcSeconds: 0}
		tInfo.LastVoteTime = &prototype.TimePointSec{UtcSeconds: 0}
	})

	alice := &prototype.AccountName{Value: "alice"}
	aliceWrap := table.NewSoAccountWrap(db, alice)
	err = aliceWrap.Create(func(tInfo *table.SoAccount) {
		tInfo.Name = alice
		tInfo.CreatedTime = &prototype.TimePointSec{UtcSeconds: 0}
		tInfo.Balance = prototype.NewCoin(0)
		tInfo.VestingShares = prototype.NewVest(0)
		tInfo.LastPostTime = &prototype.TimePointSec{UtcSeconds: 0}
		tInfo.LastVoteTime = &prototype.TimePointSec{UtcSeconds: 0}
	})

	contract := table.NewSoContractWrap(db, &prototype.ContractId{Owner: &prototype.AccountName{Value: "initminer"}, Cname: "hello"})
	err = contract.Create(func(tInfo *table.SoContract) {
		tInfo.CreatedTime = &prototype.TimePointSec{UtcSeconds: 0}
		tInfo.Balance = &prototype.Coin{Value: 2000}
	})
	if err != nil {
		t.Error(err)
	}

	wasmFile := "./testdata/contract_transfer.wasm"
	myassert := assert.New(t)
	data, _ := ioutil.ReadFile(wasmFile)
	context := vmcontext.Context{Code: data, Caller: &prototype.AccountName{Value: "alice"},
		Owner: &prototype.AccountName{Value: "initminer"}, Amount: &prototype.Coin{Value: 100},
		Gas: &prototype.Coin{Value: math.MaxUint64}, Contract: "hello",
		Injector: &FakeInjector{db: db}}
	vm := NewCosVM(&context, db, nil, logrus.New())
	ret, err := vm.Run()
	myassert.NoError(err)
	myassert.Equal(ret, uint32(0))
}

//func TestCosVM_OperateWithFund(t *testing.T) {
//	db, err := storage.NewDatabase(dbPath)
//	defer func() {
//		_ = db.Stop()
//		_ = os.RemoveAll(dbPath)
//	}()
//	if err != nil {
//		t.Error(err)
//	}
//	err = db.Start(nil)
//	if err != nil {
//		t.Error(err)
//	}
//	name := &prototype.AccountName{Value: "initminer"}
//	newAccountWrap := table.NewSoAccountWrap(db, name)
//	err = newAccountWrap.Create(func(tInfo *table.SoAccount) {
//		tInfo.Name = name
//		tInfo.CreatedTime = &prototype.TimePointSec{UtcSeconds: 0}
//		tInfo.Balance = prototype.NewCoin(100000)
//		tInfo.VestingShares = prototype.NewVest(1000)
//		tInfo.LastPostTime = &prototype.TimePointSec{UtcSeconds: 0}
//		tInfo.LastVoteTime = &prototype.TimePointSec{UtcSeconds: 0}
//	})
//
//	contract := table.NewSoContractWrap(db, &prototype.ContractId{Owner: &prototype.AccountName{Value: "initminer"}, Cname: "hello"})
//	err = contract.Create(func(tInfo *table.SoContract) {
//		tInfo.CreatedTime = &prototype.TimePointSec{UtcSeconds: 0}
//		tInfo.Balance = &prototype.Coin{Value: 2000}
//	})
//	if err != nil {
//		t.Error(err)
//	}
//	wasmFile := "./testdata/contract_data.wasm"
//	myassert := assert.New(t)
//	data, _ := ioutil.ReadFile(wasmFile)
//	context := vmcontext.Context{Code: data, Caller: &prototype.AccountName{Value: "alice"}, Contract: "hello",
//		Owner: &prototype.AccountName{Value: "initminer"}, Amount: &prototype.Coin{Value: 100}, Gas: &prototype.Coin{Value: math.MaxUint64},
//		Injector: &FakeInjector{nil}}
//	vm := NewCosVM(&context, db, nil, logrus.New())
//	ret, err := vm.Run()
//	myassert.NoError(err)
//	myassert.Equal(ret, uint32(0))
//}

// todo
// require_auth
