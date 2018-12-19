package vme

import (
	"encoding/json"
	"fmt"
	"github.com/coschain/contentos-go/common"
	"github.com/pkg/errors"
	"reflect"
	"strings"
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
	return Encode(args...)
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

func TypeSignatureEncode(targetTypes []reflect.Type) (string, error) {
	sigs := []string{}
	for _, t := range targetTypes {
		if s, err := typeSigEncode(t); err != nil {
			return "", err
		} else {
			sigs = append(sigs, s)
		}
	}
	return strings.Join(sigs, ";"), nil
}

func TypeSignatureDecode(sig string) ([]reflect.Type, error) {
	sigs := strings.Split(sig, ";")
	types := []reflect.Type{}
	for _, s := range sigs {
		if t, err := typeSigDecode(s); err != nil {
			return nil, err
		} else {
			types = append(types, t)
		}
	}
	return types, nil
}

func typeSigEncode(typ reflect.Type) (string, error) {
	sig := []string{}
	t := typ
	for {
		k := t.Kind()
		s, ok := type2sig[k]
		if ok {
			sig = append(sig, s)
			break
		}
		if k == reflect.Slice || k == reflect.Array {
			sig = append(sig, "[")
			t = t.Elem()
		} else {
			return "", errors.New("type signature doesn't support type: " + typ.String())
		}
	}
	return strings.Join(sig, ""), nil
}

func typeSigDecode(sig string) (reflect.Type, error) {
	sliceDepth := 0
	for strings.HasPrefix(sig, "[") {
		sig = sig[1:]
		sliceDepth++
	}
	typ, ok := sig2type[sig[:1]]
	if !ok {
		return nil, errors.New("invalid type signature.")
	}
	for sliceDepth > 0 {
		typ = reflect.SliceOf(typ)
		sliceDepth--
	}
	return typ, nil
}

var type2sig = map[reflect.Kind]string {
	reflect.Bool:		"Z",
	reflect.Int8:		"b",
	reflect.Int16:		"w",
	reflect.Int32:		"d",
	reflect.Int64:		"q",
	reflect.Uint8:		"B",
	reflect.Uint16:		"W",
	reflect.Uint32:		"D",
	reflect.Uint64:		"Q",
	reflect.Float32:	"f",
	reflect.Float64:	"F",
	reflect.String:		"s",
}

var sig2type = map[string]reflect.Type {
	"Z":	reflect.TypeOf(false),
	"b":	reflect.TypeOf(int8(0)),
	"w":	reflect.TypeOf(int16(0)),
	"d":	reflect.TypeOf(int32(0)),
	"q":	reflect.TypeOf(int64(0)),
	"B":	reflect.TypeOf(uint8(0)),
	"W":	reflect.TypeOf(uint16(0)),
	"D":	reflect.TypeOf(uint32(0)),
	"Q":	reflect.TypeOf(uint64(0)),
	"f":	reflect.TypeOf(float32(0)),
	"F":	reflect.TypeOf(float64(0)),
	"s":	reflect.TypeOf(""),
}

func init() {
	if common.Is32bitPlatform {
		type2sig[reflect.Int] = type2sig[reflect.Int32]
		type2sig[reflect.Uint] = type2sig[reflect.Uint32]
	} else {
		type2sig[reflect.Int] = type2sig[reflect.Int64]
		type2sig[reflect.Uint] = type2sig[reflect.Uint64]
	}
	type2sig[reflect.Uintptr] = type2sig[reflect.Uint]
}
