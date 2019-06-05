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
	StakeRecordRecordUniTable    uint32 = 832689285
	StakeRecordLastStakeTimeCell uint32 = 3228055551
	StakeRecordRecordCell        uint32 = 2514771326
	StakeRecordStakeAmountCell   uint32 = 906061269
)

////////////// SECTION Wrap Define ///////////////
type SoStakeRecordWrap struct {
	dba      iservices.IDatabaseRW
	mainKey  *prototype.StakeRecord
	mKeyFlag int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf  []byte //the buffer after the main key is encoded with prefix
	mBuf     []byte //the value after the main key is encoded
}

func NewSoStakeRecordWrap(dba iservices.IDatabaseRW, key *prototype.StakeRecord) *SoStakeRecordWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoStakeRecordWrap{dba, key, -1, nil, nil}
	return result
}

func (s *SoStakeRecordWrap) CheckExist() bool {
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

func (s *SoStakeRecordWrap) Create(f func(tInfo *SoStakeRecord)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoStakeRecord{}
	f(val)
	if val.Record == nil {
		val.Record = s.mainKey
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

func (s *SoStakeRecordWrap) getMainKeyBuf() ([]byte, error) {
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

func (s *SoStakeRecordWrap) delAllSortKeys(br bool, val *SoStakeRecord) bool {
	if s.dba == nil {
		return false
	}
	res := true

	return res
}

func (s *SoStakeRecordWrap) insertAllSortKeys(val *SoStakeRecord) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoStakeRecord fail ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoStakeRecordWrap) RemoveStakeRecord() bool {
	if s.dba == nil {
		return false
	}
	val := &SoStakeRecord{}
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
func (s *SoStakeRecordWrap) getMemKeyPrefix(fName string) uint32 {
	if fName == "LastStakeTime" {
		return StakeRecordLastStakeTimeCell
	}
	if fName == "Record" {
		return StakeRecordRecordCell
	}
	if fName == "StakeAmount" {
		return StakeRecordStakeAmountCell
	}

	return 0
}

func (s *SoStakeRecordWrap) encodeMemKey(fName string) ([]byte, error) {
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

func (s *SoStakeRecordWrap) saveAllMemKeys(tInfo *SoStakeRecord, br bool) error {
	if s.dba == nil {
		return errors.New("save member Field fail , the db is nil")
	}

	if tInfo == nil {
		return errors.New("save member Field fail , the data is nil ")
	}
	var err error = nil
	errDes := ""
	if err = s.saveMemKeyLastStakeTime(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "LastStakeTime", err)
		}
	}
	if err = s.saveMemKeyRecord(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Record", err)
		}
	}
	if err = s.saveMemKeyStakeAmount(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "StakeAmount", err)
		}
	}

	if len(errDes) > 0 {
		return errors.New(errDes)
	}
	return err
}

func (s *SoStakeRecordWrap) delAllMemKeys(br bool, tInfo *SoStakeRecord) error {
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

func (s *SoStakeRecordWrap) delMemKey(fName string) error {
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

func (s *SoStakeRecordWrap) saveMemKeyLastStakeTime(tInfo *SoStakeRecord) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemStakeRecordByLastStakeTime{}
	val.LastStakeTime = tInfo.LastStakeTime
	key, err := s.encodeMemKey("LastStakeTime")
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

func (s *SoStakeRecordWrap) GetLastStakeTime() *prototype.TimePointSec {
	res := true
	msg := &SoMemStakeRecordByLastStakeTime{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("LastStakeTime")
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
				return msg.LastStakeTime
			}
		}
	}
	if !res {
		return nil

	}
	return msg.LastStakeTime
}

func (s *SoStakeRecordWrap) MdLastStakeTime(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("LastStakeTime")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemStakeRecordByLastStakeTime{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoStakeRecord{}
	sa.Record = s.mainKey

	sa.LastStakeTime = ori.LastStakeTime

	ori.LastStakeTime = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.LastStakeTime = p

	return true
}

func (s *SoStakeRecordWrap) saveMemKeyRecord(tInfo *SoStakeRecord) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemStakeRecordByRecord{}
	val.Record = tInfo.Record
	key, err := s.encodeMemKey("Record")
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

func (s *SoStakeRecordWrap) GetRecord() *prototype.StakeRecord {
	res := true
	msg := &SoMemStakeRecordByRecord{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Record")
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
				return msg.Record
			}
		}
	}
	if !res {
		return nil

	}
	return msg.Record
}

func (s *SoStakeRecordWrap) saveMemKeyStakeAmount(tInfo *SoStakeRecord) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemStakeRecordByStakeAmount{}
	val.StakeAmount = tInfo.StakeAmount
	key, err := s.encodeMemKey("StakeAmount")
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

func (s *SoStakeRecordWrap) GetStakeAmount() *prototype.Vest {
	res := true
	msg := &SoMemStakeRecordByStakeAmount{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("StakeAmount")
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
				return msg.StakeAmount
			}
		}
	}
	if !res {
		return nil

	}
	return msg.StakeAmount
}

func (s *SoStakeRecordWrap) MdStakeAmount(p *prototype.Vest) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("StakeAmount")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemStakeRecordByStakeAmount{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoStakeRecord{}
	sa.Record = s.mainKey

	sa.StakeAmount = ori.StakeAmount

	ori.StakeAmount = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.StakeAmount = p

	return true
}

/////////////// SECTION Private function ////////////////

func (s *SoStakeRecordWrap) update(sa *SoStakeRecord) bool {
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

func (s *SoStakeRecordWrap) getStakeRecord() *SoStakeRecord {
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

	res := &SoStakeRecord{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoStakeRecordWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := s.getMemKeyPrefix("Record")
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

func (s *SoStakeRecordWrap) delAllUniKeys(br bool, val *SoStakeRecord) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delUniKeyRecord(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoStakeRecordWrap) delUniKeysWithNames(names map[string]string, val *SoStakeRecord) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if len(names["Record"]) > 0 {
		if !s.delUniKeyRecord(val) {
			res = false
		}
	}

	return res
}

func (s *SoStakeRecordWrap) insertAllUniKeys(val *SoStakeRecord) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoStakeRecord fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyRecord(val) {
		return sucFields, errors.New("insert unique Field Record fail while insert table ")
	}
	sucFields["Record"] = "Record"

	return sucFields, nil
}

func (s *SoStakeRecordWrap) delUniKeyRecord(sa *SoStakeRecord) bool {
	if s.dba == nil {
		return false
	}
	pre := StakeRecordRecordUniTable
	kList := []interface{}{pre}
	if sa != nil {

		if sa.Record == nil {
			return false
		}

		sub := sa.Record
		kList = append(kList, sub)
	} else {
		key, err := s.encodeMemKey("Record")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemStakeRecordByRecord{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		sub := ori.Record
		kList = append(kList, sub)

	}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoStakeRecordWrap) insertUniKeyRecord(sa *SoStakeRecord) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	pre := StakeRecordRecordUniTable
	sub := sa.Record
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
	val := SoUniqueStakeRecordByRecord{}
	val.Record = sa.Record

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniStakeRecordRecordWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniStakeRecordRecordWrap(db iservices.IDatabaseRW) *UniStakeRecordRecordWrap {
	if db == nil {
		return nil
	}
	wrap := UniStakeRecordRecordWrap{Dba: db}
	return &wrap
}

func (s *UniStakeRecordRecordWrap) UniQueryRecord(start *prototype.StakeRecord) *SoStakeRecordWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := StakeRecordRecordUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueStakeRecordByRecord{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoStakeRecordWrap(s.Dba, res.Record)

			return wrap
		}
	}
	return nil
}
