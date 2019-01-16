package table

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/common/encoding/kope"
	"github.com/coschain/contentos-go/common/encoding/vme"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/vm/contract/abi"
	"reflect"
	"strings"
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

func (t *ContractTable) fieldValue(record reflect.Value, i int) reflect.Value {
	f := t.abiTable.Record().Field(i)
	v := record
	for i := 0; i < f.Depth(); i++ {
		v = v.Field(0)
	}
	return v.Field(f.Ordinal())
}

func (t *ContractTable) NewRecord(encodedRecord []byte) error {
	r, err := t.decodeRecord(encodedRecord)
	if err != nil {
		return err
	}
	p := t.fieldValue(reflect.ValueOf(r), t.abiTable.PrimaryIndex()).Interface()
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
	oldPk, err := t.primaryKey(encodedPK)
	if err != nil {
		return err
	}

	newPkVal := t.fieldValue(reflect.ValueOf(newRec), t.abiTable.PrimaryIndex()).Interface()
	newPk := kope.AppendKey(t.primary, newPkVal)
	pkChanged := bytes.Compare(oldPk, newPk) != 0

	b := t.db.NewBatch()
	defer t.db.DeleteBatch(b)

	if pkChanged {
		if err = b.Delete(oldPk); err != nil {
			return err
		}
	}
	if err = b.Put(newPk, encodedRecord); err != nil {
		return err
	}
	if err = t.deleteSecondaryIndices(b, oldRec, oldPk); err != nil {
		return err
	}
	if err = t.writeSecondaryIndices(b, newRec, newPk); err != nil {
		return err
	}
	return b.Write()
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
	return b.Write()
}

func (t *ContractTable) EnumRecords(field string, start interface{}, limit interface{}, reverse bool, maxCount int, callback func(r interface{})bool) int {
	count, _ := t.enumRecords(field, start, limit, reverse, maxCount, callback)
	return count
}

func (t *ContractTable) QueryRecords(field string, start interface{}, limit interface{}, reverse bool, maxCount int) ([]interface{}, error) {
	records := []interface{}{}
	_, err := t.enumRecords(field, start, limit, reverse, maxCount, func(r interface{}) bool {
		records = append(records, r)
		return true
	})
	return records, err
}

func (t *ContractTable) QueryRecordsJson(field string, startJson string, limitJson string, reverse bool, maxCount int) (string, error) {
	var (
		start, limit interface{}
		records []interface{}
		result []byte
		err error
	)
	if len(startJson) == 0 {
		startJson = "null"
	}
	if len(limitJson) == 0 {
		limitJson = "null"
	}
	if err = json.Unmarshal([]byte(startJson), &start); err != nil {
		return "", errors.New(fmt.Sprintf("failed to decode json: \"%s\". %s", startJson, err.Error()))
	}
	if err = json.Unmarshal([]byte(limitJson), &limit); err != nil {
		return "", errors.New(fmt.Sprintf("failed to decode json: \"%s\". %s", limitJson, err.Error()))
	}
	if records, err = t.QueryRecords(field, start, limit, reverse, maxCount); err != nil {
		return "", errors.New(fmt.Sprintf("failed to query: %s", err.Error()))
	}
	if result, err = json.MarshalIndent(records, "", strings.Repeat(" ", 4)); err != nil {
		return "", errors.New(fmt.Sprintf("failed to encode result to json: %s", err.Error()))
	}
	return string(result), nil
}

func (t *ContractTable) queryValue(val interface{}, typ reflect.Type) (interface{}, error) {
	rv := reflect.ValueOf(val)
	if rv.Kind() == reflect.Invalid {
		return nil, nil
	}
	if rv.Type().ConvertibleTo(typ) {
		qv := reflect.New(typ).Elem()
		qv.Set(rv.Convert(typ))
		return qv.Interface(), nil
	}
	return nil, errors.New("incompatible query value.")
}

func (t *ContractTable) enumRecords(field string, start interface{}, limit interface{}, reverse bool, maxCount int, callback func(r interface{})bool) (int, error) {
	var (
		qStart, qLimit interface{}
		ft reflect.Type
		err error
	)

	st := t.abiTable.Record()
	if st.Field(t.abiTable.PrimaryIndex()).Name() == field {
		ft = st.Field(t.abiTable.PrimaryIndex()).Type().Type()
		if qStart, err = t.queryValue(start, ft); err != nil {
			return 0, err
		}
		if qLimit, err = t.queryValue(limit, ft); err != nil {
			return 0, err
		}
		return t.scanDatabase(t.primary, qStart, qLimit, reverse, maxCount, func(k, v []byte) (bool, error) {
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
		if st.Field(si[i]).Name() == field {
			idx = i
			break
		}
	}
	if idx < 0 {
		return 0, errors.New("unknown query field: " + field)
	}
	ft = st.Field(si[idx]).Type().Type()
	if qStart, err = t.queryValue(start, ft); err != nil {
		return 0, err
	}
	if qLimit, err = t.queryValue(limit, ft); err != nil {
		return 0, err
	}
	return t.scanDatabase(t.secondaries[idx], qStart, qLimit, reverse, maxCount, func(k, v []byte) (bool, error) {
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
	p, err := vme.DecodeWithType(encodedPK, t.abiTable.Record().Field(t.abiTable.PrimaryIndex()).Type().Type())
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
		if err := callback(i, t.fieldValue(rv, j).Interface()); err != nil {
			return err
		}
	}
	return nil
}

func (t *ContractTable) scanDatabase(prefix kope.Key, start interface{}, limit interface{}, reverse bool, maxCount int, callback func(k, v []byte)(bool, error)) (int, error) {
	var (
		startKey, limitKey kope.Key
		it iservices.IDatabaseIterator
		k, v []byte
		err error
		goAhead bool
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
		if k, err = it.Key(); err == nil {
			if v, err = it.Value(); err == nil {
				goAhead, err = callback(k, v)
				if err == nil {
					count++
				}
				if !goAhead || err != nil {
					break
				}
			}
		}
	}
	return count, err
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
