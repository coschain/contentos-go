package table

import (
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/encoding/kope"
	"github.com/coschain/contentos-go/db/storage"
)

type TableIndexType int
const (
	NotIndexed TableIndexType = iota
	Primary
	Unique
	Nonunique
)

func (idxType TableIndexType) String() string {
	switch idxType {
	case NotIndexed:
		return "NotIndexed"
	case Primary:
		return "Primary"
	case Unique:
		return "Unique"
	case Nonunique:
		return "Nonunique"
	}
	return "<unexpected>"
}

type TableIndex struct {
	name string
	table *Table
	column *TableColumn
	typ TableIndexType
	prefix []byte
	err error
}

func errorTableIndex(err error) *TableIndex {
	return &TableIndex{err:err}
}

func (idx *TableIndex) Row(value...interface{}) *TableRows {
	if idx.err != nil {
		return errorTableRows(idx.err)
	}
	valueCount := len(value)
	if valueCount > 1 {
		return errorTableRows(errors.New(fmt.Sprintf("index \"%s\" takes at most 1 argument. %d argumets were given.", idx.name, valueCount)))
	} else if valueCount == 0 {
		return &TableRows{
			index:    idx,
			keyStart: kope.MinKey(idx.prefix),
			keyLimit: kope.MaxKey(idx.prefix),
		}
	}
	val := value[0]
	if err := idx.column.checkValueType(val); err != nil {
		return errorTableRows(err)
	}

	var rows *TableRows

	switch idx.typ {
	case Primary:
		rows = &TableRows{ index: idx, key: kope.AppendKey(idx.prefix, val) }
	case Unique:
		rows = &TableRows{ index: idx, key: kope.AppendKey(idx.prefix, val) }
	case Nonunique:
		vk := kope.AppendKey(idx.prefix, val)
		rows = &TableRows{ index: idx, keyStart: kope.MinKey(vk), keyLimit: kope.MaxKey(vk) }
	}
	return rows
}

func (idx *TableIndex) addIndex(dbPutter storage.DatabasePutter, value interface{}, rowKey []byte) error {
	var dbGetter storage.DatabaseGetter
	dbGetter = idx.table.db
	if idx.typ == Primary {
		return errors.New("addIndex() not supported by primary index.")
	}
	if err := idx.column.checkValueType(value); err != nil {
		return err
	}
	if idx.typ == Unique {
		k := kope.AppendKey(idx.prefix, value)
		if duplicated, err := dbGetter.Has(k); err != nil {
			return err
		} else if duplicated {
			return errors.New(fmt.Sprintf("index \"%s\" found duplicated value: %v", idx.name, value))
		}
		return dbPutter.Put(k, rowKey)
	} else {
		k := kope.IndexKey(idx.prefix, rowKey, value)
		return dbPutter.Put(k, []byte{})
	}
}

func (idx *TableIndex) removeIndex(dbDeleter storage.DatabaseDeleter, value interface{}, rowKey []byte) error {
	if idx.typ == Primary {
		return errors.New("removeIndex() not supported by primary index.")
	}
	if err := idx.column.checkValueType(value); err != nil {
		return err
	}
	var k []byte
	if idx.typ == Unique {
		k = kope.AppendKey(idx.prefix, value)
	} else {
		k = kope.IndexKey(idx.prefix, rowKey, value)
	}
	return dbDeleter.Delete(k)
}

func (idx *TableIndex) rowKey(indexedKey []byte) ([]byte, error) {
	var dbGetter storage.DatabaseGetter
	dbGetter = idx.table.db
	if idx.typ == Primary {
		hasKey, err := dbGetter.Has(indexedKey)
		if err != nil {
			return nil, err
		}
		if hasKey {
			return indexedKey, nil
		}
		return nil, errors.New("not found")
	} else if idx.typ == Unique {
		hasKey, err := dbGetter.Has(indexedKey)
		if err != nil {
			return nil, err
		}
		if hasKey {
			k, err := dbGetter.Get(indexedKey)
			if err != nil {
				return nil, err
			}
			return k, nil
		}
		return nil, errors.New("not found")
	} else {
		return nil, errors.New("rowKey() not supported by index type: " + idx.typ.String())
	}
}

func (idx *TableIndex) rowKeyScan(indexedKeyStart []byte, indexedKeyLimit []byte) ([][]byte, error) {
	var (
		rowKeys [][]byte
		k, v, rk []byte
		err error
		dbScanner storage.DatabaseScanner
	)
	dbScanner = idx.table.db
	it := dbScanner.NewIterator(indexedKeyStart, indexedKeyLimit)
	for it.Next() {
		if k, err = it.Key(); err != nil {
			return nil, err
		}
		if v, err = it.Value(); err != nil {
			return nil, err
		}
		rk = nil
		if idx.typ == Primary {
			rk = k
		} else if idx.typ == Unique {
			rk = v
		} else if idx.typ == Nonunique {
			rk = kope.IndexedPrimaryKey(k)
		}
		if len(rk) > 0 {
			rowKeys = append(rowKeys, common.CopyBytes(rk))
		}
	}
	dbScanner.DeleteIterator(it)
	return rowKeys, nil
}
