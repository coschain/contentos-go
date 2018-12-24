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
	kope bool
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

func (t *ABIBaseType) SupportsKope() bool {
	return t.kope
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
			Tag: reflect.StructTag(fmt.Sprintf(`json:"[%s]"`, base.name)),
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
		ABIBaseType: ABIBaseType{ name: name, rt: reflect.StructOf(sf), kope: false },
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

func (a *abiTypeAlias) SupportsKope() bool {
	return a.origin.SupportsKope()
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
	ja := &JsonABI{
		Types: []jsonAbiTypedef{},
		Structs: []jsonAbiStruct{},
		Methods: []jsonAbiMethod{},
		Tables: []jsonAbiTable{},
	}
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
	"bool": 	&ABIBaseType{ name: "bool", 	rt: vme.BoolType(),		kope: true },
	"int8":  	&ABIBaseType{ name: "int8", 	rt: vme.Int8Type(), 	kope: true },
	"int16":  	&ABIBaseType{ name: "int16", 	rt: vme.Int16Type(), 	kope: true },
	"int32":  	&ABIBaseType{ name: "int32", 	rt: vme.Int32Type(), 	kope: true },
	"int64":  	&ABIBaseType{ name: "int64", 	rt: vme.Int64Type(), 	kope: true },
	"uint8":  	&ABIBaseType{ name: "uint8", 	rt: vme.Uint8Type(), 	kope: true },
	"uint16":  	&ABIBaseType{ name: "uint16", 	rt: vme.Uint16Type(), 	kope: true },
	"uint32":  	&ABIBaseType{ name: "uint32", 	rt: vme.Uint32Type(), 	kope: true },
	"uint64":  	&ABIBaseType{ name: "uint64", 	rt: vme.Uint64Type(), 	kope: true },
	"float":  	&ABIBaseType{ name: "float", 	rt: vme.Float32Type(), 	kope: true },
	"double":  	&ABIBaseType{ name: "double", 	rt: vme.Float64Type(), 	kope: true },

	"std::string":  			&ABIBaseType{ name: "std::string", 				rt: vme.StringType(),		kope: true },

	"cosio::account_name": 		&ABIBaseType{ name: "cosio::account_name", 		rt: vme.StringType(), 		kope: true },
	"cosio::contract_name": 	&ABIBaseType{ name: "cosio::contract_name", 	rt: vme.StringType(), 		kope: true },
	"cosio::method_name": 		&ABIBaseType{ name: "cosio::method_name", 		rt: vme.StringType(), 		kope: true },
	"cosio::coin_amount": 		&ABIBaseType{ name: "cosio::coin_amount", 		rt: vme.Uint64Type(), 		kope: true },
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
