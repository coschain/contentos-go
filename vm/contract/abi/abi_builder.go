package abi

import (
	"errors"
	"fmt"
	"strings"
)

// ABI builder
type abiBuilder struct {
	version string								// abi version
	typedef map[string]string					// typedefs: new_name -> old_name
	structs map[string][]abiField				// structs: struct_name -> []struct_fields
	bases map[string]string						// bases: struct_name -> base_struct
	methods map[string]string					// methods: method_name -> argument_struct
	tables map[string]string					// tables: table_name -> record_struct
	primaries map[string]string					// primary keys of tables: table_name -> primary_index_field
	secondaries map[string]map[string]bool		// secondary keys of tables: table_name -> indexing_fields
}

// abiField is a field of a struct
type abiField struct {
	name string				// field name
	typ string				// field type
}

// NewAbiBuilder() creates an ABI builder
func NewAbiBuilder() *abiBuilder {
	return &abiBuilder{
		typedef: make(map[string]string),
		structs: make(map[string][]abiField),
		bases: make(map[string]string),
		methods: make(map[string]string),
		tables: make(map[string]string),
		primaries: make(map[string]string),
		secondaries: make(map[string]map[string]bool),
	}
}

// Version() sets the ABI version.
func (b *abiBuilder) Version(version string) *abiBuilder {
	b.version = version
	return b
}

// Typedef() adds a typedef.
func (b *abiBuilder) Typedef(newName string, oldName string) *abiBuilder {
	b.typedef[newName] = oldName
	return b
}

// Struct() adds a struct type.
func (b *abiBuilder) Struct(name string, base string, nameTypePairs...string) *abiBuilder {
	pairs := len(nameTypePairs)
	count := pairs / 2
	fs := make([]abiField, count)
	for i := range fs {
		fs[i] = abiField{ nameTypePairs[i * 2], nameTypePairs[i * 2 + 1] }
	}
	b.structs[name] = fs
	if len(base) > 0 {
		b.bases[name] = base
	}
	return b
}

// Method() adds a method.
func (b *abiBuilder) Method(name string, args string) *abiBuilder {
	b.methods[name] = args
	return b
}

// Table() adds a table.
func (b *abiBuilder) Table(name string, record string, primary string, secondaries...string) *abiBuilder {
	si := make(map[string]bool)
	for _, fn := range secondaries {
		si[fn] = true
	}
	b.tables[name] = record
	b.primaries[name] = primary
	b.secondaries[name] = si
	return b
}

// Build() validates inputs and creates an ABI.
func (b *abiBuilder) Build() (*abi, error) {
	// context will cache mediate processing results, and help validate input data.
	ctx := &abiBuildContext{
		b: b,
		realTypeNames: make(map[string]string),
		resolvedTypes: make(map[string]IContractType),
		typeFieldNames: make(map[string]map[string]int),
	}
	// ctx.prepare() processes and validates input data.
	// if input data are valid, prepare() returns nil. otherwise, it returns the error.
	if err := ctx.prepare(); err != nil {
		return nil, err
	}

	// now, we create the ABI based on the context.
	abi := new(abi)

	// version
	if len(b.version) == 0 {
		return nil, errors.New("abiBuilder: empty version string.")
	}
	abi.version = b.version

	// typedefs
	abi.typedefs = b.buildTypedefs(ctx)

	// structs
	abi.types = b.buildStructs(ctx)

	// append all type aliases to types
	for t := range abi.typedefs {
		abi.types = append(abi.types, &abiTypeAlias{
			name: t,
			origin: ctx.resolvedTypes[t],
		})
	}

	// set up by_name index of types
	abi.typeByName = make(map[string]int)
	for i, t := range abi.types {
		abi.typeByName[t.Name()] = i
	}

	// methods and their by_name indices
	abi.methods = b.buildMethods(ctx)
	abi.methodByName = make(map[string]int)
	for i, m := range abi.methods {
		abi.methodByName[m.name] = i
	}

	// tables and their by_name indices
	abi.tables = b.buildTables(ctx)
	abi.tableByName = make(map[string]int)
	for i, t := range abi.tables {
		abi.tableByName[t.name] = i
	}
	return abi, nil
}

// buildTypedefs() returns final typedefs based on context data.
func (b *abiBuilder) buildTypedefs(ctx *abiBuildContext) map[string]string {
	result := make(map[string]string)
	for t := range b.typedef {
		result[t] = ctx.realTypeNames[t]
	}
	return result
}

// buildStructs() returns structs based on context data.
func (b *abiBuilder) buildStructs(ctx *abiBuildContext) []IContractType {
	result := make([]IContractType, len(b.structs))
	idx := 0
	for s := range b.structs {
		result[idx] = ctx.resolvedTypes[s]
		idx++
	}
	return result
}

// buildMethods() returns methods based on context data.
func (b *abiBuilder) buildMethods(ctx *abiBuildContext) []*abiMethod {
	result := make([]*abiMethod, len(b.methods))
	idx := 0
	for m := range b.methods {
		result[idx] = &abiMethod{
			name: m,
			args: ctx.resolvedTypes[m].(*ABIStructType),
		}
		idx++
	}
	return result
}

// buildTables() returns tables based on context data.
func (b *abiBuilder) buildTables(ctx *abiBuildContext) []*abiTable {
	result := make([]*abiTable, len(b.tables))
	idx := 0
	for name, t := range b.tables {
		// prepare secondary indices
		s := b.secondaries[name]
		si := make([]int, len(s))
		cnt := 0
		for f := range s {
			si[cnt] = ctx.typeFieldNames[t][f]
			cnt++
		}
		result[idx] = &abiTable{
			name:      name,
			record:    ctx.resolvedTypes[t].(*ABIStructType),
			primary:   ctx.typeFieldNames[t][b.primaries[name]],
			secondary: si,
		}
		idx++
	}
	return result
}

// building context
type abiBuildContext struct {
	b *abiBuilder								// the builder
	realTypeNames map[string]string				// type name mapping for all types
	resolvedTypes map[string]IContractType		// name->IContractType mapping for all types
	typeFieldNames map[string]map[string]int	// name->fields mapping for all structs
}

// prepare() takes original inputs from builder, process on them and fill up context object.
// it reports an error if anything wrong occurred.
func (ctx *abiBuildContext) prepare() error {
	// step 1, we collect all referenced types
	types := make(map[string]bool)

	// types seen in a type alias.
	for newName, oldName := range ctx.b.typedef {
		// an array can't be an alias
		if arr, _ := ctx.isArray(newName); arr {
			return errors.New(newName + " can't be an type alias.")
		}
		types[newName] = true
		types[oldName] = true
	}
	// types declared as a struct, and all field types.
	for t, fs := range ctx.b.structs {
		types[t] = true
		for _, f := range fs {
			types[f.typ] = true
		}
	}
	// types declared as a base type of some struct.
	for t, base := range ctx.b.bases {
		types[t] = true
		types[base] = true
	}
	// method argument list types.
	for _, t := range ctx.b.methods {
		types[t] = true
	}
	// table record types.
	for _, t := range ctx.b.tables {
		types[t] = true
	}
	// check if we're building primary index on non-existent tables.
	for t := range ctx.b.primaries {
		if _, ok := ctx.b.tables[t]; !ok {
			return errors.New("abiBuilder: unknown table name: " + t)
		}
	}
	// check if we're building secondary indices on non-existent tables.
	for t := range ctx.b.secondaries {
		if _, ok := ctx.b.tables[t]; !ok {
			return errors.New("abiBuilder: unknown table name: " + t)
		}
	}
	// add element types of arrays
	ctx.resolveArrays(types)

	// step 2, we find the real names of all types, and save'em in ctx.realTypeNames.
	if err := ctx.resolveRealNames(types); err != nil {
		return err
	}

	// step 3, we build IContractType interfaces for all types, and save'em in ctx.resolvedTypes.
	if err := ctx.resolveTypes(types); err != nil {
		return err
	}

	// step 4, we build fields information of all structs, and save'em in ctx.typeFieldNames.
	if err := ctx.resolveFields(); err != nil {
		return err
	}

	// step 5, final logic checks.
	if err := ctx.validate(); err != nil {
		return err
	}
	return nil
}

func (ctx *abiBuildContext) isArray(name string) (bool, string) {
	if strings.HasSuffix(name, "[]") {
		return true, name[:len(name) - 2]
	}
	return false, ""
}

// pick up array types and feed their element types back
func (ctx *abiBuildContext) resolveArrays(types map[string]bool) {
	for {
		changed := false
		newTypes := make(map[string]bool)
		for t := range types {
			if !types[t] {
				continue
			}
			if arr, e := ctx.isArray(t); arr {
				newTypes[e] = true
			}
		}
		for t := range newTypes {
			if !types[t] {
				types[t] = true
				changed = true
			}
		}
		if !changed {
			break
		}
	}
}

// find the real names of given types
func (ctx *abiBuildContext) resolveRealNames(types map[string]bool) error {
	// find the real name of given types one by one
	for name := range types {
		if origin, err := ctx.realName(name); err != nil {
			return err
		} else {
			ctx.realTypeNames[name] = origin
		}
	}
	return nil
}

// get the real name of the given type
func (ctx *abiBuildContext) realName(name string) (string, error) {
	// if we've done the job before, just return the result.
	origin, ok := ctx.realTypeNames[name]
	if ok {
		return origin, nil
	}

	// an array can't be an alias.
	if arr, _ := ctx.isArray(name); arr {
		return name, nil
	}

	// working map is used for cyclic reference detection
	working := make(map[string]bool)
	for {
		// we are already working on this type, which means we've found a circle on the typedef chain.
		if working[name] {
			return "", errors.New("abiBuilder: found cyclic typedef: " + name)
		}
		// mark that we are working on the type
		working[name] = true

		if origin, ok = ctx.b.typedef[name]; ok {
			// follow the typedef chain
			name = origin
		} else {
			// the type is not an alias, so it must be a builtin type or a custom struct.
			// if not, report an error.
			if t := ABIBuiltinType(name); t == nil {
				if _, ok = ctx.b.structs[name]; !ok {
					return "", errors.New("abiBuilder: unknown type: " + name)
				}
			}
			break
		}
	}
	return name, nil
}

const (
	unresolved = iota
	resolving
	resolved
)

// build IContractType interfaces for given types
func (ctx *abiBuildContext) resolveTypes(types map[string]bool) error {
	// flags is used for cyclic reference detection
	flags := make(map[string]int)

	// first, we resolve all builtin types
	ABIBuiltinTypeEnum(func(t IContractType) {
		ctx.resolvedTypes[t.Name()] = t
		flags[t.Name()] = resolved
	})

	// resolve given types one by one
	for name := range types {
		if err := ctx.resolveType(name, flags); err != nil {
			return err
		}
	}
	return nil
}

// build IContract interface for a given type
func (ctx *abiBuildContext) resolveType(name string, flags map[string]int) error {
	if flags[name] == resolved {
		// we've done the job before, do nothing and return.
		return nil
	} else if flags[name] == resolving {
		// we are already working on this type. report cyclic reference error.
		return errors.New("abiBuilder: found cyclic type dependency: " + name)
	}

	// mark that we're working on this type
	flags[name] = resolving
	realName := ctx.realTypeNames[name]

	// if the type is an alias, we will resolve its original type and create an alias.
	if realName != name {
		// resolve the original type
		if err := ctx.resolveType(realName, flags); err != nil {
			flags[name] = unresolved
			return err
		}
		// this type is an alias pointing to the original one
		ctx.resolvedTypes[name] = &abiTypeAlias{ name: name, origin: ctx.resolvedTypes[realName] }
		flags[name] = resolved
		return nil
	}

	// if the type is an array, we will resolve its element type first,
	// and then create an array based on the element type.
	if arr, e := ctx.isArray(name); arr {
		if err := ctx.resolveType(e, flags); err != nil {
			flags[name] = unresolved
			return err
		}
		ctx.resolvedTypes[name] = NewArray(ctx.resolvedTypes[e])
		flags[name] = resolved
		return nil
	}

	// the type is not an alias, not an array, and not a builtin type since all builtin types were resolved beforehand.
	// hence it must be a custom struct, which depends on its base type and field types.

	// resolve the base type
	var baseStruct *ABIStructType
	if base := ctx.b.bases[name]; len(base) > 0 {
		if err := ctx.resolveType(base, flags); err != nil {
			flags[name] = unresolved
			return err
		}
		baseType := ctx.resolvedTypes[base]

		// the base type must be inheritable. if not, reports an error.
		if !baseType.IsStruct() || ABIBuiltinNonInheritableType(base) != nil {
			flags[name] = unresolved
			return errors.New(fmt.Sprintf("abiBuilder: base of %s is a non-inheritable type: %s", name, base))
		}
		baseStruct = baseType.(*ABIStructType)
	}

	// resolve field types
	fs := ctx.b.structs[name]
	for i := range fs {
		if err := ctx.resolveType(fs[i].typ, flags); err != nil {
			flags[name] = unresolved
			return err
		}
	}

	// build the IContractType interface for the type
	fields := make([]ABIStructField, len(fs))
	for i := range fields {
		fields[i] = ABIStructField{
			Name: fs[i].name,
			Type: ctx.resolvedTypes[fs[i].typ],
		}
	}
	s := NewStruct(name, baseStruct, fields...)
	if s == nil {
		flags[name] = unresolved
		return errors.New(fmt.Sprintf("abiBuilder: failed creating struct %s. duplicate field names.", name))
	}
	ctx.resolvedTypes[name] = s
	flags[name] = resolved
	return nil
}

// build fields information for all structs.
func (ctx *abiBuildContext) resolveFields() error {
	for name, t := range ctx.resolvedTypes {
		if !t.IsStruct() {
			continue
		}
		fields := make(map[string]int)
		s := t.(IContractStruct)
		count := s.FieldNum()
		for i := 0; i < count; i++ {
			fields[s.Field(i).Name()] = i
		}
		ctx.typeFieldNames[name] = fields
	}
	return nil
}

// logic checks
func (ctx *abiBuildContext) validate() error {
	// method's argument list type must be a struct.
	for name, t := range ctx.b.methods {
		if !ctx.resolvedTypes[t].IsStruct() {
			return errors.New(fmt.Sprintf("abiBuilder: arguments type of method %s is not a struct: %s", name, t))
		}
	}
	// table's record type must be a struct, and indexing fields must be valid.
	for name, t := range ctx.b.tables {
		// record type must be a struct
		rt := ctx.resolvedTypes[t]
		if !rt.IsStruct() {
			return errors.New(fmt.Sprintf("abiBuilder: record type of table %s is not a struct: %s", name, t))
		}
		st := rt.(IContractStruct)

		// primary key field must be valid.
		primary := ctx.b.primaries[name]
		if ord, ok := ctx.typeFieldNames[t][primary]; !ok {
			return errors.New(fmt.Sprintf("abiBuilder: unknown primary field %s of table %s", primary, name))
		} else if !st.Field(ord).Type().SupportsKope() {
			return errors.New(fmt.Sprintf("abiBuilder: primary field %s of table %s cannot be indexed.", primary, name))
		}
		// secondary indexing fields must be valid, and not the same as primary key.
		for f, ok := range ctx.b.secondaries[name] {
			if !ok {
				continue
			}
			if f == primary {
				return errors.New(fmt.Sprintf("abiBuilder: field %s used in both primary and secondary indices of table %s", primary, name))
			}
			if ord, ok := ctx.typeFieldNames[t][f]; !ok {
				return errors.New(fmt.Sprintf("abiBuilder: unknown secondary index field %s of table %s", f, name))
			} else if !st.Field(ord).Type().SupportsKope() {
				return errors.New(fmt.Sprintf("abiBuilder: secondary index field %s of table %s cannot be indexed.", f, name))
			}
		}
	}
	return nil
}
