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
	"reflect"
	"strings"
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

func decodeRecord(a *assert.Assertions, table *ContractTable, data []byte) string {
	data, err := vme.DecodeToJson(data, table.abiTable.Record().Type(), true)
	a.NoError(err)
	a.NotNil(data)
	return string(data)
}

func testTableGreetings(a *assert.Assertions, table *ContractTable) {
	a.NotNil(table)

	var (
		data []byte
		err error
	)

	// normal insertions
	for i := 0; i < 100; i++ {
		jsonStr := fmt.Sprintf(`["account%d",%d,%d]`, i, i, i * 2)
		a.NoError(table.NewRecord(encodedRecord(a, table, jsonStr)))
	}

	// duplicate primary key
	a.Error(table.NewRecord(encodedRecord(a, table, `["account7",102,233]`)))

	// normal query
	data, err = table.GetRecord(encodedPrimaryKey(a, table, `"account10"`))
	a.NoError(err)
	a.NotNil(data)
	a.Equal(`["account10",10,20]`, decodeRecord(a, table, data))

	// query non-existent records
	data, err = table.GetRecord(encodedPrimaryKey(a, table, `"sldkfjs"`))
	a.Error(err)

	// updates
	a.NoError(table.UpdateRecord(
		encodedPrimaryKey(a, table, `"account40"`),
		encodedRecord(a, table, `["account40",40000,80000]`),
		))

	// update non-existent records
	a.Error(table.UpdateRecord(
		encodedPrimaryKey(a, table, `"sldkfjs"`),
		encodedRecord(a, table, `["sldkfjs",40000,80000]`),
	))

	// range scan by secondary index
	var result []string
	a.Equal(5,
		table.EnumRecords("count", 20, nil, true, 5, func(r interface{})bool {
			result = append(result, reflect.ValueOf(r).Field(table.abiTable.PrimaryIndex()).String())
			return true
		} ))
	a.Equal("account40,account99,account98,account97,account96", strings.Join(result, ","))

	// range scan by primary key
	result = result[:0]
	a.Equal(3,
		table.EnumRecords("name", "account60", "account63", false, 5, func(r interface{})bool {
			result = append(result, reflect.ValueOf(r).Field(table.abiTable.PrimaryIndex()).String())
			return true
		} ))
	a.Equal("account60,account61,account62", strings.Join(result, ","))

	// range scan by invalid field "xxxx"
	a.Equal(0,
		table.EnumRecords("xxxx", nil, nil, false, 5, func(r interface{})bool {
			return true
		} ))

	// delete
	a.NoError(table.DeleteRecord(encodedPrimaryKey(a, table, `"account55"`)))
	_, err = table.GetRecord(encodedPrimaryKey(a, table, `"account55"`))
	a.Error(err)
	a.Equal(8,
		table.EnumRecords("name", "account50", "account59", false, 100, func(r interface{})bool {
			return true
		} ))

	// deleting non-existent records must return success.
	a.NoError(table.DeleteRecord(encodedPrimaryKey(a, table, `"sdfsdfsdf"`)))
}
