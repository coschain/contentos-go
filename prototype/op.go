package prototype

import (
	"fmt"
	"reflect"
	"regexp"
)

type opRegistry struct {
	Name string
	Type reflect.Type
	Wrapper reflect.Type
	WrapperPtr reflect.Type
	WrapperField int
	Meta map[string]interface{}
}

const (
	operationOpField = "Op"
	operationGetterPattern = "GetOp\\d+"
)
var (
	sBaseOperationType = reflect.TypeOf((*BaseOperation)(nil)).Elem()
	sOperationPtrType = reflect.TypeOf((*Operation)(nil))
	sOperationType = sOperationPtrType.Elem()
	sOperationOpField, _ = sOperationType.FieldByName(operationOpField)
	sOperationGetters = make(map[reflect.Type]int)
	sRegisteredOps = make(map[reflect.Type]opRegistry)
	sRegisteredOpsByWrapper = make(map[reflect.Type]opRegistry)
)

func registerOperation(name string, wrapperPtr interface{}, opPtr interface{}) {
	wrapperPtrType := reflect.TypeOf(wrapperPtr)
	opPtrType := reflect.TypeOf(opPtr)

	if opPtrType.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("operation type '%s' is not a pointer type", opPtrType.Name()))
	}
	if !opPtrType.ConvertibleTo(sBaseOperationType) {
		panic(fmt.Sprintf("operation type '%s' must implement '%s'", opPtrType.Name(), sBaseOperationType.Name()))
	}
	if _, found := sOperationGetters[opPtrType]; !found {
		panic(fmt.Sprintf("unknown operation type '%s'", opPtrType.Name()))
	}
	if wrapperPtrType.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("wrapper type '%s' is not a pointer type", wrapperPtrType.Name()))
	}
	if !wrapperPtrType.ConvertibleTo(sOperationOpField.Type) {
		panic(fmt.Sprintf("wrapper type '%s' is not convertible to '%s'", wrapperPtrType.Name(), sOperationOpField.Type.Name()))
	}
	wrapperType := wrapperPtrType.Elem()
	if wrapperType.Kind() != reflect.Struct {
		panic(fmt.Sprintf("wrapper type '%s' is not a struct pointer type", wrapperPtrType.Name()))
	}
	wrapperField := -1
	for i := 0; i < wrapperType.NumField(); i++ {
		field := wrapperType.Field(i)
		if field.Type == opPtrType {
			wrapperField = i
			break
		}
	}
	if wrapperField < 0 {
		panic(fmt.Sprintf("cannot find a field of type '%s' in wrapper type '%s'", opPtrType, wrapperType))
	}
	entry := opRegistry{
		Name: name,
		Type: opPtrType,
		Wrapper: wrapperType,
		WrapperPtr: wrapperPtrType,
		WrapperField: wrapperField,
		Meta: make(map[string]interface{}),
	}
	sRegisteredOps[entry.Type] = entry
	sRegisteredOpsByWrapper[entry.WrapperPtr] = entry
}

func toGenericOperation(opPtr interface{}) *Operation {
	if entry, ok := sRegisteredOps[reflect.TypeOf(opPtr)]; ok {
		wrapperPtr := reflect.New(entry.Wrapper)
		wrapperPtr.Elem().Field(entry.WrapperField).Set(reflect.ValueOf(opPtr))
		op := reflect.New(sOperationType)
		op.Elem().Field(sOperationOpField.Index[0]).Set(wrapperPtr.Convert(sOperationOpField.Type))
		return op.Interface().(*Operation)
	}
	return nil
}

func fromGenericOperation(generic *Operation) interface{} {
	opField := reflect.ValueOf(generic).Elem().Field(sOperationOpField.Index[0]).Elem()
	if entry, ok := sRegisteredOpsByWrapper[opField.Type()]; ok {
		return opField.Elem().Field(entry.WrapperField).Interface()
	}
	return nil
}

func fromGenericOperationWithType(generic *Operation, opPtr interface{}) bool {
	if getter, ok := sOperationGetters[reflect.TypeOf(opPtr)]; ok {
		value := reflect.ValueOf(generic).Method(getter).Call(nil)[0].Elem()
		reflect.ValueOf(opPtr).Elem().Set(value)
		return true
	}
	return false
}

func getOperationProp(opPtr interface{}, propGetter func (opRegistry) interface{}) interface{} {
	if entry, ok := sRegisteredOps[reflect.TypeOf(opPtr)]; ok {
		return propGetter(entry)
	}
	return nil
}

func getGenericOperationProp(generic *Operation, propGetter func (opRegistry) interface{}) interface{} {
	opField := reflect.ValueOf(generic).Elem().Field(sOperationOpField.Index[0]).Elem()
	if entry, ok := sRegisteredOpsByWrapper[opField.Type()]; ok {
		return propGetter(entry)
	}
	return nil
}

func GetOperationName(opPtr interface{}) string {
	prop := getOperationProp(opPtr, func(e opRegistry) interface{} {
		return e.Name
	})
	if name, ok := prop.(string); ok {
		return name
	}
	return ""
}

func GetGenericOperationName(generic *Operation) string {
	prop := getGenericOperationProp(generic, func(e opRegistry) interface{} {
		return e.Name
	})
	if name, ok := prop.(string); ok {
		return name
	}
	return ""
}

func RegisterOperationMeta(opPtr interface{}, key string, value interface{}) {
	if entry, ok := sRegisteredOps[reflect.TypeOf(opPtr)]; ok {
		entry.Meta[key] = value
	} else {
		panic(fmt.Sprintf("unknown operation type '%T'", opPtr))
	}
}

func GetOperationMeta(opPtr interface{}, key string) interface{} {
	if entry, ok := sRegisteredOps[reflect.TypeOf(opPtr)]; ok {
		return entry.Meta[key]
	} else {
		panic(fmt.Sprintf("unknown operation type '%T'", opPtr))
	}
}

func GetGenericOperationMeta(generic *Operation, key string) interface{} {
	return getGenericOperationProp(generic, func(e opRegistry) interface{} {
		return e.Meta[key]
	})
}

func init() {
	for i := 0; i < sOperationPtrType.NumMethod(); i++ {
		method := sOperationPtrType.Method(i)
		if nameOK, _ := regexp.MatchString(operationGetterPattern, method.Name); !nameOK {
			continue
		}
		if method.Type.NumOut() != 1 || method.Type.Out(0).Kind() != reflect.Ptr {
			continue
		}
		sOperationGetters[method.Type.Out(0)] = method.Index
	}
}
