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
	ExtFollowerTable                = []byte("ExtFollowerTable")
	ExtFollowerFollowerInfoTable    = []byte("ExtFollowerFollowerInfoTable")
	ExtFollowerFollowerInfoUniTable = []byte("ExtFollowerFollowerInfoUniTable")
)

////////////// SECTION Wrap Define ///////////////
type SoExtFollowerWrap struct {
	dba     iservices.IDatabaseService
	mainKey *prototype.FollowerRelation
}

func NewSoExtFollowerWrap(dba iservices.IDatabaseService, key *prototype.FollowerRelation) *SoExtFollowerWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoExtFollowerWrap{dba, key}
	return result
}

func (s *SoExtFollowerWrap) CheckExist() bool {
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

func (s *SoExtFollowerWrap) Create(f func(tInfo *SoExtFollower)) error {
	val := &SoExtFollower{}
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

func (s *SoExtFollowerWrap) delSortKeyFollowerInfo(sa *SoExtFollower) bool {
	if s.dba == nil {
		return false
	}
	val := SoListExtFollowerByFollowerInfo{}
	val.FollowerInfo = sa.FollowerInfo
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoExtFollowerWrap) insertSortKeyFollowerInfo(sa *SoExtFollower) bool {
	if s.dba == nil {
		return false
	}
	val := SoListExtFollowerByFollowerInfo{}
	val.FollowerInfo = sa.FollowerInfo
	buf, err := proto.Marshal(&val)
	if err != nil {
		return false
	}
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Put(subBuf, buf)
	return ordErr == nil
}

func (s *SoExtFollowerWrap) delAllSortKeys(br bool, val *SoExtFollower) bool {
	if s.dba == nil {
		return false
	}
	if val == nil {
		return false
	}
	res := true
	if !s.delSortKeyFollowerInfo(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoExtFollowerWrap) insertAllSortKeys(val *SoExtFollower) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoExtFollower fail ")
	}
	if !s.insertSortKeyFollowerInfo(val) {
		return errors.New("insert sort Field FollowerInfo while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoExtFollowerWrap) RemoveExtFollower() bool {
	if s.dba == nil {
		return false
	}
	val := s.getExtFollower()
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
func (s *SoExtFollowerWrap) GetFollowerInfo() *prototype.FollowerRelation {
	res := s.getExtFollower()

	if res == nil {
		return nil

	}
	return res.FollowerInfo
}

////////////// SECTION List Keys ///////////////
type SExtFollowerFollowerInfoWrap struct {
	Dba iservices.IDatabaseService
}

func NewExtFollowerFollowerInfoWrap(db iservices.IDatabaseService) *SExtFollowerFollowerInfoWrap {
	if db == nil {
		return nil
	}
	wrap := SExtFollowerFollowerInfoWrap{Dba: db}
	return &wrap
}

func (s *SExtFollowerFollowerInfoWrap) DelIterater(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SExtFollowerFollowerInfoWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.FollowerRelation {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListExtFollowerByFollowerInfo{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.FollowerInfo

}

func (s *SExtFollowerFollowerInfoWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.FollowerRelation {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListExtFollowerByFollowerInfo{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.FollowerInfo

}

func (m *SoListExtFollowerByFollowerInfo) OpeEncode() ([]byte, error) {
	pre := ExtFollowerFollowerInfoTable
	sub := m.FollowerInfo
	if sub == nil {
		return nil, errors.New("the pro FollowerInfo is nil")
	}
	sub1 := m.FollowerInfo
	if sub1 == nil {
		return nil, errors.New("the mainkey FollowerInfo is nil")
	}
	kList := []interface{}{pre, sub, sub1}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
}

//Query sort by order
//start = nil  end = nil (query the db from start to end)
//start = nil (query from start the db)
//end = nil (query to the end of db)
func (s *SExtFollowerFollowerInfoWrap) QueryListByOrder(start *prototype.FollowerRelation, end *prototype.FollowerRelation) iservices.IDatabaseIterator {
	if s.Dba == nil {
		return nil
	}
	pre := ExtFollowerFollowerInfoTable
	skeyList := []interface{}{pre}
	if start != nil {
		skeyList = append(skeyList, start)
	}
	sBuf, cErr := kope.EncodeSlice(skeyList)
	if cErr != nil {
		return nil
	}
	eKeyList := []interface{}{pre}
	if end != nil {
		eKeyList = append(eKeyList, end)
	} else {
		eKeyList = append(eKeyList, kope.MaximumKey)
	}
	eBuf, cErr := kope.EncodeSlice(eKeyList)
	if cErr != nil {
		return nil
	}
	return s.Dba.NewIterator(sBuf, eBuf)
}

/////////////// SECTION Private function ////////////////

func (s *SoExtFollowerWrap) update(sa *SoExtFollower) bool {
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

func (s *SoExtFollowerWrap) getExtFollower() *SoExtFollower {
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

	res := &SoExtFollower{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoExtFollowerWrap) encodeMainKey() ([]byte, error) {
	pre := ExtFollowerTable
	sub := s.mainKey
	if sub == nil {
		return nil, errors.New("the mainKey is nil")
	}
	kList := []interface{}{pre, sub}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
}

////////////// Unique Query delete/insert/query ///////////////

func (s *SoExtFollowerWrap) delAllUniKeys(br bool, val *SoExtFollower) bool {
	if s.dba == nil {
		return false
	}
	if val == nil {
		return false
	}
	res := true
	if !s.delUniKeyFollowerInfo(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoExtFollowerWrap) insertAllUniKeys(val *SoExtFollower) error {
	if s.dba == nil {
		return errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert uniuqe Field fail,get the SoExtFollower fail ")
	}
	if !s.insertUniKeyFollowerInfo(val) {
		return errors.New("insert unique Field prototype.FollowerRelation while insert table ")
	}

	return nil
}

func (s *SoExtFollowerWrap) delUniKeyFollowerInfo(sa *SoExtFollower) bool {
	if s.dba == nil {
		return false
	}
	pre := ExtFollowerFollowerInfoUniTable
	sub := sa.FollowerInfo
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoExtFollowerWrap) insertUniKeyFollowerInfo(sa *SoExtFollower) bool {
	if s.dba == nil {
		return false
	}
	uniWrap := UniExtFollowerFollowerInfoWrap{}
	uniWrap.Dba = s.dba

	res := uniWrap.UniQueryFollowerInfo(sa.FollowerInfo)
	if res != nil {
		//the unique key is already exist
		return false
	}
	val := SoUniqueExtFollowerByFollowerInfo{}
	val.FollowerInfo = sa.FollowerInfo

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	pre := ExtFollowerFollowerInfoUniTable
	sub := sa.FollowerInfo
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniExtFollowerFollowerInfoWrap struct {
	Dba iservices.IDatabaseService
}

func NewUniExtFollowerFollowerInfoWrap(db iservices.IDatabaseService) *UniExtFollowerFollowerInfoWrap {
	if db == nil {
		return nil
	}
	wrap := UniExtFollowerFollowerInfoWrap{Dba: db}
	return &wrap
}

func (s *UniExtFollowerFollowerInfoWrap) UniQueryFollowerInfo(start *prototype.FollowerRelation) *SoExtFollowerWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := ExtFollowerFollowerInfoUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueExtFollowerByFollowerInfo{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoExtFollowerWrap(s.Dba, res.FollowerInfo)

			return wrap
		}
	}
	return nil
}
