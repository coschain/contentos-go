package table

import (
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/common/encoding/kope"
	"github.com/coschain/contentos-go/common/encoding/vme"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/vm/contract/abi"
	"reflect"
)

const (
	ContractTablePrefix = "_ContractTable"
)

type ContractTable struct {
	abiTable abi.IContractTable
	primary kope.Key
	secondaries []kope.Key
	db iservices.IDatabaseService
}

func (t *ContractTable) NewRecord(encodedRecord []byte) error {
	r, err := t.decodeRecord(encodedRecord)
	if err != nil {
		return err
	}
	p := reflect.ValueOf(r).Field(t.abiTable.PrimaryIndex()).Interface()
	pk := kope.AppendKey(t.primary, p)
	if dup, err := t.db.Has(pk); err != nil {
		return err
	} else if dup {
		return errors.New(fmt.Sprintf("contract table: duplicate primary key: %v", p))
	}

	b := t.db.NewBatch()
	defer t.db.DeleteBatch(b)

	if err = b.Put(pk, encodedRecord); err != nil {
		return err
	}
	if err = t.writeSecondaryIndices(b, r, pk); err != nil {
		return err
	}
	if err = b.Write(); err != nil {
		return err
	}
	return nil
}

func (t *ContractTable) GetRecord(encodedPK []byte) ([]byte, error) {
	pk, err := t.primaryKey(encodedPK)
	if err != nil {
		return nil, err
	}
	data, err := t.db.Get(pk)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (t *ContractTable) UpdateRecord(encodedPK []byte, encodedRecord []byte) error {
	old, err := t.GetRecord(encodedPK)
	if err != nil {
		return err
	}
	oldRec, err := t.decodeRecord(old)
	if err != nil {
		return err
	}
	newRec, err := t.decodeRecord(encodedRecord)
	if err != nil {
		return err
	}
	pk, err := t.primaryKey(encodedPK)
	if err != nil {
		return err
	}
	b := t.db.NewBatch()
	defer t.db.DeleteBatch(b)
	if err = b.Put(pk, encodedRecord); err != nil {
		return err
	}
	if err = t.deleteSecondaryIndices(b, oldRec, pk); err != nil {
		return err
	}
	if err = t.writeSecondaryIndices(b, newRec, pk); err != nil {
		return err
	}
	return nil
}

func (t *ContractTable) DeleteRecord(encodedPK []byte) error {
	old, err := t.GetRecord(encodedPK)
	if err != nil {
		return nil
	}
	oldRec, err := t.decodeRecord(old)
	if err != nil {
		return err
	}
	pk, err := t.primaryKey(encodedPK)
	if err != nil {
		return err
	}
	b := t.db.NewBatch()
	defer t.db.DeleteBatch(b)
	if err = b.Delete(pk); err != nil {
		return err
	}
	if err = t.deleteSecondaryIndices(b, oldRec, pk); err != nil {
		return err
	}
	return nil
}

func (t *ContractTable) EnumRecords(field string, start interface{}, limit interface{}, reverse bool, maxCount int, callback func(r interface{})bool) int {
	st := t.abiTable.Record()
	if st.FieldType(t.abiTable.PrimaryIndex()).Name() == field {
		return t.scanDatabase(t.primary, start, limit, reverse, maxCount, func(k, v []byte) (bool, error) {
			if r, err := t.decodeRecord(v); err != nil {
				return false, err
			} else {
				return callback(r), nil
			}
		})
	}
	si := t.abiTable.SecondaryIndices()
	idx := -1
	for i := range si {
		if st.FieldType(si[i]).Name() == field {
			idx = i
			break
		}
	}
	if idx < 0 {
		return 0
	}
	return t.scanDatabase(t.secondaries[idx], start, limit, reverse, maxCount, func(k, v []byte) (bool, error) {
		pk := kope.IndexedPrimaryKey(k)
		if data, err := t.db.Get(pk); err != nil {
			return false, err
		} else {
			if r, err := t.decodeRecord(data); err != nil {
				return false, err
			} else {
				return callback(r), nil
			}
		}
	})
}

func (t *ContractTable) primaryKey(encodedPK []byte) (kope.Key, error) {
	p, err := vme.DecodeWithType(encodedPK, t.abiTable.Record().FieldType(t.abiTable.PrimaryIndex()).Type())
	if err != nil {
		return nil, err
	}
	return kope.AppendKey(t.primary, p), nil
}

func (t *ContractTable) decodeRecord(encodedRecord []byte) (interface{}, error) {
	return vme.DecodeWithType(encodedRecord, t.abiTable.Record().Type())
}

func (t *ContractTable) writeSecondaryIndices(batch iservices.IDatabaseBatch, record interface{}, primaryKey kope.Key) error {
	return t.enumSecondaryIndexFields(record, func(idx int, v interface{}) error {
		return batch.Put(kope.IndexKey(t.secondaries[idx], primaryKey, v), []byte{})
	})
}

func (t *ContractTable) deleteSecondaryIndices(batch iservices.IDatabaseBatch, record interface{}, primaryKey kope.Key) error {
	return t.enumSecondaryIndexFields(record, func(idx int, v interface{}) error {
		return batch.Delete(kope.IndexKey(t.secondaries[idx], primaryKey, v))
	})
}

func (t *ContractTable) enumSecondaryIndexFields(record interface{}, callback func(idx int, v interface{})error) error {
	rv := reflect.ValueOf(record)
	si := t.abiTable.SecondaryIndices()
	for i, j := range si {
		if err := callback(i, rv.Field(j).Interface()); err != nil {
			return err
		}
	}
	return nil
}

func (t *ContractTable) scanDatabase(prefix kope.Key, start interface{}, limit interface{}, reverse bool, maxCount int, callback func(k, v []byte)(bool, error)) int {
	var (
		startKey, limitKey kope.Key
		it iservices.IDatabaseIterator
	)
	if start != nil {
		startKey = kope.AppendKey(prefix, start)
	} else {
		startKey = kope.MinKey(prefix)
	}
	if limit != nil {
		limitKey = kope.AppendKey(prefix, limit)
	} else {
		limitKey = kope.MaxKey(prefix)
	}
	if reverse {
		it = t.db.NewReversedIterator(startKey, limitKey)
	} else {
		it = t.db.NewIterator(startKey, limitKey)
	}
	defer t.db.DeleteIterator(it)
	count := 0
	for it.Next() {
		if count >= maxCount && maxCount > 0 {
			break
		}
		if k, err := it.Key(); err == nil {
			if v, err := it.Value(); err == nil {
				goAhead, err := callback(k, v)
				if err == nil {
					count++
				}
				if !goAhead || err != nil {
					break
				}
			}
		}
	}
	return count
}


type ContractTables struct {
	tables map[string]*ContractTable
}

func NewContractTables(owner string, contract string, abi abi.IContractABI, db iservices.IDatabaseService) *ContractTables {
	tables := &ContractTables{
		tables: make(map[string]*ContractTable),
	}
	prefix := kope.NewKey(ContractTablePrefix, owner, contract)
	count := abi.TablesCount()
	for i := 0; i < count; i++ {
		abiTable := abi.TableByIndex(i)
		si := abiTable.SecondaryIndices()
		sk := make([]kope.Key, len(si))
		for j := range sk {
			sk[j] = kope.AppendKey(prefix, "ix", si[j])
		}
		tables.tables[abiTable.Name()] = &ContractTable{
			abiTable: abiTable,
			primary: kope.AppendKey(prefix, "pk"),
			secondaries: sk,
			db: db,
		}
	}
	return tables
}

func (tables *ContractTables) Table(name string) *ContractTable {
	return tables.tables[name]
}
