package table

import (
	"errors"
	fmt "fmt"
	"reflect"

	"github.com/coschain/contentos-go/common/encoding/kope"
	"github.com/coschain/contentos-go/iservices"
	prototype "github.com/coschain/contentos-go/prototype"
	proto "github.com/golang/protobuf/proto"
)

////////////// SECTION Prefix Mark ///////////////
var (
	ExtHourTrxHourTable    uint32 = 2691214849
	ExtHourTrxCountTable   uint32 = 1734812738
	ExtHourTrxHourUniTable uint32 = 2092663070

	ExtHourTrxHourRow uint32 = 55872904
)

////////////// SECTION Wrap Define ///////////////
type SoExtHourTrxWrap struct {
	dba       iservices.IDatabaseRW
	mainKey   *prototype.TimePointSec
	mKeyFlag  int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf   []byte //the buffer after the main key is encoded with prefix
	mBuf      []byte //the value after the main key is encoded
	mdFuncMap map[string]interface{}
}

func NewSoExtHourTrxWrap(dba iservices.IDatabaseRW, key *prototype.TimePointSec) *SoExtHourTrxWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoExtHourTrxWrap{dba, key, -1, nil, nil, nil}
	return result
}

func (s *SoExtHourTrxWrap) CheckExist() bool {
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

func (s *SoExtHourTrxWrap) Create(f func(tInfo *SoExtHourTrx)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoExtHourTrx{}
	f(val)
	if val.Hour == nil {
		val.Hour = s.mainKey
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

func (s *SoExtHourTrxWrap) getMainKeyBuf() ([]byte, error) {
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

func (s *SoExtHourTrxWrap) Modify(f func(tInfo *SoExtHourTrx)) error {
	if !s.CheckExist() {
		return errors.New("the SoExtHourTrx table does not exist. Please create a table first")
	}
	oriTable := s.getExtHourTrx()
	if oriTable == nil {
		return errors.New("fail to get origin table SoExtHourTrx")
	}
	curTable := *oriTable
	f(&curTable)

	//the main key is not support modify
	if !reflect.DeepEqual(curTable.Hour, oriTable.Hour) {
		return errors.New("primary key does not support modification")
	}

	fieldSli, err := s.getModifiedFields(oriTable, &curTable)
	if err != nil {
		return err
	}

	if fieldSli == nil || len(fieldSli) < 1 {
		return nil
	}

	//check whether modify sort and unique field to nil
	err = s.checkSortAndUniFieldValidity(&curTable, fieldSli)
	if err != nil {
		return err
	}

	//check unique
	err = s.handleFieldMd(FieldMdHandleTypeCheck, &curTable, fieldSli)
	if err != nil {
		return err
	}

	//delete sort and unique key
	err = s.handleFieldMd(FieldMdHandleTypeDel, oriTable, fieldSli)
	if err != nil {
		return err
	}

	//update table
	err = s.updateExtHourTrx(&curTable)
	if err != nil {
		return err
	}

	//insert sort and unique key
	err = s.handleFieldMd(FieldMdHandleTypeInsert, &curTable, fieldSli)
	if err != nil {
		return err
	}

	return nil

}

func (s *SoExtHourTrxWrap) checkSortAndUniFieldValidity(curTable *SoExtHourTrx, fieldSli []string) error {
	if curTable != nil && fieldSli != nil && len(fieldSli) > 0 {
		for _, fName := range fieldSli {
			if len(fName) > 0 {

			}
		}
	}
	return nil
}

//Get all the modified fields in the table
func (s *SoExtHourTrxWrap) getModifiedFields(oriTable *SoExtHourTrx, curTable *SoExtHourTrx) ([]string, error) {
	if oriTable == nil {
		return nil, errors.New("table info is nil, can't get modified fields")
	}
	var list []string

	if !reflect.DeepEqual(oriTable.Count, curTable.Count) {
		list = append(list, "Count")
	}

	return list, nil
}

func (s *SoExtHourTrxWrap) handleFieldMd(t FieldMdHandleType, so *SoExtHourTrx, fSli []string) error {
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

func (s *SoExtHourTrxWrap) delSortKeyHour(sa *SoExtHourTrx) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListExtHourTrxByHour{}
	if sa == nil {
		val.Hour = s.GetHour()
	} else {
		val.Hour = sa.Hour
	}
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoExtHourTrxWrap) insertSortKeyHour(sa *SoExtHourTrx) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListExtHourTrxByHour{}
	val.Hour = sa.Hour
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

func (s *SoExtHourTrxWrap) delSortKeyCount(sa *SoExtHourTrx) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListExtHourTrxByCount{}
	if sa == nil {
		val.Count = s.GetCount()
		val.Hour = s.mainKey

	} else {
		val.Count = sa.Count
		val.Hour = sa.Hour
	}
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoExtHourTrxWrap) insertSortKeyCount(sa *SoExtHourTrx) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListExtHourTrxByCount{}
	val.Hour = sa.Hour
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

func (s *SoExtHourTrxWrap) delAllSortKeys(br bool, val *SoExtHourTrx) bool {
	if s.dba == nil {
		return false
	}
	res := true

	if !s.delSortKeyCount(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoExtHourTrxWrap) insertAllSortKeys(val *SoExtHourTrx) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoExtHourTrx fail ")
	}

	if !s.insertSortKeyCount(val) {
		return errors.New("insert sort Field Count fail while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoExtHourTrxWrap) RemoveExtHourTrx() bool {
	if s.dba == nil {
		return false
	}
	//delete sort list key
	if res := s.delAllSortKeys(true, nil); !res {
		return false
	}

	//delete unique list
	if res := s.delAllUniKeys(true, nil); !res {
		return false
	}

	//delete table
	key, err := s.encodeMainKey()
	if err != nil {
		return false
	}
	err = s.dba.Delete(key)
	if err == nil {
		s.mKeyBuf = nil
		s.mKeyFlag = -1
		return true
	} else {
		return false
	}
}

////////////// SECTION Members Get/Modify ///////////////

func (s *SoExtHourTrxWrap) GetCount() uint64 {
	res := true
	msg := &SoExtHourTrx{}
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

func (s *SoExtHourTrxWrap) mdFieldCount(p uint64, isCheck bool, isDel bool, isInsert bool,
	so *SoExtHourTrx) bool {
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

func (s *SoExtHourTrxWrap) delFieldCount(so *SoExtHourTrx) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyCount(so) {
		return false
	}

	return true
}

func (s *SoExtHourTrxWrap) insertFieldCount(so *SoExtHourTrx) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyCount(so) {
		return false
	}

	return true
}

func (s *SoExtHourTrxWrap) checkCountIsMetMdCondition(p uint64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoExtHourTrxWrap) GetHour() *prototype.TimePointSec {
	res := true
	msg := &SoExtHourTrx{}
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
				return msg.Hour
			}
		}
	}
	if !res {
		return nil

	}
	return msg.Hour
}

////////////// SECTION List Keys ///////////////
type SExtHourTrxHourWrap struct {
	Dba iservices.IDatabaseRW
}

func NewExtHourTrxHourWrap(db iservices.IDatabaseRW) *SExtHourTrxHourWrap {
	if db == nil {
		return nil
	}
	wrap := SExtHourTrxHourWrap{Dba: db}
	return &wrap
}

func (s *SExtHourTrxHourWrap) GetMainVal(val []byte) *prototype.TimePointSec {
	res := &SoListExtHourTrxByHour{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Hour

}

func (s *SExtHourTrxHourWrap) GetSubVal(val []byte) *prototype.TimePointSec {
	res := &SoListExtHourTrxByHour{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.Hour

}

func (m *SoListExtHourTrxByHour) OpeEncode() ([]byte, error) {
	pre := ExtHourTrxHourTable
	sub := m.Hour
	if sub == nil {
		return nil, errors.New("the pro Hour is nil")
	}
	sub1 := m.Hour
	if sub1 == nil {
		return nil, errors.New("the mainkey Hour is nil")
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
func (s *SExtHourTrxHourWrap) ForEachByOrder(start *prototype.TimePointSec, end *prototype.TimePointSec, lastMainKey *prototype.TimePointSec,
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
	pre := ExtHourTrxHourTable
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
type SExtHourTrxCountWrap struct {
	Dba iservices.IDatabaseRW
}

func NewExtHourTrxCountWrap(db iservices.IDatabaseRW) *SExtHourTrxCountWrap {
	if db == nil {
		return nil
	}
	wrap := SExtHourTrxCountWrap{Dba: db}
	return &wrap
}

func (s *SExtHourTrxCountWrap) GetMainVal(val []byte) *prototype.TimePointSec {
	res := &SoListExtHourTrxByCount{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Hour

}

func (s *SExtHourTrxCountWrap) GetSubVal(val []byte) *uint64 {
	res := &SoListExtHourTrxByCount{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return &res.Count

}

func (m *SoListExtHourTrxByCount) OpeEncode() ([]byte, error) {
	pre := ExtHourTrxCountTable
	sub := m.Count

	sub1 := m.Hour
	if sub1 == nil {
		return nil, errors.New("the mainkey Hour is nil")
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
func (s *SExtHourTrxCountWrap) ForEachByOrder(start *uint64, end *uint64, lastMainKey *prototype.TimePointSec,
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
	pre := ExtHourTrxCountTable
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

func (s *SoExtHourTrxWrap) update(sa *SoExtHourTrx) bool {
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

func (s *SoExtHourTrxWrap) getExtHourTrx() *SoExtHourTrx {
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

	res := &SoExtHourTrx{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoExtHourTrxWrap) updateExtHourTrx(so *SoExtHourTrx) error {
	if s.dba == nil {
		return errors.New("update fail:the db is nil")
	}

	if so == nil {
		return errors.New("update fail: the SoExtHourTrx is nil")
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

func (s *SoExtHourTrxWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := ExtHourTrxHourRow
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

func (s *SoExtHourTrxWrap) delAllUniKeys(br bool, val *SoExtHourTrx) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delUniKeyHour(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoExtHourTrxWrap) delUniKeysWithNames(names map[string]string, val *SoExtHourTrx) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if len(names["Hour"]) > 0 {
		if !s.delUniKeyHour(val) {
			res = false
		}
	}

	return res
}

func (s *SoExtHourTrxWrap) insertAllUniKeys(val *SoExtHourTrx) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoExtHourTrx fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyHour(val) {
		return sucFields, errors.New("insert unique Field Hour fail while insert table ")
	}
	sucFields["Hour"] = "Hour"

	return sucFields, nil
}

func (s *SoExtHourTrxWrap) delUniKeyHour(sa *SoExtHourTrx) bool {
	if s.dba == nil {
		return false
	}
	pre := ExtHourTrxHourUniTable
	kList := []interface{}{pre}
	if sa != nil {
		if sa.Hour == nil {
			return false
		}

		sub := sa.Hour
		kList = append(kList, sub)
	} else {
		sub := s.GetHour()
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

func (s *SoExtHourTrxWrap) insertUniKeyHour(sa *SoExtHourTrx) bool {
	if s.dba == nil || sa == nil {
		return false
	}

	pre := ExtHourTrxHourUniTable
	sub := sa.Hour
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
	val := SoUniqueExtHourTrxByHour{}
	val.Hour = sa.Hour

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniExtHourTrxHourWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniExtHourTrxHourWrap(db iservices.IDatabaseRW) *UniExtHourTrxHourWrap {
	if db == nil {
		return nil
	}
	wrap := UniExtHourTrxHourWrap{Dba: db}
	return &wrap
}

func (s *UniExtHourTrxHourWrap) UniQueryHour(start *prototype.TimePointSec) *SoExtHourTrxWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := ExtHourTrxHourUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueExtHourTrxByHour{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoExtHourTrxWrap(s.Dba, res.Hour)

			return wrap
		}
	}
	return nil
}
