package table

import (
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/encoding/kope"
	"github.com/coschain/contentos-go/db/storage"
)

type Table struct {
	name string
	db storage.Database
	valueIO TableValueIO
	columns []*TableColumn
	columnByName map[string]int
	indices []*TableIndex
	indexByName map[string]int
	indicesByType map[TableIndexType][]int
	primaryIndex *TableIndex
	prefix kope.Key
	err error
}

func (t *Table) NewRow(columnValues...interface{}) *TableRows {
	if t.err != nil {
		return errorTableRows(t.err)
	}

	batch := t.db.NewBatch()
	defer t.db.DeleteBatch(batch)

	rk, err := t.valueIO.NewRow(t.db, t.db, batch, batch, columnValues)
	if err == nil {
		for _, idx := range t.indices {
			if idx.typ == Primary {
				continue
			}
			if err = idx.addIndex(batch, columnValues[idx.column.ordinal], rk); err != nil {
				break
			}
		}
		err = batch.Write()
	}
	if err != nil {
		return errorTableRows(err)
	}
	return &TableRows{
		index: t.primaryIndex,
		key: common.CopyBytes(rk),
	}
}

func (t *Table) Row(value...interface{}) *TableRows {
	if t.err != nil {
		return errorTableRows(t.err)
	}
	return t.primaryIndex.Row(value...)
}

func (t *Table) Index(name string) *TableIndex {
	if t.err != nil {
		return errorTableIndex(t.err)
	}
	if idx, ok := t.indexByName[name]; ok {
		return t.indices[idx]
	}
	return errorTableIndex(errors.New(fmt.Sprintf("index named \"%s\" not found.", name)))
}
