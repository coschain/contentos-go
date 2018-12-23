package abi

import (
	"fmt"
	"github.com/coschain/contentos-go/common/encoding/vme"
	"reflect"
	"strings"
)

type ABIBaseType struct {
	name string
	rt reflect.Type
}

func (t *ABIBaseType) Name() string {
	return t.name
}

func (t *ABIBaseType) Type() reflect.Type {
	return t.rt
}

func (t *ABIBaseType) IsStruct() bool {
	return t.rt.Kind() == reflect.Struct
}

type ABIStructType struct {
	ABIBaseType
	base *ABIStructType
	fields []ABIStructField
}
type ABIStructField struct {
	Name string
	Type IContractType
}

func (t *ABIStructType) FieldNum() int {
	if t.base != nil {
		return len(t.fields) + 1
	}
	return len(t.fields)
}

func (t *ABIStructType) FieldType(i int) IContractType {
	if i >= 0 && i < t.FieldNum() {
		if t.base != nil {
			if i == 0 {
				return t.base
			} else {
				return t.fields[i - 1].Type
			}
		}
		return t.fields[i].Type
	}
	return nil
}

func NewStruct(name string, base *ABIStructType, fields...ABIStructField) *ABIStructType {
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
			Name: strings.Title(fields[i].Name),
			Type: fields[i].Type.Type(),
			Tag: reflect.StructTag(fmt.Sprintf(`json:"%s"`, fields[i].Name)),
		})
	}
	return &ABIStructType{
		ABIBaseType: ABIBaseType{name, reflect.StructOf(sf) },
		fields:      fields,
	}
}

type abiMethod struct {
	name string
	args *ABIStructType
}

func (m *abiMethod) Name() string {
	return m.name
}

func (m *abiMethod) Args() IContractStruct {
	return m.args
}

type abiTable struct {
	name string
	record *ABIStructType
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

type abiTypeAlias struct {
	name string
	origin IContractType
}

func (a *abiTypeAlias) Name() string {
	return a.name
}

func (a *abiTypeAlias) Type() reflect.Type {
	return a.origin.Type()
}

func (a *abiTypeAlias) IsStruct() bool {
	return a.origin.IsStruct()
}

func (a *abiTypeAlias) FieldNum() int {
	if s, ok := a.origin.(IContractStruct); ok {
		return s.FieldNum()
	} else {
		return 0
	}
}

func (a *abiTypeAlias) FieldType(i int) IContractType {
	if s, ok := a.origin.(IContractStruct); ok {
		return s.FieldType(i)
	} else {
		return nil
	}
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
		} else if s, isStruct := t.(*ABIStructType); isStruct {
			fs := make([]jsonAbiStructField, len(s.fields))
			for i := range fs {
				fs[i] = jsonAbiStructField{
					Name: s.fields[i].Name,
					Type: s.fields[i].Type.Name(),
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
			si[i] = t.record.fields[t.secondary[i]].Name
		}
		ja.Tables = append(ja.Tables, jsonAbiTable{
			Name: t.name,
			Type: t.record.name,
			Primary: t.record.fields[t.primary].Name,
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

var builtinNonInheritableTypes = map[string]IContractType {
	"bool": 	&ABIBaseType{"bool", 	vme.BoolType() },
	"int8":  	&ABIBaseType{"int8", 	vme.Int8Type() },
	"int16":  	&ABIBaseType{"int16", 	vme.Int16Type() },
	"int32":  	&ABIBaseType{"int32", 	vme.Int32Type() },
	"int64":  	&ABIBaseType{"int64", 	vme.Int64Type() },
	"uint8":  	&ABIBaseType{"uint8", 	vme.Uint8Type() },
	"uint16":  	&ABIBaseType{"uint16", 	vme.Uint16Type() },
	"uint32":  	&ABIBaseType{"uint32", 	vme.Uint32Type() },
	"uint64":  	&ABIBaseType{"uint64", 	vme.Uint64Type() },
	"float":  	&ABIBaseType{"float", 	vme.Float32Type() },
	"double":  	&ABIBaseType{"double", 	vme.Float64Type() },

	"string":  	&ABIBaseType{"string", 	vme.StringType() },
	"cosio::account_name": &ABIBaseType{"cosio::account_name", vme.StringType() },
}

var builtinInheritableTypes = map[string]IContractType {

}

func abiBuiltinType(name string, inheritable bool) IContractType {
	types := builtinNonInheritableTypes
	if inheritable {
		types = builtinInheritableTypes
	}
	if t, ok := types[name]; ok {
		return t
	}
	return nil
}

func ABIBuiltinInheritableType(name string) IContractType {
	return abiBuiltinType(name, true)
}

func ABIBuiltinNonInheritableType(name string) IContractType {
	return abiBuiltinType(name, false)
}

func ABIBuiltinType(name string) IContractType {
	if t := ABIBuiltinInheritableType(name); t != nil {
		return t
	}
	return ABIBuiltinNonInheritableType(name)
}

func ABIBuiltinTypeEnum(callback func(t IContractType)) {
	for _, t := range builtinNonInheritableTypes {
		callback(t)
	}
	for _, t := range builtinInheritableTypes {
		callback(t)
	}
}
