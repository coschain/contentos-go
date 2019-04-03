package table

import (
	"errors"
	fmt "fmt"
	"reflect"
	"strings"

	"github.com/coschain/contentos-go/common/encoding/kope"
	"github.com/coschain/contentos-go/iservices"
	proto "github.com/golang/protobuf/proto"
)

////////////// SECTION Prefix Mark ///////////////
var (
	BlocktrxsBlockUniTable uint32 = 3461050414
	BlocktrxsBlockCell     uint32 = 2415577191
	BlocktrxsTrxsCell      uint32 = 3955135682
)

////////////// SECTION Wrap Define ///////////////
type SoBlocktrxsWrap struct {
	dba      iservices.IDatabaseRW
	mainKey  *uint64
	mKeyFlag int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf  []byte //the buffer after the main key is encoded with prefix
	mBuf     []byte //the value after the main key is encoded
}

func NewSoBlocktrxsWrap(dba iservices.IDatabaseRW, key *uint64) *SoBlocktrxsWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoBlocktrxsWrap{dba, key, -1, nil, nil}
	return result
}

func (s *SoBlocktrxsWrap) CheckExist() bool {
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

func (s *SoBlocktrxsWrap) Create(f func(tInfo *SoBlocktrxs)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoBlocktrxs{}
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
		s.delAllMemKeys(false, val)
		return err
	}

	// update srt list keys
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

func (s *SoBlocktrxsWrap) getMainKeyBuf() ([]byte, error) {
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

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoBlocktrxsWrap) delAllSortKeys(br bool, val *SoBlocktrxs) bool {
	if s.dba == nil {
		return false
	}
	res := true

	return res
}

func (s *SoBlocktrxsWrap) insertAllSortKeys(val *SoBlocktrxs) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoBlocktrxs fail ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoBlocktrxsWrap) RemoveBlocktrxs() bool {
	if s.dba == nil {
		return false
	}
	val := &SoBlocktrxs{}
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
func (s *SoBlocktrxsWrap) getMemKeyPrefix(fName string) uint32 {
	if fName == "Block" {
		return BlocktrxsBlockCell
	}
	if fName == "Trxs" {
		return BlocktrxsTrxsCell
	}

	return 0
}

func (s *SoBlocktrxsWrap) encodeMemKey(fName string) ([]byte, error) {
	if len(fName) < 1 || s.mainKey == nil {
		return nil, errors.New("field name or main key is empty")
	}
	pre := s.getMemKeyPrefix(fName)
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

func (s *SoBlocktrxsWrap) saveAllMemKeys(tInfo *SoBlocktrxs, br bool) error {
	if s.dba == nil {
		return errors.New("save member Field fail , the db is nil")
	}

	if tInfo == nil {
		return errors.New("save member Field fail , the data is nil ")
	}
	var err error = nil
	errDes := ""
	if err = s.saveMemKeyBlock(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Block", err)
		}
	}
	if err = s.saveMemKeyTrxs(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Trxs", err)
		}
	}

	if len(errDes) > 0 {
		return errors.New(errDes)
	}
	return err
}

func (s *SoBlocktrxsWrap) delAllMemKeys(br bool, tInfo *SoBlocktrxs) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	t := reflect.TypeOf(*tInfo)
	errDesc := ""
	for k := 0; k < t.NumField(); k++ {
		name := t.Field(k).Name
		if len(name) > 0 && !strings.HasPrefix(name, "XXX_") {
			err := s.delMemKey(name)
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

func (s *SoBlocktrxsWrap) delMemKey(fName string) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if len(fName) <= 0 {
		return errors.New("the field name is empty ")
	}
	key, err := s.encodeMemKey(fName)
	if err != nil {
		return err
	}
	err = s.dba.Delete(key)
	return err
}

func (s *SoBlocktrxsWrap) saveMemKeyBlock(tInfo *SoBlocktrxs) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemBlocktrxsByBlock{}
	val.Block = tInfo.Block
	key, err := s.encodeMemKey("Block")
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

func (s *SoBlocktrxsWrap) GetBlock() uint64 {
	res := true
	msg := &SoMemBlocktrxsByBlock{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Block")
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
				return msg.Block
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.Block
}

func (s *SoBlocktrxsWrap) saveMemKeyTrxs(tInfo *SoBlocktrxs) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemBlocktrxsByTrxs{}
	val.Trxs = tInfo.Trxs
	key, err := s.encodeMemKey("Trxs")
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

func (s *SoBlocktrxsWrap) GetTrxs() []byte {
	res := true
	msg := &SoMemBlocktrxsByTrxs{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Trxs")
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
				return msg.Trxs
			}
		}
	}
	if !res {
		var tmpValue []byte
		return tmpValue
	}
	return msg.Trxs
}

func (s *SoBlocktrxsWrap) MdTrxs(p []byte) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Trxs")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemBlocktrxsByTrxs{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoBlocktrxs{}
	sa.Block = *s.mainKey
	sa.Trxs = ori.Trxs

	ori.Trxs = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Trxs = p

	return true
}

/////////////// SECTION Private function ////////////////

func (s *SoBlocktrxsWrap) update(sa *SoBlocktrxs) bool {
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

func (s *SoBlocktrxsWrap) getBlocktrxs() *SoBlocktrxs {
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

	res := &SoBlocktrxs{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoBlocktrxsWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := s.getMemKeyPrefix("Block")
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

func (s *SoBlocktrxsWrap) delAllUniKeys(br bool, val *SoBlocktrxs) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delUniKeyBlock(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoBlocktrxsWrap) delUniKeysWithNames(names map[string]string, val *SoBlocktrxs) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if len(names["Block"]) > 0 {
		if !s.delUniKeyBlock(val) {
			res = false
		}
	}

	return res
}

func (s *SoBlocktrxsWrap) insertAllUniKeys(val *SoBlocktrxs) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoBlocktrxs fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyBlock(val) {
		return sucFields, errors.New("insert unique Field Block fail while insert table ")
	}
	sucFields["Block"] = "Block"

	return sucFields, nil
}

func (s *SoBlocktrxsWrap) delUniKeyBlock(sa *SoBlocktrxs) bool {
	if s.dba == nil {
		return false
	}
	pre := BlocktrxsBlockUniTable
	kList := []interface{}{pre}
	if sa != nil {

		sub := sa.Block
		kList = append(kList, sub)
	} else {
		key, err := s.encodeMemKey("Block")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemBlocktrxsByBlock{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		sub := ori.Block
		kList = append(kList, sub)

	}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoBlocktrxsWrap) insertUniKeyBlock(sa *SoBlocktrxs) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	pre := BlocktrxsBlockUniTable
	sub := sa.Block
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
	val := SoUniqueBlocktrxsByBlock{}
	val.Block = sa.Block

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniBlocktrxsBlockWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniBlocktrxsBlockWrap(db iservices.IDatabaseRW) *UniBlocktrxsBlockWrap {
	if db == nil {
		return nil
	}
	wrap := UniBlocktrxsBlockWrap{Dba: db}
	return &wrap
}

func (s *UniBlocktrxsBlockWrap) UniQueryBlock(start *uint64) *SoBlocktrxsWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := BlocktrxsBlockUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueBlocktrxsByBlock{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoBlocktrxsWrap(s.Dba, &res.Block)
			return wrap
		}
	}
	return nil
}
