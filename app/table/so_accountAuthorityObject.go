package table

import (
	"errors"

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
	resBuf, err := proto.Marshal(val)
	if err != nil {
		return err
	}
	err = s.dba.Put(keyBuf, resBuf)
	if err != nil {
		return err
	}

	// update sort list keys
	if err = s.insertAllSortKeys(val); err != nil {
		s.delAllSortKeys(false, val)
		s.dba.Delete(keyBuf)
		return err
	}

	//update unique list
	if err = s.insertAllUniKeys(val); err != nil {
		s.delAllSortKeys(false, val)
		s.delAllUniKeys(false, val)
		s.dba.Delete(keyBuf)
		return err
	}

	return nil
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoAccountAuthorityObjectWrap) delAllSortKeys(br bool, val *SoAccountAuthorityObject) bool {
	if s.dba == nil || val == nil {
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
	val := s.getAccountAuthorityObject()
	if val == nil {
		return false
	}
	//delete sort list key
	if res := s.delAllSortKeys(true, val); !res {
		return false
	}

	//delete unique list
	if res := s.delAllUniKeys(true, val); !res {
		return false
	}

	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return false
	}
	return s.dba.Delete(keyBuf) == nil
}

////////////// SECTION Members Get/Modify ///////////////
func (s *SoAccountAuthorityObjectWrap) GetAccount() *prototype.AccountName {
	res := s.getAccountAuthorityObject()

	if res == nil {
		return nil

	}
	return res.Account
}

func (s *SoAccountAuthorityObjectWrap) GetActive() *prototype.Authority {
	res := s.getAccountAuthorityObject()

	if res == nil {
		return nil

	}
	return res.Active
}

func (s *SoAccountAuthorityObjectWrap) MdActive(p *prototype.Authority) bool {
	if s.dba == nil {
		return false
	}
	sa := s.getAccountAuthorityObject()
	if sa == nil {
		return false
	}

	sa.Active = p
	if !s.update(sa) {
		return false
	}

	return true
}

func (s *SoAccountAuthorityObjectWrap) GetLastOwnerUpdate() *prototype.TimePointSec {
	res := s.getAccountAuthorityObject()

	if res == nil {
		return nil

	}
	return res.LastOwnerUpdate
}

func (s *SoAccountAuthorityObjectWrap) MdLastOwnerUpdate(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}
	sa := s.getAccountAuthorityObject()
	if sa == nil {
		return false
	}

	sa.LastOwnerUpdate = p
	if !s.update(sa) {
		return false
	}

	return true
}

func (s *SoAccountAuthorityObjectWrap) GetOwner() *prototype.Authority {
	res := s.getAccountAuthorityObject()

	if res == nil {
		return nil

	}
	return res.Owner
}

func (s *SoAccountAuthorityObjectWrap) MdOwner(p *prototype.Authority) bool {
	if s.dba == nil {
		return false
	}
	sa := s.getAccountAuthorityObject()
	if sa == nil {
		return false
	}

	sa.Owner = p
	if !s.update(sa) {
		return false
	}

	return true
}

func (s *SoAccountAuthorityObjectWrap) GetPosting() *prototype.Authority {
	res := s.getAccountAuthorityObject()

	if res == nil {
		return nil

	}
	return res.Posting
}

func (s *SoAccountAuthorityObjectWrap) MdPosting(p *prototype.Authority) bool {
	if s.dba == nil {
		return false
	}
	sa := s.getAccountAuthorityObject()
	if sa == nil {
		return false
	}

	sa.Posting = p
	if !s.update(sa) {
		return false
	}

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
	pre := AccountAuthorityObjectTable
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
	if s.dba == nil || val == nil {
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

func (s *SoAccountAuthorityObjectWrap) insertAllUniKeys(val *SoAccountAuthorityObject) error {
	if s.dba == nil {
		return errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert uniuqe Field fail,get the SoAccountAuthorityObject fail ")
	}
	if !s.insertUniKeyAccount(val) {
		return errors.New("insert unique Field Account fail while insert table ")
	}

	return nil
}

func (s *SoAccountAuthorityObjectWrap) delUniKeyAccount(sa *SoAccountAuthorityObject) bool {
	if s.dba == nil {
		return false
	}

	if sa.Account == nil {
		return false
	}

	pre := AccountAuthorityObjectAccountUniTable
	sub := sa.Account
	kList := []interface{}{pre, sub}
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
