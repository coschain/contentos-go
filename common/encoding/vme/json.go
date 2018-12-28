package vme

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
)

func EncodeFromJson(jsonBytes []byte, valueType reflect.Type) ([]byte, error) {
	if val, err := jsonUnmarshalWithType(jsonBytes, valueType); err != nil {
		return nil, err
	} else {
		return Encode(val)
	}
}

func jsonUnmarshalWithType(jsonBytes []byte, valueType reflect.Type) (interface{}, error) {
	var jsonVal interface{}
	if err := json.Unmarshal(jsonBytes, &jsonVal); err != nil {
		return nil, err
	}
	return fromJsonValue(reflect.ValueOf(jsonVal), valueType)
}

func fromJsonValue(jval reflect.Value, typ reflect.Type) (interface{}, error) {
	jt := jval.Type()
	if jt.ConvertibleTo(typ) {
		return jval.Convert(typ).Interface(), nil
	}
	if jt.Kind() == reflect.Slice && jt.Elem().Kind() == reflect.Interface {
		count := jval.Len()
		jslice := jval.Interface().([]interface{})

		if typ.Kind() == reflect.Struct && count == typ.NumField() {
			v := reflect.New(typ).Elem()
			for i := 0; i < count; i++ {
				fv, err := fromJsonValue(reflect.ValueOf(jslice[i]), typ.Field(i).Type)
				if err != nil {
					return nil, errors.New("vme-json: incompatible json.")
				}
				v.Field(i).Set(reflect.ValueOf(fv))
			}
			return v.Interface(), nil
		}
		if typ.Kind() == reflect.Slice {
			et := typ.Elem()
			v := reflect.MakeSlice(typ, count, count)
			for i := 0; i < count; i++ {
				ev, err := fromJsonValue(reflect.ValueOf(jslice[i]), et)
				if err != nil {
					return nil, errors.New("vme-json: incompatible json.")
				}
				v.Index(i).Set(reflect.ValueOf(ev))
			}
			return v.Interface(), nil
		}
	}
	return nil, errors.New("vme-json: incompatible json.")
}

func DecodeToJson(data []byte, valueType reflect.Type, compact bool) ([]byte, error) {
	if val, err := DecodeWithType(data, valueType); err != nil {
		return nil, err
	} else {
		js, err := jsonMarshal(val)
		if err == nil && compact {
			buf := new(bytes.Buffer)
			if err = json.Compact(buf, js); err == nil {
				js = buf.Bytes()
			}
		}
		return js, err
	}
}

func jsonMarshal(value interface{}) ([]byte, error) {
	return json.Marshal(toJsonValue(reflect.ValueOf(value)))
}

func toJsonValue(value reflect.Value) interface{} {
	if value.Kind() == reflect.Struct {
		jval := make([]interface{}, value.NumField())
		for i := range jval {
			jval[i] = toJsonValue(value.Field(i))
		}
		return jval
	}
	if value.Kind() == reflect.Slice {
		jval := make([]interface{}, value.Len())
		for i := range jval {
			jval[i] = toJsonValue(value.Index(i))
		}
		return jval
	}
	return value.Interface()
}
