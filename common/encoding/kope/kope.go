package kope

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"sync"
)

const (
	is32bitPlatform = ^uint(0)>>32 == 0
	escapeMark 		= byte(1)
	prefixMin 		= byte(2)
	prefix 			= byte(3)
	prefixMax 		= byte(4)
)

var (
	MinKey = &minKeyPlaceholder
	MaxKey = &maxKeyPlaceholder
	minKeyPlaceholder = "I'm lesser than all real keys"
	maxKeyPlaceholder = "I'm greater than all real keys"
	sliceSeparator = []byte{0, 0}
)


// bigEndianBytes() returns memory bytes of the given value in big-endian byte order.
func bigEndianBytes(value interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, value)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func escapedWithPrefix(src []byte, errs...error) ([]byte, error) {
	var err error
	if len(errs) > 0 {
		err = errs[0]
	}
	if err != nil {
		return src, err
	}
	buf := new(bytes.Buffer)
	err = buf.WriteByte(prefix)
	if err != nil {
		return nil, err
	}
	mark := escapeMark
	for _, b := range src {
		if err = binary.Write(buf, binary.LittleEndian, b); err != nil {
			return nil, err
		}
		if b == 0 {
			if err = binary.Write(buf, binary.LittleEndian, mark); err != nil {
				return nil, err
			}
		}
	}
	return buf.Bytes(), nil
}

func encodeSignedInteger(value interface{}) ([]byte, error) {
	data, err := bigEndianBytes(value)
	if err == nil {
		data[0] ^= 0x80
	}
	return escapedWithPrefix(data, err)
}

func encodeUnsignedInteger(value interface{}) ([]byte, error) {
	return escapedWithPrefix(bigEndianBytes(value))
}

func encodeFloat(value interface{}) ([]byte, error) {
	data, err := bigEndianBytes(value)
	if err == nil {
		isZero := true
		for _, b := range data[1:] {
			if b != 0 {
				isZero = false
				break
			}
		}
		if isZero = isZero && (data[0]&0x7f) == 0; isZero {
			data[0] = 0
		}

		if isNegative := (data[0] & 0x80) != 0; isNegative {
			for i, b := range data {
				data[i] = ^b
			}
		} else {
			data[0] ^= 0x80
		}
	}
	return escapedWithPrefix(data, err)
}

// Encode a signed integer
func EncodeInt(value int) ([]byte, error) {
	if is32bitPlatform {
		return EncodeInt32(int32(value))
	}
	return EncodeInt64(int64(value))
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
	if is32bitPlatform {
		return EncodeUint32(uint32(value))
	}
	return EncodeUint64(uint64(value))
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
	return EncodeUint(uint(value))
}

// Encode a bool
func EncodeBool(value bool) ([]byte, error) {
	b := uint8(0)
	if value {
		b = 1
	}
	return EncodeUint8(b)
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
	return EncodeBytes([]byte(value))
}

// Encode a byte slice
func EncodeBytes(value []byte) ([]byte, error) {
	return escapedWithPrefix(value, nil)
}

func EncodeValue(rv reflect.Value) ([]byte, error) {
	k := rv.Kind()
	if k == reflect.Invalid {
		return nil, errors.New("kope: cannot encode nil values")
	}
	rt := rv.Type()

	if encFunc, ok := registeredEncoders.Load(rt); ok {
		return encFunc.(CustomEncoderFunc)(rv)
	}

	if k == reflect.Ptr {
		if rv.IsNil() {
			return nil, errors.New("kope: cannot encode nil pointers")
		}
		if rt.Elem().Kind() == reflect.String {
			ptr := rv.Interface().(*string)
			if ptr == MinKey {
				return encodeMinMaxKey(true)
			} else if ptr == MaxKey {
				return encodeMinMaxKey(false)
			}
		}
		return EncodeValue(reflect.Indirect(rv))
	}

	switch k {
	case reflect.Bool:
		return EncodeBool(rv.Interface().(bool))
	case reflect.Int:
		return EncodeInt(rv.Interface().(int))
	case reflect.Int8:
		return EncodeInt8(rv.Interface().(int8))
	case reflect.Int16:
		return EncodeInt16(rv.Interface().(int16))
	case reflect.Int32:
		return EncodeInt32(rv.Interface().(int32))
	case reflect.Int64:
		return EncodeInt64(rv.Interface().(int64))
	case reflect.Uint:
		return EncodeUint(rv.Interface().(uint))
	case reflect.Uint8:
		return EncodeUint8(rv.Interface().(uint8))
	case reflect.Uint16:
		return EncodeUint16(rv.Interface().(uint16))
	case reflect.Uint32:
		return EncodeUint32(rv.Interface().(uint32))
	case reflect.Uint64:
		return EncodeUint64(rv.Interface().(uint64))
	case reflect.Uintptr:
		return EncodeUintPtr(rv.Interface().(uintptr))
	case reflect.Float32:
		return EncodeFloat32(rv.Interface().(float32))
	case reflect.Float64:
		return EncodeFloat64(rv.Interface().(float64))
	case reflect.String:
		return EncodeString(rv.Interface().(string))
	case reflect.Array, reflect.Slice:
		if rt.Elem().Kind() == reflect.Uint8 {
			return EncodeBytes(rv.Interface().([]byte))
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
func Encode(value interface{}) ([]byte, error) {
	return EncodeValue(reflect.ValueOf(value))
}

func encodeMinMaxKey(min bool) ([]byte, error) {
	p := prefixMin
	if !min {
		p = prefixMax
	}
	return []byte{p}, nil
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
	return escapedWithPrefix(bytes.Join(children, sliceSeparator))
}

func unescapedWithoutPrefix(src []byte) ([]byte, error) {
	if len(src) < 1 {
		return nil, errors.New("unescapedWithPrefix: data too short")
	}
	lastByte, e := src[0], src[1:]
	buf := new(bytes.Buffer)
	for _, b := range e {
		if !(b == escapeMark && lastByte == 0) {
			if err := buf.WriteByte(b); err != nil {
				return nil, err
			}
		}
		lastByte = b
	}
	return buf.Bytes(), nil
}

func DecodeSlice(enc []byte) ([][]byte, error) {
	d, err := unescapedWithoutPrefix(enc)
	if err != nil {
		return nil, err
	}
	b := 0
	pos := []int{0}
	sepLen := len(sliceSeparator)
	for {
		p := bytes.Index(d[b:], sliceSeparator)
		if p < 0 {
			break
		}
		b += p + sepLen
		pos = append(pos, b - sepLen, b)
	}
	pos = append(pos, len(d))

	var result [][]byte
	for i := 0; i < len(pos); i += 2 {
		result = append(result, d[pos[i]:pos[i + 1]])
	}
	return result, nil
}

func Complement(enc []byte, errs ...error) ([]byte, error) {
	var err error
	if len(errs) > 0 {
		err = errs[0]
	}
	if err != nil {
		return enc, err
	}
	if len(enc) > 1 {
		buf := new(bytes.Buffer)
		err := buf.WriteByte(enc[0])
		if err != nil {
			return nil, err
		}
		for _, b := range enc[1:] {
			if err = buf.WriteByte(^b); err != nil {
				return nil, err
			}
			if b == 0 {
				if err = buf.WriteByte(0xfe); err != nil {
					return nil, err
				}
			}
		}
		if _, err = buf.Write([]byte{0xff, 0xff}); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}
	return enc, err
}

func Uncomplement(src []byte) ([]byte, error) {
	if len(src) < 3 {
		return nil, errors.New("Uncomplement: data too short")
	}
	lastByte, e := src[0], src[1:len(src)-2]
	buf := new(bytes.Buffer)
	err := buf.WriteByte(lastByte)
	if err != nil {
		return nil, err
	}
	for _, b := range e {
		if !(b == 0xfe && lastByte == 0xff) {
			if err := buf.WriteByte(^b); err != nil {
				return nil, err
			}
		}
		lastByte = b
	}
	return buf.Bytes(), nil
}

var (
	registeredEncoders sync.Map
)

type CustomEncoderFunc func(reflect.Value)([]byte, error)
type CustomPrimaryKeysFunc func(reflect.Value) []interface{}

func RegisterTypeEncoder(rt reflect.Type, encoderFunc CustomEncoderFunc) {
	registeredEncoders.Store(rt, encoderFunc)
}

func RegisterTypePrimaryKeys(rt reflect.Type, pkExtractor CustomPrimaryKeysFunc) {
	RegisterTypeEncoder(rt, func(value reflect.Value) ([]byte, error) {
		return EncodeSlice(pkExtractor(value))
	})
}
