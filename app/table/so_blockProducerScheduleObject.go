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
	BlockProducerScheduleObjectIdUniTable uint32 = 1798653281

	BlockProducerScheduleObjectIdRow uint32 = 3218627324
)

////////////// SECTION Wrap Define ///////////////
type SoBlockProducerScheduleObjectWrap struct {
	dba       iservices.IDatabaseRW
	mainKey   *int32
	mKeyFlag  int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf   []byte //the buffer after the main key is encoded with prefix
	mBuf      []byte //the value after the main key is encoded
	mdFuncMap map[string]interface{}
}

func NewSoBlockProducerScheduleObjectWrap(dba iservices.IDatabaseRW, key *int32) *SoBlockProducerScheduleObjectWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoBlockProducerScheduleObjectWrap{dba, key, -1, nil, nil, nil}
	return result
}

func (s *SoBlockProducerScheduleObjectWrap) CheckExist() bool {
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

func (s *SoBlockProducerScheduleObjectWrap) Create(f func(tInfo *SoBlockProducerScheduleObject)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoBlockProducerScheduleObject{}
	f(val)
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

func (s *SoBlockProducerScheduleObjectWrap) getMainKeyBuf() ([]byte, error) {
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

func (s *SoBlockProducerScheduleObjectWrap) Modify(f func(tInfo *SoBlockProducerScheduleObject)) error {
	if !s.CheckExist() {
		return errors.New("the SoBlockProducerScheduleObject table does not exist. Please create a table first")
	}
	oriTable := s.getBlockProducerScheduleObject()
	if oriTable == nil {
		return errors.New("fail to get origin table SoBlockProducerScheduleObject")
	}
	curTable := *oriTable
	f(&curTable)

	//the main key is not support modify
	if !reflect.DeepEqual(curTable.Id, oriTable.Id) {
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
	err = s.updateBlockProducerScheduleObject(&curTable)
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

func (s *SoBlockProducerScheduleObjectWrap) SetCurrentShuffledBlockProducer(p []string) bool {
	err := s.Modify(func(r *SoBlockProducerScheduleObject) {
		r.CurrentShuffledBlockProducer = p
	})
	return err == nil
}

func (s *SoBlockProducerScheduleObjectWrap) SetPubKey(p []*prototype.PublicKeyType) bool {
	err := s.Modify(func(r *SoBlockProducerScheduleObject) {
		r.PubKey = p
	})
	return err == nil
}

func (s *SoBlockProducerScheduleObjectWrap) checkSortAndUniFieldValidity(curTable *SoBlockProducerScheduleObject, fieldSli []string) error {
	if curTable != nil && fieldSli != nil && len(fieldSli) > 0 {
		for _, fName := range fieldSli {
			if len(fName) > 0 {

			}
		}
	}
	return nil
}

//Get all the modified fields in the table
func (s *SoBlockProducerScheduleObjectWrap) getModifiedFields(oriTable *SoBlockProducerScheduleObject, curTable *SoBlockProducerScheduleObject) ([]string, error) {
	if oriTable == nil {
		return nil, errors.New("table info is nil, can't get modified fields")
	}
	var list []string

	if !reflect.DeepEqual(oriTable.CurrentShuffledBlockProducer, curTable.CurrentShuffledBlockProducer) {
		list = append(list, "CurrentShuffledBlockProducer")
	}

	if !reflect.DeepEqual(oriTable.PubKey, curTable.PubKey) {
		list = append(list, "PubKey")
	}

	return list, nil
}

func (s *SoBlockProducerScheduleObjectWrap) handleFieldMd(t FieldMdHandleType, so *SoBlockProducerScheduleObject, fSli []string) error {
	if so == nil {
		return errors.New("fail to modify empty table")
	}

	//there is no field need to modify
	if fSli == nil || len(fSli) < 1 {
		return nil
	}

	errStr := ""
	for _, fName := range fSli {

		if fName == "CurrentShuffledBlockProducer" {
			res := true
			if t == FieldMdHandleTypeCheck {
				res = s.mdFieldCurrentShuffledBlockProducer(so.CurrentShuffledBlockProducer, true, false, false, so)
				errStr = fmt.Sprintf("fail to modify exist value of %v", fName)
			} else if t == FieldMdHandleTypeDel {
				res = s.mdFieldCurrentShuffledBlockProducer(so.CurrentShuffledBlockProducer, false, true, false, so)
				errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", fName)
			} else if t == FieldMdHandleTypeInsert {
				res = s.mdFieldCurrentShuffledBlockProducer(so.CurrentShuffledBlockProducer, false, false, true, so)
				errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", fName)
			}
			if !res {
				return errors.New(errStr)
			}
		}

		if fName == "PubKey" {
			res := true
			if t == FieldMdHandleTypeCheck {
				res = s.mdFieldPubKey(so.PubKey, true, false, false, so)
				errStr = fmt.Sprintf("fail to modify exist value of %v", fName)
			} else if t == FieldMdHandleTypeDel {
				res = s.mdFieldPubKey(so.PubKey, false, true, false, so)
				errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", fName)
			} else if t == FieldMdHandleTypeInsert {
				res = s.mdFieldPubKey(so.PubKey, false, false, true, so)
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

func (s *SoBlockProducerScheduleObjectWrap) delAllSortKeys(br bool, val *SoBlockProducerScheduleObject) bool {
	if s.dba == nil {
		return false
	}
	res := true

	return res
}

func (s *SoBlockProducerScheduleObjectWrap) insertAllSortKeys(val *SoBlockProducerScheduleObject) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoBlockProducerScheduleObject fail ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoBlockProducerScheduleObjectWrap) RemoveBlockProducerScheduleObject() bool {
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

func (s *SoBlockProducerScheduleObjectWrap) GetCurrentShuffledBlockProducer() []string {
	res := true
	msg := &SoBlockProducerScheduleObject{}
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
				return msg.CurrentShuffledBlockProducer
			}
		}
	}
	if !res {
		var tmpValue []string
		return tmpValue
	}
	return msg.CurrentShuffledBlockProducer
}

func (s *SoBlockProducerScheduleObjectWrap) mdFieldCurrentShuffledBlockProducer(p []string, isCheck bool, isDel bool, isInsert bool,
	so *SoBlockProducerScheduleObject) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkCurrentShuffledBlockProducerIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldCurrentShuffledBlockProducer(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldCurrentShuffledBlockProducer(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoBlockProducerScheduleObjectWrap) delFieldCurrentShuffledBlockProducer(so *SoBlockProducerScheduleObject) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerScheduleObjectWrap) insertFieldCurrentShuffledBlockProducer(so *SoBlockProducerScheduleObject) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerScheduleObjectWrap) checkCurrentShuffledBlockProducerIsMetMdCondition(p []string) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerScheduleObjectWrap) GetId() int32 {
	res := true
	msg := &SoBlockProducerScheduleObject{}
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
				return msg.Id
			}
		}
	}
	if !res {
		var tmpValue int32
		return tmpValue
	}
	return msg.Id
}

func (s *SoBlockProducerScheduleObjectWrap) GetPubKey() []*prototype.PublicKeyType {
	res := true
	msg := &SoBlockProducerScheduleObject{}
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
				return msg.PubKey
			}
		}
	}
	if !res {
		var tmpValue []*prototype.PublicKeyType
		return tmpValue
	}
	return msg.PubKey
}

func (s *SoBlockProducerScheduleObjectWrap) mdFieldPubKey(p []*prototype.PublicKeyType, isCheck bool, isDel bool, isInsert bool,
	so *SoBlockProducerScheduleObject) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkPubKeyIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldPubKey(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldPubKey(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoBlockProducerScheduleObjectWrap) delFieldPubKey(so *SoBlockProducerScheduleObject) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerScheduleObjectWrap) insertFieldPubKey(so *SoBlockProducerScheduleObject) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerScheduleObjectWrap) checkPubKeyIsMetMdCondition(p []*prototype.PublicKeyType) bool {
	if s.dba == nil {
		return false
	}

	return true
}

/////////////// SECTION Private function ////////////////

func (s *SoBlockProducerScheduleObjectWrap) update(sa *SoBlockProducerScheduleObject) bool {
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

func (s *SoBlockProducerScheduleObjectWrap) getBlockProducerScheduleObject() *SoBlockProducerScheduleObject {
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

	res := &SoBlockProducerScheduleObject{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoBlockProducerScheduleObjectWrap) updateBlockProducerScheduleObject(so *SoBlockProducerScheduleObject) error {
	if s.dba == nil {
		return errors.New("update fail:the db is nil")
	}

	if so == nil {
		return errors.New("update fail: the SoBlockProducerScheduleObject is nil")
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

func (s *SoBlockProducerScheduleObjectWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := BlockProducerScheduleObjectIdRow
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

func (s *SoBlockProducerScheduleObjectWrap) delAllUniKeys(br bool, val *SoBlockProducerScheduleObject) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delUniKeyId(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoBlockProducerScheduleObjectWrap) delUniKeysWithNames(names map[string]string, val *SoBlockProducerScheduleObject) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if len(names["Id"]) > 0 {
		if !s.delUniKeyId(val) {
			res = false
		}
	}

	return res
}

func (s *SoBlockProducerScheduleObjectWrap) insertAllUniKeys(val *SoBlockProducerScheduleObject) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoBlockProducerScheduleObject fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyId(val) {
		return sucFields, errors.New("insert unique Field Id fail while insert table ")
	}
	sucFields["Id"] = "Id"

	return sucFields, nil
}

func (s *SoBlockProducerScheduleObjectWrap) delUniKeyId(sa *SoBlockProducerScheduleObject) bool {
	if s.dba == nil {
		return false
	}
	pre := BlockProducerScheduleObjectIdUniTable
	kList := []interface{}{pre}
	if sa != nil {

		sub := sa.Id
		kList = append(kList, sub)
	} else {
		sub := s.GetId()

		kList = append(kList, sub)

	}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoBlockProducerScheduleObjectWrap) insertUniKeyId(sa *SoBlockProducerScheduleObject) bool {
	if s.dba == nil || sa == nil {
		return false
	}

	pre := BlockProducerScheduleObjectIdUniTable
	sub := sa.Id
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
	val := SoUniqueBlockProducerScheduleObjectById{}
	val.Id = sa.Id

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniBlockProducerScheduleObjectIdWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniBlockProducerScheduleObjectIdWrap(db iservices.IDatabaseRW) *UniBlockProducerScheduleObjectIdWrap {
	if db == nil {
		return nil
	}
	wrap := UniBlockProducerScheduleObjectIdWrap{Dba: db}
	return &wrap
}

func (s *UniBlockProducerScheduleObjectIdWrap) UniQueryId(start *int32) *SoBlockProducerScheduleObjectWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := BlockProducerScheduleObjectIdUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueBlockProducerScheduleObjectById{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoBlockProducerScheduleObjectWrap(s.Dba, &res.Id)
			return wrap
		}
	}
	return nil
}
