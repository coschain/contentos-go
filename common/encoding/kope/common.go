package kope

import (
	"bytes"
	"encoding/binary"
)

// type marks
const (
	typeInvalid = byte(iota)
	typeMin
	typeNormal
	typeMax = ^typeMin
)

// extended alphabet
const (
	extListBegin = byte(0x01)
	extListEnd   = byte(0x02)
	extSeparator = byte(0x80)
	extZero      = byte(0xff)
)

var (
	MinimalKey        = &minKeyPlaceholder
	MaximumKey        = &maxKeyPlaceholder
	minKeyPlaceholder = "I'm lesser than all real keys"
	maxKeyPlaceholder = "I'm greater than all real keys"
	separator         = []byte{0, extSeparator}
	listBegin         = []byte{0, extListBegin}
	listEnd           = []byte{0, extListEnd}
)

func packString(s []byte) []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(typeNormal)
	for _, b := range s {
		buf.WriteByte(b)
		if b == 0 {
			buf.WriteByte(extZero)
		}
	}
	return buf.Bytes()
}

func packList(e [][]byte) []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(typeNormal)
	buf.Write(listBegin)
	buf.Write(bytes.Join(e, separator))
	buf.Write(listEnd)
	return buf.Bytes()
}

func PackList(e [][]byte) []byte{
	return packList(e)
}

func unpackList(d []byte) [][]byte {
	s := len(d)
	r := [][]byte{}
	if s >= 5 && d[0] == typeNormal && bytes.HasPrefix(d[1:], listBegin) && bytes.HasSuffix(d, listEnd) {
		p, last, depth, begin := 0, -1, 0, -1
		for p < s {
			c := d[p]
			if last == 0 {
				if c == extListBegin {
					depth++
					if depth == 1 {
						begin = p + 1
					}
				} else if c == extListEnd {
					depth--
					if depth == 0 {
						r = append(r, d[begin:p-1])
						begin = -1
					}
				} else if c == extSeparator {
					if depth == 1 {
						r = append(r, d[begin:p-1])
						begin = p + 1
					}
				}
			}
			last = int(c)
			p++
		}
	}
	return r
}

func catLists(lists [][]byte) []byte {
	e := [][]byte{}
	for _, lst := range lists {
		e = append(e, unpackList(lst)...)
	}
	return packList(e)
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
		return packString(data), nil
	}
	return nil, err
}

func encodeUnsignedInteger(value interface{}) ([]byte, error) {
	data, err := bigEndianBytes(value)
	if err == nil {
		return packString(data), nil
	}
	return nil, err
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
		return packString(data), nil
	}
	return nil, err
}

func encodeMinMaxKey(min bool) ([]byte, error) {
	p := uint8(typeMin)
	if !min {
		p = typeMax
	}
	return []byte{p}, nil
}

func encodeSlice(encodedElements [][]byte) ([]byte, error) {
	return packList(encodedElements), nil
}

func encodeNil() ([]byte, error) {
	return []byte{typeNormal}, nil
}
