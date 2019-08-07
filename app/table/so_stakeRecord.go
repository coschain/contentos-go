package table

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/coschain/contentos-go/common/encoding/kope"
	"github.com/coschain/contentos-go/iservices"
	prototype "github.com/coschain/contentos-go/prototype"
	proto "github.com/golang/protobuf/proto"
)

////////////// SECTION Prefix Mark ///////////////
var (
	StakeRecordRecordTable        uint32 = 265171955
	StakeRecordRecordReverseTable uint32 = 2606609996
	StakeRecordRecordUniTable     uint32 = 832689285

	StakeRecordRecordRow uint32 = 957572259
)

////////////// SECTION Wrap Define ///////////////
type SoStakeRecordWrap struct {
	dba       iservices.IDatabaseRW
	mainKey   *prototype.StakeRecord
	mKeyFlag  int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf   []byte //the buffer after the main key is encoded with prefix
	mBuf      []byte //the value after the main key is encoded
	mdFuncMap map[string]interface{}
}

func NewSoStakeRecordWrap(dba iservices.IDatabaseRW, key *prototype.StakeRecord) *SoStakeRecordWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoStakeRecordWrap{dba, key, -1, nil, nil, nil}
	return result
}

func (s *SoStakeRecordWrap) CheckExist() bool {
	if s.dba == nil {
		return false
	}
	if s.mKeyFlag != -1 {
		//if you have already obtained the existence status of the primary key, use it directly
		if s.mKeyFlag == 0 {
			return false
		}
		return true
	}
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return false
	}

	res, err := s.dba.Has(keyBuf)
	if err != nil {
		return false
	}
	if res == false {
		s.mKeyFlag = 0
	} else {
		s.mKeyFlag = 1
	}
	return res
}

func (s *SoStakeRecordWrap) MustExist(errMsgs ...interface{}) *SoStakeRecordWrap {
	if !s.CheckExist() {
		panic(bindErrorInfo(fmt.Sprintf("SoStakeRecordWrap.MustExist: %v not found", s.mainKey), errMsgs...))
	}
	return s
}

func (s *SoStakeRecordWrap) MustNotExist(errMsgs ...interface{}) *SoStakeRecordWrap {
	if s.CheckExist() {
		panic(bindErrorInfo(fmt.Sprintf("SoStakeRecordWrap.MustNotExist: %v already exists", s.mainKey), errMsgs...))
	}
	return s
}

func (s *SoStakeRecordWrap) create(f func(tInfo *SoStakeRecord)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoStakeRecord{}
	f(val)
	if val.Record == nil {
		val.Record = s.mainKey
	}
	if s.CheckExist() {
		return errors.New("the main key is already exist")
	}
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return err

	}

	buf, err := proto.Marshal(val)
	if err != nil {
		return err
	}
	err = s.dba.Put(keyBuf, buf)
	if err != nil {
		return err
	}

	// update srt list keys
	if err = s.insertAllSortKeys(val); err != nil {
		s.delAllSortKeys(false, val)
		s.dba.Delete(keyBuf)
		return err
	}

	//update unique list
	if sucNames, err := s.insertAllUniKeys(val); err != nil {
		s.delAllSortKeys(false, val)
		s.delUniKeysWithNames(sucNames, val)
		s.dba.Delete(keyBuf)
		return err
	}

	s.mKeyFlag = 1
	return nil
}

func (s *SoStakeRecordWrap) Create(f func(tInfo *SoStakeRecord), errArgs ...interface{}) *SoStakeRecordWrap {
	err := s.create(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Errorf("SoStakeRecordWrap.Create failed: %s", err.Error()), errArgs...))
	}
	return s
}

func (s *SoStakeRecordWrap) getMainKeyBuf() ([]byte, error) {
	if s.mainKey == nil {
		return nil, errors.New("the main key is nil")
	}
	if s.mBuf == nil {
		var err error = nil
		s.mBuf, err = kope.Encode(s.mainKey)
		if err != nil {
			return nil, err
		}
	}
	return s.mBuf, nil
}

func (s *SoStakeRecordWrap) modify(f func(tInfo *SoStakeRecord)) error {
	if !s.CheckExist() {
		return errors.New("the SoStakeRecord table does not exist. Please create a table first")
	}
	oriTable := s.getStakeRecord()
	if oriTable == nil {
		return errors.New("fail to get origin table SoStakeRecord")
	}

	curTable := s.getStakeRecord()
	if curTable == nil {
		return errors.New("fail to create current table SoStakeRecord")
	}
	f(curTable)

	//the main key is not support modify
	if !reflect.DeepEqual(curTable.Record, oriTable.Record) {
		return errors.New("primary key does not support modification")
	}

	fieldSli, err := s.getModifiedFields(oriTable, curTable)
	if err != nil {
		return err
	}

	if fieldSli == nil || len(fieldSli) < 1 {
		return nil
	}

	//check whether modify sort and unique field to nil
	err = s.checkSortAndUniFieldValidity(curTable, fieldSli)
	if err != nil {
		return err
	}

	//check unique
	err = s.handleFieldMd(FieldMdHandleTypeCheck, curTable, fieldSli)
	if err != nil {
		return err
	}

	//delete sort and unique key
	err = s.handleFieldMd(FieldMdHandleTypeDel, oriTable, fieldSli)
	if err != nil {
		return err
	}

	//update table
	err = s.updateStakeRecord(curTable)
	if err != nil {
		return err
	}

	//insert sort and unique key
	err = s.handleFieldMd(FieldMdHandleTypeInsert, curTable, fieldSli)
	if err != nil {
		return err
	}

	return nil

}

func (s *SoStakeRecordWrap) Modify(f func(tInfo *SoStakeRecord), errArgs ...interface{}) *SoStakeRecordWrap {
	err := s.modify(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoStakeRecordWrap.Modify failed: %s", err.Error()), errArgs...))
	}
	return s
}

func (s *SoStakeRecordWrap) SetLastStakeTime(p *prototype.TimePointSec, errArgs ...interface{}) *SoStakeRecordWrap {
	err := s.modify(func(r *SoStakeRecord) {
		r.LastStakeTime = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoStakeRecordWrap.SetLastStakeTime( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoStakeRecordWrap) SetRecordReverse(p *prototype.StakeRecordReverse, errArgs ...interface{}) *SoStakeRecordWrap {
	err := s.modify(func(r *SoStakeRecord) {
		r.RecordReverse = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoStakeRecordWrap.SetRecordReverse( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoStakeRecordWrap) SetStakeAmount(p *prototype.Vest, errArgs ...interface{}) *SoStakeRecordWrap {
	err := s.modify(func(r *SoStakeRecord) {
		r.StakeAmount = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoStakeRecordWrap.SetStakeAmount( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoStakeRecordWrap) checkSortAndUniFieldValidity(curTable *SoStakeRecord, fieldSli []string) error {
	if curTable != nil && fieldSli != nil && len(fieldSli) > 0 {
		for _, fName := range fieldSli {
			if len(fName) > 0 {

				if fName == "RecordReverse" && curTable.RecordReverse == nil {
					return errors.New("sort field RecordReverse can't be modified to nil")
				}

			}
		}
	}
	return nil
}

//Get all the modified fields in the table
func (s *SoStakeRecordWrap) getModifiedFields(oriTable *SoStakeRecord, curTable *SoStakeRecord) ([]string, error) {
	if oriTable == nil {
		return nil, errors.New("table info is nil, can't get modified fields")
	}
	var list []string

	if !reflect.DeepEqual(oriTable.LastStakeTime, curTable.LastStakeTime) {
		list = append(list, "LastStakeTime")
	}

	if !reflect.DeepEqual(oriTable.RecordReverse, curTable.RecordReverse) {
		list = append(list, "RecordReverse")
	}

	if !reflect.DeepEqual(oriTable.StakeAmount, curTable.StakeAmount) {
		list = append(list, "StakeAmount")
	}

	return list, nil
}

func (s *SoStakeRecordWrap) handleFieldMd(t FieldMdHandleType, so *SoStakeRecord, fSli []string) error {
	if so == nil {
		return errors.New("fail to modify empty table")
	}

	//there is no field need to modify
	if fSli == nil || len(fSli) < 1 {
		return nil
	}

	errStr := ""
	for _, fName := range fSli {

		if fName == "LastStakeTime" {
			res := true
			if t == FieldMdHandleTypeCheck {
				res = s.mdFieldLastStakeTime(so.LastStakeTime, true, false, false, so)
				errStr = fmt.Sprintf("fail to modify exist value of %v", fName)
			} else if t == FieldMdHandleTypeDel {
				res = s.mdFieldLastStakeTime(so.LastStakeTime, false, true, false, so)
				errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", fName)
			} else if t == FieldMdHandleTypeInsert {
				res = s.mdFieldLastStakeTime(so.LastStakeTime, false, false, true, so)
				errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", fName)
			}
			if !res {
				return errors.New(errStr)
			}
		}

		if fName == "RecordReverse" {
			res := true
			if t == FieldMdHandleTypeCheck {
				res = s.mdFieldRecordReverse(so.RecordReverse, true, false, false, so)
				errStr = fmt.Sprintf("fail to modify exist value of %v", fName)
			} else if t == FieldMdHandleTypeDel {
				res = s.mdFieldRecordReverse(so.RecordReverse, false, true, false, so)
				errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", fName)
			} else if t == FieldMdHandleTypeInsert {
				res = s.mdFieldRecordReverse(so.RecordReverse, false, false, true, so)
				errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", fName)
			}
			if !res {
				return errors.New(errStr)
			}
		}

		if fName == "StakeAmount" {
			res := true
			if t == FieldMdHandleTypeCheck {
				res = s.mdFieldStakeAmount(so.StakeAmount, true, false, false, so)
				errStr = fmt.Sprintf("fail to modify exist value of %v", fName)
			} else if t == FieldMdHandleTypeDel {
				res = s.mdFieldStakeAmount(so.StakeAmount, false, true, false, so)
				errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", fName)
			} else if t == FieldMdHandleTypeInsert {
				res = s.mdFieldStakeAmount(so.StakeAmount, false, false, true, so)
				errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", fName)
			}
			if !res {
				return errors.New(errStr)
			}
		}

	}

	return nil
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoStakeRecordWrap) delSortKeyRecord(sa *SoStakeRecord) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListStakeRecordByRecord{}
	if sa == nil {
		val.Record = s.GetRecord()
	} else {
		val.Record = sa.Record
	}
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoStakeRecordWrap) insertSortKeyRecord(sa *SoStakeRecord) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListStakeRecordByRecord{}
	val.Record = sa.Record
	buf, err := proto.Marshal(&val)
	if err != nil {
		return false
	}
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Put(subBuf, buf)
	return ordErr == nil
}

func (s *SoStakeRecordWrap) delSortKeyRecordReverse(sa *SoStakeRecord) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListStakeRecordByRecordReverse{}
	if sa == nil {
		val.RecordReverse = s.GetRecordReverse()
		val.Record = s.mainKey

	} else {
		val.RecordReverse = sa.RecordReverse
		val.Record = sa.Record
	}
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoStakeRecordWrap) insertSortKeyRecordReverse(sa *SoStakeRecord) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListStakeRecordByRecordReverse{}
	val.Record = sa.Record
	val.RecordReverse = sa.RecordReverse
	buf, err := proto.Marshal(&val)
	if err != nil {
		return false
	}
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Put(subBuf, buf)
	return ordErr == nil
}

func (s *SoStakeRecordWrap) delAllSortKeys(br bool, val *SoStakeRecord) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delSortKeyRecord(val) {
		if br {
			return false
		} else {
			res = false
		}
	}
	if !s.delSortKeyRecordReverse(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoStakeRecordWrap) insertAllSortKeys(val *SoStakeRecord) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoStakeRecord fail ")
	}
	if !s.insertSortKeyRecord(val) {
		return errors.New("insert sort Field Record fail while insert table ")
	}
	if !s.insertSortKeyRecordReverse(val) {
		return errors.New("insert sort Field RecordReverse fail while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoStakeRecordWrap) removeStakeRecord() error {
	if s.dba == nil {
		return errors.New("database is nil")
	}
	//delete sort list key
	if res := s.delAllSortKeys(true, nil); !res {
		return errors.New("delAllSortKeys failed")
	}

	//delete unique list
	if res := s.delAllUniKeys(true, nil); !res {
		return errors.New("delAllUniKeys failed")
	}

	//delete table
	key, err := s.encodeMainKey()
	if err != nil {
		return fmt.Errorf("encodeMainKey failed: %s", err.Error())
	}
	err = s.dba.Delete(key)
	if err == nil {
		s.mKeyBuf = nil
		s.mKeyFlag = -1
		return nil
	} else {
		return fmt.Errorf("database.Delete failed: %s", err.Error())
	}
}

func (s *SoStakeRecordWrap) RemoveStakeRecord(errMsgs ...interface{}) *SoStakeRecordWrap {
	err := s.removeStakeRecord()
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoStakeRecordWrap.RemoveStakeRecord failed: %s", err.Error()), errMsgs...))
	}
	return s
}

////////////// SECTION Members Get/Modify ///////////////

func (s *SoStakeRecordWrap) GetLastStakeTime() *prototype.TimePointSec {
	res := true
	msg := &SoStakeRecord{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMainKey()
		if err != nil {
			res = false
		} else {
			buf, err := s.dba.Get(key)
			if err != nil {
				res = false
			}
			err = proto.Unmarshal(buf, msg)
			if err != nil {
				res = false
			} else {
				return msg.LastStakeTime
			}
		}
	}
	if !res {
		return nil

	}
	return msg.LastStakeTime
}

func (s *SoStakeRecordWrap) mdFieldLastStakeTime(p *prototype.TimePointSec, isCheck bool, isDel bool, isInsert bool,
	so *SoStakeRecord) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkLastStakeTimeIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldLastStakeTime(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldLastStakeTime(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoStakeRecordWrap) delFieldLastStakeTime(so *SoStakeRecord) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoStakeRecordWrap) insertFieldLastStakeTime(so *SoStakeRecord) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoStakeRecordWrap) checkLastStakeTimeIsMetMdCondition(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoStakeRecordWrap) GetRecord() *prototype.StakeRecord {
	res := true
	msg := &SoStakeRecord{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMainKey()
		if err != nil {
			res = false
		} else {
			buf, err := s.dba.Get(key)
			if err != nil {
				res = false
			}
			err = proto.Unmarshal(buf, msg)
			if err != nil {
				res = false
			} else {
				return msg.Record
			}
		}
	}
	if !res {
		return nil

	}
	return msg.Record
}

func (s *SoStakeRecordWrap) GetRecordReverse() *prototype.StakeRecordReverse {
	res := true
	msg := &SoStakeRecord{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMainKey()
		if err != nil {
			res = false
		} else {
			buf, err := s.dba.Get(key)
			if err != nil {
				res = false
			}
			err = proto.Unmarshal(buf, msg)
			if err != nil {
				res = false
			} else {
				return msg.RecordReverse
			}
		}
	}
	if !res {
		return nil

	}
	return msg.RecordReverse
}

func (s *SoStakeRecordWrap) mdFieldRecordReverse(p *prototype.StakeRecordReverse, isCheck bool, isDel bool, isInsert bool,
	so *SoStakeRecord) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkRecordReverseIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldRecordReverse(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldRecordReverse(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoStakeRecordWrap) delFieldRecordReverse(so *SoStakeRecord) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyRecordReverse(so) {
		return false
	}

	return true
}

func (s *SoStakeRecordWrap) insertFieldRecordReverse(so *SoStakeRecord) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyRecordReverse(so) {
		return false
	}

	return true
}

func (s *SoStakeRecordWrap) checkRecordReverseIsMetMdCondition(p *prototype.StakeRecordReverse) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoStakeRecordWrap) GetStakeAmount() *prototype.Vest {
	res := true
	msg := &SoStakeRecord{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMainKey()
		if err != nil {
			res = false
		} else {
			buf, err := s.dba.Get(key)
			if err != nil {
				res = false
			}
			err = proto.Unmarshal(buf, msg)
			if err != nil {
				res = false
			} else {
				return msg.StakeAmount
			}
		}
	}
	if !res {
		return nil

	}
	return msg.StakeAmount
}

func (s *SoStakeRecordWrap) mdFieldStakeAmount(p *prototype.Vest, isCheck bool, isDel bool, isInsert bool,
	so *SoStakeRecord) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkStakeAmountIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldStakeAmount(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldStakeAmount(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoStakeRecordWrap) delFieldStakeAmount(so *SoStakeRecord) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoStakeRecordWrap) insertFieldStakeAmount(so *SoStakeRecord) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoStakeRecordWrap) checkStakeAmountIsMetMdCondition(p *prototype.Vest) bool {
	if s.dba == nil {
		return false
	}

	return true
}

////////////// SECTION List Keys ///////////////
type SStakeRecordRecordWrap struct {
	Dba iservices.IDatabaseRW
}

func NewStakeRecordRecordWrap(db iservices.IDatabaseRW) *SStakeRecordRecordWrap {
	if db == nil {
		return nil
	}
	wrap := SStakeRecordRecordWrap{Dba: db}
	return &wrap
}

func (s *SStakeRecordRecordWrap) GetMainVal(val []byte) *prototype.StakeRecord {
	res := &SoListStakeRecordByRecord{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Record

}

func (s *SStakeRecordRecordWrap) GetSubVal(val []byte) *prototype.StakeRecord {
	res := &SoListStakeRecordByRecord{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.Record

}

func (m *SoListStakeRecordByRecord) OpeEncode() ([]byte, error) {
	pre := StakeRecordRecordTable
	sub := m.Record
	if sub == nil {
		return nil, errors.New("the pro Record is nil")
	}
	sub1 := m.Record
	if sub1 == nil {
		return nil, errors.New("the mainkey Record is nil")
	}
	kList := []interface{}{pre, sub, sub1}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
}

//Query srt by order
//
//start = nil  end = nil (query the db from start to end)
//start = nil (query from start the db)
//end = nil (query to the end of db)
//
//f: callback for each traversal , primary 、sub key、idx(the number of times it has been iterated)
//as arguments to the callback function
//if the return value of f is true,continue iterating until the end iteration;
//otherwise stop iteration immediately
//
//lastMainKey: the main key of the last one of last page
//lastSubVal: the value  of the last one of last page
//
func (s *SStakeRecordRecordWrap) ForEachByOrder(start *prototype.StakeRecord, end *prototype.StakeRecord, lastMainKey *prototype.StakeRecord,
	lastSubVal *prototype.StakeRecord, f func(mVal *prototype.StakeRecord, sVal *prototype.StakeRecord, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := StakeRecordRecordTable
	skeyList := []interface{}{pre}
	if start != nil {
		skeyList = append(skeyList, start)
		if lastMainKey != nil {
			skeyList = append(skeyList, lastMainKey, kope.MinimalKey)
		}
	} else {
		if lastMainKey != nil && lastSubVal != nil {
			skeyList = append(skeyList, lastSubVal, lastMainKey, kope.MinimalKey)
		}
		skeyList = append(skeyList, kope.MinimalKey)
	}
	sBuf, cErr := kope.EncodeSlice(skeyList)
	if cErr != nil {
		return cErr
	}
	eKeyList := []interface{}{pre}
	if end != nil {
		eKeyList = append(eKeyList, end)
	} else {
		eKeyList = append(eKeyList, kope.MaximumKey)
	}
	eBuf, cErr := kope.EncodeSlice(eKeyList)
	if cErr != nil {
		return cErr
	}
	var idx uint32 = 0
	s.Dba.Iterate(sBuf, eBuf, false, func(key, value []byte) bool {
		idx++
		return f(s.GetMainVal(value), s.GetSubVal(value), idx)
	})
	return nil
}

////////////// SECTION List Keys ///////////////
type SStakeRecordRecordReverseWrap struct {
	Dba iservices.IDatabaseRW
}

func NewStakeRecordRecordReverseWrap(db iservices.IDatabaseRW) *SStakeRecordRecordReverseWrap {
	if db == nil {
		return nil
	}
	wrap := SStakeRecordRecordReverseWrap{Dba: db}
	return &wrap
}

func (s *SStakeRecordRecordReverseWrap) GetMainVal(val []byte) *prototype.StakeRecord {
	res := &SoListStakeRecordByRecordReverse{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Record

}

func (s *SStakeRecordRecordReverseWrap) GetSubVal(val []byte) *prototype.StakeRecordReverse {
	res := &SoListStakeRecordByRecordReverse{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.RecordReverse

}

func (m *SoListStakeRecordByRecordReverse) OpeEncode() ([]byte, error) {
	pre := StakeRecordRecordReverseTable
	sub := m.RecordReverse
	if sub == nil {
		return nil, errors.New("the pro RecordReverse is nil")
	}
	sub1 := m.Record
	if sub1 == nil {
		return nil, errors.New("the mainkey Record is nil")
	}
	kList := []interface{}{pre, sub, sub1}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
}

//Query srt by order
//
//start = nil  end = nil (query the db from start to end)
//start = nil (query from start the db)
//end = nil (query to the end of db)
//
//f: callback for each traversal , primary 、sub key、idx(the number of times it has been iterated)
//as arguments to the callback function
//if the return value of f is true,continue iterating until the end iteration;
//otherwise stop iteration immediately
//
//lastMainKey: the main key of the last one of last page
//lastSubVal: the value  of the last one of last page
//
func (s *SStakeRecordRecordReverseWrap) ForEachByOrder(start *prototype.StakeRecordReverse, end *prototype.StakeRecordReverse, lastMainKey *prototype.StakeRecord,
	lastSubVal *prototype.StakeRecordReverse, f func(mVal *prototype.StakeRecord, sVal *prototype.StakeRecordReverse, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := StakeRecordRecordReverseTable
	skeyList := []interface{}{pre}
	if start != nil {
		skeyList = append(skeyList, start)
		if lastMainKey != nil {
			skeyList = append(skeyList, lastMainKey, kope.MinimalKey)
		}
	} else {
		if lastMainKey != nil && lastSubVal != nil {
			skeyList = append(skeyList, lastSubVal, lastMainKey, kope.MinimalKey)
		}
		skeyList = append(skeyList, kope.MinimalKey)
	}
	sBuf, cErr := kope.EncodeSlice(skeyList)
	if cErr != nil {
		return cErr
	}
	eKeyList := []interface{}{pre}
	if end != nil {
		eKeyList = append(eKeyList, end)
	} else {
		eKeyList = append(eKeyList, kope.MaximumKey)
	}
	eBuf, cErr := kope.EncodeSlice(eKeyList)
	if cErr != nil {
		return cErr
	}
	var idx uint32 = 0
	s.Dba.Iterate(sBuf, eBuf, false, func(key, value []byte) bool {
		idx++
		return f(s.GetMainVal(value), s.GetSubVal(value), idx)
	})
	return nil
}

/////////////// SECTION Private function ////////////////

func (s *SoStakeRecordWrap) update(sa *SoStakeRecord) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	buf, err := proto.Marshal(sa)
	if err != nil {
		return false
	}

	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return false
	}

	return s.dba.Put(keyBuf, buf) == nil
}

func (s *SoStakeRecordWrap) getStakeRecord() *SoStakeRecord {
	if s.dba == nil {
		return nil
	}
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return nil
	}
	resBuf, err := s.dba.Get(keyBuf)

	if err != nil {
		return nil
	}

	res := &SoStakeRecord{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoStakeRecordWrap) updateStakeRecord(so *SoStakeRecord) error {
	if s.dba == nil {
		return errors.New("update fail:the db is nil")
	}

	if so == nil {
		return errors.New("update fail: the SoStakeRecord is nil")
	}

	key, err := s.encodeMainKey()
	if err != nil {
		return nil
	}

	buf, err := proto.Marshal(so)
	if err != nil {
		return err
	}

	err = s.dba.Put(key, buf)
	if err != nil {
		return err
	}

	return nil
}

func (s *SoStakeRecordWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := StakeRecordRecordRow
	sub := s.mainKey
	if sub == nil {
		return nil, errors.New("the mainKey is nil")
	}
	preBuf, err := kope.Encode(pre)
	if err != nil {
		return nil, err
	}
	mBuf, err := s.getMainKeyBuf()
	if err != nil {
		return nil, err
	}
	list := make([][]byte, 2)
	list[0] = preBuf
	list[1] = mBuf
	s.mKeyBuf = kope.PackList(list)
	return s.mKeyBuf, nil
}

////////////// Unique Query delete/insert/query ///////////////

func (s *SoStakeRecordWrap) delAllUniKeys(br bool, val *SoStakeRecord) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delUniKeyRecord(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoStakeRecordWrap) delUniKeysWithNames(names map[string]string, val *SoStakeRecord) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if len(names["Record"]) > 0 {
		if !s.delUniKeyRecord(val) {
			res = false
		}
	}

	return res
}

func (s *SoStakeRecordWrap) insertAllUniKeys(val *SoStakeRecord) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoStakeRecord fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyRecord(val) {
		return sucFields, errors.New("insert unique Field Record fail while insert table ")
	}
	sucFields["Record"] = "Record"

	return sucFields, nil
}

func (s *SoStakeRecordWrap) delUniKeyRecord(sa *SoStakeRecord) bool {
	if s.dba == nil {
		return false
	}
	pre := StakeRecordRecordUniTable
	kList := []interface{}{pre}
	if sa != nil {
		if sa.Record == nil {
			return false
		}

		sub := sa.Record
		kList = append(kList, sub)
	} else {
		sub := s.GetRecord()
		if sub == nil {
			return true
		}

		kList = append(kList, sub)

	}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoStakeRecordWrap) insertUniKeyRecord(sa *SoStakeRecord) bool {
	if s.dba == nil || sa == nil {
		return false
	}

	pre := StakeRecordRecordUniTable
	sub := sa.Record
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	res, err := s.dba.Has(kBuf)
	if err == nil && res == true {
		//the unique key is already exist
		return false
	}
	val := SoUniqueStakeRecordByRecord{}
	val.Record = sa.Record

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniStakeRecordRecordWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniStakeRecordRecordWrap(db iservices.IDatabaseRW) *UniStakeRecordRecordWrap {
	if db == nil {
		return nil
	}
	wrap := UniStakeRecordRecordWrap{Dba: db}
	return &wrap
}

func (s *UniStakeRecordRecordWrap) UniQueryRecord(start *prototype.StakeRecord) *SoStakeRecordWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := StakeRecordRecordUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueStakeRecordByRecord{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoStakeRecordWrap(s.Dba, res.Record)

			return wrap
		}
	}
	return nil
}
