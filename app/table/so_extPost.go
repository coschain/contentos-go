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
	ExtPostTable             = []byte("ExtPostTable")
	ExtPostCreatedOrderTable = []byte("ExtPostCreatedOrderTable")
	ExtPostReplyOrderTable   = []byte("ExtPostReplyOrderTable")
	ExtPostPostIdUniTable    = []byte("ExtPostPostIdUniTable")
)

////////////// SECTION Wrap Define ///////////////
type SoExtPostWrap struct {
	dba     iservices.IDatabaseService
	mainKey *uint64
}

func NewSoExtPostWrap(dba iservices.IDatabaseService, key *uint64) *SoExtPostWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoExtPostWrap{dba, key}
	return result
}

func (s *SoExtPostWrap) CheckExist() bool {
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

func (s *SoExtPostWrap) Create(f func(tInfo *SoExtPost)) error {
	val := &SoExtPost{}
	f(val)
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

func (s *SoExtPostWrap) delSortKeyCreatedOrder(sa *SoExtPost) bool {
	if s.dba == nil {
		return false
	}
	val := SoListExtPostByCreatedOrder{}
	val.CreatedOrder = sa.CreatedOrder
	val.PostId = sa.PostId
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoExtPostWrap) insertSortKeyCreatedOrder(sa *SoExtPost) bool {
	if s.dba == nil {
		return false
	}
	val := SoListExtPostByCreatedOrder{}
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

func (s *SoExtPostWrap) delSortKeyReplyOrder(sa *SoExtPost) bool {
	if s.dba == nil {
		return false
	}
	val := SoListExtPostByReplyOrder{}
	val.ReplyOrder = sa.ReplyOrder
	val.PostId = sa.PostId
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoExtPostWrap) insertSortKeyReplyOrder(sa *SoExtPost) bool {
	if s.dba == nil {
		return false
	}
	val := SoListExtPostByReplyOrder{}
	val.PostId = sa.PostId
	val.ReplyOrder = sa.ReplyOrder
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

func (s *SoExtPostWrap) delAllSortKeys(br bool, val *SoExtPost) bool {
	if s.dba == nil {
		return false
	}
	if val == nil {
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
	if !s.delSortKeyReplyOrder(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoExtPostWrap) insertAllSortKeys(val *SoExtPost) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoExtPost fail ")
	}
	if !s.insertSortKeyCreatedOrder(val) {
		return errors.New("insert sort Field CreatedOrder while insert table ")
	}
	if !s.insertSortKeyReplyOrder(val) {
		return errors.New("insert sort Field ReplyOrder while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoExtPostWrap) RemoveExtPost() bool {
	if s.dba == nil {
		return false
	}
	val := s.getExtPost()
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
func (s *SoExtPostWrap) GetCreatedOrder() *prototype.PostCreatedOrder {
	res := s.getExtPost()

	if res == nil {
		return nil

	}
	return res.CreatedOrder
}

func (s *SoExtPostWrap) MdCreatedOrder(p *prototype.PostCreatedOrder) bool {
	if s.dba == nil {
		return false
	}
	sa := s.getExtPost()
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

func (s *SoExtPostWrap) GetPostId() uint64 {
	res := s.getExtPost()

	if res == nil {
		var tmpValue uint64
		return tmpValue
	}
	return res.PostId
}

func (s *SoExtPostWrap) GetReplyOrder() *prototype.PostReplyOrder {
	res := s.getExtPost()

	if res == nil {
		return nil

	}
	return res.ReplyOrder
}

func (s *SoExtPostWrap) MdReplyOrder(p *prototype.PostReplyOrder) bool {
	if s.dba == nil {
		return false
	}
	sa := s.getExtPost()
	if sa == nil {
		return false
	}

	if !s.delSortKeyReplyOrder(sa) {
		return false
	}
	sa.ReplyOrder = p
	if !s.update(sa) {
		return false
	}

	if !s.insertSortKeyReplyOrder(sa) {
		return false
	}

	return true
}

////////////// SECTION List Keys ///////////////
type SExtPostCreatedOrderWrap struct {
	Dba iservices.IDatabaseService
}

func NewExtPostCreatedOrderWrap(db iservices.IDatabaseService) *SExtPostCreatedOrderWrap {
	if db == nil {
		return nil
	}
	wrap := SExtPostCreatedOrderWrap{Dba: db}
	return &wrap
}

func (s *SExtPostCreatedOrderWrap) DelIterater(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SExtPostCreatedOrderWrap) GetMainVal(iterator iservices.IDatabaseIterator) *uint64 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListExtPostByCreatedOrder{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return &res.PostId

}

func (s *SExtPostCreatedOrderWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.PostCreatedOrder {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListExtPostByCreatedOrder{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.CreatedOrder

}

func (m *SoListExtPostByCreatedOrder) OpeEncode() ([]byte, error) {
	pre := ExtPostCreatedOrderTable
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
func (s *SExtPostCreatedOrderWrap) QueryListByRevOrder(start *prototype.PostCreatedOrder, end *prototype.PostCreatedOrder) iservices.IDatabaseIterator {
	if s.Dba == nil {
		return nil
	}
	pre := ExtPostCreatedOrderTable
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

////////////// SECTION List Keys ///////////////
type SExtPostReplyOrderWrap struct {
	Dba iservices.IDatabaseService
}

func NewExtPostReplyOrderWrap(db iservices.IDatabaseService) *SExtPostReplyOrderWrap {
	if db == nil {
		return nil
	}
	wrap := SExtPostReplyOrderWrap{Dba: db}
	return &wrap
}

func (s *SExtPostReplyOrderWrap) DelIterater(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SExtPostReplyOrderWrap) GetMainVal(iterator iservices.IDatabaseIterator) *uint64 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListExtPostByReplyOrder{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return &res.PostId

}

func (s *SExtPostReplyOrderWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.PostReplyOrder {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListExtPostByReplyOrder{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.ReplyOrder

}

func (m *SoListExtPostByReplyOrder) OpeEncode() ([]byte, error) {
	pre := ExtPostReplyOrderTable
	sub := m.ReplyOrder
	if sub == nil {
		return nil, errors.New("the pro ReplyOrder is nil")
	}
	sub1 := m.PostId

	kList := []interface{}{pre, sub, sub1}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
}

//Query sort by reverse order
func (s *SExtPostReplyOrderWrap) QueryListByRevOrder(start *prototype.PostReplyOrder, end *prototype.PostReplyOrder) iservices.IDatabaseIterator {
	if s.Dba == nil {
		return nil
	}
	pre := ExtPostReplyOrderTable
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

func (s *SoExtPostWrap) update(sa *SoExtPost) bool {
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

func (s *SoExtPostWrap) getExtPost() *SoExtPost {
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

	res := &SoExtPost{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoExtPostWrap) encodeMainKey() ([]byte, error) {
	pre := ExtPostTable
	sub := s.mainKey
	if sub == nil {
		return nil, errors.New("the mainKey is nil")
	}
	kList := []interface{}{pre, sub}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
}

////////////// Unique Query delete/insert/query ///////////////

func (s *SoExtPostWrap) delAllUniKeys(br bool, val *SoExtPost) bool {
	if s.dba == nil {
		return false
	}
	if val == nil {
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

func (s *SoExtPostWrap) insertAllUniKeys(val *SoExtPost) error {
	if s.dba == nil {
		return errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert uniuqe Field fail,get the SoExtPost fail ")
	}
	if !s.insertUniKeyPostId(val) {
		return errors.New("insert unique Field uint64 while insert table ")
	}

	return nil
}

func (s *SoExtPostWrap) delUniKeyPostId(sa *SoExtPost) bool {
	if s.dba == nil {
		return false
	}
	pre := ExtPostPostIdUniTable
	sub := sa.PostId
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoExtPostWrap) insertUniKeyPostId(sa *SoExtPost) bool {
	if s.dba == nil {
		return false
	}
	uniWrap := UniExtPostPostIdWrap{}
	uniWrap.Dba = s.dba
	res := uniWrap.UniQueryPostId(&sa.PostId)

	if res != nil {
		//the unique key is already exist
		return false
	}
	val := SoUniqueExtPostByPostId{}
	val.PostId = sa.PostId

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	pre := ExtPostPostIdUniTable
	sub := sa.PostId
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniExtPostPostIdWrap struct {
	Dba iservices.IDatabaseService
}

func NewUniExtPostPostIdWrap(db iservices.IDatabaseService) *UniExtPostPostIdWrap {
	if db == nil {
		return nil
	}
	wrap := UniExtPostPostIdWrap{Dba: db}
	return &wrap
}

func (s *UniExtPostPostIdWrap) UniQueryPostId(start *uint64) *SoExtPostWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := ExtPostPostIdUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueExtPostByPostId{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoExtPostWrap(s.Dba, &res.PostId)
			return wrap
		}
	}
	return nil
}
