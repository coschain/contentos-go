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
	FollowerTable                = []byte("FollowerTable")
	FollowerFollowerInfoUniTable = []byte("FollowerFollowerInfoUniTable")
)

////////////// SECTION Wrap Define ///////////////
type SoFollowerWrap struct {
	dba     iservices.IDatabaseService
	mainKey *prototype.FollowerRelation
}

func NewSoFollowerWrap(dba iservices.IDatabaseService, key *prototype.FollowerRelation) *SoFollowerWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoFollowerWrap{dba, key}
	return result
}

func (s *SoFollowerWrap) CheckExist() bool {
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

func (s *SoFollowerWrap) Create(f func(tInfo *SoFollower)) error {
	val := &SoFollower{}
	f(val)
	if val.FollowerInfo == nil {
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
	if !s.insertUniKeyFollowerInfo(val) {
		return errors.New("insert unique Field prototype.FollowerRelation while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert ///////////////

////////////// SECTION LKeys delete/insert //////////////

func (s *SoFollowerWrap) RemoveFollower() bool {
	sa := s.getFollower()
	if sa == nil {
		return false
	}
	//delete sort list key

	//delete unique list
	if !s.delUniKeyFollowerInfo(sa) {
		return false
	}

	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return false
	}
	return s.dba.Delete(keyBuf) == nil
}

////////////// SECTION Members Get/Modify ///////////////
func (s *SoFollowerWrap) GetCreatedTime() *prototype.TimePointSec {
	res := s.getFollower()

	if res == nil {
		return nil

	}
	return res.CreatedTime
}

func (s *SoFollowerWrap) MdCreatedTime(p *prototype.TimePointSec) bool {
	sa := s.getFollower()
	if sa == nil {
		return false
	}

	sa.CreatedTime = p
	if !s.update(sa) {
		return false
	}

	return true
}

func (s *SoFollowerWrap) GetFollowerInfo() *prototype.FollowerRelation {
	res := s.getFollower()

	if res == nil {
		return nil

	}
	return res.FollowerInfo
}

/////////////// SECTION Private function ////////////////

func (s *SoFollowerWrap) update(sa *SoFollower) bool {
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

func (s *SoFollowerWrap) getFollower() *SoFollower {
	keyBuf, err := s.encodeMainKey()

	if err != nil {
		return nil
	}

	resBuf, err := s.dba.Get(keyBuf)

	if err != nil {
		return nil
	}

	res := &SoFollower{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoFollowerWrap) encodeMainKey() ([]byte, error) {
	pre := FollowerTable
	sub := s.mainKey
	if sub == nil {
		return nil, errors.New("the mainKey is nil")
	}
	kList := []interface{}{pre, sub}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
}

////////////// Unique Query delete/insert/query ///////////////

func (s *SoFollowerWrap) delUniKeyFollowerInfo(sa *SoFollower) bool {
	pre := FollowerFollowerInfoUniTable
	sub := sa.FollowerInfo
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoFollowerWrap) insertUniKeyFollowerInfo(sa *SoFollower) bool {
	uniWrap := UniFollowerFollowerInfoWrap{}
	uniWrap.Dba = s.dba

	res := uniWrap.UniQueryFollowerInfo(sa.FollowerInfo)
	if res != nil {
		//the unique key is already exist
		return false
	}
	val := SoUniqueFollowerByFollowerInfo{}
	val.FollowerInfo = sa.FollowerInfo

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	pre := FollowerFollowerInfoUniTable
	sub := sa.FollowerInfo
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniFollowerFollowerInfoWrap struct {
	Dba iservices.IDatabaseService
}

func NewUniFollowerFollowerInfoWrap(db iservices.IDatabaseService) *UniFollowerFollowerInfoWrap {
	if db == nil {
		return nil
	}
	wrap := UniFollowerFollowerInfoWrap{Dba: db}
	return &wrap
}

func (s *UniFollowerFollowerInfoWrap) UniQueryFollowerInfo(start *prototype.FollowerRelation) *SoFollowerWrap {
	if start == nil {
		return nil
	}
	pre := FollowerFollowerInfoUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueFollowerByFollowerInfo{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoFollowerWrap(s.Dba, res.FollowerInfo)

			return wrap
		}
	}
	return nil
}
