package abi

import (
	"fmt"
	"github.com/pkg/errors"
)

type abiBuilder struct {
	version string
	typedef map[string]string
	structs map[string][]abiField
	bases map[string]string
	methods map[string]string
	tables map[string]string
	primaries map[string]string
	secondaries map[string][]string
}

type abiField struct {
	name string
	typ string
}

func NewAbiBuilder() *abiBuilder {
	return nil
}

func (b *abiBuilder) Version(version string) *abiBuilder {
	b.version = version
	return b
}

func (b *abiBuilder) Typedef(newName string, oldName string) *abiBuilder {
	b.typedef[newName] = oldName
	return b
}

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

func (b *abiBuilder) Method(name string, args string) *abiBuilder {
	b.methods[name] = args
	return b
}

func (b *abiBuilder) Table(name string, record string, primary string, secondaries...string) *abiBuilder {
	b.tables[name] = record
	b.primaries[name] = primary
	b.secondaries[name] = secondaries
	return b
}

func (b *abiBuilder) Build() (*abi, error) {
	abi := &abi{ version: b.version }

	if r, err := b.buildTypedefs(); err != nil {
		return nil, err
	} else {
		abi.typedefs = r
	}

	if r, err := b.buildStructs(); err != nil {
		return nil, err
	} else {
		abi.types = r

	}

	return abi, nil
}

func (b *abiBuilder) buildTypedefs() (map[string]string, error) {
	result := make(map[string]string)
	for t := range b.typedef {
		if origin, err := b.realType(t); err != nil {
			return nil, err
		} else {
			result[t] = origin
		}
	}
	return nil, nil
}

func (b *abiBuilder) realType(name string) (string, error) {
	checked := make(map[string]bool)
	for {
		if checked[name] {
			return "", errors.New("abiBuilder: cyclic typedef's.")
		}
		checked[name] = true
		origin, ok := b.typedef[name]
		if !ok {
			if t := ABIBuiltinType(name); t == nil {
				if _, isStruct := b.structs[name]; !isStruct {
					return "", errors.New("abiBuilder: unknown type: " + name)
				}
			}
			break
		}
		name = origin
	}
	return name, nil
}

func (b *abiBuilder) buildStructs() ([]IContractType, error) {
	count := len(b.structs)
	result, flags := make([]IContractType, 0, count), make(map[string]int)

	for t := range b.structs {
		if err := b.buildStruct(t, &result, flags); err != nil {
			return nil, err
		}
	}
	return result, nil
}

const (
	initial = iota
	working
	done
)

func (b *abiBuilder) buildStruct(name string, result *[]IContractType, flags map[string]int) error {
	if flags[name] == done {
		return nil
	} else if flags[name] == working {
		return errors.New("abiBuilder: cyclic reference of structs: " + name)
	}
	flags[name] = working
	base := b.bases[name]
	if realBase, err := b.realType(base); err != nil {
		return errors.New(fmt.Sprintf("abiBuilder: unknown base type of struct %s: %s", name, base))
	} else if ABIBuiltinPrimitiveType(realBase) != nil {
		return errors.New(fmt.Sprintf("abiBuilder: struct %s based on a primitive type: %s", name, realBase))
	} else if ABIBuiltinType(realBase) == nil {
		if err := b.buildStruct(realBase, result, flags); err != nil {
			return err
		}
	}
	// TODO: wip...
	return nil
}
