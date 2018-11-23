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
	FollowCountTable           = []byte("FollowCountTable")
	FollowCountAccountUniTable = []byte("FollowCountAccountUniTable")
)

////////////// SECTION Wrap Define ///////////////
type SoFollowCountWrap struct {
	dba     iservices.IDatabaseService
	mainKey *prototype.AccountName
}

func NewSoFollowCountWrap(dba iservices.IDatabaseService, key *prototype.AccountName) *SoFollowCountWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoFollowCountWrap{dba, key}
	return result
}

func (s *SoFollowCountWrap) CheckExist() bool {
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

func (s *SoFollowCountWrap) Create(f func(tInfo *SoFollowCount)) error {
	val := &SoFollowCount{}
	f(val)
	if val.Account == nil {
		return errors.New("the mainkey is nil")
	}
	if s.CheckExist() {
		return errors.New("the mainkey is already exist")
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

	//update unique list
	if !s.insertUniKeyAccount(val) {
		return errors.New("insert unique Field prototype.AccountName while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert ///////////////

////////////// SECTION LKeys delete/insert //////////////

func (s *SoFollowCountWrap) RemoveFollowCount() bool {
	if s.dba == nil {
		return false
	}
	sa := s.getFollowCount()
	if sa == nil {
		return false
	}
	//delete sort list key

	//delete unique list
	if !s.delUniKeyAccount(sa) {
		return false
	}

	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return false
	}
	return s.dba.Delete(keyBuf) == nil
}

////////////// SECTION Members Get/Modify ///////////////
func (s *SoFollowCountWrap) GetAccount() *prototype.AccountName {
	res := s.getFollowCount()

	if res == nil {
		return nil

	}
	return res.Account
}

func (s *SoFollowCountWrap) GetFollowerCnt() uint32 {
	res := s.getFollowCount()

	if res == nil {
		var tmpValue uint32
		return tmpValue
	}
	return res.FollowerCnt
}

func (s *SoFollowCountWrap) MdFollowerCnt(p uint32) bool {
	if s.dba == nil {
		return false
	}
	sa := s.getFollowCount()
	if sa == nil {
		return false
	}

	sa.FollowerCnt = p
	if !s.update(sa) {
		return false
	}

	return true
}

func (s *SoFollowCountWrap) GetFollowingCnt() uint32 {
	res := s.getFollowCount()

	if res == nil {
		var tmpValue uint32
		return tmpValue
	}
	return res.FollowingCnt
}

func (s *SoFollowCountWrap) MdFollowingCnt(p uint32) bool {
	if s.dba == nil {
		return false
	}
	sa := s.getFollowCount()
	if sa == nil {
		return false
	}

	sa.FollowingCnt = p
	if !s.update(sa) {
		return false
	}

	return true
}

func (s *SoFollowCountWrap) GetUpdateTime() *prototype.TimePointSec {
	res := s.getFollowCount()

	if res == nil {
		return nil

	}
	return res.UpdateTime
}

func (s *SoFollowCountWrap) MdUpdateTime(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}
	sa := s.getFollowCount()
	if sa == nil {
		return false
	}

	sa.UpdateTime = p
	if !s.update(sa) {
		return false
	}

	return true
}

/////////////// SECTION Private function ////////////////

func (s *SoFollowCountWrap) update(sa *SoFollowCount) bool {
	if s.dba == nil {
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

func (s *SoFollowCountWrap) getFollowCount() *SoFollowCount {
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

	res := &SoFollowCount{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoFollowCountWrap) encodeMainKey() ([]byte, error) {
	pre := FollowCountTable
	sub := s.mainKey
	if sub == nil {
		return nil, errors.New("the mainKey is nil")
	}
	kList := []interface{}{pre, sub}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
}

////////////// Unique Query delete/insert/query ///////////////

func (s *SoFollowCountWrap) delUniKeyAccount(sa *SoFollowCount) bool {
	if s.dba == nil {
		return false
	}
	pre := FollowCountAccountUniTable
	sub := sa.Account
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoFollowCountWrap) insertUniKeyAccount(sa *SoFollowCount) bool {
	if s.dba == nil {
		return false
	}
	uniWrap := UniFollowCountAccountWrap{}
	uniWrap.Dba = s.dba

	res := uniWrap.UniQueryAccount(sa.Account)
	if res != nil {
		//the unique key is already exist
		return false
	}
	val := SoUniqueFollowCountByAccount{}
	val.Account = sa.Account

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	pre := FollowCountAccountUniTable
	sub := sa.Account
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniFollowCountAccountWrap struct {
	Dba iservices.IDatabaseService
}

func NewUniFollowCountAccountWrap(db iservices.IDatabaseService) *UniFollowCountAccountWrap {
	if db == nil {
		return nil
	}
	wrap := UniFollowCountAccountWrap{Dba: db}
	return &wrap
}

func (s *UniFollowCountAccountWrap) UniQueryAccount(start *prototype.AccountName) *SoFollowCountWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := FollowCountAccountUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueFollowCountByAccount{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoFollowCountWrap(s.Dba, res.Account)

			return wrap
		}
	}
	return nil
}
