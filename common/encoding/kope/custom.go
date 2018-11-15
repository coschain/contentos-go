package kope

import (
	"bytes"
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
			typeHash, _ := EncodeUint64(h)
			return pack(typeCustom, true, bytes.Join([][]byte{typeHash, data}, separator))
		}
		return nil, err
	}
}

func makeDecoder(dec CustomDecoderFunc) CustomDecoderFunc {
	return func(e []byte) (interface{}, error) {
		return dec(e)
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

func customDecode(enc []byte) (interface{}, error) {
	_, typ, data, err := unpack(enc)
	if err != nil {
		return nil, err
	}
	if typ != typeCustom {
		return nil, errors.New("invalid encoded data")
	}
	if pos := bytes.Index(data, separator); pos > 0 {
		h, err := DecodeUint64(data[:pos])
		if err == nil {
			if codec := customCodecByHash(h); codec != nil {
				return codec.decoder(data[pos + len(separator):])
			}
		}
	}
	return nil, errors.New("invalid encoded data")
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
