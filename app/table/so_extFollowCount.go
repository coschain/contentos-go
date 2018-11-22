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
	ExtFollowCountTable           = []byte("ExtFollowCountTable")
	ExtFollowCountAccountUniTable = []byte("ExtFollowCountAccountUniTable")
)

////////////// SECTION Wrap Define ///////////////
type SoExtFollowCountWrap struct {
	dba     iservices.IDatabaseService
	mainKey *prototype.AccountName
}

func NewSoExtFollowCountWrap(dba iservices.IDatabaseService, key *prototype.AccountName) *SoExtFollowCountWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoExtFollowCountWrap{dba, key}
	return result
}

func (s *SoExtFollowCountWrap) CheckExist() bool {
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

func (s *SoExtFollowCountWrap) Create(f func(tInfo *SoExtFollowCount)) error {
	val := &SoExtFollowCount{}
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

func (s *SoExtFollowCountWrap) RemoveExtFollowCount() bool {
	sa := s.getExtFollowCount()
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
func (s *SoExtFollowCountWrap) GetAccount() *prototype.AccountName {
	res := s.getExtFollowCount()

	if res == nil {
		return nil

	}
	return res.Account
}

func (s *SoExtFollowCountWrap) GetFollowerCnt() uint32 {
	res := s.getExtFollowCount()

	if res == nil {
		var tmpValue uint32
		return tmpValue
	}
	return res.FollowerCnt
}

func (s *SoExtFollowCountWrap) MdFollowerCnt(p uint32) bool {
	sa := s.getExtFollowCount()
	if sa == nil {
		return false
	}

	sa.FollowerCnt = p
	if !s.update(sa) {
		return false
	}

	return true
}

func (s *SoExtFollowCountWrap) GetFollowingCnt() uint32 {
	res := s.getExtFollowCount()

	if res == nil {
		var tmpValue uint32
		return tmpValue
	}
	return res.FollowingCnt
}

func (s *SoExtFollowCountWrap) MdFollowingCnt(p uint32) bool {
	sa := s.getExtFollowCount()
	if sa == nil {
		return false
	}

	sa.FollowingCnt = p
	if !s.update(sa) {
		return false
	}

	return true
}

func (s *SoExtFollowCountWrap) GetUpdateTime() *prototype.TimePointSec {
	res := s.getExtFollowCount()

	if res == nil {
		return nil

	}
	return res.UpdateTime
}

func (s *SoExtFollowCountWrap) MdUpdateTime(p *prototype.TimePointSec) bool {
	sa := s.getExtFollowCount()
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

func (s *SoExtFollowCountWrap) update(sa *SoExtFollowCount) bool {
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
	pre := ExtFollowCountTable
	sub := s.mainKey
	if sub == nil {
		return nil, errors.New("the mainKey is nil")
	}
	kList := []interface{}{pre, sub}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
}

////////////// Unique Query delete/insert/query ///////////////

func (s *SoExtFollowCountWrap) delUniKeyAccount(sa *SoExtFollowCount) bool {
	pre := ExtFollowCountAccountUniTable
	sub := sa.Account
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoExtFollowCountWrap) insertUniKeyAccount(sa *SoExtFollowCount) bool {
	uniWrap := UniExtFollowCountAccountWrap{}
	uniWrap.Dba = s.dba

	res := uniWrap.UniQueryAccount(sa.Account)
	if res != nil {
		//the unique key is already exist
		return false
	}
	val := SoUniqueExtFollowCountByAccount{}
	val.Account = sa.Account

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	pre := ExtFollowCountAccountUniTable
	sub := sa.Account
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
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
	if start == nil {
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
