package kope

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/coschain/contentos-go/common"
)

const (
	is32bitPlatform = ^uint(0)>>32 == 0
)

// type marks
const (
	typeInvalid			= byte(iota)
	typeMin
	typeBool
	typeInt
	typeInt8
	typeInt16
	typeInt32
	typeInt64
	typeUint
	typeUint8
	typeUint16
	typeUint32
	typeUint64
	typeUintptr
	typeFloat32
	typeFloat64
	typeString
	typeBytes
	typeList
	typeCustom
	typeMax				= ^typeMin

	typeReversedFlag	= 0x80
)

// extended alphabet
const (
	extSeparator	= byte(0x01)
	extEnding		= byte(0x02)
	extZero			= byte(0xff)
)

var (
	MinimalKey        = &minKeyPlaceholder
	MaximumKey        = &maxKeyPlaceholder
	minKeyPlaceholder = "I'm lesser than all real keys"
	maxKeyPlaceholder = "I'm greater than all real keys"
	separator         = []byte{0, extSeparator}
)

func pack(typ byte, ending bool, src []byte) ([]byte, error) {
	var err error
	buf := new(bytes.Buffer)
	err = buf.WriteByte(typ)
	if err != nil {
		return nil, err
	}
	ez := extZero
	for _, b := range src {
		if err = binary.Write(buf, binary.LittleEndian, b); err != nil {
			return nil, err
		}
		if b == 0 {
			if err = binary.Write(buf, binary.LittleEndian, ez); err != nil {
				return nil, err
			}
		}
	}
	if ending {
		if _, err = buf.Write([]byte{0, extEnding}); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func unpack(src []byte) (bool, byte, []byte, error) {
	srcLen := len(src)
	flip := false
	if srcLen < 1 {
		return flip, typeInvalid, nil, errors.New("packed data too short")
	}
	data := src
	typ := data[0]
	if typ, flip = typeId(src[0]); flip {
		data = make([]byte, len(src))
		for i, b := range src {
			data[i] = ^b
		}
	}
	if bytes.HasSuffix(data, []byte{0, extEnding}) {
		srcLen -= 2
	}
	buf := new(bytes.Buffer)
	lastByte := data[0]
	for _, b := range data[1:srcLen] {
		if !(b == extZero && lastByte == 0) {
			if err := buf.WriteByte(b); err != nil {
				return flip, typeInvalid, nil, err
			}
		}
		lastByte = b
	}
	return flip, typ, buf.Bytes(), nil
}

func bigEndianBytes(value interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, value)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodeSignedInteger(value interface{}) ([]byte, error) {
	data, err := bigEndianBytes(value)
	if err == nil {
		data[0] ^= 0x80
		var t byte
		switch len(data) {
		case 1:
			t = typeInt8
		case 2:
			t = typeInt16
		case 4:
			t = typeInt32
		case 8:
			t = typeInt64
		}
		return pack(t, false, data)
	}
	return nil, err
}

func decodeSignedInteger(enc []byte, valuePtr interface{}) error {
	_, typ, data, err := unpack(enc)
	if err == nil {
		if typ == typeInt8 || typ == typeInt16 || typ == typeInt32 || typ == typeInt64 || typ == typeInt {
			e := common.CopyBytes(data)
			e[0] ^= 0x80
			return binary.Read(bytes.NewBuffer(e), binary.BigEndian, valuePtr)
		} else {
			return errors.New("invalid encoded data")
		}
	}
	return err
}

func encodeUnsignedInteger(value interface{}) ([]byte, error) {
	data, err := bigEndianBytes(value)
	if err == nil {
		var t byte
		switch len(data) {
		case 1:
			t = typeUint8
		case 2:
			t = typeUint16
		case 4:
			t = typeUint32
		case 8:
			t = typeUint64
		}
		return pack(t, false, data)
	}
	return nil, err
}

func decodeUnsignedInteger(enc []byte, valuePtr interface{}) error {
	_, typ, data, err := unpack(enc)
	if err == nil {
		if typ == typeUint8 || typ == typeUint16 || typ == typeUint32 || typ == typeUint64 || typ == typeUint || typ == typeUintptr {
			return binary.Read(bytes.NewBuffer(data), binary.BigEndian, valuePtr)
		} else {
			return errors.New("invalid encoded data")
		}
	}
	return err
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
		t := byte(typeFloat32)
		if len(data) == 8 {
			t = typeFloat64
		}
		return pack(t, false, data)
	}
	return nil, err
}

func decodeFloat(enc []byte, valuePtr interface{}) error {
	_, typ, data, err := unpack(enc)
	if err == nil {
		if typ == typeFloat32 || typ == typeFloat64 {
			e := common.CopyBytes(data)
			if negative := (e[0] & 0x80) == 0; negative {
				for i, b := range e {
					e[i] = ^b
				}
			} else {
				e[0] ^= 0x80
			}
			return binary.Read(bytes.NewBuffer(e), binary.BigEndian, valuePtr)
		} else {
			return errors.New("invalid encoded data")
		}
	}
	return err
}

func encodeMinMaxKey(min bool) ([]byte, error) {
	p := uint8(typeMin)
	if !min {
		p = typeMax
	}
	return []byte{p}, nil
}

func decodeMinMaxKey(enc []byte) (*string, error) {
	if len(enc) == 1 {
		switch enc[0] {
		case typeMin:
			return MinimalKey, nil
		case typeMax:
			return MaximumKey, nil
		}
	}
	return nil, errors.New("invalid encoded data")
}

func encodeSlice(encodedElements [][]byte) ([]byte, error) {
	return pack(typeList, true, bytes.Join(encodedElements, separator))
}

func decodeSlice(enc []byte) ([][]byte, error) {
	_, typ, data, err := unpack(enc)
	if err == nil {
		if typ == typeList {
			b := 0
			pos := []int{0}
			sepLen := len(separator)
			for {
				p := bytes.Index(data[b:], separator)
				if p < 0 {
					break
				}
				b += p + sepLen
				pos = append(pos, b - sepLen, b)
			}
			pos = append(pos, len(data))

			var result [][]byte
			for i := 0; i < len(pos); i += 2 {
				result = append(result, data[pos[i]:pos[i + 1]])
			}
			return result, nil
		}
		return nil, errors.New("invalid encoded data")
	}
	return nil, err
}

func flipped(data []byte, errs...error) ([]byte, error) {
	var err error
	if len(errs) > 0 {
		err = errs[0]
	}
	if err != nil {
		return nil, err
	}
	if len(data) < 1 {
		return nil, errors.New("invalid data")
	}
	buf := new(bytes.Buffer)
	for _, b := range data {
		if err = buf.WriteByte(^b); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func typeId(typ byte) (byte, bool) {
	if f := (typ & typeReversedFlag) != 0; f {
		return ^typ, f
	} else {
		return typ, f
	}
}
