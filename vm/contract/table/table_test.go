package table

import (
	"fmt"
	"github.com/coschain/contentos-go/common/encoding/vme"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/coschain/contentos-go/vm/contract/abi"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)


const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randomString(size uint) string {
	b := make([]byte, size)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func TestContractTable(t *testing.T) {
	a := assert.New(t)

	dir, err := ioutil.TempDir("", "contract_table")
	a.NoError(err)
	defer os.RemoveAll(dir)

	fn := filepath.Join(dir, randomString(8))
	db, err := storage.NewDatabase(fn)
	a.NoError(err)
	a.NotNil(db)
	a.NoError(db.Start(nil))
	defer db.Close()

	data, err := ioutil.ReadFile("testdata/hello.abi")
	a.NoError(err)
	helloAbi, err := abi.UnmarshalABI(data)
	a.NoError(err)

	tables := NewContractTables("someone", "hello", helloAbi, db)
	testTableGreetings(a, tables.Table("table_greetings"))
	testTableHello(a, tables.Table("hello"))
	testTableGlobalCounters(a, tables.Table("global_counters"))
}

func encodedPrimaryKey(a *assert.Assertions, table *ContractTable, jsonstr string) []byte {
	data, err := vme.EncodeFromJson([]byte(jsonstr), table.abiTable.Record().Field(table.abiTable.PrimaryIndex()).Type().Type())
	a.NoError(err)
	a.NotNil(data)
	return data
}

func encodedRecord(a *assert.Assertions, table *ContractTable, jsonstr string) []byte {
	data, err := vme.EncodeFromJson([]byte(jsonstr), table.abiTable.Record().Type())
	a.NoError(err)
	a.NotNil(data)
	return data
}

func testTableGreetings(a *assert.Assertions, table *ContractTable) {
	a.NotNil(table)

	for i := 0; i < 1000; i++ {
		jsonStr := fmt.Sprintf(`["account%d",%d,%d]`, i, i, i * 2)
		a.NoError(table.NewRecord(encodedRecord(a, table, jsonStr)))
	}
}

func testTableHello(a *assert.Assertions, table *ContractTable) {
	a.NotNil(table)
}

func testTableGlobalCounters(a *assert.Assertions, table *ContractTable) {
	a.NotNil(table)
}
