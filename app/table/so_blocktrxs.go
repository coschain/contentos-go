package table

import (
	"errors"
	fmt "fmt"
	"reflect"

	"github.com/coschain/contentos-go/common/encoding/kope"
	"github.com/coschain/contentos-go/iservices"
	proto "github.com/golang/protobuf/proto"
)

////////////// SECTION Prefix Mark ///////////////
var (
	BlocktrxsBlockUniTable uint32 = 3461050414

	BlocktrxsBlockRow uint32 = 4250009783
)

////////////// SECTION Wrap Define ///////////////
type SoBlocktrxsWrap struct {
	dba       iservices.IDatabaseRW
	mainKey   *uint64
	mKeyFlag  int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf   []byte //the buffer after the main key is encoded with prefix
	mBuf      []byte //the value after the main key is encoded
	mdFuncMap map[string]interface{}
}

func NewSoBlocktrxsWrap(dba iservices.IDatabaseRW, key *uint64) *SoBlocktrxsWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoBlocktrxsWrap{dba, key, -1, nil, nil, nil}
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

func (s *SoBlocktrxsWrap) Md(f func(tInfo *SoBlocktrxs)) error {
	if !s.CheckExist() {
		return errors.New("the SoBlocktrxs table does not exist. Please create a table first")
	}
	oriTable := s.getBlocktrxs()
	if oriTable == nil {
		return errors.New("fail to get origin table SoBlocktrxs")
	}
	curTable := *oriTable
	f(&curTable)

	//the main key is not support modify
	if !reflect.DeepEqual(curTable.Block, oriTable.Block) {
		curTable.Block = oriTable.Block
	}

	fieldSli, err := s.getModifiedFields(oriTable, &curTable)
	if err != nil {
		return err
	}

	if fieldSli == nil || len(fieldSli) < 1 {
		return nil
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
	err = s.updateBlocktrxs(&curTable)
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

//Get all the modified fields in the table
func (s *SoBlocktrxsWrap) getModifiedFields(oriTable *SoBlocktrxs, curTable *SoBlocktrxs) ([]string, error) {
	if oriTable == nil {
		return nil, errors.New("table info is nil, can't get modified fields")
	}
	var list []string

	if !reflect.DeepEqual(oriTable.Trxs, curTable.Trxs) {
		list = append(list, "Trxs")
	}

	return list, nil
}

func (s *SoBlocktrxsWrap) handleFieldMd(t FieldMdHandleType, so *SoBlocktrxs, fSli []string) error {
	if so == nil {
		return errors.New("fail to modify empty table")
	}

	//there is no field need to modify
	if fSli == nil || len(fSli) < 1 {
		return nil
	}

	errStr := ""
	for _, fName := range fSli {

		if fName == "Trxs" {
			res := true
			if t == FieldMdHandleTypeCheck {
				res = s.mdFieldTrxs(so.Trxs, true, false, false, so)
				errStr = fmt.Sprintf("fail to modify exist value of %v", fName)
			} else if t == FieldMdHandleTypeDel {
				res = s.mdFieldTrxs(so.Trxs, false, true, false, so)
				errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", fName)
			} else if t == FieldMdHandleTypeInsert {
				res = s.mdFieldTrxs(so.Trxs, false, false, true, so)
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

func (s *SoBlocktrxsWrap) GetBlock() uint64 {
	res := true
	msg := &SoBlocktrxs{}
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

func (s *SoBlocktrxsWrap) GetTrxs() []byte {
	res := true
	msg := &SoBlocktrxs{}
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

func (s *SoBlocktrxsWrap) mdFieldTrxs(p []byte, isCheck bool, isDel bool, isInsert bool,
	so *SoBlocktrxs) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkTrxsIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldTrxs(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldTrxs(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoBlocktrxsWrap) delFieldTrxs(so *SoBlocktrxs) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlocktrxsWrap) insertFieldTrxs(so *SoBlocktrxs) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlocktrxsWrap) checkTrxsIsMetMdCondition(p []byte) bool {
	if s.dba == nil {
		return false
	}

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

func (s *SoBlocktrxsWrap) updateBlocktrxs(so *SoBlocktrxs) error {
	if s.dba == nil {
		return errors.New("update fail:the db is nil")
	}

	if so == nil {
		return errors.New("update fail: the SoBlocktrxs is nil")
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

func (s *SoBlocktrxsWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := BlocktrxsBlockRow
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
		sub := s.GetBlock()

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
