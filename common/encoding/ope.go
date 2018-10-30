package encoding

//
// This file implemented a simple way of OPE (Order Preserving Encoding).
//
// OPE preserves dictionary order of origin values, which means that, given two values x, y of same type,
// compare(x, y) == compare(OPE(x), OPE(y)) is always true. OPE is useful in database key encoding, where
// a key is an encoded list of values, and keys order must be the same as original non-encoded lists.
//
// Order preserving is the only requirement for OPE algorithms. When a data type has a primary key combined
// by some fields, it's safe to encode primary key only. So that OPE can be data loss.
//
// Here we present a OPE method for integers, floats, strings, slices and custom user types,
// 1, numbers: big-endian memory bytes with some bit flips
// 2, strings: memory bytes are ok
// 3, user types: OpeEncoder interface must be implemented
// 4, arrays and slices: join each encoded element with a separator
//

import (
	"reflect"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
)


// User types must implement OpeEncoder interface if they decide to support OPE.
// Encode(value) will call OpeEncode() if the given value type implemented OpeEncoder interface.
type OpeEncoder interface {
	OpeEncode() ([]byte, error)
}

// bigEndianBytes returns memory bytes of the given value in big-endian byte order.
func bigEndianBytes(value interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, value)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// encodeSignedInteger is a generic encoder for any signed integer values.
func encodeSignedInteger(value interface{}) ([]byte, error) {
	data, err := bigEndianBytes(value)
	if err == nil {
		data[0] ^= 0x80
	}
	return data, err
}

func encodeUnsignedInteger(value interface{}) ([]byte, error) {
	// memory bytes reserve the order of unsigned integers
	return bigEndianBytes(value)
}

func encodeFloat(value interface{}) ([]byte, error) {
	data, err := bigEndianBytes(value)
	if err == nil {
		// 0 has two legal IEEE754 encodings: +0 and -0.
		// We choose +0 as the only valid one, and convert -0 to +0 if found.
		// This avoid the result of +0 > -0 when comparing.
		isZero := true
		for _, b := range data[1:] {
			if b != 0 {
				isZero = false
				break
			}
		}
		if isZero = isZero && (data[0] & 0x7f) == 0; isZero {
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
	return data, err
}

const host32bit = ^uint(0) >> 32 == 0

// Encode a signed integer
func EncodeInt(value int) ([]byte, error) {
	if host32bit {
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
	if host32bit {
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
	return value, nil
}

// escape() replaces "\x00" with "\x00\x01".
// two escape()'ed slices remain their original dictionary order without containing "\x00\x00".
// "\x00\x00" is the list separator, as it's safe to be inserted between escape()'ed elements.
func escape(src []byte) ([]byte, error) {
	buf := new(bytes.Buffer)
	var err error
	one := uint8(1)
	for _, b := range src {
		if err = binary.Write(buf, binary.LittleEndian, b); err != nil {
			return nil, err
		}
		if b == 0 {
			if err = binary.Write(buf, binary.LittleEndian, one); err != nil {
				return nil, err
			}
		}
	}
	return buf.Bytes(), nil
}

// Encode a list of values
func EncodeSlice(value []interface{}, fixedElemSize bool) ([]byte, error) {
	children := make([][]byte, len(value))
	var err error
	var d, sep []byte
	if fixedElemSize {
		sep = []byte {}
	} else {
		sep = []byte {0, 0}
	}
	for i, c := range value {
		if d, err = Encode(c); err != nil {
			return nil, err
		}
		if !fixedElemSize {
			if d, err = escape(d); err != nil {
				return nil, err
			}
		}
		children[i] = d
	}
	return bytes.Join(children, sep), nil
}

// Encode a OpeEncoder
func EncodeOpeEncoder(value OpeEncoder) ([]byte, error) {
	return value.OpeEncode()
}

// Encode a big integer
func EncodeBigInt(value big.Int) ([]byte, error) {
	data := value.Bytes()
	size, _ := EncodeUint32(uint32(len(data)))
	sign := uint8(1)
	data = bytes.Join([][]byte{ {sign}, size, data}, []byte{})
	if value.Cmp(big.NewInt(0)) < 0 {
		for i:=1; i < len(data); i++ {
			data[i] ^= 0xff
		}
		data[0] = 0
	}
	return data, nil
}


var opeEncoderInterfaceType = reflect.TypeOf((*OpeEncoder)(nil)).Elem()

// Encode a reflected value
func Encode(value interface{}) ([]byte, error) {
	// basic data types
	switch value.(type) {
	case bool:
		return EncodeBool(value.(bool))
	case int:
		return EncodeInt(value.(int))
	case int8:
		return EncodeInt8(value.(int8))
	case int16:
		return EncodeInt16(value.(int16))
	case int32:
		return EncodeInt32(value.(int32))
	case int64:
		return EncodeInt64(value.(int64))
	case uint:
		return EncodeUint(value.(uint))
	case uint8:
		return EncodeUint8(value.(uint8))
	case uint16:
		return EncodeUint16(value.(uint16))
	case uint32:
		return EncodeUint32(value.(uint32))
	case uint64:
		return EncodeUint64(value.(uint64))
	case uintptr:
		return EncodeUintPtr(value.(uintptr))
	case float32:
		return EncodeFloat32(value.(float32))
	case float64:
		return EncodeFloat64(value.(float64))
	case string:
		return EncodeString(value.(string))
	case []byte:
		return EncodeBytes(value.([]byte))
	case big.Int:
		return EncodeBigInt(value.(big.Int))
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Invalid {
		return nil, errors.New("ope: cannot encode nil value")
	}
	// check if OpeEncoder interface implemented
	if rv.Type().Implements(opeEncoderInterfaceType) {
		return EncodeOpeEncoder(value.(OpeEncoder))
	}

	switch rv.Kind() {
	case reflect.Ptr:
		if rv.IsNil() {
			return nil, errors.New("ope: cannot encode a nil pointer")
		}
		// try to encode the value pointed by the pointer
		return Encode(reflect.Indirect(rv).Interface())
	case reflect.Slice, reflect.Array:
		// make a []interface{}, and call EncodeSlice()
		size := rv.Len()
		elements := make([]interface{}, size)
		for i := 0; i < size; i++ {
			elements[i] = rv.Index(i).Interface()
		}
		fixedElementSize := false
		switch rv.Type().Elem().Kind() {
		case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Uintptr, reflect.Float32, reflect.Float64:
				fixedElementSize = true
		}

		return EncodeSlice(elements, fixedElementSize)
	}

	return nil, errors.New(fmt.Sprintf("ope: cannot encode values of type %T", value))
}
