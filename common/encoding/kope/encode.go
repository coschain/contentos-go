package kope

import (
	"errors"
	"fmt"
	"reflect"
)

// Encode a signed integer
func EncodeInt(value int) ([]byte, error) {
	var data []byte
	var err error
	if is32bitPlatform {
		data, err = EncodeInt32(int32(value))
	} else {
		data, err = EncodeInt64(int64(value))
	}
	if err == nil {
		data[0] = typeInt
	}
	return data, err
}

// Encode a signed 8-bit integer
func EncodeInt8(value int8) ([]byte, error) {
	return encodeSignedInteger(value)
}

// Encode a signed 16-bit integer
func EncodeInt16(value int16) ([]byte, error) {
	return encodeSignedInteger(value)
}

// Encode a signed 32-bit integer
func EncodeInt32(value int32) ([]byte, error) {
	return encodeSignedInteger(value)
}

// Encode a signed 64-bit integer
func EncodeInt64(value int64) ([]byte, error) {
	return encodeSignedInteger(value)
}

// Encode an unsigned integer
func EncodeUint(value uint) ([]byte, error) {
	var data []byte
	var err error
	if is32bitPlatform {
		data, err = EncodeUint32(uint32(value))
	} else {
		data, err = EncodeUint64(uint64(value))
	}
	if err == nil {
		data[0] = typeUint
	}
	return data, err
}

// Encode an unsigned 8-bit integer
func EncodeUint8(value uint8) ([]byte, error) {
	return encodeUnsignedInteger(value)
}

// Encode an unsigned 16-bit integer
func EncodeUint16(value uint16) ([]byte, error) {
	return encodeUnsignedInteger(value)
}

// Encode an unsigned 32-bit integer
func EncodeUint32(value uint32) ([]byte, error) {
	return encodeUnsignedInteger(value)
}

// Encode an unsigned 64-bit integer
func EncodeUint64(value uint64) ([]byte, error) {
	return encodeUnsignedInteger(value)
}

// Encode an unsigned pointer integer
func EncodeUintPtr(value uintptr) ([]byte, error) {
	data, err := EncodeUint(uint(value))
	if err == nil {
		data[0] = typeUintptr
	}
	return data, err
}

// Encode a bool
func EncodeBool(value bool) ([]byte, error) {
	b := uint8(0)
	if value {
		b = 1
	}
	return pack(typeBool, false, []byte{b})
}

// Encode a float32
func EncodeFloat32(value float32) ([]byte, error) {
	return encodeFloat(value)
}

// Encode a float64
func EncodeFloat64(value float64) ([]byte, error) {
	return encodeFloat(value)
}

// Encode a string
func EncodeString(value string) ([]byte, error) {
	return pack(typeString, true, []byte(value))
}

// Encode a byte slice
func EncodeBytes(value []byte) ([]byte, error) {
	return pack(typeBytes, true, value)
}

func EncodeSlice(value []interface{}) ([]byte, error) {
	children := make([][]byte, len(value))
	var err error
	var d []byte
	for i, c := range value {
		if d, err = Encode(c); err != nil {
			return nil, err
		}
		children[i] = d
	}
	return encodeSlice(children)
}

func EncodeValue(rv reflect.Value) ([]byte, error) {
	k := rv.Kind()
	if k == reflect.Invalid {
		return nil, errors.New("kope: cannot encode nil values")
	}
	rt := rv.Type()

	if codec := customCodecByType(rt); codec != nil {
		return codec.encoder(rv)
	}

	if k == reflect.Ptr {
		if rv.IsNil() {
			return nil, errors.New("kope: cannot encode nil pointers")
		}
		if rt.Elem().Kind() == reflect.String {
			ptr := rv.Interface().(*string)
			if ptr == MinimalKey {
				return encodeMinMaxKey(true)
			} else if ptr == MaximumKey {
				return encodeMinMaxKey(false)
			}
		}
		return EncodeValue(reflect.Indirect(rv))
	}

	switch k {
	case reflect.Bool:
		return EncodeBool(rv.Bool())
	case reflect.Int:
		return EncodeInt(int(rv.Int()))
	case reflect.Int8:
		return EncodeInt8(int8(rv.Int()))
	case reflect.Int16:
		return EncodeInt16(int16(rv.Int()))
	case reflect.Int32:
		return EncodeInt32(int32(rv.Int()))
	case reflect.Int64:
		return EncodeInt64(rv.Int())
	case reflect.Uint:
		return EncodeUint(uint(rv.Uint()))
	case reflect.Uint8:
		return EncodeUint8(uint8(rv.Uint()))
	case reflect.Uint16:
		return EncodeUint16(uint16(rv.Uint()))
	case reflect.Uint32:
		return EncodeUint32(uint32(rv.Uint()))
	case reflect.Uint64:
		return EncodeUint64(rv.Uint())
	case reflect.Uintptr:
		return EncodeUintPtr(uintptr(rv.Uint()))
	case reflect.Float32:
		return EncodeFloat32(float32(rv.Float()))
	case reflect.Float64:
		return EncodeFloat64(rv.Float())
	case reflect.String:
		return EncodeString(rv.String())
	case reflect.Array, reflect.Slice:
		if rt.Elem().Kind() == reflect.Uint8 {
			if rt.PkgPath() + "." + rt.Name() == keyTypePkgName {
				return rv.Bytes(), nil
			}
			return EncodeBytes(rv.Bytes())
		}
		size := rv.Len()
		elements := make([]interface{}, size)
		for i := 0; i < size; i++ {
			elements[i] = rv.Index(i).Interface()
		}
		return EncodeSlice(elements)
	}
	return nil, errors.New(fmt.Sprintf("kope: cannot encode values of type %s", rt.Name()))
}

// Encode a value
func encode(value interface{}) ([]byte, error) {
	return EncodeValue(reflect.ValueOf(value))
}

// Encode values
func Encode(values...interface{}) ([]byte, error) {
	if len(values) == 0 {
		return nil, errors.New("nothing to encode")
	}
	if len(values) == 1 {
		return encode(values[0])
	} else {
		return encode(values)
	}
}

func Complement(enc []byte, errs ...error) ([]byte, error) {
	return flipped(enc, errs...)
}
