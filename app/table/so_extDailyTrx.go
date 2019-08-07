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
	ExtDailyTrxDateTable    uint32 = 4241530934
	ExtDailyTrxCountTable   uint32 = 1672192129
	ExtDailyTrxDateUniTable uint32 = 111567975

	ExtDailyTrxDateRow uint32 = 3166181191
)

////////////// SECTION Wrap Define ///////////////
type SoExtDailyTrxWrap struct {
	dba       iservices.IDatabaseRW
	mainKey   *prototype.TimePointSec
	mKeyFlag  int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf   []byte //the buffer after the main key is encoded with prefix
	mBuf      []byte //the value after the main key is encoded
	mdFuncMap map[string]interface{}
}

func NewSoExtDailyTrxWrap(dba iservices.IDatabaseRW, key *prototype.TimePointSec) *SoExtDailyTrxWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoExtDailyTrxWrap{dba, key, -1, nil, nil, nil}
	return result
}

func (s *SoExtDailyTrxWrap) CheckExist() bool {
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

func (s *SoExtDailyTrxWrap) MustExist(errMsgs ...interface{}) *SoExtDailyTrxWrap {
	if !s.CheckExist() {
		panic(bindErrorInfo(fmt.Sprintf("SoExtDailyTrxWrap.MustExist: %v not found", s.mainKey), errMsgs...))
	}
	return s
}

func (s *SoExtDailyTrxWrap) MustNotExist(errMsgs ...interface{}) *SoExtDailyTrxWrap {
	if s.CheckExist() {
		panic(bindErrorInfo(fmt.Sprintf("SoExtDailyTrxWrap.MustNotExist: %v already exists", s.mainKey), errMsgs...))
	}
	return s
}

func (s *SoExtDailyTrxWrap) create(f func(tInfo *SoExtDailyTrx)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoExtDailyTrx{}
	f(val)
	if val.Date == nil {
		val.Date = s.mainKey
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

func (s *SoExtDailyTrxWrap) Create(f func(tInfo *SoExtDailyTrx), errArgs ...interface{}) *SoExtDailyTrxWrap {
	err := s.create(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Errorf("SoExtDailyTrxWrap.Create failed: %s", err.Error()), errArgs...))
	}
	return s
}

func (s *SoExtDailyTrxWrap) getMainKeyBuf() ([]byte, error) {
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

func (s *SoExtDailyTrxWrap) modify(f func(tInfo *SoExtDailyTrx)) error {
	if !s.CheckExist() {
		return errors.New("the SoExtDailyTrx table does not exist. Please create a table first")
	}
	oriTable := s.getExtDailyTrx()
	if oriTable == nil {
		return errors.New("fail to get origin table SoExtDailyTrx")
	}

	curTable := s.getExtDailyTrx()
	if curTable == nil {
		return errors.New("fail to create current table SoExtDailyTrx")
	}
	f(curTable)

	//the main key is not support modify
	if !reflect.DeepEqual(curTable.Date, oriTable.Date) {
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
	err = s.updateExtDailyTrx(curTable)
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

func (s *SoExtDailyTrxWrap) Modify(f func(tInfo *SoExtDailyTrx), errArgs ...interface{}) *SoExtDailyTrxWrap {
	err := s.modify(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoExtDailyTrxWrap.Modify failed: %s", err.Error()), errArgs...))
	}
	return s
}

func (s *SoExtDailyTrxWrap) SetCount(p uint64, errArgs ...interface{}) *SoExtDailyTrxWrap {
	err := s.modify(func(r *SoExtDailyTrx) {
		r.Count = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoExtDailyTrxWrap.SetCount( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoExtDailyTrxWrap) checkSortAndUniFieldValidity(curTable *SoExtDailyTrx, fieldSli []string) error {
	if curTable != nil && fieldSli != nil && len(fieldSli) > 0 {
		for _, fName := range fieldSli {
			if len(fName) > 0 {

			}
		}
	}
	return nil
}

//Get all the modified fields in the table
func (s *SoExtDailyTrxWrap) getModifiedFields(oriTable *SoExtDailyTrx, curTable *SoExtDailyTrx) ([]string, error) {
	if oriTable == nil {
		return nil, errors.New("table info is nil, can't get modified fields")
	}
	var list []string

	if !reflect.DeepEqual(oriTable.Count, curTable.Count) {
		list = append(list, "Count")
	}

	return list, nil
}

func (s *SoExtDailyTrxWrap) handleFieldMd(t FieldMdHandleType, so *SoExtDailyTrx, fSli []string) error {
	if so == nil {
		return errors.New("fail to modify empty table")
	}

	//there is no field need to modify
	if fSli == nil || len(fSli) < 1 {
		return nil
	}

	errStr := ""
	for _, fName := range fSli {

		if fName == "Count" {
			res := true
			if t == FieldMdHandleTypeCheck {
				res = s.mdFieldCount(so.Count, true, false, false, so)
				errStr = fmt.Sprintf("fail to modify exist value of %v", fName)
			} else if t == FieldMdHandleTypeDel {
				res = s.mdFieldCount(so.Count, false, true, false, so)
				errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", fName)
			} else if t == FieldMdHandleTypeInsert {
				res = s.mdFieldCount(so.Count, false, false, true, so)
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

func (s *SoExtDailyTrxWrap) delSortKeyDate(sa *SoExtDailyTrx) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListExtDailyTrxByDate{}
	if sa == nil {
		val.Date = s.GetDate()
	} else {
		val.Date = sa.Date
	}
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoExtDailyTrxWrap) insertSortKeyDate(sa *SoExtDailyTrx) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListExtDailyTrxByDate{}
	val.Date = sa.Date
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

func (s *SoExtDailyTrxWrap) delSortKeyCount(sa *SoExtDailyTrx) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListExtDailyTrxByCount{}
	if sa == nil {
		val.Count = s.GetCount()
		val.Date = s.mainKey

	} else {
		val.Count = sa.Count
		val.Date = sa.Date
	}
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoExtDailyTrxWrap) insertSortKeyCount(sa *SoExtDailyTrx) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListExtDailyTrxByCount{}
	val.Date = sa.Date
	val.Count = sa.Count
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

func (s *SoExtDailyTrxWrap) delAllSortKeys(br bool, val *SoExtDailyTrx) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delSortKeyDate(val) {
		if br {
			return false
		} else {
			res = false
		}
	}
	if !s.delSortKeyCount(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoExtDailyTrxWrap) insertAllSortKeys(val *SoExtDailyTrx) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoExtDailyTrx fail ")
	}
	if !s.insertSortKeyDate(val) {
		return errors.New("insert sort Field Date fail while insert table ")
	}
	if !s.insertSortKeyCount(val) {
		return errors.New("insert sort Field Count fail while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoExtDailyTrxWrap) removeExtDailyTrx() error {
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

func (s *SoExtDailyTrxWrap) RemoveExtDailyTrx(errMsgs ...interface{}) *SoExtDailyTrxWrap {
	err := s.removeExtDailyTrx()
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoExtDailyTrxWrap.RemoveExtDailyTrx failed: %s", err.Error()), errMsgs...))
	}
	return s
}

////////////// SECTION Members Get/Modify ///////////////

func (s *SoExtDailyTrxWrap) GetCount() uint64 {
	res := true
	msg := &SoExtDailyTrx{}
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
				return msg.Count
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.Count
}

func (s *SoExtDailyTrxWrap) mdFieldCount(p uint64, isCheck bool, isDel bool, isInsert bool,
	so *SoExtDailyTrx) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkCountIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldCount(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldCount(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoExtDailyTrxWrap) delFieldCount(so *SoExtDailyTrx) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyCount(so) {
		return false
	}

	return true
}

func (s *SoExtDailyTrxWrap) insertFieldCount(so *SoExtDailyTrx) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyCount(so) {
		return false
	}

	return true
}

func (s *SoExtDailyTrxWrap) checkCountIsMetMdCondition(p uint64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoExtDailyTrxWrap) GetDate() *prototype.TimePointSec {
	res := true
	msg := &SoExtDailyTrx{}
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
				return msg.Date
			}
		}
	}
	if !res {
		return nil

	}
	return msg.Date
}

////////////// SECTION List Keys ///////////////
type SExtDailyTrxDateWrap struct {
	Dba iservices.IDatabaseRW
}

func NewExtDailyTrxDateWrap(db iservices.IDatabaseRW) *SExtDailyTrxDateWrap {
	if db == nil {
		return nil
	}
	wrap := SExtDailyTrxDateWrap{Dba: db}
	return &wrap
}

func (s *SExtDailyTrxDateWrap) GetMainVal(val []byte) *prototype.TimePointSec {
	res := &SoListExtDailyTrxByDate{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Date

}

func (s *SExtDailyTrxDateWrap) GetSubVal(val []byte) *prototype.TimePointSec {
	res := &SoListExtDailyTrxByDate{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.Date

}

func (m *SoListExtDailyTrxByDate) OpeEncode() ([]byte, error) {
	pre := ExtDailyTrxDateTable
	sub := m.Date
	if sub == nil {
		return nil, errors.New("the pro Date is nil")
	}
	sub1 := m.Date
	if sub1 == nil {
		return nil, errors.New("the mainkey Date is nil")
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
func (s *SExtDailyTrxDateWrap) ForEachByOrder(start *prototype.TimePointSec, end *prototype.TimePointSec, lastMainKey *prototype.TimePointSec,
	lastSubVal *prototype.TimePointSec, f func(mVal *prototype.TimePointSec, sVal *prototype.TimePointSec, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := ExtDailyTrxDateTable
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
type SExtDailyTrxCountWrap struct {
	Dba iservices.IDatabaseRW
}

func NewExtDailyTrxCountWrap(db iservices.IDatabaseRW) *SExtDailyTrxCountWrap {
	if db == nil {
		return nil
	}
	wrap := SExtDailyTrxCountWrap{Dba: db}
	return &wrap
}

func (s *SExtDailyTrxCountWrap) GetMainVal(val []byte) *prototype.TimePointSec {
	res := &SoListExtDailyTrxByCount{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Date

}

func (s *SExtDailyTrxCountWrap) GetSubVal(val []byte) *uint64 {
	res := &SoListExtDailyTrxByCount{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return &res.Count

}

func (m *SoListExtDailyTrxByCount) OpeEncode() ([]byte, error) {
	pre := ExtDailyTrxCountTable
	sub := m.Count

	sub1 := m.Date
	if sub1 == nil {
		return nil, errors.New("the mainkey Date is nil")
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
func (s *SExtDailyTrxCountWrap) ForEachByOrder(start *uint64, end *uint64, lastMainKey *prototype.TimePointSec,
	lastSubVal *uint64, f func(mVal *prototype.TimePointSec, sVal *uint64, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := ExtDailyTrxCountTable
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

func (s *SoExtDailyTrxWrap) update(sa *SoExtDailyTrx) bool {
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

func (s *SoExtDailyTrxWrap) getExtDailyTrx() *SoExtDailyTrx {
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

	res := &SoExtDailyTrx{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoExtDailyTrxWrap) updateExtDailyTrx(so *SoExtDailyTrx) error {
	if s.dba == nil {
		return errors.New("update fail:the db is nil")
	}

	if so == nil {
		return errors.New("update fail: the SoExtDailyTrx is nil")
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

func (s *SoExtDailyTrxWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := ExtDailyTrxDateRow
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

func (s *SoExtDailyTrxWrap) delAllUniKeys(br bool, val *SoExtDailyTrx) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delUniKeyDate(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoExtDailyTrxWrap) delUniKeysWithNames(names map[string]string, val *SoExtDailyTrx) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if len(names["Date"]) > 0 {
		if !s.delUniKeyDate(val) {
			res = false
		}
	}

	return res
}

func (s *SoExtDailyTrxWrap) insertAllUniKeys(val *SoExtDailyTrx) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoExtDailyTrx fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyDate(val) {
		return sucFields, errors.New("insert unique Field Date fail while insert table ")
	}
	sucFields["Date"] = "Date"

	return sucFields, nil
}

func (s *SoExtDailyTrxWrap) delUniKeyDate(sa *SoExtDailyTrx) bool {
	if s.dba == nil {
		return false
	}
	pre := ExtDailyTrxDateUniTable
	kList := []interface{}{pre}
	if sa != nil {
		if sa.Date == nil {
			return false
		}

		sub := sa.Date
		kList = append(kList, sub)
	} else {
		sub := s.GetDate()
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

func (s *SoExtDailyTrxWrap) insertUniKeyDate(sa *SoExtDailyTrx) bool {
	if s.dba == nil || sa == nil {
		return false
	}

	pre := ExtDailyTrxDateUniTable
	sub := sa.Date
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
	val := SoUniqueExtDailyTrxByDate{}
	val.Date = sa.Date

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniExtDailyTrxDateWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniExtDailyTrxDateWrap(db iservices.IDatabaseRW) *UniExtDailyTrxDateWrap {
	if db == nil {
		return nil
	}
	wrap := UniExtDailyTrxDateWrap{Dba: db}
	return &wrap
}

func (s *UniExtDailyTrxDateWrap) UniQueryDate(start *prototype.TimePointSec) *SoExtDailyTrxWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := ExtDailyTrxDateUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueExtDailyTrxByDate{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoExtDailyTrxWrap(s.Dba, res.Date)

			return wrap
		}
	}
	return nil
}
