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
	ExtFollowCountTable           = []byte("ExtFollowCountTable")
	ExtFollowCountAccountUniTable = []byte("ExtFollowCountAccountUniTable")
)

////////////// SECTION Wrap Define ///////////////
type SoExtFollowCountWrap struct {
	dba      iservices.IDatabaseService
	mainKey  *prototype.AccountName
	mKeyFlag int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf  []byte //the buffer after the main key is encoded
}

func NewSoExtFollowCountWrap(dba iservices.IDatabaseService, key *prototype.AccountName) *SoExtFollowCountWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoExtFollowCountWrap{dba, key, -1, nil}
	return result
}

func (s *SoExtFollowCountWrap) CheckExist() bool {
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

func (s *SoExtFollowCountWrap) Create(f func(tInfo *SoExtFollowCount)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoExtFollowCount{}
	f(val)
	if val.Account == nil {
		val.Account = s.mainKey
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

func (s *SoExtFollowCountWrap) encodeMemKey(fName string) ([]byte, error) {
	if len(fName) < 1 || s.mainKey == nil {
		return nil, errors.New("field name or main key is empty")
	}
	pre := "ExtFollowCount" + fName + "cell"
	kList := []interface{}{pre, s.mainKey}
	key, err := kope.EncodeSlice(kList)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (so *SoExtFollowCountWrap) saveAllMemKeys(tInfo *SoExtFollowCount, br bool) error {
	if so.dba == nil {
		return errors.New("save member Field fail , the db is nil")
	}

	if tInfo == nil {
		return errors.New("save member Field fail , the data is nil ")
	}
	var err error = nil
	errDes := ""
	if err = so.saveMemKeyAccount(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Account", err)
		}
	}
	if err = so.saveMemKeyFollowerCnt(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "FollowerCnt", err)
		}
	}
	if err = so.saveMemKeyFollowingCnt(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "FollowingCnt", err)
		}
	}
	if err = so.saveMemKeyUpdateTime(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "UpdateTime", err)
		}
	}

	if len(errDes) > 0 {
		return errors.New(errDes)
	}
	return err
}

func (so *SoExtFollowCountWrap) delAllMemKeys(br bool, tInfo *SoExtFollowCount) error {
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

func (so *SoExtFollowCountWrap) delMemKey(fName string) error {
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

func (s *SoExtFollowCountWrap) delAllSortKeys(br bool, val *SoExtFollowCount) bool {
	if s.dba == nil {
		return false
	}
	res := true

	return res
}

func (s *SoExtFollowCountWrap) insertAllSortKeys(val *SoExtFollowCount) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoExtFollowCount fail ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoExtFollowCountWrap) RemoveExtFollowCount() bool {
	if s.dba == nil {
		return false
	}
	val := &SoExtFollowCount{}
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
func (s *SoExtFollowCountWrap) saveMemKeyAccount(tInfo *SoExtFollowCount) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemExtFollowCountByAccount{}
	val.Account = tInfo.Account
	key, err := s.encodeMemKey("Account")
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

func (s *SoExtFollowCountWrap) GetAccount() *prototype.AccountName {
	res := true
	msg := &SoMemExtFollowCountByAccount{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Account")
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
				return msg.Account
			}
		}
	}
	if !res {
		return nil

	}
	return msg.Account
}

func (s *SoExtFollowCountWrap) saveMemKeyFollowerCnt(tInfo *SoExtFollowCount) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemExtFollowCountByFollowerCnt{}
	val.FollowerCnt = tInfo.FollowerCnt
	key, err := s.encodeMemKey("FollowerCnt")
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

func (s *SoExtFollowCountWrap) GetFollowerCnt() uint32 {
	res := true
	msg := &SoMemExtFollowCountByFollowerCnt{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("FollowerCnt")
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
				return msg.FollowerCnt
			}
		}
	}
	if !res {
		var tmpValue uint32
		return tmpValue
	}
	return msg.FollowerCnt
}

func (s *SoExtFollowCountWrap) MdFollowerCnt(p uint32) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("FollowerCnt")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemExtFollowCountByFollowerCnt{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoExtFollowCount{}
	sa.Account = s.mainKey

	sa.FollowerCnt = ori.FollowerCnt

	ori.FollowerCnt = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.FollowerCnt = p

	return true
}

func (s *SoExtFollowCountWrap) saveMemKeyFollowingCnt(tInfo *SoExtFollowCount) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemExtFollowCountByFollowingCnt{}
	val.FollowingCnt = tInfo.FollowingCnt
	key, err := s.encodeMemKey("FollowingCnt")
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

func (s *SoExtFollowCountWrap) GetFollowingCnt() uint32 {
	res := true
	msg := &SoMemExtFollowCountByFollowingCnt{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("FollowingCnt")
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
				return msg.FollowingCnt
			}
		}
	}
	if !res {
		var tmpValue uint32
		return tmpValue
	}
	return msg.FollowingCnt
}

func (s *SoExtFollowCountWrap) MdFollowingCnt(p uint32) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("FollowingCnt")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemExtFollowCountByFollowingCnt{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoExtFollowCount{}
	sa.Account = s.mainKey

	sa.FollowingCnt = ori.FollowingCnt

	ori.FollowingCnt = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.FollowingCnt = p

	return true
}

func (s *SoExtFollowCountWrap) saveMemKeyUpdateTime(tInfo *SoExtFollowCount) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemExtFollowCountByUpdateTime{}
	val.UpdateTime = tInfo.UpdateTime
	key, err := s.encodeMemKey("UpdateTime")
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

func (s *SoExtFollowCountWrap) GetUpdateTime() *prototype.TimePointSec {
	res := true
	msg := &SoMemExtFollowCountByUpdateTime{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("UpdateTime")
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
				return msg.UpdateTime
			}
		}
	}
	if !res {
		return nil

	}
	return msg.UpdateTime
}

func (s *SoExtFollowCountWrap) MdUpdateTime(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("UpdateTime")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemExtFollowCountByUpdateTime{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoExtFollowCount{}
	sa.Account = s.mainKey

	sa.UpdateTime = ori.UpdateTime

	ori.UpdateTime = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.UpdateTime = p

	return true
}

/////////////// SECTION Private function ////////////////

func (s *SoExtFollowCountWrap) update(sa *SoExtFollowCount) bool {
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

func (s *SoExtFollowCountWrap) getExtFollowCount() *SoExtFollowCount {
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

	res := &SoExtFollowCount{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoExtFollowCountWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := "ExtFollowCount" + "Account" + "cell"
	sub := s.mainKey
	if sub == nil {
		return nil, errors.New("the mainKey is nil")
	}
	kList := []interface{}{pre, sub}
	var cErr error = nil
	s.mKeyBuf, cErr = kope.EncodeSlice(kList)
	return s.mKeyBuf, cErr
}

////////////// Unique Query delete/insert/query ///////////////

func (s *SoExtFollowCountWrap) delAllUniKeys(br bool, val *SoExtFollowCount) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delUniKeyAccount(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoExtFollowCountWrap) delUniKeysWithNames(names map[string]string, val *SoExtFollowCount) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if len(names["Account"]) > 0 {
		if !s.delUniKeyAccount(val) {
			res = false
		}
	}

	return res
}

func (s *SoExtFollowCountWrap) insertAllUniKeys(val *SoExtFollowCount) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoExtFollowCount fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyAccount(val) {
		return sucFields, errors.New("insert unique Field Account fail while insert table ")
	}
	sucFields["Account"] = "Account"

	return sucFields, nil
}

func (s *SoExtFollowCountWrap) delUniKeyAccount(sa *SoExtFollowCount) bool {
	if s.dba == nil {
		return false
	}
	pre := ExtFollowCountAccountUniTable
	kList := []interface{}{pre}
	if sa != nil {

		if sa.Account == nil {
			return false
		}

		sub := sa.Account
		kList = append(kList, sub)
	} else {
		key, err := s.encodeMemKey("Account")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemExtFollowCountByAccount{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		sub := ori.Account
		kList = append(kList, sub)

	}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoExtFollowCountWrap) insertUniKeyAccount(sa *SoExtFollowCount) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	pre := ExtFollowCountAccountUniTable
	sub := sa.Account
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
	val := SoUniqueExtFollowCountByAccount{}
	val.Account = sa.Account

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniExtFollowCountAccountWrap struct {
	Dba iservices.IDatabaseService
}

func NewUniExtFollowCountAccountWrap(db iservices.IDatabaseService) *UniExtFollowCountAccountWrap {
	if db == nil {
		return nil
	}
	wrap := UniExtFollowCountAccountWrap{Dba: db}
	return &wrap
}

func (s *UniExtFollowCountAccountWrap) UniQueryAccount(start *prototype.AccountName) *SoExtFollowCountWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := ExtFollowCountAccountUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueExtFollowCountByAccount{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoExtFollowCountWrap(s.Dba, res.Account)

			return wrap
		}
	}
	return nil
}
