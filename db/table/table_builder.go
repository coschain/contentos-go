package table

import (
	"errors"
	"github.com/coschain/contentos-go/common/encoding/kope"
	"github.com/coschain/contentos-go/db/storage"
)

type tableBuilder struct {
	t   Table
	err error
}

func TableBuilder() *tableBuilder {
	return &tableBuilder{
		t: Table{
			columnByName:  make(map[string]int),
			indexByName:   make(map[string]int),
			indicesByType: make(map[TableIndexType][]int),
		},
	}
}

func (b *tableBuilder) Database(db storage.Database) *tableBuilder {
	b.t.db = db
	return b
}

func (b *tableBuilder) ValueIO(valueIO TableValueIO) *tableBuilder {
	b.t.valueIO = valueIO
	return b
}

func (b *tableBuilder) Name(name string) *tableBuilder {
	b.t.name = name
	return b
}

func (b *tableBuilder) Column(name string, typeId string, indexType ...TableIndexType) *tableBuilder {
	if _, hasColumn := b.t.columnByName[name]; !hasColumn {
		if ti, typeOK := typeInfoByName(typeId); typeOK {
			idx := NotIndexed
			if len(indexType) > 0 {
				idx = indexType[0]
			}
			index := (*TableIndex)(nil)
			if idx != NotIndexed {
				if _, hasIndex := b.t.indexByName[name]; !hasIndex {
					isDupPrimary := idx == Primary && len(b.t.indicesByType[Primary]) > 0
					if !isDupPrimary {
						if ti.kope {
							index = &TableIndex{
								name:  name,
								table: &b.t,
								typ:   idx,
							}
							b.t.indices = append(b.t.indices, index)
							b.t.indexByName[index.name] = len(b.t.indices) - 1
							b.t.indicesByType[idx] = append(b.t.indicesByType[idx], len(b.t.indices)-1)
						} else {
							b.err = errors.New("column " + name + " can't be indexed, its type " + typeId + " doesn't support kope.")
						}
					} else {
						b.err = errors.New("duplicate primary key: " + name)
					}
				} else {
					b.err = errors.New("duplicate index: " + name)
				}
			}
			column := &TableColumn{
				name:    name,
				table:   &b.t,
				ordinal: len(b.t.columns),
				ti:      ti,
				index:   index,
			}
			b.t.columns = append(b.t.columns, column)
			b.t.columnByName[name] = column.ordinal
			if index != nil {
				index.column = column
			}
		} else {
			b.err = errors.New("column type not registered: " + typeId)
		}
	} else {
		b.err = errors.New("duplicate column: " + name)
	}
	return b
}

func (b *tableBuilder) Build() (*Table, error) {
	if b.err != nil {
		return nil, b.err
	}
	if len(b.t.name) == 0 {
		return nil, errors.New("table name not set")
	}
	if b.t.db == nil {
		return nil, errors.New("Database interface not set")
	}
	if b.t.valueIO == nil {
		return nil, errors.New("TableValueIO interface not set")
	}
	if len(b.t.columns) == 0 {
		return nil, errors.New("table columns not defined")
	}
	if len(b.t.indicesByType[Primary]) != 1 {
		return nil, errors.New("table primary key not defined")
	}

	b.t.prefix = kope.NewKey(b.t.name)
	for _, index := range b.t.indices {
		if index.typ == Primary {
			index.prefix = kope.AppendKey(b.t.prefix, "pk")
		} else {
			index.prefix = kope.AppendKey(b.t.prefix, "ix", index.name)
		}
	}
	b.t.primaryIndex = b.t.indices[b.t.indicesByType[Primary][0]]
	return &b.t, nil
}
