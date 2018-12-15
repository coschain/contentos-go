package vme

//
// vme is for contract parameter encoding.
// It implements the same serialization protocol used by smart contracts.
//
// Single parameter
// A single parameter can be an instance of,
//  - primitive types (bool/integers/floats/string) or
//  - single or multiple dimensional slice of primitive types
//
// Integers and floats are encoded by simply using their memory bytes, in host machine's byte order.
// Booleans are treated as special bytes. byte(1) represents true, and byte(0) represents false.
// Strings are treated as byte slices.
// Slices are encoded as a variant-length-integer encoded length followed by encoded elements one by one.
//
//
// Multiple parameters
// Multiple parameters are encoded by concatenating encoded parameters.
//
//
// Example
// To encode a single parameter,
//   Encode(10)
//   Encode("hello")
//   Encode([]int{1, 2, 3})         // a single parameter of type []int.
//
// To encode multiple parameters,
//   Encode(1, 2, 3)                // 3 separate int parameters
//   Encode("world", 3.14, byte(20), true)
//

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"unsafe"
)

func memoryBytes(ptr unsafe.Pointer, size uintptr) []byte {
	buf := make([]byte, size)
	for i := range buf {
		buf[i] = *(*byte)(unsafe.Pointer(uintptr(ptr) + uintptr(i)))
	}
	return buf
}

func encodeBool(value bool) ([]byte, error) {
	b := uint8(0)
	if value {
		b = 1
	}
	return encodeUint8(b)
}

func encodeInt(value int) ([]byte, error) {
	return memoryBytes(unsafe.Pointer(&value), unsafe.Sizeof(value)), nil
}

func encodeInt8(value int8) ([]byte, error) {
	return []byte{ byte(value) }, nil
}

func encodeInt16(value int16) ([]byte, error) {
	return memoryBytes(unsafe.Pointer(&value), unsafe.Sizeof(value)), nil
}

func encodeInt32(value int32) ([]byte, error) {
	return memoryBytes(unsafe.Pointer(&value), unsafe.Sizeof(value)), nil
}

func encodeInt64(value int64) ([]byte, error) {
	return memoryBytes(unsafe.Pointer(&value), unsafe.Sizeof(value)), nil
}

func encodeUint(value uint) ([]byte, error) {
	return memoryBytes(unsafe.Pointer(&value), unsafe.Sizeof(value)), nil
}

func encodeUint8(value uint8) ([]byte, error) {
	return []byte{ value }, nil
}

func encodeUint16(value uint16) ([]byte, error) {
	return memoryBytes(unsafe.Pointer(&value), unsafe.Sizeof(value)), nil
}

func encodeUint32(value uint32) ([]byte, error) {
	return memoryBytes(unsafe.Pointer(&value), unsafe.Sizeof(value)), nil
}

func encodeUint64(value uint64) ([]byte, error) {
	return memoryBytes(unsafe.Pointer(&value), unsafe.Sizeof(value)), nil
}

func encodeFloat32(value float32) ([]byte, error) {
	return memoryBytes(unsafe.Pointer(&value), unsafe.Sizeof(value)), nil
}

func encodeFloat64(value float64) ([]byte, error) {
	return memoryBytes(unsafe.Pointer(&value), unsafe.Sizeof(value)), nil
}

func varInt(value uint64) []byte {
	buf := make([]byte, 10)
	return buf[:binary.PutUvarint(buf, value)]
}

func encodeString(value string) ([]byte, error) {
	return encodeBytes([]byte(value))
}

func encodeBytes(value []byte) ([]byte, error) {
	return bytes.Join([][]byte{ varInt(uint64(len(value))), value }, nil), nil
}

func encodeSlice(value []interface{}) ([]byte, error) {
	count := len(value)
	parts := make([][]byte, 0, count + 1)
	parts = append(parts, varInt(uint64(count)))
	for _, e := range value {
		if data, err := Encode(e); err != nil {
			return nil, err
		} else {
			parts = append(parts, data)
		}
	}
	return bytes.Join(parts, nil), nil
}

func encodeValue(rv reflect.Value) ([]byte, error) {
	k := rv.Kind()
	if k == reflect.Invalid {
		return nil, errors.New("vme: cannot encode zero values")
	}
	if k == reflect.Ptr {
		if rv.IsNil() {
			return nil, errors.New("vme: cannot encode nil pointers")
		}
		return encodeValue(reflect.Indirect(rv))
	}
	rt := rv.Type()
	switch k {
	case reflect.Bool:
		return encodeBool(rv.Bool())
	case reflect.Int:
		return encodeInt(int(rv.Int()))
	case reflect.Int8:
		return encodeInt8(int8(rv.Int()))
	case reflect.Int16:
		return encodeInt16(int16(rv.Int()))
	case reflect.Int32:
		return encodeInt32(int32(rv.Int()))
	case reflect.Int64:
		return encodeInt64(rv.Int())
	case reflect.Uint:
		return encodeUint(uint(rv.Uint()))
	case reflect.Uint8:
		return encodeUint8(uint8(rv.Uint()))
	case reflect.Uint16:
		return encodeUint16(uint16(rv.Uint()))
	case reflect.Uint32:
		return encodeUint32(uint32(rv.Uint()))
	case reflect.Uint64:
		return encodeUint64(rv.Uint())
	case reflect.Float32:
		return encodeFloat32(float32(rv.Float()))
	case reflect.Float64:
		return encodeFloat64(rv.Float())
	case reflect.String:
		return encodeString(rv.String())
	case reflect.Array, reflect.Slice:
		if rt.Elem().Kind() == reflect.Uint8 {
			return encodeBytes(rv.Bytes())
		}
		size := rv.Len()
		elements := make([]interface{}, size)
		for i := 0; i < size; i++ {
			elements[i] = rv.Index(i).Interface()
		}
		return encodeSlice(elements)
	}
	return nil, errors.New(fmt.Sprintf("vme: cannot encode values of type %s", rt.Name()))
}

func encode(value interface{}) ([]byte, error) {
	return encodeValue(reflect.ValueOf(value))
}

func Encode(values...interface{}) ([]byte, error) {
	count := len(values)
	if count == 0 {
		return nil, errors.New("vme: nothing to encode")
	}
	r := make([][]byte, count)
	for i, v := range values {
		data, err := encode(v)
		if err != nil {
			return nil, err
		}
		r[i] = data
	}
	return bytes.Join(r, nil), nil
}

func String(value...interface{}) (string, error) {
	if data, err := Encode(value...); err != nil {
		return "", err
	} else {
		return string(data), nil
	}
}
