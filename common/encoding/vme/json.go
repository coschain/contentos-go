package vme

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"reflect"
)

func EncodeJsonArray(jsonStr string, targetTypes []reflect.Type) ([]byte, error) {
	jsonArgs := []interface{}{}
	if err := json.Unmarshal([]byte(jsonStr), &jsonArgs); err != nil {
		return nil, err
	}
	argCount := len(targetTypes)
	if argCount != len(jsonArgs) {
		return nil, errors.New(fmt.Sprintf("vme: incorrect argument count: %d. expecting %d.", len(jsonArgs), argCount))
	}
	args := make([]interface{}, argCount)
	for i, v := range jsonArgs {
		if cv, err := convertJsonArg(v, targetTypes[i]); err != nil {
			return nil, errors.New(fmt.Sprintf("vme: argument #%d: %s", i, err.Error()))
		} else {
			args[i] = cv
		}
	}
	return EncodeMany(args...)
}

func convertJsonArg(src interface{}, dstType reflect.Type) (interface{}, error) {
	srcValue := reflect.ValueOf(src)
	srcType := srcValue.Type()
	if srcType.ConvertibleTo(dstType) {
		return srcValue.Convert(dstType).Interface(), nil
	}
	if srcType.Kind() == reflect.Slice && srcType.Elem().Kind() == reflect.Interface && dstType.Kind() == reflect.Slice {
		count := srcValue.Len()
		dstValue := make([]interface{}, count)
		srcSlice := src.([]interface{})
		for i, v := range srcSlice {
			if d, err := convertJsonArg(v, dstType.Elem()); err != nil {
				return nil, err
			} else {
				dstValue[i] = d
			}
		}
		return dstValue, nil
	}
	return nil, errors.New(fmt.Sprintf("cannot convert %s to %s", srcType.String(), dstType.String()))
}

func EncodeJsonArrayWithTypeSig(jsonStr string, sig string) ([]byte, error) {
	if types, err := TypeSignatureDecode(sig); err != nil {
		return nil, err
	} else {
		return EncodeJsonArray(jsonStr, types)
	}
}
