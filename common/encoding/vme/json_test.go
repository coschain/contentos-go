package vme

import (
	"bytes"
	"github.com/coschain/contentos-go/common"
	"reflect"
	"testing"
)

func requireEncodeJsonError(t *testing.T, jsonStr string, typ reflect.Type) {
	_, err := EncodeFromJson([]byte(jsonStr), typ)
	if err == nil {
		t.Fatalf("encoding should fail, but succeeded. json = %v", jsonStr)
	}
}

func requireEncodeJsonOK(t *testing.T, jsonStr string, typ reflect.Type) []byte {
	data, err := EncodeFromJson([]byte(jsonStr), typ)
	if err != nil {
		t.Fatalf("encoding json should succeed, but got error = %v, json = %v", err, jsonStr)
	}
	return data
}

func requireEncodeJsonResult(t *testing.T, jsonStr string, typ reflect.Type, leBytes...byte) {
	enc := requireEncodeJsonOK(t, jsonStr, typ)
	r := leBytes
	if !common.IsLittleEndianPlatform() {
		n := len(leBytes)
		r = make([]byte, n)
		for i := range r {
			r[i] = leBytes[n - i]
		}
	}
	if bytes.Compare(r, enc) != 0 {
		t.Fatalf("encoding json result error. got %v, expecting %v", enc, r)
	}
}

func requireDecodeJsonOK(t *testing.T, typ reflect.Type, leBytes...byte) string {
	enc := leBytes
	if !common.IsLittleEndianPlatform() {
		n := len(leBytes)
		enc = make([]byte, n)
		for i := range enc {
			enc[i] = leBytes[n - i]
		}
	}
	d, err := DecodeToJson(enc, typ, true)
	if err != nil {
		t.Fatalf("decoding json failed. type = %s, encoded bytes = %v", typ.String(), enc)
	}
	return string(d)
}

func requireDecodeJsonResult(t *testing.T, jsonStr string, typ reflect.Type, leBytes...byte) {
	j := requireDecodeJsonOK(t, typ, leBytes...)
	if j != jsonStr {
		t.Fatalf("decoding result error. got %v, expecting %v", j, jsonStr)
	}
}


func TestEncodeFromJson(t *testing.T) {
	requireEncodeJsonResult(t, `true`, BoolType(), 1)
	requireEncodeJsonResult(t, `false`, BoolType(), 0)
	requireEncodeJsonResult(t, `10`, Int8Type(), 10)
	requireEncodeJsonResult(t, `-2`, Int8Type(), 0xfe)
	requireEncodeJsonResult(t, `100`, Int16Type(), 100, 0)
	requireEncodeJsonResult(t, `-10`, Int32Type(), 0xf6, 0xff, 0xff, 0xff)
	requireEncodeJsonResult(t, `12345678`, Int64Type(), 0x4e, 0x61, 0xbc, 0, 0, 0, 0, 0)
	requireEncodeJsonResult(t, `10`, Uint8Type(), 10)
	requireEncodeJsonResult(t, `4321`, Uint16Type(), 0xe1, 0x10)
	requireEncodeJsonResult(t, `645322`, Uint32Type(), 0xca, 0xd8, 0x09, 0)
	requireEncodeJsonResult(t, `987654321`, Uint64Type(), 0xb1, 0x68, 0xde, 0x3a, 0, 0, 0, 0)
	requireEncodeJsonResult(t, `3.14159`, Float32Type(), 0xd0, 0x0f, 0x49, 0x40)
	requireEncodeJsonResult(t, `3.14159265359`, Float64Type(), 0xea, 0x2e, 0x44, 0x54, 0xfb, 0x21, 0x09, 0x40)
	requireEncodeJsonResult(t, `"hello"`, StringType(), []byte("\x05hello")...)

	requireEncodeJsonResult(t,
		`["alice", 100, 3.14159]`,
		StructOf(StringType(), Int16Type(), Float32Type()),
		[]byte("\x03\x05alice\x64\x00\xd0\x0f\x49\x40")...)

	requireEncodeJsonResult(t,
		`["bob", ["alice", 100, 3.14159]]`,
		StructOf(StringType(), StructOf(StringType(), Int16Type(), Float32Type())),
		[]byte("\x02\x03bob\x03\x05alice\x64\x00\xd0\x0f\x49\x40")...)

	requireEncodeJsonError(t, `0`, BoolType())
	requireEncodeJsonError(t, `123`, StringType())
}

func TestDecodeToJson(t *testing.T) {
	requireDecodeJsonResult(t, `true`, BoolType(), 1)
	requireDecodeJsonResult(t, `false`, BoolType(), 0)
	requireDecodeJsonResult(t, `10`, Int8Type(), 10)
	requireDecodeJsonResult(t, `-2`, Int8Type(), 0xfe)
	requireDecodeJsonResult(t, `100`, Int16Type(), 100, 0)
	requireDecodeJsonResult(t, `-10`, Int32Type(), 0xf6, 0xff, 0xff, 0xff)
	requireDecodeJsonResult(t, `12345678`, Int64Type(), 0x4e, 0x61, 0xbc, 0, 0, 0, 0, 0)
	requireDecodeJsonResult(t, `10`, Uint8Type(), 10)
	requireDecodeJsonResult(t, `4321`, Uint16Type(), 0xe1, 0x10)
	requireDecodeJsonResult(t, `645322`, Uint32Type(), 0xca, 0xd8, 0x09, 0)
	requireDecodeJsonResult(t, `987654321`, Uint64Type(), 0xb1, 0x68, 0xde, 0x3a, 0, 0, 0, 0)
	requireDecodeJsonResult(t, `3.14159`, Float32Type(), 0xd0, 0x0f, 0x49, 0x40)
	requireDecodeJsonResult(t, `3.14159265359`, Float64Type(), 0xea, 0x2e, 0x44, 0x54, 0xfb, 0x21, 0x09, 0x40)
	requireDecodeJsonResult(t, `"hello"`, StringType(), []byte("\x05hello")...)

	requireDecodeJsonResult(t,
		`["alice",100,3.14159]`,
		StructOf(StringType(), Int16Type(), Float32Type()),
		[]byte("\x03\x05alice\x64\x00\xd0\x0f\x49\x40")...)

	requireDecodeJsonResult(t,
		`["bob",["alice",100,3.14159]]`,
		StructOf(StringType(), StructOf(StringType(), Int16Type(), Float32Type())),
		[]byte("\x02\x03bob\x03\x05alice\x64\x00\xd0\x0f\x49\x40")...)
}
