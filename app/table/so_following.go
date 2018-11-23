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
	FollowingTable                 = []byte("FollowingTable")
	FollowingFollowingInfoUniTable = []byte("FollowingFollowingInfoUniTable")
)

////////////// SECTION Wrap Define ///////////////
type SoFollowingWrap struct {
	dba     iservices.IDatabaseService
	mainKey *prototype.FollowingRelation
}

func NewSoFollowingWrap(dba iservices.IDatabaseService, key *prototype.FollowingRelation) *SoFollowingWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoFollowingWrap{dba, key}
	return result
}

func (s *SoFollowingWrap) CheckExist() bool {
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

func (s *SoFollowingWrap) Create(f func(tInfo *SoFollowing)) error {
	val := &SoFollowing{}
	f(val)
	if val.FollowingInfo == nil {
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
	if !s.insertUniKeyFollowingInfo(val) {
		return errors.New("insert unique Field prototype.FollowingRelation while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert ///////////////

////////////// SECTION LKeys delete/insert //////////////

func (s *SoFollowingWrap) RemoveFollowing() bool {
	if s.dba == nil {
		return false
	}
	sa := s.getFollowing()
	if sa == nil {
		return false
	}
	//delete sort list key

	//delete unique list
	if !s.delUniKeyFollowingInfo(sa) {
		return false
	}

	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return false
	}
	return s.dba.Delete(keyBuf) == nil
}

////////////// SECTION Members Get/Modify ///////////////
func (s *SoFollowingWrap) GetCreatedTime() *prototype.TimePointSec {
	res := s.getFollowing()

	if res == nil {
		return nil

	}
	return res.CreatedTime
}

func (s *SoFollowingWrap) MdCreatedTime(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}
	sa := s.getFollowing()
	if sa == nil {
		return false
	}

	sa.CreatedTime = p
	if !s.update(sa) {
		return false
	}

	return true
}

func (s *SoFollowingWrap) GetFollowingInfo() *prototype.FollowingRelation {
	res := s.getFollowing()

	if res == nil {
		return nil

	}
	return res.FollowingInfo
}

/////////////// SECTION Private function ////////////////

func (s *SoFollowingWrap) update(sa *SoFollowing) bool {
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

func (s *SoFollowingWrap) getFollowing() *SoFollowing {
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

	res := &SoFollowing{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoFollowingWrap) encodeMainKey() ([]byte, error) {
	pre := FollowingTable
	sub := s.mainKey
	if sub == nil {
		return nil, errors.New("the mainKey is nil")
	}
	kList := []interface{}{pre, sub}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
}

////////////// Unique Query delete/insert/query ///////////////

func (s *SoFollowingWrap) delUniKeyFollowingInfo(sa *SoFollowing) bool {
	if s.dba == nil {
		return false
	}
	pre := FollowingFollowingInfoUniTable
	sub := sa.FollowingInfo
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoFollowingWrap) insertUniKeyFollowingInfo(sa *SoFollowing) bool {
	if s.dba == nil {
		return false
	}
	uniWrap := UniFollowingFollowingInfoWrap{}
	uniWrap.Dba = s.dba

	res := uniWrap.UniQueryFollowingInfo(sa.FollowingInfo)
	if res != nil {
		//the unique key is already exist
		return false
	}
	val := SoUniqueFollowingByFollowingInfo{}
	val.FollowingInfo = sa.FollowingInfo

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	pre := FollowingFollowingInfoUniTable
	sub := sa.FollowingInfo
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniFollowingFollowingInfoWrap struct {
	Dba iservices.IDatabaseService
}

func NewUniFollowingFollowingInfoWrap(db iservices.IDatabaseService) *UniFollowingFollowingInfoWrap {
	if db == nil {
		return nil
	}
	wrap := UniFollowingFollowingInfoWrap{Dba: db}
	return &wrap
}

func (s *UniFollowingFollowingInfoWrap) UniQueryFollowingInfo(start *prototype.FollowingRelation) *SoFollowingWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := FollowingFollowingInfoUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueFollowingByFollowingInfo{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoFollowingWrap(s.Dba, res.FollowingInfo)

			return wrap
		}
	}
	return nil
}
