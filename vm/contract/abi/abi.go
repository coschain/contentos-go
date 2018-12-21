package abi

import (
	"fmt"
	"github.com/coschain/contentos-go/common/encoding/vme"
	"reflect"
	"strings"
)

type abiBaseType struct {
	name string
	rt reflect.Type
}

func (t *abiBaseType) Name() string {
	return t.name
}

func (t *abiBaseType) Type() reflect.Type {
	return t.rt
}

func (t *abiBaseType) IsStruct() bool {
	return t.rt.Kind() == reflect.Struct
}

type abiStructType struct {
	abiBaseType
	base *abiStructType
	fields []abiStructField
}
type abiStructField struct {
	name string
	typ IContractType
}

func (t *abiStructType) FieldNum() int {
	if t.base != nil {
		return len(t.fields) + 1
	}
	return len(t.fields)
}

func (t *abiStructType) FieldType(i int) IContractType {
	if i >= 0 && i < t.FieldNum() {
		if t.base != nil {
			if i == 0 {
				return t.base
			} else {
				return t.fields[i - 1].typ
			}
		}
		return t.fields[i].typ
	}
	return nil
}

func NewStruct(name string, base *abiStructType, fields...abiStructField) *abiStructType {
	sf := make([]reflect.StructField, 0, len(fields) + 1)
	if base != nil {
		sf = append(sf, reflect.StructField{
			Name: "Base__",
			Type: base.Type(),
			Tag: reflect.StructTag(fmt.Sprintf(`json:"__base__"`)),
		})
	}
	for i := range fields {
		sf = append(sf, reflect.StructField{
			Name: strings.Title(fields[i].name),
			Type: fields[i].typ.Type(),
			Tag: reflect.StructTag(fmt.Sprintf(`json:"%s"`, fields[i].name)),
		})
	}
	return &abiStructType{
		abiBaseType: abiBaseType{ name, reflect.StructOf(sf) },
		fields: fields,
	}
}

type abiMethod struct {
	name string
	args *abiStructType
}

func (m *abiMethod) Name() string {
	return m.name
}

func (m *abiMethod) Args() IContractStruct {
	return m.args
}

type abiTable struct {
	name string
	record *abiStructType
	primary int
	secondary []int
}

func (t *abiTable) Name() string {
	return t.name
}

func (t *abiTable) Record() IContractStruct {
	return t.record
}

func (t *abiTable) PrimaryIndex() int {
	return t.primary
}

func (t *abiTable) SecondaryIndices() []int {
	return t.secondary
}


type abi struct {
	version string
	types []IContractType
	typeByName map[string]int
	typedefs map[string]string
	methods []*abiMethod
	methodByName map[string]int
	tables []*abiTable
	tableByName map[string]int
}

func (abi *abi) TypesCount() int {
	return len(abi.types)
}

func (abi *abi) TypeByIndex(i int) IContractType {
	if i >= 0 && i < abi.TypesCount() {
		return abi.types[i]
	}
	return nil
}

func (abi *abi) TypeByName(name string) IContractType {
	if i, ok := abi.typeByName[name]; ok {
		return abi.TypeByIndex(i)
	}
	return nil
}

func (abi *abi) MethodsCount() int {
	return len(abi.methods)
}

func (abi *abi) MethodByIndex(i int) IContractMethod {
	if i >= 0 && i < abi.MethodsCount() {
		return abi.methods[i]
	}
	return nil
}

func (abi *abi) MethodByName(name string) IContractMethod {
	if i, ok := abi.methodByName[name]; ok {
		return abi.MethodByIndex(i)
	}
	return nil
}

func (abi *abi) TablesCount() int {
	return len(abi.tables)
}

func (abi *abi) TableByIndex(i int) IContractTable {
	if i >= 0 && i < abi.TablesCount() {
		return abi.tables[i]
	}
	return nil
}

func (abi *abi) TableByName(name string) IContractTable {
	if i, ok := abi.tableByName[name]; ok {
		return abi.TableByIndex(i)
	}
	return nil
}

func (abi *abi) Marshal() ([]byte, error) {
	ja := new(JsonABI)
	ja.Version = abi.version
	for _, t := range abi.types {
		if ot, isAlias := abi.typedefs[t.Name()]; isAlias {
			ja.Types = append(ja.Types, jsonAbiTypedef{
				Name: t.Name(),
				Type: ot,
			})
		} else if s, isStruct := t.(*abiStructType); isStruct {
			fs := make([]jsonAbiStructField, len(s.fields))
			for i := range fs {
				fs[i] = jsonAbiStructField{
					Name: s.fields[i].name,
					Type: s.fields[i].typ.Name(),
				}
			}
			ja.Structs = append(ja.Structs, jsonAbiStruct{
				Name: s.name,
				Base: s.base.name,
				Fields: fs,
			})
		}
	}
	for _, m := range abi.methods {
		ja.Methods = append(ja.Methods, jsonAbiMethod{
			Name: m.name,
			Type: m.args.name,
		})
	}
	for _, t := range abi.tables {
		si := make([]string, len(t.secondary))
		for i := range si {
			si[i] = t.record.fields[t.secondary[i]].name
		}
		ja.Tables = append(ja.Tables, jsonAbiTable{
			Name: t.name,
			Type: t.record.name,
			Primary: t.record.fields[t.primary].name,
			Secondary: si,
		})
	}
	return ja.Marshal()
}

func (abi *abi) Unmarshal(data []byte) error {
	ja := new(JsonABI)
	err := ja.Unmarshal(data)
	if err != nil {
		return err
	}
	b := NewAbiBuilder()
	b.Version(abi.version)
	for _, t := range ja.Types {
		b.Typedef(t.Name, t.Type)
	}
	for _, s := range ja.Structs {
		fs := make([]string, len(s.Fields) * 2)
		for i := range s.Fields {
			fs[i * 2] = s.Fields[i].Name
			fs[i * 2 + 1] = s.Fields[i].Type
		}
		b.Struct(s.Name, s.Base, fs...)
	}
	for _, m := range ja.Methods {
		b.Method(m.Name, m.Type)
	}
	for _, t := range ja.Tables {
		b.Table(t.Name, t.Type, t.Primary, t.Secondary...)
	}

	if newAbi, err := b.Build(); err != nil {
		return err
	} else {
		*abi = *newAbi
		return nil
	}
}


//
// built-in definitions
//

var builtinPrimitiveTypes = map[string]IContractType {
	"bool": 	&abiBaseType{ "bool", 	vme.BoolType() },
	"int8":  	&abiBaseType{ "int8", 	vme.Int8Type() },
	"int16":  	&abiBaseType{ "int16", 	vme.Int16Type() },
	"int32":  	&abiBaseType{ "int32", 	vme.Int32Type() },
	"int64":  	&abiBaseType{ "int64", 	vme.Int64Type() },
	"uint8":  	&abiBaseType{ "uint8", 	vme.Uint8Type() },
	"uint16":  	&abiBaseType{ "uint16", 	vme.Uint16Type() },
	"uint32":  	&abiBaseType{ "uint32", 	vme.Uint32Type() },
	"uint64":  	&abiBaseType{ "uint64", 	vme.Uint64Type() },
	"float":  	&abiBaseType{ "float", 	vme.Float32Type() },
	"double":  	&abiBaseType{ "double", 	vme.Float64Type() },

	"string":  	&abiBaseType{ "string", 	vme.StringType() },
	"cosio::account_name": &abiBaseType{ "string", vme.StringType() },
}

var builtinClasses = map[string]IContractType {
	"string":  	&abiBaseType{ "string", 	vme.StringType() },
	"cosio::account_name": &abiBaseType{ "cosio::account_name", vme.StringType() },
}

func ABIBuiltinPrimitiveType(name string) IContractType {
	if t, ok := builtinPrimitiveTypes[name]; ok {
		return t
	}
	return nil
}

func ABIBuiltinType(name string) IContractType {
	if t := ABIBuiltinPrimitiveType(name); t != nil {
		return t
	}
	if t, ok := builtinClasses[name]; ok {
		return t
	}
	return nil
}
