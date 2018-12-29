package abi

import (
	"github.com/coschain/contentos-go/common/encoding/vme"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)


func TestAbi(t *testing.T) {
	r := assert.New(t)

	data, _ := ioutil.ReadFile("testdata/hello.abi")
	abi, err := UnmarshalABI(data)
	r.NoError(err, "Unmarshal failed.")

	data, err = abi.Marshal()
	r.NoError(err, "Marshal failed.")

	for i := 0; i < 100; i++ {
		x, err := UnmarshalABI(data)
		r.NoError(err, "Unmarshal failed.")
		r.Equal(abi.TypesCount(), x.TypesCount())
		r.Equal(abi.MethodsCount(), x.MethodsCount())
		r.Equal(abi.TablesCount(), x.TablesCount())

		data, err = x.Marshal()
		r.NoError(err, "Marshal failed.")
	}

	r.Equal(6, abi.TypesCount())
	for i := 0; i < 6; i++ {
		r.NotNil(abi.TypeByIndex(i))
	}
	r.NotNil(abi.TypeByName("timestamp_t"))
	r.NotNil(abi.TypeByName("hi"))
	r.NotNil(abi.TypeByName("greeting"))
	r.NotNil(abi.TypeByName("stats"))
	r.NotNil(abi.TypeByName("str"))
	r.NotNil(abi.TypeByName("test_array"))

	r.Equal(1, abi.MethodsCount())
	for i := 0; i < 1; i++ {
		r.NotNil(abi.MethodByIndex(i))
	}
	r.NotNil(abi.MethodByName("hi"))

	r.Equal(4, abi.TablesCount())
	for i := 0; i < 4; i++ {
		r.NotNil(abi.TableByIndex(i))
	}
	r.NotNil(abi.TableByName("table_greetings"))
	r.NotNil(abi.TableByName("hello"))
	r.NotNil(abi.TableByName("global_counters"))
	r.NotNil(abi.TableByName("table_test_array"))

	jsonStr := `[[1],300,400]`
	data, err = vme.EncodeFromJson([]byte(jsonStr), abi.TypeByName("stats").Type())
	r.NoError(err)
	data, err = vme.DecodeToJson(data, abi.TypeByName("stats").Type(), true)
	r.NoError(err)
	r.Equal(jsonStr, string(data))

	jsonStr = `[456,["nice","to","meet","you"]]`
	data, err = vme.EncodeFromJson([]byte(jsonStr), abi.TypeByName("test_array").Type())
	r.NoError(err)
	data, err = vme.DecodeToJson(data, abi.TypeByName("test_array").Type(), true)
	r.NoError(err)
	r.Equal(jsonStr, string(data))
}
