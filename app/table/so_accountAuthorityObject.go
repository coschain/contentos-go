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
	AccountAuthorityObjectTable           = []byte("AccountAuthorityObjectTable")
	AccountAuthorityObjectAccountUniTable = []byte("AccountAuthorityObjectAccountUniTable")
)

////////////// SECTION Wrap Define ///////////////
type SoAccountAuthorityObjectWrap struct {
	dba     iservices.IDatabaseService
	mainKey *prototype.AccountName
}

func NewSoAccountAuthorityObjectWrap(dba iservices.IDatabaseService, key *prototype.AccountName) *SoAccountAuthorityObjectWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoAccountAuthorityObjectWrap{dba, key}
	return result
}

func (s *SoAccountAuthorityObjectWrap) CheckExist() bool {
	if s.dba == nil {
		return false
	}
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return false
	}

	res, err := s.dba.Has(keyBuf)
	if err != nil {
		return false
	}

	return res
}

func (s *SoAccountAuthorityObjectWrap) Create(f func(tInfo *SoAccountAuthorityObject)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoAccountAuthorityObject{}
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

func (s *SoAccountAuthorityObjectWrap) encodeMemKey(fName string) ([]byte, error) {
	if len(fName) < 1 || s.mainKey == nil {
		return nil, errors.New("field name or main key is empty")
	}
	pre := "AccountAuthorityObject" + fName + "cell"
	kList := []interface{}{pre, s.mainKey}
	key, err := kope.EncodeSlice(kList)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (so *SoAccountAuthorityObjectWrap) saveAllMemKeys(tInfo *SoAccountAuthorityObject, br bool) error {
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
	if err = so.saveMemKeyActive(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Active", err)
		}
	}
	if err = so.saveMemKeyLastOwnerUpdate(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "LastOwnerUpdate", err)
		}
	}
	if err = so.saveMemKeyOwner(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Owner", err)
		}
	}
	if err = so.saveMemKeyPosting(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Posting", err)
		}
	}

	if len(errDes) > 0 {
		return errors.New(errDes)
	}
	return err
}

func (so *SoAccountAuthorityObjectWrap) delAllMemKeys(br bool, tInfo *SoAccountAuthorityObject) error {
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

func (so *SoAccountAuthorityObjectWrap) delMemKey(fName string) error {
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

func (s *SoAccountAuthorityObjectWrap) delAllSortKeys(br bool, val *SoAccountAuthorityObject) bool {
	if s.dba == nil {
		return false
	}
	res := true

	return res
}

func (s *SoAccountAuthorityObjectWrap) insertAllSortKeys(val *SoAccountAuthorityObject) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoAccountAuthorityObject fail ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoAccountAuthorityObjectWrap) RemoveAccountAuthorityObject() bool {
	if s.dba == nil {
		return false
	}
	val := &SoAccountAuthorityObject{}
	//delete sort list key
	if res := s.delAllSortKeys(true, nil); !res {
		return false
	}

	//delete unique list
	if res := s.delAllUniKeys(true, nil); !res {
		return false
	}

	err := s.delAllMemKeys(true, val)
	return err == nil
}

////////////// SECTION Members Get/Modify ///////////////
func (s *SoAccountAuthorityObjectWrap) saveMemKeyAccount(tInfo *SoAccountAuthorityObject) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemAccountAuthorityObjectByAccount{}
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

func (s *SoAccountAuthorityObjectWrap) GetAccount() *prototype.AccountName {
	res := true
	msg := &SoMemAccountAuthorityObjectByAccount{}
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

func (s *SoAccountAuthorityObjectWrap) saveMemKeyActive(tInfo *SoAccountAuthorityObject) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemAccountAuthorityObjectByActive{}
	val.Active = tInfo.Active
	key, err := s.encodeMemKey("Active")
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

func (s *SoAccountAuthorityObjectWrap) GetActive() *prototype.Authority {
	res := true
	msg := &SoMemAccountAuthorityObjectByActive{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Active")
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
				return msg.Active
			}
		}
	}
	if !res {
		return nil

	}
	return msg.Active
}

func (s *SoAccountAuthorityObjectWrap) MdActive(p *prototype.Authority) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Active")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemAccountAuthorityObjectByActive{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoAccountAuthorityObject{}
	sa.Account = s.mainKey

	sa.Active = ori.Active

	ori.Active = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Active = p

	return true
}

func (s *SoAccountAuthorityObjectWrap) saveMemKeyLastOwnerUpdate(tInfo *SoAccountAuthorityObject) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemAccountAuthorityObjectByLastOwnerUpdate{}
	val.LastOwnerUpdate = tInfo.LastOwnerUpdate
	key, err := s.encodeMemKey("LastOwnerUpdate")
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

func (s *SoAccountAuthorityObjectWrap) GetLastOwnerUpdate() *prototype.TimePointSec {
	res := true
	msg := &SoMemAccountAuthorityObjectByLastOwnerUpdate{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("LastOwnerUpdate")
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
				return msg.LastOwnerUpdate
			}
		}
	}
	if !res {
		return nil

	}
	return msg.LastOwnerUpdate
}

func (s *SoAccountAuthorityObjectWrap) MdLastOwnerUpdate(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("LastOwnerUpdate")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemAccountAuthorityObjectByLastOwnerUpdate{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoAccountAuthorityObject{}
	sa.Account = s.mainKey

	sa.LastOwnerUpdate = ori.LastOwnerUpdate

	ori.LastOwnerUpdate = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.LastOwnerUpdate = p

	return true
}

func (s *SoAccountAuthorityObjectWrap) saveMemKeyOwner(tInfo *SoAccountAuthorityObject) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemAccountAuthorityObjectByOwner{}
	val.Owner = tInfo.Owner
	key, err := s.encodeMemKey("Owner")
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

func (s *SoAccountAuthorityObjectWrap) GetOwner() *prototype.Authority {
	res := true
	msg := &SoMemAccountAuthorityObjectByOwner{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Owner")
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
				return msg.Owner
			}
		}
	}
	if !res {
		return nil

	}
	return msg.Owner
}

func (s *SoAccountAuthorityObjectWrap) MdOwner(p *prototype.Authority) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Owner")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemAccountAuthorityObjectByOwner{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoAccountAuthorityObject{}
	sa.Account = s.mainKey

	sa.Owner = ori.Owner

	ori.Owner = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Owner = p

	return true
}

func (s *SoAccountAuthorityObjectWrap) saveMemKeyPosting(tInfo *SoAccountAuthorityObject) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemAccountAuthorityObjectByPosting{}
	val.Posting = tInfo.Posting
	key, err := s.encodeMemKey("Posting")
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

func (s *SoAccountAuthorityObjectWrap) GetPosting() *prototype.Authority {
	res := true
	msg := &SoMemAccountAuthorityObjectByPosting{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Posting")
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
				return msg.Posting
			}
		}
	}
	if !res {
		return nil

	}
	return msg.Posting
}

func (s *SoAccountAuthorityObjectWrap) MdPosting(p *prototype.Authority) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Posting")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemAccountAuthorityObjectByPosting{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoAccountAuthorityObject{}
	sa.Account = s.mainKey

	sa.Posting = ori.Posting

	ori.Posting = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Posting = p

	return true
}

/////////////// SECTION Private function ////////////////

func (s *SoAccountAuthorityObjectWrap) update(sa *SoAccountAuthorityObject) bool {
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

func (s *SoAccountAuthorityObjectWrap) getAccountAuthorityObject() *SoAccountAuthorityObject {
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

	res := &SoAccountAuthorityObject{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoAccountAuthorityObjectWrap) encodeMainKey() ([]byte, error) {
	pre := "AccountAuthorityObject" + "Account" + "cell"
	sub := s.mainKey
	if sub == nil {
		return nil, errors.New("the mainKey is nil")
	}
	kList := []interface{}{pre, sub}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
}

////////////// Unique Query delete/insert/query ///////////////

func (s *SoAccountAuthorityObjectWrap) delAllUniKeys(br bool, val *SoAccountAuthorityObject) bool {
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

func (s *SoAccountAuthorityObjectWrap) delUniKeysWithNames(names map[string]string, val *SoAccountAuthorityObject) bool {
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

func (s *SoAccountAuthorityObjectWrap) insertAllUniKeys(val *SoAccountAuthorityObject) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoAccountAuthorityObject fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyAccount(val) {
		return sucFields, errors.New("insert unique Field Account fail while insert table ")
	}
	sucFields["Account"] = "Account"

	return sucFields, nil
}

func (s *SoAccountAuthorityObjectWrap) delUniKeyAccount(sa *SoAccountAuthorityObject) bool {
	if s.dba == nil {
		return false
	}
	pre := AccountAuthorityObjectAccountUniTable
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
		ori := &SoMemAccountAuthorityObjectByAccount{}
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

func (s *SoAccountAuthorityObjectWrap) insertUniKeyAccount(sa *SoAccountAuthorityObject) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	uniWrap := UniAccountAuthorityObjectAccountWrap{}
	uniWrap.Dba = s.dba

	res := uniWrap.UniQueryAccount(sa.Account)
	if res != nil {
		//the unique key is already exist
		return false
	}
	val := SoUniqueAccountAuthorityObjectByAccount{}
	val.Account = sa.Account

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	pre := AccountAuthorityObjectAccountUniTable
	sub := sa.Account
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniAccountAuthorityObjectAccountWrap struct {
	Dba iservices.IDatabaseService
}

func NewUniAccountAuthorityObjectAccountWrap(db iservices.IDatabaseService) *UniAccountAuthorityObjectAccountWrap {
	if db == nil {
		return nil
	}
	wrap := UniAccountAuthorityObjectAccountWrap{Dba: db}
	return &wrap
}

func (s *UniAccountAuthorityObjectAccountWrap) UniQueryAccount(start *prototype.AccountName) *SoAccountAuthorityObjectWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := AccountAuthorityObjectAccountUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueAccountAuthorityObjectByAccount{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoAccountAuthorityObjectWrap(s.Dba, res.Account)

			return wrap
		}
	}
	return nil
}
