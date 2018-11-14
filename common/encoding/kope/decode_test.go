package kope

import (
	"reflect"
	"testing"
)

func requireOK(t *testing.T, value interface{}, complement bool) {
	enc, err := Encode(value)
	if err != nil {
		t.Fatalf("encoding failed: %v", err)
	}
	if complement {
		enc, _ = Complement(enc)
	}
	dv, err := Decode(enc)
	if err != nil {
		t.Fatalf("decoding failed: %v", err)
	}
	if !testEqual(dv, value) {
		t.Fatalf("not equal. decoded = %v, original = %v", dv, value)
	}
}

func testEqual(s1 interface{}, s2 interface{}) bool {
	rv1, rv2 := reflect.ValueOf(s1), reflect.ValueOf(s2)
	if rv1.Kind() == reflect.Slice && rv2.Kind() == reflect.Slice {
		if rv1.Len() != rv2.Len() {
			return false
		}
		for i := 0; i < rv1.Len(); i++ {
			e1, e2 := rv1.Index(i), rv2.Index(i)
			if !testEqual(e1.Interface(), e2.Interface()) {
				return false
			}
		}
		return true
	} else {
		return s1 == s2
	}
}

type bar struct {
	x, y int
	name string
}

func barEncode(value reflect.Value) ([]byte, error) {
	ptr := value.Interface().(bar)
	return Encode(ptr.x, ptr.y, ptr.name)
}

func barDecode(enc []byte) (interface{}, error) {
	d, err := Decode(enc)
	if err != nil {
		return nil, err
	}
	s := d.([]interface{})
	return bar{ x: s[0].(int), y: s[1].(int), name: s[2].(string) }, nil
}

func TestDecode(t *testing.T) {
	requireOK(t, int8(22), false)
	requireOK(t, int8(-22), false)
	requireOK(t, int16(1200), false)
	requireOK(t, int16(-1200), false)
	requireOK(t, int32(323), false)
	requireOK(t, int32(-323), false)
	requireOK(t, int64(233323), false)
	requireOK(t, int64(-233323), false)
	requireOK(t, int(7434), false)
	requireOK(t, int(-7434), false)
	requireOK(t, uint8(22), false)
	requireOK(t, uint16(1200), false)
	requireOK(t, uint32(323), false)
	requireOK(t, uint64(233323), false)
	requireOK(t, uint(7434), false)
	requireOK(t, uintptr(237434), false)
	requireOK(t, float32(2.4e19), false)
	requireOK(t, float32(-2.4e-19), false)
	requireOK(t, float64(934.54), false)
	requireOK(t, float64(2.71828e-32), false)
	requireOK(t, "this is a string", false)
	requireOK(t, []byte{1,2,3,4,5,6,7}, false)
	requireOK(t, []int{1,2,3,4,5,6,7}, false)
	requireOK(t, [][]string{ {"one", "two"}, {"three"} }, false)

	requireOK(t, int8(22), true)
	requireOK(t, int8(-22), true)
	requireOK(t, int16(1200), true)
	requireOK(t, int16(-1200), true)
	requireOK(t, int32(323), true)
	requireOK(t, int32(-323), true)
	requireOK(t, int64(233323), true)
	requireOK(t, int64(-233323), true)
	requireOK(t, int(7434), true)
	requireOK(t, int(-7434), true)
	requireOK(t, uint8(22), true)
	requireOK(t, uint16(1200), true)
	requireOK(t, uint32(323), true)
	requireOK(t, uint64(233323), true)
	requireOK(t, uint(7434), true)
	requireOK(t, uintptr(237434), true)
	requireOK(t, float32(2.4e19), true)
	requireOK(t, float32(-2.4e-19), true)
	requireOK(t, float64(934.54), true)
	requireOK(t, float64(2.71828e-32), true)
	requireOK(t, "this is a string", true)
	requireOK(t, []byte{1,2,3,4,5,6,7}, true)
	requireOK(t, []int{1,2,3,4,5,6,7}, true)
	requireOK(t, [][]string{ {"one", "two"}, {"three"} }, true)

	RegisterType(reflect.TypeOf((*bar)(nil)).Elem(), "decode_test.bar", barEncode, barDecode)
	mybar := bar{1, 2, "mybar"}
	enc, _ := Encode(mybar)
	dbar, _ := Decode(enc)
	if mybar != dbar {
		t.Fatalf("not equal. decoded = %v, original = %v", dbar, mybar)
	}
}
