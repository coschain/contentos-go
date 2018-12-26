package table

import (
	"errors"
	fmt "fmt"
	"reflect"
	"strings"

	"github.com/coschain/contentos-go/common/encoding/kope"
	"github.com/coschain/contentos-go/iservices"
	prototype "github.com/coschain/contentos-go/prototype"
	proto "github.com/golang/protobuf/proto"
)

////////////// SECTION Prefix Mark ///////////////
var (
	ContractDataTable      = []byte("ContractDataTable")
	ContractDataIdUniTable = []byte("ContractDataIdUniTable")
)

////////////// SECTION Wrap Define ///////////////
type SoContractDataWrap struct {
	dba      iservices.IDatabaseService
	mainKey  *prototype.ContractDataId
	mKeyFlag int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf  []byte //the buffer after the main key is encoded with prefix
	mBuf     []byte //the value after the main key is encoded
}

func NewSoContractDataWrap(dba iservices.IDatabaseService, key *prototype.ContractDataId) *SoContractDataWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoContractDataWrap{dba, key, -1, nil, nil}
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
	err = s.saveAllMemKeys(val, true)
	if err != nil {
		return err
	}

	// update sort list keys
	if err = s.insertAllSortKeys(val); err != nil {
		s.delAllSortKeys(false, val)
		s.dba.Delete(keyBuf)
		s.delAllMemKeys(false, val)
		return err
	}

	//update unique list
	if sucNames, err := s.insertAllUniKeys(val); err != nil {
		s.delAllSortKeys(false, val)
		s.delUniKeysWithNames(sucNames, val)
		s.dba.Delete(keyBuf)
		s.delAllMemKeys(false, val)
		return err
	}

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

func (s *SoContractDataWrap) encodeMemKey(fName string) ([]byte, error) {
	if len(fName) < 1 || s.mainKey == nil {
		return nil, errors.New("field name or main key is empty")
	}
	pre := "ContractData" + fName + "cell"
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
	return kope.PackList(list), nil
}

func (so *SoContractDataWrap) saveAllMemKeys(tInfo *SoContractData, br bool) error {
	if so.dba == nil {
		return errors.New("save member Field fail , the db is nil")
	}

	if tInfo == nil {
		return errors.New("save member Field fail , the data is nil ")
	}
	var err error = nil
	errDes := ""
	if err = so.saveMemKeyId(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Id", err)
		}
	}
	if err = so.saveMemKeyKey(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Key", err)
		}
	}
	if err = so.saveMemKeyValue(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Value", err)
		}
	}

	if len(errDes) > 0 {
		return errors.New(errDes)
	}
	return err
}

func (so *SoContractDataWrap) delAllMemKeys(br bool, tInfo *SoContractData) error {
	if so.dba == nil {
		return errors.New("the db is nil")
	}
	t := reflect.TypeOf(*tInfo)
	errDesc := ""
	for k := 0; k < t.NumField(); k++ {
		name := t.Field(k).Name
		if len(name) > 0 && !strings.HasPrefix(name, "XXX_") {
			err := so.delMemKey(name)
			if err != nil {
				if br {
					return err
				}
				errDesc += fmt.Sprintf("delete the Field %s fail,error is %s;\n", name, err)
			}
		}
	}
	if len(errDesc) > 0 {
		return errors.New(errDesc)
	}
	return nil
}

func (so *SoContractDataWrap) delMemKey(fName string) error {
	if so.dba == nil {
		return errors.New("the db is nil")
	}
	if len(fName) <= 0 {
		return errors.New("the field name is empty ")
	}
	key, err := so.encodeMemKey(fName)
	if err != nil {
		return err
	}
	err = so.dba.Delete(key)
	return err
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
	val := &SoContractData{}
	//delete sort list key
	if res := s.delAllSortKeys(true, nil); !res {
		return false
	}

	//delete unique list
	if res := s.delAllUniKeys(true, nil); !res {
		return false
	}

	err := s.delAllMemKeys(true, val)
	if err == nil {
		s.mKeyBuf = nil
		s.mKeyFlag = -1
		return true
	} else {
		return false
	}
}

////////////// SECTION Members Get/Modify ///////////////
func (s *SoContractDataWrap) saveMemKeyId(tInfo *SoContractData) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemContractDataById{}
	val.Id = tInfo.Id
	key, err := s.encodeMemKey("Id")
	if err != nil {
		return err
	}
	buf, err := proto.Marshal(&val)
	if err != nil {
		return err
	}
	err = s.dba.Put(key, buf)
	return err
}

func (s *SoContractDataWrap) GetId() *prototype.ContractDataId {
	res := true
	msg := &SoMemContractDataById{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Id")
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

func (s *SoContractDataWrap) saveMemKeyKey(tInfo *SoContractData) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemContractDataByKey{}
	val.Key = tInfo.Key
	key, err := s.encodeMemKey("Key")
	if err != nil {
		return err
	}
	buf, err := proto.Marshal(&val)
	if err != nil {
		return err
	}
	err = s.dba.Put(key, buf)
	return err
}

func (s *SoContractDataWrap) GetKey() []byte {
	res := true
	msg := &SoMemContractDataByKey{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Key")
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

func (s *SoContractDataWrap) MdKey(p []byte) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Key")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemContractDataByKey{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoContractData{}
	sa.Id = s.mainKey

	sa.Key = ori.Key

	ori.Key = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Key = p

	return true
}

func (s *SoContractDataWrap) saveMemKeyValue(tInfo *SoContractData) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemContractDataByValue{}
	val.Value = tInfo.Value
	key, err := s.encodeMemKey("Value")
	if err != nil {
		return err
	}
	buf, err := proto.Marshal(&val)
	if err != nil {
		return err
	}
	err = s.dba.Put(key, buf)
	return err
}

func (s *SoContractDataWrap) GetValue() []byte {
	res := true
	msg := &SoMemContractDataByValue{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Value")
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

func (s *SoContractDataWrap) MdValue(p []byte) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Value")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemContractDataByValue{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoContractData{}
	sa.Id = s.mainKey

	sa.Value = ori.Value

	ori.Value = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Value = p

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

func (s *SoContractDataWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := "ContractData" + "Id" + "cell"
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
		key, err := s.encodeMemKey("Id")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemContractDataById{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		sub := ori.Id
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
	Dba iservices.IDatabaseService
}

func NewUniContractDataIdWrap(db iservices.IDatabaseService) *UniContractDataIdWrap {
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
