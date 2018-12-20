package vme

import (
	"encoding/binary"
	"errors"
	"github.com/coschain/contentos-go/common"
	"reflect"
	"unsafe"
)

func decodeBool(data []byte) (bool, int) {
	if len(data) >= 1 {
		return data[0] != 0, 1
	}
	return false, 0
}

func decodeInt8(data []byte) (int8, int) {
	if len(data) >= 1 {
		return int8(data[0]), 1
	}
	return 0, 0
}

func decodeInt16(data []byte) (int16, int) {
	if len(data) >= 2 {
		return int16(common.HostByteOrder().Uint16(data)), 2
	}
	return 0, 0
}

func decodeInt32(data []byte) (int32, int) {
	if len(data) >= 4 {
		return int32(common.HostByteOrder().Uint32(data)), 4
	}
	return 0, 0
}

func decodeInt64(data []byte) (int64, int) {
	if len(data) >= 8 {
		return int64(common.HostByteOrder().Uint64(data)), 8
	}
	return 0, 0
}

func decodeInt(data []byte) (int, int) {
	if common.Is32bitPlatform {
		if v, n := decodeInt32(data); n > 0 {
			return int(v), n
		} else {
			return 0, 0
		}
	} else {
		if v, n := decodeInt64(data); n > 0 {
			return int(v), n
		} else {
			return 0, 0
		}
	}
}

func decodeUint8(data []byte) (uint8, int) {
	if len(data) >= 1 {
		return data[0], 1
	}
	return 0, 0
}

func decodeUint16(data []byte) (uint16, int) {
	if len(data) >= 2 {
		return common.HostByteOrder().Uint16(data), 2
	}
	return 0, 0
}

func decodeUint32(data []byte) (uint32, int) {
	if len(data) >= 4 {
		return common.HostByteOrder().Uint32(data), 4
	}
	return 0, 0
}

func decodeUint64(data []byte) (uint64, int) {
	if len(data) >= 8 {
		return common.HostByteOrder().Uint64(data), 8
	}
	return 0, 0
}

func decodeUint(data []byte) (uint, int) {
	if common.Is32bitPlatform {
		if v, n := decodeUint32(data); n > 0 {
			return uint(v), n
		} else {
			return 0, 0
		}
	} else {
		if v, n := decodeUint64(data); n > 0 {
			return uint(v), n
		} else {
			return 0, 0
		}
	}
}

func decodeFloat32(data []byte) (float32, int) {
	if len(data) >= 4 {
		x := common.HostByteOrder().Uint32(data)
		return *(*float32)(unsafe.Pointer(&x)), 4
	}
	return 0, 0
}

func decodeFloat64(data []byte) (float64, int) {
	if len(data) >= 8 {
		x := common.HostByteOrder().Uint64(data)
		return *(*float64)(unsafe.Pointer(&x)), 8
	}
	return 0, 0
}

func decodeCount(data []byte) (uint64, int) {
	return binary.Uvarint(data)
}

func decodeString(data []byte) (string, int) {
	count, offset := decodeCount(data)
	if offset <= 0 {
		return "", 0
	}
	strSize, dataSize := int(count), len(data)
	if offset < dataSize && strSize > 0 && strSize < dataSize && offset + strSize <= dataSize {
		return string(data[offset:offset + strSize]), offset + strSize
	} else {
		return "", 0
	}
}

func decodeBytes(data []byte) ([]byte, int) {
	s, n := decodeString(data)
	if n > 0 {
		return []byte(s), n
	} else {
		return nil, 0
	}
}

func decodeStruct(data []byte, typ reflect.Type) (interface{}, int) {
	count, offset := decodeCount(data)
	if offset <= 0 {
		return nil, 0
	}
	size := int(count)
	rv := reflect.New(typ).Elem()
	for i := 0; i < size; i++ {
		dv, n := decodeValue(data[offset:], typ.Field(i).Type)
		if n <= 0 {
			return nil, n
		}
		rv.Field(i).Set(reflect.ValueOf(dv))
		offset += n
	}
	return rv.Interface(), offset
}

func decodeValue(data []byte, typ reflect.Type) (interface{}, int) {
	switch typ.Kind() {
	case reflect.Bool:
		return decodeBool(data)
	case reflect.Int:
		return decodeInt(data)
	case reflect.Int8:
		return decodeInt8(data)
	case reflect.Int16:
		return decodeInt16(data)
	case reflect.Int32:
		return decodeInt32(data)
	case reflect.Int64:
		return decodeInt64(data)
	case reflect.Uint:
		return decodeUint(data)
	case reflect.Uint8:
		return decodeUint8(data)
	case reflect.Uint16:
		return decodeUint16(data)
	case reflect.Uint32:
		return decodeUint32(data)
	case reflect.Uint64:
		return decodeUint64(data)
	case reflect.Float32:
		return decodeFloat32(data)
	case reflect.Float64:
		return decodeFloat64(data)
	case reflect.String:
		return decodeString(data)
	case reflect.Slice:
		et := typ.Elem()
		if et.Kind() == reflect.Uint8 {
			return decodeBytes(data)
		}
	case reflect.Struct:
		return decodeStruct(data, typ)
	}
	return nil, -1
}

func decodingError(n int, typ reflect.Type) error {
	if n == 0 {
		return errors.New("vme: Invalid encoded data.")
	} else if n < 0 {
		return errors.New("vme: Unsupported data type: " + typ.String())
	}
	return nil
}

func Decode(data []byte, outPtr interface{}) error {
	out := reflect.ValueOf(outPtr)
	if out.Kind() != reflect.Ptr {
		return errors.New("vme: Decode() needs an output pointer.")
	}
	dt := out.Type().Elem()
	dv, n := decodeValue(data, dt)
	if n > 0 {
		out.Set(reflect.ValueOf(dv).Addr())
	}
	return decodingError(n, dt)
}

func DecodeWithType(data []byte, valueType reflect.Type) (interface{}, error) {
	dv, n := decodeValue(data, valueType)
	return dv, decodingError(n, valueType)
}
