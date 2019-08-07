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
	TransactionObjectExpirationTable uint32 = 3273070683
	TransactionObjectTrxIdUniTable   uint32 = 482982412

	TransactionObjectTrxIdRow uint32 = 3516269592
)

////////////// SECTION Wrap Define ///////////////
type SoTransactionObjectWrap struct {
	dba       iservices.IDatabaseRW
	mainKey   *prototype.Sha256
	mKeyFlag  int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf   []byte //the buffer after the main key is encoded with prefix
	mBuf      []byte //the value after the main key is encoded
	mdFuncMap map[string]interface{}
}

func NewSoTransactionObjectWrap(dba iservices.IDatabaseRW, key *prototype.Sha256) *SoTransactionObjectWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoTransactionObjectWrap{dba, key, -1, nil, nil, nil}
	return result
}

func (s *SoTransactionObjectWrap) CheckExist() bool {
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

func (s *SoTransactionObjectWrap) MustExist() *SoTransactionObjectWrap {
	if !s.CheckExist() {
		panic(fmt.Errorf("SoTransactionObjectWrap.MustExist: %v not found", s.mainKey))
	}
	return s
}

func (s *SoTransactionObjectWrap) MustNotExist() *SoTransactionObjectWrap {
	if s.CheckExist() {
		panic(fmt.Errorf("SoTransactionObjectWrap.MustNotExist: %v already exists", s.mainKey))
	}
	return s
}

func (s *SoTransactionObjectWrap) create(f func(tInfo *SoTransactionObject)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoTransactionObject{}
	f(val)
	if val.TrxId == nil {
		val.TrxId = s.mainKey
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

func (s *SoTransactionObjectWrap) Create(f func(tInfo *SoTransactionObject), errArgs ...interface{}) *SoTransactionObjectWrap {
	err := s.create(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Errorf("SoTransactionObjectWrap.Create failed: %s", err.Error()), errArgs...))
	}
	return s
}

func (s *SoTransactionObjectWrap) getMainKeyBuf() ([]byte, error) {
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

func (s *SoTransactionObjectWrap) modify(f func(tInfo *SoTransactionObject)) error {
	if !s.CheckExist() {
		return errors.New("the SoTransactionObject table does not exist. Please create a table first")
	}
	oriTable := s.getTransactionObject()
	if oriTable == nil {
		return errors.New("fail to get origin table SoTransactionObject")
	}
	curTable := *oriTable
	f(&curTable)

	//the main key is not support modify
	if !reflect.DeepEqual(curTable.TrxId, oriTable.TrxId) {
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
	err = s.updateTransactionObject(&curTable)
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

func (s *SoTransactionObjectWrap) Modify(f func(tInfo *SoTransactionObject), errArgs ...interface{}) *SoTransactionObjectWrap {
	err := s.modify(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoTransactionObjectWrap.Modify failed: %s", err.Error()), errArgs...))
	}
	return s
}

func (s *SoTransactionObjectWrap) SetExpiration(p *prototype.TimePointSec, errArgs ...interface{}) *SoTransactionObjectWrap {
	err := s.modify(func(r *SoTransactionObject) {
		r.Expiration = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoTransactionObjectWrap.SetExpiration( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoTransactionObjectWrap) checkSortAndUniFieldValidity(curTable *SoTransactionObject, fieldSli []string) error {
	if curTable != nil && fieldSli != nil && len(fieldSli) > 0 {
		for _, fName := range fieldSli {
			if len(fName) > 0 {

				if fName == "Expiration" && curTable.Expiration == nil {
					return errors.New("sort field Expiration can't be modified to nil")
				}

			}
		}
	}
	return nil
}

//Get all the modified fields in the table
func (s *SoTransactionObjectWrap) getModifiedFields(oriTable *SoTransactionObject, curTable *SoTransactionObject) ([]string, error) {
	if oriTable == nil {
		return nil, errors.New("table info is nil, can't get modified fields")
	}
	var list []string

	if !reflect.DeepEqual(oriTable.Expiration, curTable.Expiration) {
		list = append(list, "Expiration")
	}

	return list, nil
}

func (s *SoTransactionObjectWrap) handleFieldMd(t FieldMdHandleType, so *SoTransactionObject, fSli []string) error {
	if so == nil {
		return errors.New("fail to modify empty table")
	}

	//there is no field need to modify
	if fSli == nil || len(fSli) < 1 {
		return nil
	}

	errStr := ""
	for _, fName := range fSli {

		if fName == "Expiration" {
			res := true
			if t == FieldMdHandleTypeCheck {
				res = s.mdFieldExpiration(so.Expiration, true, false, false, so)
				errStr = fmt.Sprintf("fail to modify exist value of %v", fName)
			} else if t == FieldMdHandleTypeDel {
				res = s.mdFieldExpiration(so.Expiration, false, true, false, so)
				errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", fName)
			} else if t == FieldMdHandleTypeInsert {
				res = s.mdFieldExpiration(so.Expiration, false, false, true, so)
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

func (s *SoTransactionObjectWrap) delSortKeyExpiration(sa *SoTransactionObject) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListTransactionObjectByExpiration{}
	if sa == nil {
		val.Expiration = s.GetExpiration()
		val.TrxId = s.mainKey

	} else {
		val.Expiration = sa.Expiration
		val.TrxId = sa.TrxId
	}
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoTransactionObjectWrap) insertSortKeyExpiration(sa *SoTransactionObject) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListTransactionObjectByExpiration{}
	val.TrxId = sa.TrxId
	val.Expiration = sa.Expiration
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

func (s *SoTransactionObjectWrap) delAllSortKeys(br bool, val *SoTransactionObject) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delSortKeyExpiration(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoTransactionObjectWrap) insertAllSortKeys(val *SoTransactionObject) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoTransactionObject fail ")
	}
	if !s.insertSortKeyExpiration(val) {
		return errors.New("insert sort Field Expiration fail while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoTransactionObjectWrap) removeTransactionObject() error {
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

func (s *SoTransactionObjectWrap) RemoveTransactionObject(errMsgs ...interface{}) *SoTransactionObjectWrap {
	err := s.removeTransactionObject()
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoTransactionObjectWrap.RemoveTransactionObject failed: %s", err.Error()), errMsgs...))
	}
	return s
}

////////////// SECTION Members Get/Modify ///////////////

func (s *SoTransactionObjectWrap) GetExpiration() *prototype.TimePointSec {
	res := true
	msg := &SoTransactionObject{}
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
				return msg.Expiration
			}
		}
	}
	if !res {
		return nil

	}
	return msg.Expiration
}

func (s *SoTransactionObjectWrap) mdFieldExpiration(p *prototype.TimePointSec, isCheck bool, isDel bool, isInsert bool,
	so *SoTransactionObject) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkExpirationIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldExpiration(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldExpiration(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoTransactionObjectWrap) delFieldExpiration(so *SoTransactionObject) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyExpiration(so) {
		return false
	}

	return true
}

func (s *SoTransactionObjectWrap) insertFieldExpiration(so *SoTransactionObject) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyExpiration(so) {
		return false
	}

	return true
}

func (s *SoTransactionObjectWrap) checkExpirationIsMetMdCondition(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoTransactionObjectWrap) GetTrxId() *prototype.Sha256 {
	res := true
	msg := &SoTransactionObject{}
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
				return msg.TrxId
			}
		}
	}
	if !res {
		return nil

	}
	return msg.TrxId
}

////////////// SECTION List Keys ///////////////
type STransactionObjectExpirationWrap struct {
	Dba iservices.IDatabaseRW
}

func NewTransactionObjectExpirationWrap(db iservices.IDatabaseRW) *STransactionObjectExpirationWrap {
	if db == nil {
		return nil
	}
	wrap := STransactionObjectExpirationWrap{Dba: db}
	return &wrap
}

func (s *STransactionObjectExpirationWrap) GetMainVal(val []byte) *prototype.Sha256 {
	res := &SoListTransactionObjectByExpiration{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.TrxId

}

func (s *STransactionObjectExpirationWrap) GetSubVal(val []byte) *prototype.TimePointSec {
	res := &SoListTransactionObjectByExpiration{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.Expiration

}

func (m *SoListTransactionObjectByExpiration) OpeEncode() ([]byte, error) {
	pre := TransactionObjectExpirationTable
	sub := m.Expiration
	if sub == nil {
		return nil, errors.New("the pro Expiration is nil")
	}
	sub1 := m.TrxId
	if sub1 == nil {
		return nil, errors.New("the mainkey TrxId is nil")
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
func (s *STransactionObjectExpirationWrap) ForEachByOrder(start *prototype.TimePointSec, end *prototype.TimePointSec, lastMainKey *prototype.Sha256,
	lastSubVal *prototype.TimePointSec, f func(mVal *prototype.Sha256, sVal *prototype.TimePointSec, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := TransactionObjectExpirationTable
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

func (s *SoTransactionObjectWrap) update(sa *SoTransactionObject) bool {
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

func (s *SoTransactionObjectWrap) getTransactionObject() *SoTransactionObject {
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

	res := &SoTransactionObject{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoTransactionObjectWrap) updateTransactionObject(so *SoTransactionObject) error {
	if s.dba == nil {
		return errors.New("update fail:the db is nil")
	}

	if so == nil {
		return errors.New("update fail: the SoTransactionObject is nil")
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

func (s *SoTransactionObjectWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := TransactionObjectTrxIdRow
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

func (s *SoTransactionObjectWrap) delAllUniKeys(br bool, val *SoTransactionObject) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delUniKeyTrxId(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoTransactionObjectWrap) delUniKeysWithNames(names map[string]string, val *SoTransactionObject) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if len(names["TrxId"]) > 0 {
		if !s.delUniKeyTrxId(val) {
			res = false
		}
	}

	return res
}

func (s *SoTransactionObjectWrap) insertAllUniKeys(val *SoTransactionObject) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoTransactionObject fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyTrxId(val) {
		return sucFields, errors.New("insert unique Field TrxId fail while insert table ")
	}
	sucFields["TrxId"] = "TrxId"

	return sucFields, nil
}

func (s *SoTransactionObjectWrap) delUniKeyTrxId(sa *SoTransactionObject) bool {
	if s.dba == nil {
		return false
	}
	pre := TransactionObjectTrxIdUniTable
	kList := []interface{}{pre}
	if sa != nil {
		if sa.TrxId == nil {
			return false
		}

		sub := sa.TrxId
		kList = append(kList, sub)
	} else {
		sub := s.GetTrxId()
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

func (s *SoTransactionObjectWrap) insertUniKeyTrxId(sa *SoTransactionObject) bool {
	if s.dba == nil || sa == nil {
		return false
	}

	pre := TransactionObjectTrxIdUniTable
	sub := sa.TrxId
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
	val := SoUniqueTransactionObjectByTrxId{}
	val.TrxId = sa.TrxId

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniTransactionObjectTrxIdWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniTransactionObjectTrxIdWrap(db iservices.IDatabaseRW) *UniTransactionObjectTrxIdWrap {
	if db == nil {
		return nil
	}
	wrap := UniTransactionObjectTrxIdWrap{Dba: db}
	return &wrap
}

func (s *UniTransactionObjectTrxIdWrap) UniQueryTrxId(start *prototype.Sha256) *SoTransactionObjectWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := TransactionObjectTrxIdUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueTransactionObjectByTrxId{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoTransactionObjectWrap(s.Dba, res.TrxId)

			return wrap
		}
	}
	return nil
}
