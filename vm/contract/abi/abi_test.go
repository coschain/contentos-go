package abi

import (
	"github.com/coschain/contentos-go/common/encoding/vme"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"math/rand"
	"reflect"
	"testing"
	"time"
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

	r.Equal(9, abi.TypesCount())
	for i := 0; i < 9; i++ {
		r.NotNil(abi.TypeByIndex(i))
	}
	r.NotNil(abi.TypeByName("timestamp_t"))
	r.NotNil(abi.TypeByName("hi"))
	r.NotNil(abi.TypeByName("greeting"))
	r.NotNil(abi.TypeByName("stats"))
	r.NotNil(abi.TypeByName("str"))
	r.NotNil(abi.TypeByName("test_array"))
	r.NotNil(abi.TypeByName("int2str"))
	r.NotNil(abi.TypeByName("mymap"))
	r.NotNil(abi.TypeByName("test_map"))

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

	jsonStr = `[[1,"one"],[2,"two"],[3,"three"]]`
	data, err = vme.EncodeFromJson([]byte(jsonStr), abi.TypeByName("int2str").Type())
	r.NoError(err)
	data, err = vme.DecodeToJson(data, abi.TypeByName("int2str").Type(), true)
	r.NoError(err)
	r.Equal(jsonStr, string(data))

	jsonStr = `[100,[["key1",[456,["nice","to","meet","you"]]],["key2",[123,["hello","world"]]]]]`
	data, err = vme.EncodeFromJson([]byte(jsonStr), abi.TypeByName("test_map").Type())
	r.NoError(err)
	data, err = vme.DecodeToJson(data, abi.TypeByName("test_map").Type(), true)
	r.NoError(err)
	r.Equal(jsonStr, string(data))
}

func TestAbi2(t *testing.T) {
	r := assert.New(t)

	data, _ := ioutil.ReadFile("testdata/test.abi")
	abi, err := UnmarshalABI(data)
	r.NoError(err, "Unmarshal failed.")

	jsonStr := `[[[["alice",20,false,[["maths",10],["chemistry",5]]],"swim"]]]`
	data, err = vme.EncodeFromJson([]byte(jsonStr), abi.TypeByName("test_arg").Type())
	r.NoError(err)
	data, err = vme.DecodeToJson(data, abi.TypeByName("test_arg").Type(), true)
	r.NoError(err)
	r.Equal(jsonStr, string(data))
}

func TestUint64Json(t *testing.T) {
	var (
		data []byte
		err error
	)
	a := assert.New(t)
	uint64Type := reflect.TypeOf(uint64(0))
	rand.Seed(time.Now().UnixNano())
	// try a few random uint64s
	for i := 0; i < 5; i++ {
		// generate a random decimal with 19 significands,
		// which is a uint64 and can't be represented by float64.
		digits := ""
		for j := 0; j < 19; j++ {
			digits += string('0' + rand.Intn(9) + 1)
		}
		// json -> vme encoded
		data, err = vme.EncodeFromJson([]byte(digits), uint64Type)
		a.NoError(err)

		// vme encoded -> json
		data, err = vme.DecodeToJson(data, uint64Type, true)
		a.NoError(err)

		// must be precisely equal
		a.Equal(digits, string(data))
	}
}
