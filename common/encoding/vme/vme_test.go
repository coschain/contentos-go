package vme

import (
	"bytes"
	"github.com/coschain/contentos-go/common"
	"reflect"
	"testing"
)

func requireEncodeError(t *testing.T, value interface{}) {
	_, err := Encode(value)
	if err == nil {
		t.Fatalf("encoding should fail, but succeeded. value = %v", value)
	}
}

func requireEncodeOK(t *testing.T, value interface{}) []byte {
	data, err := Encode(value)
	if err != nil {
		t.Fatalf("encoding should succeed, but got error = %v, value = %v", err, value)
	}
	return data
}

func requireEncodeResult(t *testing.T, value interface{}, leBytes...byte) {
	enc := requireEncodeOK(t, value)
	r := leBytes
	if !common.IsLittleEndianPlatform() {
		n := len(leBytes)
		r = make([]byte, n)
		n--
		for i := range r {
			r[i] = leBytes[n - i]
		}
	}
	if bytes.Compare(r, enc) != 0 {
		t.Fatalf("encoding result error. got %v, expecting %v", enc, r)
	}
}

func requireDecodeOK(t *testing.T, typ reflect.Type, leBytes...byte) interface{} {
	enc := leBytes
	if !common.IsLittleEndianPlatform() {
		n := len(leBytes)
		enc = make([]byte, n)
		n--
		for i := range enc {
			enc[i] = leBytes[n - i]
		}
	}
	d, err := DecodeWithType(enc, typ)
	if err != nil {
		t.Fatalf("decoding to %s failed. encoded bytes = %v", typ.String(), enc)
	}
	return d
}

func requireDecodeResult(t *testing.T, value interface{}, leBytes...byte) {
	d := requireDecodeOK(t, reflect.TypeOf(value), leBytes...)
	if d != value {
		t.Fatalf("decoding result error. got %v, expecting %v", d, value)
	}
}


func TestEncode(t *testing.T) {
	requireEncodeResult(t, true, 1)
	requireEncodeResult(t, false, 0)
	requireEncodeResult(t, int8(10), 10)
	requireEncodeResult(t, int8(-2), 0xfe)
	requireEncodeResult(t, int16(100), 100, 0)
	requireEncodeResult(t, int32(-10), 0xf6, 0xff, 0xff, 0xff)
	requireEncodeResult(t, int64(12345678), 0x4e, 0x61, 0xbc, 0, 0, 0, 0, 0)
	requireEncodeResult(t, uint8(10), 10)
	requireEncodeResult(t, int16(4321), 0xe1, 0x10)
	requireEncodeResult(t, uint32(645322), 0xca, 0xd8, 0x09, 0)
	requireEncodeResult(t, uint64(987654321), 0xb1, 0x68, 0xde, 0x3a, 0, 0, 0, 0)
	requireEncodeResult(t, float32(3.14159), 0xd0, 0x0f, 0x49, 0x40)
	requireEncodeResult(t, float64(3.14159265359), 0xea, 0x2e, 0x44, 0x54, 0xfb, 0x21, 0x09, 0x40)
	requireEncodeResult(t, "hello", []byte("\x05hello")...)
	requireEncodeResult(t, []byte("hello"), []byte("\x05hello")...)

	requireEncodeResult(t, StructValue("alice", int16(100), float32(3.14159)), []byte("\x03\x05alice\x64\x00\xd0\x0f\x49\x40")...)
	requireEncodeResult(t,
		StructValue("bob",
			StructValue("alice", int16(100), float32(3.14159)),
		),
		[]byte("\x02\x03bob\x03\x05alice\x64\x00\xd0\x0f\x49\x40")...)

	requireEncodeResult(t, []int32{3,4,5,6}, 4, 3,0,0,0, 4,0,0,0, 5,0,0,0, 6,0,0,0)
	requireEncodeResult(t, []string{"nice", "to", "meet", "you"}, []byte("\x04\x04nice\x02to\x04meet\x03you")...)

	requireEncodeError(t, map[int]int{1:2, 3:4})
	requireEncodeError(t, nil)
	requireEncodeError(t, []interface{}{3})
}

func TestDecode(t *testing.T) {
	requireDecodeResult(t, true, 1)
	requireDecodeResult(t, false, 0)
	requireDecodeResult(t, int8(10), 10)
	requireDecodeResult(t, int8(-2), 0xfe)
	requireDecodeResult(t, int16(100), 100, 0)
	requireDecodeResult(t, int32(-10), 0xf6, 0xff, 0xff, 0xff)
	requireDecodeResult(t, int64(12345678), 0x4e, 0x61, 0xbc, 0, 0, 0, 0, 0)
	requireDecodeResult(t, uint8(10), 10)
	requireDecodeResult(t, int16(4321), 0xe1, 0x10)
	requireDecodeResult(t, uint32(645322), 0xca, 0xd8, 0x09, 0)
	requireDecodeResult(t, uint64(987654321), 0xb1, 0x68, 0xde, 0x3a, 0, 0, 0, 0)
	requireDecodeResult(t, float32(3.14159), 0xd0, 0x0f, 0x49, 0x40)
	requireDecodeResult(t, float64(3.14159265359), 0xea, 0x2e, 0x44, 0x54, 0xfb, 0x21, 0x09, 0x40)
	requireDecodeResult(t, "hello", []byte("\x05hello")...)

	var d interface{}
	d = requireDecodeOK(t, BytesType(), []byte("\x05hello")...)
	if bytes.Compare(d.([]byte), []byte("hello")) != 0 {
		t.Fatalf("decoding result error. got %v, expecting %v", d, []byte("hello"))
	}

	sv := StructValue("bob",
		StructValue("alice", int16(100), float32(3.14159)),
	)
	d = requireDecodeOK(t, reflect.TypeOf(sv), []byte("\x02\x03bob\x03\x05alice\x64\x00\xd0\x0f\x49\x40")...)
	if reflect.ValueOf(d).Field(0).String() != "bob" ||
		reflect.ValueOf(d).Field(1).Field(0).String() != "alice" ||
		reflect.ValueOf(d).Field(1).Field(1).Int() != 100 ||
		reflect.ValueOf(d).Field(1).Field(2).Float() != float64(float32(3.14159)) {
		t.Fatal("decoding result error on structures")
	}

	v1 := int32(0)
	Decode([]byte{10, 0, 0, 0}, &v1)
	if v1 != 10 {
		t.Fatal("decoding result error on ints")
	}
	v2 := "world"
	Decode([]byte("\x05hello"), &v2)
	if v2 != "hello" {
		t.Fatal("decoding result error on strings")
	}

	t3 := StructOf(StringType(), StructOf(StringType(), Int16Type(), Float32Type()))
	p3 := reflect.New(t3).Interface()
	Decode([]byte("\x02\x03bob\x03\x05alice\x64\x00\xd0\x0f\x49\x40"), p3)
	if reflect.ValueOf(p3).Elem().Field(0).String() != "bob" {
		t.Fatal("decoding result error on structures")
	}

	ss, _ := DecodeWithType([]byte("\x04\x04nice\x02to\x04meet\x03you"), reflect.SliceOf(reflect.TypeOf("")))
	s4 := ss.([]string)
	if len(s4) != 4 || s4[0] != "nice" || s4[1] != "to" || s4[2] != "meet" || s4[3] != "you" {
		t.Fatal("decoding []string error")
	}
}
