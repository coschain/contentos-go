package vme

import (
	"fmt"
	"reflect"
)

func BoolType() reflect.Type {
	return reflect.TypeOf(false)
}
func Int8Type() reflect.Type {
	return reflect.TypeOf(int8(0))
}
func Int16Type() reflect.Type {
	return reflect.TypeOf(int16(0))
}
func Int32Type() reflect.Type {
	return reflect.TypeOf(int32(0))
}
func Int64Type() reflect.Type {
	return reflect.TypeOf(int64(0))
}
func IntType() reflect.Type {
	return reflect.TypeOf(int(0))
}
func Uint8Type() reflect.Type {
	return reflect.TypeOf(uint8(0))
}
func Uint16Type() reflect.Type {
	return reflect.TypeOf(uint16(0))
}
func Uint32Type() reflect.Type {
	return reflect.TypeOf(uint32(0))
}
func Uint64Type() reflect.Type {
	return reflect.TypeOf(uint64(0))
}
func UintType() reflect.Type {
	return reflect.TypeOf(uint(0))
}
func Float32Type() reflect.Type {
	return reflect.TypeOf(float32(0))
}
func Float64Type() reflect.Type {
	return reflect.TypeOf(float64(0))
}
func StringType() reflect.Type {
	return reflect.TypeOf("")
}
func BytesType() reflect.Type {
	return reflect.TypeOf([]byte{})
}

func StructOf(fieldTypes...reflect.Type) reflect.Type {
	count := len(fieldTypes)
	fs := make([]reflect.StructField, count)
	for i := range fs {
		fs[i] = reflect.StructField{
			Name: fmt.Sprintf("Field_%d", i),
			Type: fieldTypes[i],
		}
	}
	return reflect.StructOf(fs)
}

func StructValue(fieldValues...interface{}) interface{} {
	count := len(fieldValues)
	types := make([]reflect.Type, count)
	for i := range types {
		types[i] = reflect.TypeOf(fieldValues[i])
	}
	val := reflect.New(StructOf(types...)).Elem()
	for i := 0; i < count; i++ {
		val.Field(i).Set(reflect.ValueOf(fieldValues[i]))
	}
	return val.Interface()
}
