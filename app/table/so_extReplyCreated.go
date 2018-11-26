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
	ExtReplyCreatedTable             = []byte("ExtReplyCreatedTable")
	ExtReplyCreatedCreatedOrderTable = []byte("ExtReplyCreatedCreatedOrderTable")
	ExtReplyCreatedPostIdUniTable    = []byte("ExtReplyCreatedPostIdUniTable")
)

////////////// SECTION Wrap Define ///////////////
type SoExtReplyCreatedWrap struct {
	dba     iservices.IDatabaseService
	mainKey *uint64
}

func NewSoExtReplyCreatedWrap(dba iservices.IDatabaseService, key *uint64) *SoExtReplyCreatedWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoExtReplyCreatedWrap{dba, key}
	return result
}

func (s *SoExtReplyCreatedWrap) CheckExist() bool {
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

func (s *SoExtReplyCreatedWrap) Create(f func(tInfo *SoExtReplyCreated)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoExtReplyCreated{}
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

func (s *SoExtReplyCreatedWrap) delSortKeyCreatedOrder(sa *SoExtReplyCreated) bool {
	if s.dba == nil {
		return false
	}
	val := SoListExtReplyCreatedByCreatedOrder{}
	val.CreatedOrder = sa.CreatedOrder
	val.PostId = sa.PostId
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoExtReplyCreatedWrap) insertSortKeyCreatedOrder(sa *SoExtReplyCreated) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListExtReplyCreatedByCreatedOrder{}
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

func (s *SoExtReplyCreatedWrap) delAllSortKeys(br bool, val *SoExtReplyCreated) bool {
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

func (s *SoExtReplyCreatedWrap) insertAllSortKeys(val *SoExtReplyCreated) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoExtReplyCreated fail ")
	}
	if !s.insertSortKeyCreatedOrder(val) {
		return errors.New("insert sort Field CreatedOrder fail while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoExtReplyCreatedWrap) RemoveExtReplyCreated() bool {
	if s.dba == nil {
		return false
	}
	val := s.getExtReplyCreated()
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
func (s *SoExtReplyCreatedWrap) GetCreatedOrder() *prototype.ReplyCreatedOrder {
	res := s.getExtReplyCreated()

	if res == nil {
		return nil

	}
	return res.CreatedOrder
}

func (s *SoExtReplyCreatedWrap) MdCreatedOrder(p *prototype.ReplyCreatedOrder) bool {
	if s.dba == nil {
		return false
	}
	sa := s.getExtReplyCreated()
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

func (s *SoExtReplyCreatedWrap) GetPostId() uint64 {
	res := s.getExtReplyCreated()

	if res == nil {
		var tmpValue uint64
		return tmpValue
	}
	return res.PostId
}

////////////// SECTION List Keys ///////////////
type SExtReplyCreatedCreatedOrderWrap struct {
	Dba iservices.IDatabaseService
}

func NewExtReplyCreatedCreatedOrderWrap(db iservices.IDatabaseService) *SExtReplyCreatedCreatedOrderWrap {
	if db == nil {
		return nil
	}
	wrap := SExtReplyCreatedCreatedOrderWrap{Dba: db}
	return &wrap
}

func (s *SExtReplyCreatedCreatedOrderWrap) DelIterater(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SExtReplyCreatedCreatedOrderWrap) GetMainVal(iterator iservices.IDatabaseIterator) *uint64 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListExtReplyCreatedByCreatedOrder{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return &res.PostId

}

func (s *SExtReplyCreatedCreatedOrderWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.ReplyCreatedOrder {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListExtReplyCreatedByCreatedOrder{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.CreatedOrder

}

func (m *SoListExtReplyCreatedByCreatedOrder) OpeEncode() ([]byte, error) {
	pre := ExtReplyCreatedCreatedOrderTable
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
func (s *SExtReplyCreatedCreatedOrderWrap) QueryListByRevOrder(start *prototype.ReplyCreatedOrder, end *prototype.ReplyCreatedOrder) iservices.IDatabaseIterator {
	if s.Dba == nil {
		return nil
	}
	pre := ExtReplyCreatedCreatedOrderTable
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

func (s *SoExtReplyCreatedWrap) update(sa *SoExtReplyCreated) bool {
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

func (s *SoExtReplyCreatedWrap) getExtReplyCreated() *SoExtReplyCreated {
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

	res := &SoExtReplyCreated{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoExtReplyCreatedWrap) encodeMainKey() ([]byte, error) {
	pre := ExtReplyCreatedTable
	sub := s.mainKey
	if sub == nil {
		return nil, errors.New("the mainKey is nil")
	}
	kList := []interface{}{pre, sub}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
}

////////////// Unique Query delete/insert/query ///////////////

func (s *SoExtReplyCreatedWrap) delAllUniKeys(br bool, val *SoExtReplyCreated) bool {
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

func (s *SoExtReplyCreatedWrap) insertAllUniKeys(val *SoExtReplyCreated) error {
	if s.dba == nil {
		return errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert uniuqe Field fail,get the SoExtReplyCreated fail ")
	}
	if !s.insertUniKeyPostId(val) {
		return errors.New("insert unique Field PostId fail while insert table ")
	}

	return nil
}

func (s *SoExtReplyCreatedWrap) delUniKeyPostId(sa *SoExtReplyCreated) bool {
	if s.dba == nil {
		return false
	}

	pre := ExtReplyCreatedPostIdUniTable
	sub := sa.PostId
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoExtReplyCreatedWrap) insertUniKeyPostId(sa *SoExtReplyCreated) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	uniWrap := UniExtReplyCreatedPostIdWrap{}
	uniWrap.Dba = s.dba
	res := uniWrap.UniQueryPostId(&sa.PostId)

	if res != nil {
		//the unique key is already exist
		return false
	}
	val := SoUniqueExtReplyCreatedByPostId{}
	val.PostId = sa.PostId

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	pre := ExtReplyCreatedPostIdUniTable
	sub := sa.PostId
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniExtReplyCreatedPostIdWrap struct {
	Dba iservices.IDatabaseService
}

func NewUniExtReplyCreatedPostIdWrap(db iservices.IDatabaseService) *UniExtReplyCreatedPostIdWrap {
	if db == nil {
		return nil
	}
	wrap := UniExtReplyCreatedPostIdWrap{Dba: db}
	return &wrap
}

func (s *UniExtReplyCreatedPostIdWrap) UniQueryPostId(start *uint64) *SoExtReplyCreatedWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := ExtReplyCreatedPostIdUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueExtReplyCreatedByPostId{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoExtReplyCreatedWrap(s.Dba, &res.PostId)
			return wrap
		}
	}
	return nil
}
