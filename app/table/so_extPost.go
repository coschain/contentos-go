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

	if !s.insertSortKeyCreatedOrder(val) {
		return errors.New("insert sort Field CreatedOrder while insert table ")
	}

	if !s.insertSortKeyReplyOrder(val) {
		return errors.New("insert sort Field ReplyOrder while insert table ")
	}

	//update unique list
	if !s.insertUniKeyPostId(val) {
		return errors.New("insert unique Field uint64 while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoExtPostWrap) delSortKeyCreatedOrder(sa *SoExtPost) bool {
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

////////////// SECTION LKeys delete/insert //////////////

func (s *SoExtPostWrap) RemoveExtPost() bool {
	sa := s.getExtPost()
	if sa == nil {
		return false
	}
	//delete sort list key
	if !s.delSortKeyCreatedOrder(sa) {
		return false
	}
	if !s.delSortKeyReplyOrder(sa) {
		return false
	}

	//delete unique list
	if !s.delUniKeyPostId(sa) {
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
	kBuf, cErr := encoding.EncodeSlice(kList, false)
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
	}
	sBuf, cErr := encoding.EncodeSlice(skeyList, false)
	if cErr != nil {
		return nil
	}
	eKeyList := []interface{}{pre}
	if end != nil {
		eKeyList = append(eKeyList, end)
	}
	eBuf, cErr := encoding.EncodeSlice(eKeyList, false)
	if cErr != nil {
		return nil
	}

	if start != nil && end != nil {
		res := bytes.Compare(sBuf, eBuf)
		if res == -1 {
			// order
			return nil
		} else if res == 0 {
			sBuf = nil
		}
	} else if start == nil {
		//query to the max data
		sBuf = nil
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
	kBuf, cErr := encoding.EncodeSlice(kList, false)
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
	}
	sBuf, cErr := encoding.EncodeSlice(skeyList, false)
	if cErr != nil {
		return nil
	}
	eKeyList := []interface{}{pre}
	if end != nil {
		eKeyList = append(eKeyList, end)
	}
	eBuf, cErr := encoding.EncodeSlice(eKeyList, false)
	if cErr != nil {
		return nil
	}

	if start != nil && end != nil {
		res := bytes.Compare(sBuf, eBuf)
		if res == -1 {
			// order
			return nil
		} else if res == 0 {
			sBuf = nil
		}
	} else if start == nil {
		//query to the max data
		sBuf = nil
	}
	//reverse the start and end when create ReversedIterator to query by reverse order
	iter := s.Dba.NewReversedIterator(eBuf, sBuf)
	return iter
}

/////////////// SECTION Private function ////////////////

func (s *SoExtPostWrap) update(sa *SoExtPost) bool {
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
	kBuf, cErr := encoding.EncodeSlice(kList, false)
	return kBuf, cErr
}

////////////// Unique Query delete/insert/query ///////////////

func (s *SoExtPostWrap) delUniKeyPostId(sa *SoExtPost) bool {
	pre := ExtPostPostIdUniTable
	sub := sa.PostId
	kList := []interface{}{pre, sub}
	kBuf, err := encoding.EncodeSlice(kList, false)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoExtPostWrap) insertUniKeyPostId(sa *SoExtPost) bool {
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
	kBuf, err := encoding.EncodeSlice(kList, false)
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
	if start == nil {
		return nil
	}
	pre := ExtPostPostIdUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := encoding.EncodeSlice(kList, false)
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
