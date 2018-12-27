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


type abiStructField struct {
	name string
	typ IContractType
	depth int
	ordinal int
}

func (f *abiStructField) Name() string {
	return f.name
}

func (f *abiStructField) Type() IContractType {
	return f.typ
}

func (f *abiStructField) Depth() int {
	return f.depth
}

func (f *abiStructField) Ordinal() int {
	return f.ordinal
}


type ABIStructType struct {
	ABIBaseType
	base *abiStructField
	fields []*abiStructField
	localFields []int
}

func (t *ABIStructType) FieldNum() int {
	return len(t.fields)
}

func (t *ABIStructType) Field(i int) IContractStructField {
	if i >= 0 && i < t.FieldNum() {
		return t.fields[i]
	}
	return nil
}

func (t *ABIStructType) LocalFieldNum() int {
	return len(t.localFields)
}

func (t *ABIStructType) LocalField(i int) IContractStructField {
	if i >= 0 && i < t.LocalFieldNum() {
		return t.Field(t.localFields[i])
	}
	return nil
}

func (t *ABIStructType) Base() IContractStructField {
	return t.base
}

type ABIStructField struct {
	Name string
	Type IContractType
}

func NewStruct(name string, base *ABIStructType, fields...ABIStructField) *ABIStructType {
	var (
		locals []int
		total []*abiStructField
		rfields []reflect.StructField
		baseField *abiStructField
		seen map[string]bool
	)
	seen = make(map[string]bool)
	if base != nil {
		for i := 0; i < base.FieldNum(); i++ {
			f := base.Field(i)
			fn := f.Name()
			if seen[fn] {
				return nil
			}
			seen[fn] = true
			total = append(total, &abiStructField{
				name: fn,
				typ: f.Type(),
				depth: f.Depth() + 1,
				ordinal: f.Ordinal(),
			})
		}
		baseField = &abiStructField{
			name: "[base]",
			typ: base,
			depth: 0,
			ordinal: 0,
		}
		rfields = append(rfields, reflect.StructField{
			Name: "Base_Field__",
			Type: baseField.typ.Type(),
			Tag: reflect.StructTag(`json:"[base]"`),
		})
	}
	for _, f := range fields {
		if seen[f.Name] {
			return nil
		}
		seen[f.Name] = true
		sf := &abiStructField{
			name: f.Name,
			typ: f.Type,
			depth: 0,
			ordinal: len(locals),
		}
		locals = append(locals, len(total))
		total = append(total, sf)
		rfields = append(rfields, reflect.StructField{
			Name: strings.Title(f.Name),
			Type: f.Type.Type(),
			Tag: reflect.StructTag(fmt.Sprintf(`json:"%s"`, f.Name)),
		})
	}
	return &ABIStructType {
		ABIBaseType: ABIBaseType{
			name: name,
			rt: reflect.StructOf(rfields),
			kope: false,
		},
		base: baseField,
		fields: total,
		localFields: locals,
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

func (a *abiTypeAlias) Field(i int) IContractStructField {
	if s, ok := a.origin.(IContractStruct); ok {
		return s.Field(i)
	} else {
		return nil
	}
}

func (a *abiTypeAlias) LocalFieldNum() int {
	if s, ok := a.origin.(IContractStruct); ok {
		return s.LocalFieldNum()
	} else {
		return 0
	}
}

func (a *abiTypeAlias) LocalField(i int) IContractStructField {
	if s, ok := a.origin.(IContractStruct); ok {
		return s.LocalField(i)
	} else {
		return nil
	}
}

func (a *abiTypeAlias) Base() IContractStructField {
	if s, ok := a.origin.(IContractStruct); ok {
		return s.Base()
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
			fs := make([]jsonAbiStructField, len(s.localFields))
			for i := range fs {
				fs[i] = jsonAbiStructField{
					Name: s.fields[s.localFields[i]].Name(),
					Type: s.fields[s.localFields[i]].typ.Name(),
				}
			}
			base := ""
			if s.base != nil {
				base = s.base.Type().Name()
			}
			ja.Structs = append(ja.Structs, jsonAbiStruct{
				Name: s.name,
				Base: base,
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
			si[i] = t.record.fields[t.secondary[i]].Name()
		}
		ja.Tables = append(ja.Tables, jsonAbiTable{
			Name: t.name,
			Type: t.record.name,
			Primary: t.record.fields[t.primary].Name(),
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
	b.Version(ja.Version)
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

func UnmarshalABI(data []byte) (ISerializableContractABI, error) {
	abi := new(abi)
	if err := abi.Unmarshal(data); err != nil {
		return nil, err
	}
	return abi, nil
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
	"cosio::singleton_record": 	NewStruct("cosio::singleton_record", nil, ABIStructField{ "id", builtinNonInheritableTypes["int32"]}),

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
