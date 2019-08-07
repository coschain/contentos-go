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
	BlockSummaryObjectIdUniTable uint32 = 1471985338

	BlockSummaryObjectIdRow uint32 = 172113035
)

////////////// SECTION Wrap Define ///////////////
type SoBlockSummaryObjectWrap struct {
	dba       iservices.IDatabaseRW
	mainKey   *uint32
	mKeyFlag  int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf   []byte //the buffer after the main key is encoded with prefix
	mBuf      []byte //the value after the main key is encoded
	mdFuncMap map[string]interface{}
}

func NewSoBlockSummaryObjectWrap(dba iservices.IDatabaseRW, key *uint32) *SoBlockSummaryObjectWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoBlockSummaryObjectWrap{dba, key, -1, nil, nil, nil}
	return result
}

func (s *SoBlockSummaryObjectWrap) CheckExist() bool {
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

func (s *SoBlockSummaryObjectWrap) MustExist() *SoBlockSummaryObjectWrap {
	if !s.CheckExist() {
		panic(fmt.Errorf("SoBlockSummaryObjectWrap.MustExist: %v not found", s.mainKey))
	}
	return s
}

func (s *SoBlockSummaryObjectWrap) MustNotExist() *SoBlockSummaryObjectWrap {
	if s.CheckExist() {
		panic(fmt.Errorf("SoBlockSummaryObjectWrap.MustNotExist: %v already exists", s.mainKey))
	}
	return s
}

func (s *SoBlockSummaryObjectWrap) create(f func(tInfo *SoBlockSummaryObject)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoBlockSummaryObject{}
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

func (s *SoBlockSummaryObjectWrap) Create(f func(tInfo *SoBlockSummaryObject), errArgs ...interface{}) *SoBlockSummaryObjectWrap {
	err := s.create(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Errorf("SoBlockSummaryObjectWrap.Create failed: %s", err.Error()), errArgs...))
	}
	return s
}

func (s *SoBlockSummaryObjectWrap) getMainKeyBuf() ([]byte, error) {
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

func (s *SoBlockSummaryObjectWrap) modify(f func(tInfo *SoBlockSummaryObject)) error {
	if !s.CheckExist() {
		return errors.New("the SoBlockSummaryObject table does not exist. Please create a table first")
	}
	oriTable := s.getBlockSummaryObject()
	if oriTable == nil {
		return errors.New("fail to get origin table SoBlockSummaryObject")
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
	err = s.updateBlockSummaryObject(&curTable)
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

func (s *SoBlockSummaryObjectWrap) Modify(f func(tInfo *SoBlockSummaryObject), errArgs ...interface{}) *SoBlockSummaryObjectWrap {
	err := s.modify(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoBlockSummaryObjectWrap.Modify failed: %s", err.Error()), errArgs...))
	}
	return s
}

func (s *SoBlockSummaryObjectWrap) SetBlockId(p *prototype.Sha256, errArgs ...interface{}) *SoBlockSummaryObjectWrap {
	err := s.modify(func(r *SoBlockSummaryObject) {
		r.BlockId = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoBlockSummaryObjectWrap.SetBlockId( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoBlockSummaryObjectWrap) checkSortAndUniFieldValidity(curTable *SoBlockSummaryObject, fieldSli []string) error {
	if curTable != nil && fieldSli != nil && len(fieldSli) > 0 {
		for _, fName := range fieldSli {
			if len(fName) > 0 {

			}
		}
	}
	return nil
}

//Get all the modified fields in the table
func (s *SoBlockSummaryObjectWrap) getModifiedFields(oriTable *SoBlockSummaryObject, curTable *SoBlockSummaryObject) ([]string, error) {
	if oriTable == nil {
		return nil, errors.New("table info is nil, can't get modified fields")
	}
	var list []string

	if !reflect.DeepEqual(oriTable.BlockId, curTable.BlockId) {
		list = append(list, "BlockId")
	}

	return list, nil
}

func (s *SoBlockSummaryObjectWrap) handleFieldMd(t FieldMdHandleType, so *SoBlockSummaryObject, fSli []string) error {
	if so == nil {
		return errors.New("fail to modify empty table")
	}

	//there is no field need to modify
	if fSli == nil || len(fSli) < 1 {
		return nil
	}

	errStr := ""
	for _, fName := range fSli {

		if fName == "BlockId" {
			res := true
			if t == FieldMdHandleTypeCheck {
				res = s.mdFieldBlockId(so.BlockId, true, false, false, so)
				errStr = fmt.Sprintf("fail to modify exist value of %v", fName)
			} else if t == FieldMdHandleTypeDel {
				res = s.mdFieldBlockId(so.BlockId, false, true, false, so)
				errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", fName)
			} else if t == FieldMdHandleTypeInsert {
				res = s.mdFieldBlockId(so.BlockId, false, false, true, so)
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

func (s *SoBlockSummaryObjectWrap) delAllSortKeys(br bool, val *SoBlockSummaryObject) bool {
	if s.dba == nil {
		return false
	}
	res := true

	return res
}

func (s *SoBlockSummaryObjectWrap) insertAllSortKeys(val *SoBlockSummaryObject) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoBlockSummaryObject fail ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoBlockSummaryObjectWrap) removeBlockSummaryObject() error {
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

func (s *SoBlockSummaryObjectWrap) RemoveBlockSummaryObject(errMsgs ...interface{}) *SoBlockSummaryObjectWrap {
	err := s.removeBlockSummaryObject()
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoBlockSummaryObjectWrap.RemoveBlockSummaryObject failed: %s", err.Error()), errMsgs...))
	}
	return s
}

////////////// SECTION Members Get/Modify ///////////////

func (s *SoBlockSummaryObjectWrap) GetBlockId() *prototype.Sha256 {
	res := true
	msg := &SoBlockSummaryObject{}
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
				return msg.BlockId
			}
		}
	}
	if !res {
		return nil

	}
	return msg.BlockId
}

func (s *SoBlockSummaryObjectWrap) mdFieldBlockId(p *prototype.Sha256, isCheck bool, isDel bool, isInsert bool,
	so *SoBlockSummaryObject) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkBlockIdIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldBlockId(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldBlockId(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoBlockSummaryObjectWrap) delFieldBlockId(so *SoBlockSummaryObject) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockSummaryObjectWrap) insertFieldBlockId(so *SoBlockSummaryObject) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockSummaryObjectWrap) checkBlockIdIsMetMdCondition(p *prototype.Sha256) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockSummaryObjectWrap) GetId() uint32 {
	res := true
	msg := &SoBlockSummaryObject{}
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
		var tmpValue uint32
		return tmpValue
	}
	return msg.Id
}

/////////////// SECTION Private function ////////////////

func (s *SoBlockSummaryObjectWrap) update(sa *SoBlockSummaryObject) bool {
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

func (s *SoBlockSummaryObjectWrap) getBlockSummaryObject() *SoBlockSummaryObject {
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

	res := &SoBlockSummaryObject{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoBlockSummaryObjectWrap) updateBlockSummaryObject(so *SoBlockSummaryObject) error {
	if s.dba == nil {
		return errors.New("update fail:the db is nil")
	}

	if so == nil {
		return errors.New("update fail: the SoBlockSummaryObject is nil")
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

func (s *SoBlockSummaryObjectWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := BlockSummaryObjectIdRow
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

func (s *SoBlockSummaryObjectWrap) delAllUniKeys(br bool, val *SoBlockSummaryObject) bool {
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

func (s *SoBlockSummaryObjectWrap) delUniKeysWithNames(names map[string]string, val *SoBlockSummaryObject) bool {
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

func (s *SoBlockSummaryObjectWrap) insertAllUniKeys(val *SoBlockSummaryObject) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoBlockSummaryObject fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyId(val) {
		return sucFields, errors.New("insert unique Field Id fail while insert table ")
	}
	sucFields["Id"] = "Id"

	return sucFields, nil
}

func (s *SoBlockSummaryObjectWrap) delUniKeyId(sa *SoBlockSummaryObject) bool {
	if s.dba == nil {
		return false
	}
	pre := BlockSummaryObjectIdUniTable
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

func (s *SoBlockSummaryObjectWrap) insertUniKeyId(sa *SoBlockSummaryObject) bool {
	if s.dba == nil || sa == nil {
		return false
	}

	pre := BlockSummaryObjectIdUniTable
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
	val := SoUniqueBlockSummaryObjectById{}
	val.Id = sa.Id

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniBlockSummaryObjectIdWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniBlockSummaryObjectIdWrap(db iservices.IDatabaseRW) *UniBlockSummaryObjectIdWrap {
	if db == nil {
		return nil
	}
	wrap := UniBlockSummaryObjectIdWrap{Dba: db}
	return &wrap
}

func (s *UniBlockSummaryObjectIdWrap) UniQueryId(start *uint32) *SoBlockSummaryObjectWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := BlockSummaryObjectIdUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueBlockSummaryObjectById{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoBlockSummaryObjectWrap(s.Dba, &res.Id)
			return wrap
		}
	}
	return nil
}
