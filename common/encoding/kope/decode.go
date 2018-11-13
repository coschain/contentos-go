package kope

import (
	"errors"
)

// Decode a signed integer
func DecodeInt(enc []byte) (int, error) {
	if is32bitPlatform {
		value, err := DecodeInt32(enc)
		return int(value), err
	} else {
		value, err := DecodeInt64(enc)
		return int(value), err
	}
}

// Decode a signed 8-bit integer
func DecodeInt8(enc []byte) (int8, error) {
	var value int8
	if err := decodeSignedInteger(enc, &value); err == nil {
		return value, nil
	} else {
		return 0, err
	}
}

// Decode a signed 16-bit integer
func DecodeInt16(enc []byte) (int16, error) {
	var value int16
	if err := decodeSignedInteger(enc, &value); err == nil {
		return value, nil
	} else {
		return 0, err
	}
}

// Decode a signed 32-bit integer
func DecodeInt32(enc []byte) (int32, error) {
	var value int32
	if err := decodeSignedInteger(enc, &value); err == nil {
		return value, nil
	} else {
		return 0, err
	}
}

// Decode a signed 64-bit integer
func DecodeInt64(enc []byte) (int64, error) {
	var value int64
	if err := decodeSignedInteger(enc, &value); err == nil {
		return value, nil
	} else {
		return 0, err
	}
}

// Decode an unsigned integer
func DecodeUint(enc []byte) (uint, error) {
	if is32bitPlatform {
		value, err := DecodeUint32(enc)
		return uint(value), err
	} else {
		value, err := DecodeUint64(enc)
		return uint(value), err
	}
}

// Decode an unsigned 8-bit integer
func DecodeUint8(enc []byte) (uint8, error) {
	var value uint8
	if err := decodeUnsignedInteger(enc, &value); err == nil {
		return value, nil
	} else {
		return 0, err
	}
}

// Decode an unsigned 16-bit integer
func DecodeUint16(enc []byte) (uint16, error) {
	var value uint16
	if err := decodeUnsignedInteger(enc, &value); err == nil {
		return value, nil
	} else {
		return 0, err
	}
}

// Decode an unsigned 32-bit integer
func DecodeUint32(enc []byte) (uint32, error) {
	var value uint32
	if err := decodeUnsignedInteger(enc, &value); err == nil {
		return value, nil
	} else {
		return 0, err
	}
}

// Decode an unsigned 64-bit integer
func DecodeUint64(enc []byte) (uint64, error) {
	var value uint64
	if err := decodeUnsignedInteger(enc, &value); err == nil {
		return value, nil
	} else {
		return 0, err
	}
}

// Decode an unsigned pointer integer
func DecodeUintPtr(enc []byte) (uintptr, error) {
	value, err := DecodeUint(enc)
	return uintptr(value), err
}

// Decode a bool
func DecodeBool(enc []byte) (bool, error) {
	typ, data, err := unpack(enc)
	if err == nil {
		if typ == typeBool && len(data) == 1 {
			return data[0] != 0, nil
		}
		return false, errors.New("invalid encoded data")
	}
	return false, err
}

// Decode a float32
func DecodeFloat32(enc []byte) (float32, error) {
	var value float32
	if err := decodeFloat(enc, &value); err == nil {
		return value, nil
	} else {
		return 0, err
	}
}

// Decode a float64
func DecodeFloat64(enc []byte) (float64, error) {
	var value float64
	if err := decodeFloat(enc, &value); err == nil {
		return value, nil
	} else {
		return 0, err
	}
}

// Decode a string
func DecodeString(enc []byte) (string, error) {
	typ, data, err := unpack(enc)
	if err == nil {
		if typ == typeString {
			return string(data), nil
		}
		return "", errors.New("invalid encoded data")
	}
	return "", err
}

// Decode a byte slice
func DecodeBytes(enc []byte) ([]byte, error) {
	typ, data, err := unpack(enc)
	if err == nil {
		if typ == typeBytes {
			return data, nil
		}
		return nil, errors.New("invalid encoded data")
	}
	return nil, err
}

// Decode a slice
func DecodeSlice(enc []byte) ([]interface{}, error) {
	encElements, err := decodeSlice(enc)
	if err == nil {
		children := make([]interface{}, len(encElements))
		for i, e := range encElements {
			if d, err := Decode(e); err != nil {
				return nil, err
			} else {
				children[i] = d
			}
		}
		return children, nil
	}
	return nil, err
}

func DecodeMinMaxKey(enc []byte) (*string, error) {
	return decodeMinMaxKey(enc)
}

func DecodeCustom(enc []byte) (interface{}, error) {
	if codec := customCodecByEncodedBytes(enc); codec != nil {
		return codec.decoder(enc)
	} else {
		return nil, errors.New("invalid encoded data")
	}
}

func Decode(enc []byte) (interface{}, error) {
	if len(enc) > 0 {
		typ, _ := typeId(enc[0])
		switch typ {
		case typeBool:
			return DecodeBool(enc)
		case typeInt:
			return DecodeInt(enc)
		case typeInt8:
			return DecodeInt8(enc)
		case typeInt16:
			return DecodeInt16(enc)
		case typeInt32:
			return DecodeInt32(enc)
		case typeInt64:
			return DecodeInt64(enc)
		case typeUint:
			return DecodeUint(enc)
		case typeUint8:
			return DecodeUint8(enc)
		case typeUint16:
			return DecodeUint16(enc)
		case typeUint32:
			return DecodeUint32(enc)
		case typeUint64:
			return DecodeUint64(enc)
		case typeUintptr:
			return DecodeUintPtr(enc)
		case typeFloat32:
			return DecodeFloat32(enc)
		case typeFloat64:
			return DecodeFloat64(enc)
		case typeString:
			return DecodeString(enc)
		case typeBytes:
			return DecodeBytes(enc)
		case typeList:
			return DecodeSlice(enc)
		case typeCustom:
			return DecodeCustom(enc)
		case typeMin, typeMax:
			return DecodeMinMaxKey(enc)
		}
	}
	return nil, errors.New("invalid encoded data")
}
