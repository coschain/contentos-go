package abi

import "reflect"

//
// IContractType is a data type used in a contract.
//
type IContractType interface {
	// Name() returns name of this type.
	Name() string

	// Type() returns a reflect.Type representing this type.
	Type() reflect.Type

	// IsStruct() indicates whether it's a struct or not.
	IsStruct() bool

	// IsArray() indicates whether it's an array or not.
	IsArray() bool

	// IsMap() indicates whether it's a map or not.
	IsMap() bool

	// SupportsKope() indicates whether the type supports kope.
	SupportsKope() bool
}


//
// IContractArray is an array of some type.
//
type IContractArray interface {
	IContractType

	// Elem() return type of elements.
	Elem() IContractType
}

//
// IContractMap is a map of specified key-value types.
//
type IContractMap interface {
	IContractType

	// Key() returns key type.
	Key() IContractType

	// Value() returns value type.
	Value() IContractType
}

//
// IContractStructField is a field of a struct
//
type IContractStructField interface {
	// Name() returns name of the field.
	Name() string

	// Type() returns type of the field.
	Type() IContractType

	// Depth() returns the depth of the field.
	// e.g. 0: local fields. 1: fields of base. 2: fields of base's base...
	Depth() int

	// Ordinal() returns ordinal of the field in its defining struct.
	Ordinal() int
}

//
// IContractStruct is a struct type used in a contract.
//
type IContractStruct interface {
	IContractType

	// FieldNum() returns number of fields.
	// Involving fields are both local fields and fields of all bases in hierachy.
	FieldNum() int

	// Field(i) returns the i-th field of the struct.
	// i must be in range [ 0, FieldNum() ), otherwise Field(i) returns nil.
	Field(i int) IContractStructField

	// LocalFieldNum() returns number of local fields.
	LocalFieldNum() int

	// LocalField(i) returns the i-th local field.
	// i must be in range [ 0, LocalFieldNum() ), otherwise LocalField(i) returns nil.
	LocalField(i int) IContractStructField

	// Base() returns the base of the struct as an embedded field.
	// If the struct has no base type, Base() returns nil.
	Base() IContractStructField
}

//
// IContractMethod is a method defined in a contract.
//
type IContractMethod interface {
	// Name() returns name of this method.
	Name() string

	// Args() returns a struct type representing the argument list.
	Args() IContractStruct
}

//
// IContractTable is a table defined in a contract.
//
type IContractTable interface {
	// Name() returns name of the table.
	Name() string

	// Record() returns a struct type representing columns of the table.
	Record() IContractStruct

	// PrimaryIndex() returns the field number of the primary-key column.
	PrimaryIndex() int

	// SecondaryIndices() returns field number of each secondary index column.
	// Secondary indices are always non-unique and single. Neither unique nor composite indices are supported.
	SecondaryIndices() []int
}

//
// IContractABI is a full ABI definition of a contract.
//
type IContractABI interface {
	// TypesCount() returns total number of types used in a contract.
	TypesCount() int

	// TypeByIndex(i) returns the i-th type.
	TypeByIndex(i int) IContractType

	// TypeByName(name) returns the type of the given @name.
	TypeByName(name string) IContractType

	// MethodsCount() returns total number of methods defined in a contract.
	MethodsCount() int

	// MethodByIndex(i) returns the i-th method.
	MethodByIndex(i int) IContractMethod

	// MethodByName(name) returns the method of given @name.
	MethodByName(name string) IContractMethod

	// TableCount() returns total number of tables defined in a contract.
	TablesCount() int

	// TableByIndex(i) returns the i-th table.
	TableByIndex(i int) IContractTable

	// TableByName(name) returns the table of given @name.
	TableByName(name string) IContractTable
}

//
// ISerializableContractABI is a IContractABI which supports marshal/unmarshal.
//
type ISerializableContractABI interface {
	IContractABI

	// Marshal() encodes the given ABI to a byte slice.
	Marshal() ([]byte, error)

	// Unmarshal() decodes the ABI from a byte slice.
	Unmarshal(data []byte) error
}
