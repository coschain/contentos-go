package table

import (
	"bytes"
	"errors"

	"github.com/coschain/contentos-go/common/encoding"
	"github.com/coschain/contentos-go/iservices"
	prototype "github.com/coschain/contentos-go/prototype"
	proto "github.com/golang/protobuf/proto"
)

////////////// SECTION Prefix Mark ///////////////
var (
	ExtFollowingTable                 = []byte("ExtFollowingTable")
	ExtFollowingFollowingInfoTable    = []byte("ExtFollowingFollowingInfoTable")
	ExtFollowingFollowingInfoUniTable = []byte("ExtFollowingFollowingInfoUniTable")
)

////////////// SECTION Wrap Define ///////////////
type SoExtFollowingWrap struct {
	dba     iservices.IDatabaseService
	mainKey *prototype.FollowingCreatedOrder
}

func NewSoExtFollowingWrap(dba iservices.IDatabaseService, key *prototype.FollowingCreatedOrder) *SoExtFollowingWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoExtFollowingWrap{dba, key}
	return result
}

func (s *SoExtFollowingWrap) CheckExist() bool {
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

func (s *SoExtFollowingWrap) Create(f func(tInfo *SoExtFollowing)) error {
	val := &SoExtFollowing{}
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

	if !s.insertSortKeyFollowingInfo(val) {
		return errors.New("insert sort Field FollowingInfo while insert table ")
	}

	//update unique list
	if !s.insertUniKeyFollowingInfo(val) {
		return errors.New("insert unique Field prototype.FollowingCreatedOrder while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoExtFollowingWrap) delSortKeyFollowingInfo(sa *SoExtFollowing) bool {
	val := SoListExtFollowingByFollowingInfo{}
	val.FollowingInfo = sa.FollowingInfo
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoExtFollowingWrap) insertSortKeyFollowingInfo(sa *SoExtFollowing) bool {
	val := SoListExtFollowingByFollowingInfo{}
	val.FollowingInfo = sa.FollowingInfo
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

////////////// SECTION LKeys delete/insert //////////////

func (s *SoExtFollowingWrap) RemoveExtFollowing() bool {
	sa := s.getExtFollowing()
	if sa == nil {
		return false
	}
	//delete sort list key
	if !s.delSortKeyFollowingInfo(sa) {
		return false
	}

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
func (s *SoExtFollowingWrap) GetFollowingInfo() *prototype.FollowingCreatedOrder {
	res := s.getExtFollowing()

	if res == nil {
		return nil

	}
	return res.FollowingInfo
}

////////////// SECTION List Keys ///////////////
type SExtFollowingFollowingInfoWrap struct {
	Dba iservices.IDatabaseService
}

func NewExtFollowingFollowingInfoWrap(db iservices.IDatabaseService) *SExtFollowingFollowingInfoWrap {
	if db == nil {
		return nil
	}
	wrap := SExtFollowingFollowingInfoWrap{Dba: db}
	return &wrap
}

func (s *SExtFollowingFollowingInfoWrap) DelIterater(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SExtFollowingFollowingInfoWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.FollowingCreatedOrder {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListExtFollowingByFollowingInfo{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.FollowingInfo

}

func (s *SExtFollowingFollowingInfoWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.FollowingCreatedOrder {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListExtFollowingByFollowingInfo{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.FollowingInfo

}

func (m *SoListExtFollowingByFollowingInfo) OpeEncode() ([]byte, error) {
	pre := ExtFollowingFollowingInfoTable
	sub := m.FollowingInfo
	if sub == nil {
		return nil, errors.New("the pro FollowingInfo is nil")
	}
	sub1 := m.FollowingInfo
	if sub1 == nil {
		return nil, errors.New("the mainkey FollowingInfo is nil")
	}
	kList := []interface{}{pre, sub, sub1}
	kBuf, cErr := encoding.EncodeSlice(kList, false)
	return kBuf, cErr
}

//Query sort by order
//start = nil  end = nil (query the db from start to end)
//start = nil (query from start the db)
//end = nil (query to the end of db)
func (s *SExtFollowingFollowingInfoWrap) QueryListByOrder(start *prototype.FollowingCreatedOrder, end *prototype.FollowingCreatedOrder) iservices.IDatabaseIterator {
	if s.Dba == nil {
		return nil
	}
	pre := ExtFollowingFollowingInfoTable
	skeyList := []interface{}{pre}
	if start != nil {
		skeyList = append(skeyList, start)
	}
	sBuf, cErr := encoding.EncodeSlice(skeyList, false)
	if cErr != nil {
		return nil
	}
	if start != nil && end == nil {
		iter := s.Dba.NewIterator(sBuf, nil)
		return iter
	}
	eKeyList := []interface{}{pre}
	if end != nil {
		eKeyList = append(eKeyList, end)
	}
	eBuf, cErr := encoding.EncodeSlice(eKeyList, false)
	if cErr != nil {
		return nil
	}

	res := bytes.Compare(sBuf, eBuf)
	if res == 0 {
		eBuf = nil
	} else if res == 1 {
		//reverse order
		return nil
	}
	iter := s.Dba.NewIterator(sBuf, eBuf)

	return iter
}

/////////////// SECTION Private function ////////////////

func (s *SoExtFollowingWrap) update(sa *SoExtFollowing) bool {
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

func (s *SoExtFollowingWrap) getExtFollowing() *SoExtFollowing {
	keyBuf, err := s.encodeMainKey()

	if err != nil {
		return nil
	}

	resBuf, err := s.dba.Get(keyBuf)

	if err != nil {
		return nil
	}

	res := &SoExtFollowing{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoExtFollowingWrap) encodeMainKey() ([]byte, error) {
	pre := ExtFollowingTable
	sub := s.mainKey
	if sub == nil {
		return nil, errors.New("the mainKey is nil")
	}
	kList := []interface{}{pre, sub}
	kBuf, cErr := encoding.EncodeSlice(kList, false)
	return kBuf, cErr
}

////////////// Unique Query delete/insert/query ///////////////

func (s *SoExtFollowingWrap) delUniKeyFollowingInfo(sa *SoExtFollowing) bool {
	pre := ExtFollowingFollowingInfoUniTable
	sub := sa.FollowingInfo
	kList := []interface{}{pre, sub}
	kBuf, err := encoding.EncodeSlice(kList, false)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoExtFollowingWrap) insertUniKeyFollowingInfo(sa *SoExtFollowing) bool {
	uniWrap := UniExtFollowingFollowingInfoWrap{}
	uniWrap.Dba = s.dba

	res := uniWrap.UniQueryFollowingInfo(sa.FollowingInfo)
	if res != nil {
		//the unique key is already exist
		return false
	}
	val := SoUniqueExtFollowingByFollowingInfo{}
	val.FollowingInfo = sa.FollowingInfo

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	pre := ExtFollowingFollowingInfoUniTable
	sub := sa.FollowingInfo
	kList := []interface{}{pre, sub}
	kBuf, err := encoding.EncodeSlice(kList, false)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniExtFollowingFollowingInfoWrap struct {
	Dba iservices.IDatabaseService
}

func NewUniExtFollowingFollowingInfoWrap(db iservices.IDatabaseService) *UniExtFollowingFollowingInfoWrap {
	if db == nil {
		return nil
	}
	wrap := UniExtFollowingFollowingInfoWrap{Dba: db}
	return &wrap
}

func (s *UniExtFollowingFollowingInfoWrap) UniQueryFollowingInfo(start *prototype.FollowingCreatedOrder) *SoExtFollowingWrap {
	if start == nil {
		return nil
	}
	pre := ExtFollowingFollowingInfoUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := encoding.EncodeSlice(kList, false)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueExtFollowingByFollowingInfo{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoExtFollowingWrap(s.Dba, res.FollowingInfo)

			return wrap
		}
	}
	return nil
}
