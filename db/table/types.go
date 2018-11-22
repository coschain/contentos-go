package table

import (
	"github.com/coschain/contentos-go/common/encoding/kope"
	"github.com/golang/protobuf/proto"
	"reflect"
	"sync"
)

type typeInfo struct {
	typ reflect.Type
	name string
	kope bool
}

var (
	registeredTypesByName  	sync.Map
	registeredTypes  		sync.Map
	opeEncoderInterfaceType = reflect.TypeOf((*kope.OpeEncoder)(nil)).Elem()
)

func registerType(name string, t reflect.Type, supportKope bool) {
	if len(name) > 0 && t != nil {
		ti := typeInfo{t, name, supportKope}
		registeredTypes.Store(t, ti)
		registeredTypesByName.Store(name, ti)
	}
}

func RegisterCustomType(i interface{}, name string) {
	t := reflect.TypeOf(i)
	registerType(name, t, t.Implements(opeEncoderInterfaceType))
}

func RegisterProtoType(m proto.Message) {
	t := reflect.TypeOf(m)
	registerType(proto.MessageName(m), t, t.Implements(opeEncoderInterfaceType))
}

func RegisterProtoTypeNamed(name string) {
	if t := proto.MessageType(name); t != nil {
		registerType(name, t, t.Implements(opeEncoderInterfaceType))
	}
}

func typeInfoByType(t reflect.Type) (*typeInfo, bool) {
	if i, ok := registeredTypes.Load(t); ok {
		ti := i.(typeInfo)
		return &ti, true
	}
	return nil, false
}

func typeInfoByName(name string) (*typeInfo, bool) {
	if i, ok := registeredTypesByName.Load(name); ok {
		ti := i.(typeInfo)
		return &ti, true
	}
	return nil, false
}

func RegisteredName(t reflect.Type) string {
	if ti, ok := typeInfoByType(t); ok {
		return ti.name
	}
	return ""
}

func RegisteredType(name string) reflect.Type {
	if ti, ok := typeInfoByName(name); ok {
		return ti.typ
	}
	return nil
}

func registerBuiltinType(i interface{}) {
	t := reflect.TypeOf(i)
	registerType(t.Name(), t, true)
}

func init() {
	registerBuiltinType(false)
	registerBuiltinType(int(0))
	registerBuiltinType(int8(0))
	registerBuiltinType(int16(0))
	registerBuiltinType(int32(0))
	registerBuiltinType(int64(0))
	registerBuiltinType(uint(0))
	registerBuiltinType(uint8(0))
	registerBuiltinType(uint16(0))
	registerBuiltinType(uint32(0))
	registerBuiltinType(uint64(0))
	registerBuiltinType(uintptr(0))
	registerBuiltinType(float32(0))
	registerBuiltinType(float64(0))
	registerBuiltinType("")
}
