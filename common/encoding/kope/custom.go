package kope

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc64"
	"reflect"
	"sync"
)

var (
	registeredCodecs sync.Map
	registeredTypeIndex sync.Map
)

type CustomEncoderFunc func(reflect.Value) ([]byte, error)
type CustomDecoderFunc func([]byte) (interface{}, error)

type CustomCodec struct {
	rt reflect.Type
	name string
	encoder CustomEncoderFunc
	decoder CustomDecoderFunc
}

func makeEncoder(enc CustomEncoderFunc, h uint64) CustomEncoderFunc {
	return func(rv reflect.Value) ([]byte, error) {
		data, err := enc(rv)
		if err == nil {
			hashBytes, _ := bigEndianBytes(h)
			return pack(typeCustom, true, bytes.Join([][]byte{hashBytes, data}, nil))
		}
		return nil, err
	}
}

func makeDecoder(dec CustomDecoderFunc) CustomDecoderFunc {
	return func(e []byte) (interface{}, error) {
		typ, data, err := unpack(e)
		if err == nil && typ == typeCustom && len(data) >= 8 {
			return dec(data[8:])
		}
		return nil, errors.New("invalid encoded data")
	}
}

func defaultDecoderFunc(enc []byte) (interface{}, error) {
	return enc, nil
}

func customCodecByType(rt reflect.Type) *CustomCodec {
	if c, ok := registeredCodecs.Load(rt); ok {
		cc := c.(CustomCodec)
		return &cc
	}
	return nil
}

func customCodecByHash(h uint64) *CustomCodec {
	if rt, ok := registeredTypeIndex.Load(h); ok {
		return customCodecByType(rt.(reflect.Type))
	}
	return nil
}

func customCodecByEncodedBytes(enc []byte) *CustomCodec {
	if len(enc) < 9 || enc[0] != typeCustom {
		return nil
	}
	return customCodecByHash(binary.BigEndian.Uint64(enc[1:]))
}

func RegisterType(rt reflect.Type, name string, enc CustomEncoderFunc, dec CustomDecoderFunc) {
	if rt == nil || len(name) == 0 || enc == nil {
		return
	}
	if dec == nil {
		dec = defaultDecoderFunc
	}
	h := crc64.Checksum([]byte(name), crc64.MakeTable(crc64.ECMA))
	registeredCodecs.Store(rt, CustomCodec{
		rt: rt,
		name: name,
		encoder: makeEncoder(enc, h),
		decoder: makeDecoder(dec),
	})
	registeredTypeIndex.Store(h, rt)
}
