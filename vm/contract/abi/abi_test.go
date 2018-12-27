package abi

import (
	"fmt"
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

	r.Equal(4, abi.TypesCount())
	for i := 0; i < 4; i++ {
		r.NotNil(abi.TypeByIndex(i))
	}
	r.NotNil(abi.TypeByName("timestamp_t"))
	r.NotNil(abi.TypeByName("hi"))
	r.NotNil(abi.TypeByName("greeting"))
	r.NotNil(abi.TypeByName("stats"))

	r.Equal(1, abi.MethodsCount())
	for i := 0; i < 1; i++ {
		r.NotNil(abi.MethodByIndex(i))
	}
	r.NotNil(abi.MethodByName("hi"))

	r.Equal(3, abi.TablesCount())
	for i := 0; i < 3; i++ {
		r.NotNil(abi.TableByIndex(i))
	}
	r.NotNil(abi.TableByName("table_greetings"))
	r.NotNil(abi.TableByName("hello"))
	r.NotNil(abi.TableByName("global_counters"))

	jsonStr := `[[1],300,400]`
	data, err = vme.EncodeFromJson([]byte(jsonStr), abi.TypeByName("stats").Type())
	r.NoError(err)
	data, err = vme.DecodeToJson(data, abi.TypeByName("stats").Type(), true)
	r.NoError(err)
	fmt.Println(string(data))
	r.Equal(jsonStr, string(data))
}
