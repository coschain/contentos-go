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
	for i := 0; i < 1000; i++ {
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

func unmarshalTestAbi(abiFile string) error {
	data, _ := ioutil.ReadFile("testdata/" + abiFile)
	_, err := UnmarshalABI(data)
	return err
}

func TestInvalidAbis(t *testing.T) {
	a := assert.New(t)

	// version is a must
	a.Error(unmarshalTestAbi("no_version.abi"))

	// type of a map key is not comparable
	a.Error(unmarshalTestAbi("map_key_noncmp.abi"))

	// typedef: cyclic reference
	a.Error(unmarshalTestAbi("typedef_cyclic.abi"))

	// typedef: unknown old type
	a.Error(unmarshalTestAbi("typedef_unknown_type.abi"))

	// typedef: new type can't be an array nor a map
	a.Error(unmarshalTestAbi("typedef_newtype_array.abi"))
	a.Error(unmarshalTestAbi("typedef_newtype_map.abi"))

	// struct: unknown member type
	a.Error(unmarshalTestAbi("struct_member_unknown_type.abi"))

	// struct: duplicate member names
	a.Error(unmarshalTestAbi("struct_member_dup.abi"))

	// struct: cyclic reference of member type
	a.Error(unmarshalTestAbi("struct_member_type_cyclic.abi"))

	// struct: unknown base type
	a.Error(unmarshalTestAbi("struct_base_unknown_type.abi"))

	// struct: non-inheritable base type
	a.Error(unmarshalTestAbi("struct_base_non_inheritable.abi"))

	// struct: cyclic reference of base type
	a.Error(unmarshalTestAbi("struct_base_type_cyclic.abi"))

	// method: unknown arg type
	a.Error(unmarshalTestAbi("method_arg_unknown_type.abi"))

	// method: arg type is not a struct
	a.Error(unmarshalTestAbi("method_arg_non_struct.abi"))

	// table: unknown record type
	a.Error(unmarshalTestAbi("table_record_unknown_type.abi"))

	// table: record type is not a struct
	a.Error(unmarshalTestAbi("table_record_non_struct.abi"))

	// table: primary key is not a record member
	a.Error(unmarshalTestAbi("table_primary_unknown.abi"))

	// table: primary key member is not comparable
	a.Error(unmarshalTestAbi("table_primary_noncmp.abi"))

	// table: secondary index is not a record member
	a.Error(unmarshalTestAbi("table_secondary_unknown.abi"))

	// table: secondary index member is not comparable
	a.Error(unmarshalTestAbi("table_secondary_noncmp.abi"))

	// duplicated typedef's: allowed but only the last one is taken
	a.NoError(unmarshalTestAbi("typedef_dup.abi"))
}
