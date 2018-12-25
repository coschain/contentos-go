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
	RewardsKeeperTable      = []byte("RewardsKeeperTable")
	RewardsKeeperIdUniTable = []byte("RewardsKeeperIdUniTable")
)

////////////// SECTION Wrap Define ///////////////
type SoRewardsKeeperWrap struct {
	dba      iservices.IDatabaseService
	mainKey  *int32
	mKeyFlag int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf  []byte //the buffer after the main key is encoded
}

func NewSoRewardsKeeperWrap(dba iservices.IDatabaseService, key *int32) *SoRewardsKeeperWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoRewardsKeeperWrap{dba, key, -1, nil}
	return result
}

func (s *SoRewardsKeeperWrap) CheckExist() bool {
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

func (s *SoRewardsKeeperWrap) Create(f func(tInfo *SoRewardsKeeper)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoRewardsKeeper{}
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

func (s *SoRewardsKeeperWrap) encodeMemKey(fName string) ([]byte, error) {
	if len(fName) < 1 || s.mainKey == nil {
		return nil, errors.New("field name or main key is empty")
	}
	pre := "RewardsKeeper" + fName + "cell"
	kList := []interface{}{pre, s.mainKey}
	key, err := kope.EncodeSlice(kList)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (so *SoRewardsKeeperWrap) saveAllMemKeys(tInfo *SoRewardsKeeper, br bool) error {
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
	if err = so.saveMemKeyKeeper(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Keeper", err)
		}
	}

	if len(errDes) > 0 {
		return errors.New(errDes)
	}
	return err
}

func (so *SoRewardsKeeperWrap) delAllMemKeys(br bool, tInfo *SoRewardsKeeper) error {
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

func (so *SoRewardsKeeperWrap) delMemKey(fName string) error {
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

func (s *SoRewardsKeeperWrap) delAllSortKeys(br bool, val *SoRewardsKeeper) bool {
	if s.dba == nil {
		return false
	}
	res := true

	return res
}

func (s *SoRewardsKeeperWrap) insertAllSortKeys(val *SoRewardsKeeper) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoRewardsKeeper fail ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoRewardsKeeperWrap) RemoveRewardsKeeper() bool {
	if s.dba == nil {
		return false
	}
	val := &SoRewardsKeeper{}
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
func (s *SoRewardsKeeperWrap) saveMemKeyId(tInfo *SoRewardsKeeper) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemRewardsKeeperById{}
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

func (s *SoRewardsKeeperWrap) GetId() int32 {
	res := true
	msg := &SoMemRewardsKeeperById{}
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
		var tmpValue int32
		return tmpValue
	}
	return msg.Id
}

func (s *SoRewardsKeeperWrap) saveMemKeyKeeper(tInfo *SoRewardsKeeper) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemRewardsKeeperByKeeper{}
	val.Keeper = tInfo.Keeper
	key, err := s.encodeMemKey("Keeper")
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

func (s *SoRewardsKeeperWrap) GetKeeper() *prototype.InternalRewardsKeeper {
	res := true
	msg := &SoMemRewardsKeeperByKeeper{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Keeper")
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
				return msg.Keeper
			}
		}
	}
	if !res {
		return nil

	}
	return msg.Keeper
}

func (s *SoRewardsKeeperWrap) MdKeeper(p *prototype.InternalRewardsKeeper) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Keeper")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemRewardsKeeperByKeeper{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoRewardsKeeper{}
	sa.Id = *s.mainKey
	sa.Keeper = ori.Keeper

	ori.Keeper = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Keeper = p

	return true
}

/////////////// SECTION Private function ////////////////

func (s *SoRewardsKeeperWrap) update(sa *SoRewardsKeeper) bool {
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

func (s *SoRewardsKeeperWrap) getRewardsKeeper() *SoRewardsKeeper {
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

	res := &SoRewardsKeeper{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoRewardsKeeperWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := "RewardsKeeper" + "Id" + "cell"
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

func (s *SoRewardsKeeperWrap) delAllUniKeys(br bool, val *SoRewardsKeeper) bool {
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

func (s *SoRewardsKeeperWrap) delUniKeysWithNames(names map[string]string, val *SoRewardsKeeper) bool {
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

func (s *SoRewardsKeeperWrap) insertAllUniKeys(val *SoRewardsKeeper) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoRewardsKeeper fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyId(val) {
		return sucFields, errors.New("insert unique Field Id fail while insert table ")
	}
	sucFields["Id"] = "Id"

	return sucFields, nil
}

func (s *SoRewardsKeeperWrap) delUniKeyId(sa *SoRewardsKeeper) bool {
	if s.dba == nil {
		return false
	}
	pre := RewardsKeeperIdUniTable
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
		ori := &SoMemRewardsKeeperById{}
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

func (s *SoRewardsKeeperWrap) insertUniKeyId(sa *SoRewardsKeeper) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	pre := RewardsKeeperIdUniTable
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
	val := SoUniqueRewardsKeeperById{}
	val.Id = sa.Id

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniRewardsKeeperIdWrap struct {
	Dba iservices.IDatabaseService
}

func NewUniRewardsKeeperIdWrap(db iservices.IDatabaseService) *UniRewardsKeeperIdWrap {
	if db == nil {
		return nil
	}
	wrap := UniRewardsKeeperIdWrap{Dba: db}
	return &wrap
}

func (s *UniRewardsKeeperIdWrap) UniQueryId(start *int32) *SoRewardsKeeperWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := RewardsKeeperIdUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueRewardsKeeperById{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoRewardsKeeperWrap(s.Dba, &res.Id)
			return wrap
		}
	}
	return nil
}
