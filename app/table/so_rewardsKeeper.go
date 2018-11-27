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
	RewardsKeeperTable      = []byte("RewardsKeeperTable")
	RewardsKeeperIdUniTable = []byte("RewardsKeeperIdUniTable")
)

////////////// SECTION Wrap Define ///////////////
type SoRewardsKeeperWrap struct {
	dba     iservices.IDatabaseService
	mainKey *int32
}

func NewSoRewardsKeeperWrap(dba iservices.IDatabaseService, key *int32) *SoRewardsKeeperWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoRewardsKeeperWrap{dba, key}
	return result
}

func (s *SoRewardsKeeperWrap) CheckExist() bool {
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

func (s *SoRewardsKeeperWrap) delAllSortKeys(br bool, val *SoRewardsKeeper) bool {
	if s.dba == nil || val == nil {
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
	val := s.getRewardsKeeper()
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
func (s *SoRewardsKeeperWrap) GetId() int32 {
	res := s.getRewardsKeeper()

	if res == nil {
		var tmpValue int32
		return tmpValue
	}
	return res.Id
}

func (s *SoRewardsKeeperWrap) GetKeeper() *prototype.InternalRewardsKeeper {
	res := s.getRewardsKeeper()

	if res == nil {
		return nil

	}
	return res.Keeper
}

func (s *SoRewardsKeeperWrap) MdKeeper(p *prototype.InternalRewardsKeeper) bool {
	if s.dba == nil {
		return false
	}
	sa := s.getRewardsKeeper()
	if sa == nil {
		return false
	}

	sa.Keeper = p
	if !s.update(sa) {
		return false
	}

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
	pre := RewardsKeeperTable
	sub := s.mainKey
	if sub == nil {
		return nil, errors.New("the mainKey is nil")
	}
	kList := []interface{}{pre, sub}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
}

////////////// Unique Query delete/insert/query ///////////////

func (s *SoRewardsKeeperWrap) delAllUniKeys(br bool, val *SoRewardsKeeper) bool {
	if s.dba == nil || val == nil {
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

func (s *SoRewardsKeeperWrap) insertAllUniKeys(val *SoRewardsKeeper) error {
	if s.dba == nil {
		return errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert uniuqe Field fail,get the SoRewardsKeeper fail ")
	}
	if !s.insertUniKeyId(val) {
		return errors.New("insert unique Field Id fail while insert table ")
	}

	return nil
}

func (s *SoRewardsKeeperWrap) delUniKeyId(sa *SoRewardsKeeper) bool {
	if s.dba == nil {
		return false
	}

	pre := RewardsKeeperIdUniTable
	sub := sa.Id
	kList := []interface{}{pre, sub}
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
	uniWrap := UniRewardsKeeperIdWrap{}
	uniWrap.Dba = s.dba
	res := uniWrap.UniQueryId(&sa.Id)

	if res != nil {
		//the unique key is already exist
		return false
	}
	val := SoUniqueRewardsKeeperById{}
	val.Id = sa.Id

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	pre := RewardsKeeperIdUniTable
	sub := sa.Id
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
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
