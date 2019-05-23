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
	ContractDataIdUniTable uint32 = 3112701798

	ContractDataIdRow uint32 = 3142951837
)

////////////// SECTION Wrap Define ///////////////
type SoContractDataWrap struct {
	dba       iservices.IDatabaseRW
	mainKey   *prototype.ContractDataId
	mKeyFlag  int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf   []byte //the buffer after the main key is encoded with prefix
	mBuf      []byte //the value after the main key is encoded
	mdFuncMap map[string]interface{}
}

func NewSoContractDataWrap(dba iservices.IDatabaseRW, key *prototype.ContractDataId) *SoContractDataWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoContractDataWrap{dba, key, -1, nil, nil, nil}
	return result
}

func (s *SoContractDataWrap) CheckExist() bool {
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

func (s *SoContractDataWrap) Create(f func(tInfo *SoContractData)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoContractData{}
	f(val)
	if val.Id == nil {
		val.Id = s.mainKey
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

func (s *SoContractDataWrap) getMainKeyBuf() ([]byte, error) {
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

func (s *SoContractDataWrap) Md(f func(tInfo *SoContractData)) error {
	if !s.CheckExist() {
		return errors.New("the SoContractData table does not exist. Please create a table first")
	}
	oriTable := s.getContractData()
	if oriTable == nil {
		return errors.New("fail to get origin table SoContractData")
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
	err = s.updateContractData(&curTable)
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

func (s *SoContractDataWrap) checkSortAndUniFieldValidity(curTable *SoContractData, fieldSli []string) error {
	if curTable != nil && fieldSli != nil && len(fieldSli) > 0 {
		for _, fName := range fieldSli {
			if len(fName) > 0 {

			}
		}
	}
	return nil
}

//Get all the modified fields in the table
func (s *SoContractDataWrap) getModifiedFields(oriTable *SoContractData, curTable *SoContractData) ([]string, error) {
	if oriTable == nil {
		return nil, errors.New("table info is nil, can't get modified fields")
	}
	var list []string

	if !reflect.DeepEqual(oriTable.Key, curTable.Key) {
		list = append(list, "Key")
	}

	if !reflect.DeepEqual(oriTable.Value, curTable.Value) {
		list = append(list, "Value")
	}

	return list, nil
}

func (s *SoContractDataWrap) handleFieldMd(t FieldMdHandleType, so *SoContractData, fSli []string) error {
	if so == nil {
		return errors.New("fail to modify empty table")
	}

	//there is no field need to modify
	if fSli == nil || len(fSli) < 1 {
		return nil
	}

	errStr := ""
	for _, fName := range fSli {

		if fName == "Key" {
			res := true
			if t == FieldMdHandleTypeCheck {
				res = s.mdFieldKey(so.Key, true, false, false, so)
				errStr = fmt.Sprintf("fail to modify exist value of %v", fName)
			} else if t == FieldMdHandleTypeDel {
				res = s.mdFieldKey(so.Key, false, true, false, so)
				errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", fName)
			} else if t == FieldMdHandleTypeInsert {
				res = s.mdFieldKey(so.Key, false, false, true, so)
				errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", fName)
			}
			if !res {
				return errors.New(errStr)
			}
		}

		if fName == "Value" {
			res := true
			if t == FieldMdHandleTypeCheck {
				res = s.mdFieldValue(so.Value, true, false, false, so)
				errStr = fmt.Sprintf("fail to modify exist value of %v", fName)
			} else if t == FieldMdHandleTypeDel {
				res = s.mdFieldValue(so.Value, false, true, false, so)
				errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", fName)
			} else if t == FieldMdHandleTypeInsert {
				res = s.mdFieldValue(so.Value, false, false, true, so)
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

func (s *SoContractDataWrap) delAllSortKeys(br bool, val *SoContractData) bool {
	if s.dba == nil {
		return false
	}
	res := true

	return res
}

func (s *SoContractDataWrap) insertAllSortKeys(val *SoContractData) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoContractData fail ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoContractDataWrap) RemoveContractData() bool {
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

func (s *SoContractDataWrap) GetId() *prototype.ContractDataId {
	res := true
	msg := &SoContractData{}
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
		return nil

	}
	return msg.Id
}

func (s *SoContractDataWrap) GetKey() []byte {
	res := true
	msg := &SoContractData{}
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
				return msg.Key
			}
		}
	}
	if !res {
		var tmpValue []byte
		return tmpValue
	}
	return msg.Key
}

func (s *SoContractDataWrap) mdFieldKey(p []byte, isCheck bool, isDel bool, isInsert bool,
	so *SoContractData) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkKeyIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldKey(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldKey(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoContractDataWrap) delFieldKey(so *SoContractData) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoContractDataWrap) insertFieldKey(so *SoContractData) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoContractDataWrap) checkKeyIsMetMdCondition(p []byte) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoContractDataWrap) GetValue() []byte {
	res := true
	msg := &SoContractData{}
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
				return msg.Value
			}
		}
	}
	if !res {
		var tmpValue []byte
		return tmpValue
	}
	return msg.Value
}

func (s *SoContractDataWrap) mdFieldValue(p []byte, isCheck bool, isDel bool, isInsert bool,
	so *SoContractData) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkValueIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldValue(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldValue(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoContractDataWrap) delFieldValue(so *SoContractData) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoContractDataWrap) insertFieldValue(so *SoContractData) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoContractDataWrap) checkValueIsMetMdCondition(p []byte) bool {
	if s.dba == nil {
		return false
	}

	return true
}

/////////////// SECTION Private function ////////////////

func (s *SoContractDataWrap) update(sa *SoContractData) bool {
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

func (s *SoContractDataWrap) getContractData() *SoContractData {
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

	res := &SoContractData{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoContractDataWrap) updateContractData(so *SoContractData) error {
	if s.dba == nil {
		return errors.New("update fail:the db is nil")
	}

	if so == nil {
		return errors.New("update fail: the SoContractData is nil")
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

func (s *SoContractDataWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := ContractDataIdRow
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

func (s *SoContractDataWrap) delAllUniKeys(br bool, val *SoContractData) bool {
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

func (s *SoContractDataWrap) delUniKeysWithNames(names map[string]string, val *SoContractData) bool {
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

func (s *SoContractDataWrap) insertAllUniKeys(val *SoContractData) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoContractData fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyId(val) {
		return sucFields, errors.New("insert unique Field Id fail while insert table ")
	}
	sucFields["Id"] = "Id"

	return sucFields, nil
}

func (s *SoContractDataWrap) delUniKeyId(sa *SoContractData) bool {
	if s.dba == nil {
		return false
	}
	pre := ContractDataIdUniTable
	kList := []interface{}{pre}
	if sa != nil {
		if sa.Id == nil {
			return false
		}

		sub := sa.Id
		kList = append(kList, sub)
	} else {
		sub := s.GetId()
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

func (s *SoContractDataWrap) insertUniKeyId(sa *SoContractData) bool {
	if s.dba == nil || sa == nil {
		return false
	}

	pre := ContractDataIdUniTable
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
	val := SoUniqueContractDataById{}
	val.Id = sa.Id

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniContractDataIdWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniContractDataIdWrap(db iservices.IDatabaseRW) *UniContractDataIdWrap {
	if db == nil {
		return nil
	}
	wrap := UniContractDataIdWrap{Dba: db}
	return &wrap
}

func (s *UniContractDataIdWrap) UniQueryId(start *prototype.ContractDataId) *SoContractDataWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := ContractDataIdUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueContractDataById{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoContractDataWrap(s.Dba, res.Id)

			return wrap
		}
	}
	return nil
}
