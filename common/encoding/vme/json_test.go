package vme

import (
	"reflect"
	"testing"
)

func TestEncodeJsonArray(t *testing.T) {
	_, err := EncodeJsonArray("[3, 3.4, false, \"hello\", [2,4,5], [\"abc\",\"de\"]]", []reflect.Type{
		reflect.TypeOf(int32(0)),
		reflect.TypeOf(uint64(0)),
		reflect.TypeOf(false),
		reflect.TypeOf(""),
		reflect.TypeOf([]int16{}),
		reflect.TypeOf([][]byte{}),
	})
	if err != nil {
		t.Fatal(err)
	}
}
