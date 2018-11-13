package kope

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

func hexStr(t *testing.T, v interface{}) string {
	data, err := Encode(v)
	if err != nil {
		t.Fatalf("encoding failed. error = %v", err)
	}
	return fmt.Sprintf("%x", data)
}

func hexStrComplement(t *testing.T, v interface{}) string {
	data, err := Complement(Encode(v))
	if err != nil {
		t.Fatalf("complement encoding failed. error = %v", err)
	}
	return fmt.Sprintf("%x", data)
}

func assertLess(t *testing.T, a interface{}, b interface{}) {
	if sa, sb := hexStr(t, a), hexStr(t, b); sa >= sb {
		t.Fatalf("assertLess: %v | %v", sa, sb)
	}
}

func assertComplementGreat(t *testing.T, a interface{}, b interface{}) {
	if sa, sb := hexStrComplement(t, a), hexStrComplement(t, b); sa <= sb {
		t.Fatalf("assertComplementGreat: %v | %v", sa, sb)
	}
}

func requireError(t *testing.T, v interface{}) {
	if _, err := Encode(v); err == nil {
		t.Fatalf("requireError: %T", v)
	}
}

func requireNoError(t *testing.T, v interface{}) {
	if _, err := Encode(v); err != nil {
		t.Fatalf("requireNoError: %T (%v)", v, err)
	}
}

type foo struct {
	x, y int
	name string
}

func TestEncode(t *testing.T) {
	// basic types
	requireNoError(t, true)
	requireNoError(t, false)
	requireNoError(t, int(232))
	requireNoError(t, int8(12))
	requireNoError(t, int16(30023))
	requireNoError(t, int32(23423423))
	requireNoError(t, int64(8493409343))
	requireNoError(t, uint(232))
	requireNoError(t, uint8(12))
	requireNoError(t, uint16(30023))
	requireNoError(t, uint32(23423423))
	requireNoError(t, uint64(8493409343))
	requireNoError(t, uintptr(99343))
	requireNoError(t, float32(3.14159))
	requireNoError(t, float64(2.71828e+3))
	requireNoError(t, "hello world")
	requireNoError(t, []byte("hello world"))
	requireNoError(t, MinKey)
	requireNoError(t, MaxKey)
	requireError(t, nil)
	requireError(t, map[int]int{1: 10, 2: 20})

	// slices
	requireNoError(t, []bool{true, false, true, true, false})
	requireNoError(t, []int{0, -3, 2, 89, 900, 74, -2})
	requireNoError(t, []int8{0, -3, 2, 89, 9, 74, -2})
	requireNoError(t, []int16{0, -3, 2, 89, 900, 74, -2})
	requireNoError(t, []int32{0, -3, 2, 89, 900, 74, -2})
	requireNoError(t, []int64{0, -3, 2, 89, 900, 74, -2})
	requireNoError(t, []uint{0, 11, 22, 33, 44, 55, 66})
	requireNoError(t, []uint8{0, 11, 22, 33, 44, 55, 66})
	requireNoError(t, []uint16{0, 11, 22, 33, 44, 55, 66})
	requireNoError(t, []uint32{0, 11, 22, 33, 44, 55, 66})
	requireNoError(t, []uint64{0, 11, 22, 33, 44, 55, 66})
	requireNoError(t, []uintptr{0, 11, 22, 33, 44, 55, 66})
	requireNoError(t, []float32{-2.3, 0.0, 7.2, 99.5, 1e20, -5e-7})
	requireNoError(t, []float64{-2.3, 0.0, 7.2, 99.5, 1e20, -5e-7})
	requireNoError(t, []string{"alice", "bob", "charlie"})

	// RegisterTypeEncoder
	requireError(t, &foo{1,2, "alice"})
	RegisterType(reflect.TypeOf((*foo)(nil)), "kope_test.foo",
		func(value reflect.Value) ([]byte, error) {
			ptr := value.Interface().(*foo)
			return Encode([]interface{} {
				ptr.x, ptr.y, ptr.name,
			})
		}, nil)
	requireNoError(t, &foo{1,2, "alice"})


	// order preserving
	assertLess(t, int8(-2), int8(0))
	assertLess(t, int8(-120), int8(-23))
	assertLess(t, int8(-12), int8(23))
	assertLess(t, uint32(0), uint32(3))
	assertLess(t, uint32(232), uint32(65675))
	assertLess(t, float32(-3e30), float32(-2e20))
	assertLess(t, float32(-0.000001), float32(0))
	assertLess(t, float32(0), float32(0.000023))
	assertLess(t, float32(5.34), float32(7.3e10))
	assertLess(t, float64(-3e30), float64(-2e20))
	assertLess(t, float64(-1.3e20), float64(-1.29e20))
	assertLess(t, float64(-0.000001), float64(0))
	assertLess(t, float64(0), float64(0.000023))
	assertLess(t, float64(5.34), float64(7.3e10))
	assertLess(t, "alice", "bob")
	assertLess(t, "alice", "alice's mom")
	assertLess(t, []string{"1", "2"}, []string{"1", "2", "3"})
	assertLess(t, []string{"alice", ""}, []string{"alice", "bob"})

	// reversed order preserving
	assertComplementGreat(t, int8(-2), int8(0))
	assertComplementGreat(t, int8(-120), int8(-23))
	assertComplementGreat(t, int8(-12), int8(23))
	assertComplementGreat(t, uint32(0), uint32(3))
	assertComplementGreat(t, uint32(232), uint32(65675))
	assertComplementGreat(t, float32(-3e30), float32(-2e20))
	assertComplementGreat(t, float32(-0.000001), float32(0))
	assertComplementGreat(t, float32(0), float32(0.000023))
	assertComplementGreat(t, float32(5.34), float32(7.3e10))
	assertComplementGreat(t, float64(-3e30), float64(-2e20))
	assertComplementGreat(t, float64(-1.3e20), float64(-1.29e20))
	assertComplementGreat(t, float64(-0.000001), float64(0))
	assertComplementGreat(t, float64(0), float64(0.000023))
	assertComplementGreat(t, float64(5.34), float64(7.3e10))
	assertComplementGreat(t, "alice", "bob")
	assertComplementGreat(t, "alice", "alice's mom")
	assertComplementGreat(t, []string{"1", "2"}, []string{"1", "2", "3"})
	assertComplementGreat(t, []string{"alice", ""}, []string{"alice", "bob"})

	// Uncomplement
	e1, _ := Encode("hello, world")
	e2, _ := Complement(e1)
	e3, _ := Complement(e2)
	if bytes.Compare(e1, e3) != 0 {
		t.Fatalf("Uncomplement failed")
	}

	// DecodeSlice
	var d []interface{}
	var e []byte
	e, _ = Encode([]int{1, 2, 3})
	d, _ = DecodeSlice(e)
	if d[0] != int(1) || d[1] != int(2) || d[2] != int(3) {
		t.Fatalf("decodeSlice failed")
	}

	e, _ = Encode([]string{"one", "two", "three"})
	d, _ = DecodeSlice(e)
	if d[0] != "one" || d[1] != "two" || d[2] != "three" {
		t.Fatalf("decodeSlice failed")
	}
}
