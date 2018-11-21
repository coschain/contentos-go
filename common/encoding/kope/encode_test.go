package kope

import (
	"fmt"
	"testing"
)

func hexStr(t *testing.T, v interface{}) string {
	data, err := Encode(v)
	if err != nil {
		t.Fatalf("encoding failed. error = %v", err)
	}
	return fmt.Sprintf("%x", data)
}


func assertLess(t *testing.T, a interface{}, b interface{}) {
	if sa, sb := hexStr(t, a), hexStr(t, b); sa >= sb {
		t.Fatalf("assertLess: %v | %v", sa, sb)
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
	requireNoError(t, MinimalKey)
	requireNoError(t, MaximumKey)
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
}
