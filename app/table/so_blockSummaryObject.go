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
	BlockSummaryObjectTable      = []byte("BlockSummaryObjectTable")
	BlockSummaryObjectIdUniTable = []byte("BlockSummaryObjectIdUniTable")
)

////////////// SECTION Wrap Define ///////////////
type SoBlockSummaryObjectWrap struct {
	dba      iservices.IDatabaseService
	mainKey  *uint32
	mKeyFlag int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf  []byte //the buffer after the main key is encoded
	mBuf     []byte //the value after the main key is encoded
}

func NewSoBlockSummaryObjectWrap(dba iservices.IDatabaseService, key *uint32) *SoBlockSummaryObjectWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoBlockSummaryObjectWrap{
		dba,
		key,
		-1,
		nil,
		nil,
	}
	return result
}

func (s *SoBlockSummaryObjectWrap) CheckExist() bool {
	if s.dba == nil {
		return false
	}
	if s.mKeyFlag != -1 {
		//f you have already obtained the existence status of the primary key, use it directly
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

func (s *SoBlockSummaryObjectWrap) Create(f func(tInfo *SoBlockSummaryObject)) error {
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

func (s *SoBlockSummaryObjectWrap) encodeMemKey(fName string) ([]byte, error) {
	if len(fName) < 1 || s.mainKey == nil {
		return nil, errors.New("field name or main key is empty")
	}
	pre := "BlockSummaryObject" + fName + "cell"
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

func (so *SoBlockSummaryObjectWrap) saveAllMemKeys(tInfo *SoBlockSummaryObject, br bool) error {
	if so.dba == nil {
		return errors.New("save member Field fail , the db is nil")
	}

	if tInfo == nil {
		return errors.New("save member Field fail , the data is nil ")
	}
	var err error = nil
	errDes := ""
	if err = so.saveMemKeyBlockId(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "BlockId", err)
		}
	}
	if err = so.saveMemKeyId(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Id", err)
		}
	}

	if len(errDes) > 0 {
		return errors.New(errDes)
	}
	return err
}

func (so *SoBlockSummaryObjectWrap) delAllMemKeys(br bool, tInfo *SoBlockSummaryObject) error {
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

func (so *SoBlockSummaryObjectWrap) delMemKey(fName string) error {
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

func (s *SoBlockSummaryObjectWrap) RemoveBlockSummaryObject() bool {
	if s.dba == nil {
		return false
	}
	val := &SoBlockSummaryObject{}
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
func (s *SoBlockSummaryObjectWrap) saveMemKeyBlockId(tInfo *SoBlockSummaryObject) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemBlockSummaryObjectByBlockId{}
	val.BlockId = tInfo.BlockId
	key, err := s.encodeMemKey("BlockId")
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

func (s *SoBlockSummaryObjectWrap) GetBlockId() *prototype.Sha256 {
	res := true
	msg := &SoMemBlockSummaryObjectByBlockId{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("BlockId")
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

func (s *SoBlockSummaryObjectWrap) MdBlockId(p *prototype.Sha256) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("BlockId")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemBlockSummaryObjectByBlockId{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoBlockSummaryObject{}
	sa.Id = *s.mainKey
	sa.BlockId = ori.BlockId

	ori.BlockId = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.BlockId = p

	return true
}

func (s *SoBlockSummaryObjectWrap) saveMemKeyId(tInfo *SoBlockSummaryObject) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemBlockSummaryObjectById{}
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

func (s *SoBlockSummaryObjectWrap) GetId() uint32 {
	res := true
	msg := &SoMemBlockSummaryObjectById{}
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

func (s *SoBlockSummaryObjectWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := "BlockSummaryObject" + "Id" + "cell"
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
		key, err := s.encodeMemKey("Id")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemBlockSummaryObjectById{}
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
	Dba iservices.IDatabaseService
}

func NewUniBlockSummaryObjectIdWrap(db iservices.IDatabaseService) *UniBlockSummaryObjectIdWrap {
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
