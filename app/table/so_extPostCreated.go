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
	ExtPostCreatedTable             = []byte("ExtPostCreatedTable")
	ExtPostCreatedCreatedOrderTable = []byte("ExtPostCreatedCreatedOrderTable")
	ExtPostCreatedPostIdUniTable    = []byte("ExtPostCreatedPostIdUniTable")
)

////////////// SECTION Wrap Define ///////////////
type SoExtPostCreatedWrap struct {
	dba     iservices.IDatabaseService
	mainKey *uint64
}

func NewSoExtPostCreatedWrap(dba iservices.IDatabaseService, key *uint64) *SoExtPostCreatedWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoExtPostCreatedWrap{dba, key}
	return result
}

func (s *SoExtPostCreatedWrap) CheckExist() bool {
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

func (s *SoExtPostCreatedWrap) Create(f func(tInfo *SoExtPostCreated)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoExtPostCreated{}
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

func (s *SoExtPostCreatedWrap) delSortKeyCreatedOrder(sa *SoExtPostCreated) bool {
	if s.dba == nil {
		return false
	}
	val := SoListExtPostCreatedByCreatedOrder{}
	val.CreatedOrder = sa.CreatedOrder
	val.PostId = sa.PostId
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoExtPostCreatedWrap) insertSortKeyCreatedOrder(sa *SoExtPostCreated) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListExtPostCreatedByCreatedOrder{}
	val.PostId = sa.PostId
	val.CreatedOrder = sa.CreatedOrder
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

func (s *SoExtPostCreatedWrap) delAllSortKeys(br bool, val *SoExtPostCreated) bool {
	if s.dba == nil || val == nil {
		return false
	}
	res := true
	if !s.delSortKeyCreatedOrder(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoExtPostCreatedWrap) insertAllSortKeys(val *SoExtPostCreated) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoExtPostCreated fail ")
	}
	if !s.insertSortKeyCreatedOrder(val) {
		return errors.New("insert sort Field CreatedOrder fail while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoExtPostCreatedWrap) RemoveExtPostCreated() bool {
	if s.dba == nil {
		return false
	}
	val := s.getExtPostCreated()
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
func (s *SoExtPostCreatedWrap) GetCreatedOrder() *prototype.PostCreatedOrder {
	res := s.getExtPostCreated()

	if res == nil {
		return nil

	}
	return res.CreatedOrder
}

func (s *SoExtPostCreatedWrap) MdCreatedOrder(p *prototype.PostCreatedOrder) bool {
	if s.dba == nil {
		return false
	}
	sa := s.getExtPostCreated()
	if sa == nil {
		return false
	}

	if !s.delSortKeyCreatedOrder(sa) {
		return false
	}
	sa.CreatedOrder = p
	if !s.update(sa) {
		return false
	}

	if !s.insertSortKeyCreatedOrder(sa) {
		return false
	}

	return true
}

func (s *SoExtPostCreatedWrap) GetPostId() uint64 {
	res := s.getExtPostCreated()

	if res == nil {
		var tmpValue uint64
		return tmpValue
	}
	return res.PostId
}

////////////// SECTION List Keys ///////////////
type SExtPostCreatedCreatedOrderWrap struct {
	Dba iservices.IDatabaseService
}

func NewExtPostCreatedCreatedOrderWrap(db iservices.IDatabaseService) *SExtPostCreatedCreatedOrderWrap {
	if db == nil {
		return nil
	}
	wrap := SExtPostCreatedCreatedOrderWrap{Dba: db}
	return &wrap
}

func (s *SExtPostCreatedCreatedOrderWrap) DelIterater(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SExtPostCreatedCreatedOrderWrap) GetMainVal(iterator iservices.IDatabaseIterator) *uint64 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListExtPostCreatedByCreatedOrder{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return &res.PostId

}

func (s *SExtPostCreatedCreatedOrderWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.PostCreatedOrder {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListExtPostCreatedByCreatedOrder{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.CreatedOrder

}

func (m *SoListExtPostCreatedByCreatedOrder) OpeEncode() ([]byte, error) {
	pre := ExtPostCreatedCreatedOrderTable
	sub := m.CreatedOrder
	if sub == nil {
		return nil, errors.New("the pro CreatedOrder is nil")
	}
	sub1 := m.PostId

	kList := []interface{}{pre, sub, sub1}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
}

//Query sort by reverse order
func (s *SExtPostCreatedCreatedOrderWrap) QueryListByRevOrder(start *prototype.PostCreatedOrder, end *prototype.PostCreatedOrder) iservices.IDatabaseIterator {
	if s.Dba == nil {
		return nil
	}
	pre := ExtPostCreatedCreatedOrderTable
	skeyList := []interface{}{pre}
	if start != nil {
		skeyList = append(skeyList, start)
	} else {
		skeyList = append(skeyList, kope.MaximumKey)
	}
	sBuf, cErr := kope.EncodeSlice(skeyList)
	if cErr != nil {
		return nil
	}
	eKeyList := []interface{}{pre}
	if end != nil {
		eKeyList = append(eKeyList, end)
	}
	eBuf, cErr := kope.EncodeSlice(eKeyList)
	if cErr != nil {
		return nil
	}
	//reverse the start and end when create ReversedIterator to query by reverse order
	iter := s.Dba.NewReversedIterator(eBuf, sBuf)
	return iter
}

/////////////// SECTION Private function ////////////////

func (s *SoExtPostCreatedWrap) update(sa *SoExtPostCreated) bool {
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

func (s *SoExtPostCreatedWrap) getExtPostCreated() *SoExtPostCreated {
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

	res := &SoExtPostCreated{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoExtPostCreatedWrap) encodeMainKey() ([]byte, error) {
	pre := ExtPostCreatedTable
	sub := s.mainKey
	if sub == nil {
		return nil, errors.New("the mainKey is nil")
	}
	kList := []interface{}{pre, sub}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
}

////////////// Unique Query delete/insert/query ///////////////

func (s *SoExtPostCreatedWrap) delAllUniKeys(br bool, val *SoExtPostCreated) bool {
	if s.dba == nil || val == nil {
		return false
	}
	res := true
	if !s.delUniKeyPostId(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoExtPostCreatedWrap) insertAllUniKeys(val *SoExtPostCreated) error {
	if s.dba == nil {
		return errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert uniuqe Field fail,get the SoExtPostCreated fail ")
	}
	if !s.insertUniKeyPostId(val) {
		return errors.New("insert unique Field PostId fail while insert table ")
	}

	return nil
}

func (s *SoExtPostCreatedWrap) delUniKeyPostId(sa *SoExtPostCreated) bool {
	if s.dba == nil {
		return false
	}

	pre := ExtPostCreatedPostIdUniTable
	sub := sa.PostId
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoExtPostCreatedWrap) insertUniKeyPostId(sa *SoExtPostCreated) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	uniWrap := UniExtPostCreatedPostIdWrap{}
	uniWrap.Dba = s.dba
	res := uniWrap.UniQueryPostId(&sa.PostId)

	if res != nil {
		//the unique key is already exist
		return false
	}
	val := SoUniqueExtPostCreatedByPostId{}
	val.PostId = sa.PostId

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	pre := ExtPostCreatedPostIdUniTable
	sub := sa.PostId
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniExtPostCreatedPostIdWrap struct {
	Dba iservices.IDatabaseService
}

func NewUniExtPostCreatedPostIdWrap(db iservices.IDatabaseService) *UniExtPostCreatedPostIdWrap {
	if db == nil {
		return nil
	}
	wrap := UniExtPostCreatedPostIdWrap{Dba: db}
	return &wrap
}

func (s *UniExtPostCreatedPostIdWrap) UniQueryPostId(start *uint64) *SoExtPostCreatedWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := ExtPostCreatedPostIdUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueExtPostCreatedByPostId{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoExtPostCreatedWrap(s.Dba, &res.PostId)
			return wrap
		}
	}
	return nil
}
