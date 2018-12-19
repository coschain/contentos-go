package vm

import (
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/coschain/contentos-go/prototype"
	"github.com/inconshreveable/log15"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

const (
	dbPath = "/tmp/cos.db"
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
	myassert := assert.New(t)
	data, _ := ioutil.ReadFile(wasmFile)
	context := Context{Code: data}
	vm := NewCosVM(&context, nil, nil, log15.New())
	ret, _ := vm.Run()
	myassert.Equal(ret, uint32(0))
}

func TestCosVM_Props(t *testing.T) {
	wasmFile := "./testdata/props.wasm"
	myassert := assert.New(t)
	data, _ := ioutil.ReadFile(wasmFile)
	context := Context{Code: data}
	props := &prototype.DynamicProperties{CurrentWitness: &prototype.AccountName{Value: "initminer"}, HeadBlockNumber: 1,
		Time: &prototype.TimePointSec{UtcSeconds: 42}}
	vm := NewCosVM(&context, nil, props, log15.New())
	ret, _ := vm.Run()
	myassert.Equal(ret, uint32(0))
}

func TestCosVM_CosAssert(t *testing.T) {
	wasmFile := "./testdata/props.wasm"
	myassert := assert.New(t)
	data, _ := ioutil.ReadFile(wasmFile)
	context := Context{Code: data}
	vm := NewCosVM(&context, nil, nil, log15.New())
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
	context := Context{Code: data}
	vm := NewCosVM(&context, db, nil, log15.New())
	ret, _ := vm.Run()
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
	context := Context{Code: data, Caller: &prototype.AccountName{Value: "alice"},
		Owner: &prototype.AccountName{Value: "initminer"}, Amount: &prototype.Coin{Value: 100}}
	vm := NewCosVM(&context, db, nil, log15.New())
	ret, _ := vm.Run()
	myassert.Equal(ret, uint32(0))
}
