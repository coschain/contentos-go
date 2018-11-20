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
	PostTable             = []byte("PostTable")
	PostCreatedTable      = []byte("PostCreatedTable")
	PostCreatedOrderTable = []byte("PostCreatedOrderTable")
	PostReplyOrderTable   = []byte("PostReplyOrderTable")
	PostPostIdUniTable    = []byte("PostPostIdUniTable")
)

////////////// SECTION Wrap Define ///////////////
type SoPostWrap struct {
	dba     iservices.IDatabaseService
	mainKey *uint64
}

func NewSoPostWrap(dba iservices.IDatabaseService, key *uint64) *SoPostWrap {
	result := &SoPostWrap{dba, key}
	return result
}

func (s *SoPostWrap) CheckExist() bool {
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

func (s *SoPostWrap) Create(f func(tInfo *SoPost)) error {
	val := &SoPost{}
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

	if !s.insertSortKeyCreated(val) {
		return errors.New("insert sort Field Created while insert table ")
	}

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

func (s *SoPostWrap) delSortKeyCreated(sa *SoPost) bool {
	val := SoListPostByCreated{}
	val.Created = sa.Created
	val.PostId = sa.PostId
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoPostWrap) insertSortKeyCreated(sa *SoPost) bool {
	val := SoListPostByCreated{}
	val.PostId = sa.PostId
	val.Created = sa.Created
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

func (s *SoPostWrap) delSortKeyCreatedOrder(sa *SoPost) bool {
	val := SoListPostByCreatedOrder{}
	val.CreatedOrder = sa.CreatedOrder
	val.PostId = sa.PostId
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoPostWrap) insertSortKeyCreatedOrder(sa *SoPost) bool {
	val := SoListPostByCreatedOrder{}
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

func (s *SoPostWrap) delSortKeyReplyOrder(sa *SoPost) bool {
	val := SoListPostByReplyOrder{}
	val.ReplyOrder = sa.ReplyOrder
	val.PostId = sa.PostId
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoPostWrap) insertSortKeyReplyOrder(sa *SoPost) bool {
	val := SoListPostByReplyOrder{}
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

func (s *SoPostWrap) RemovePost() bool {
	sa := s.getPost()
	if sa == nil {
		return false
	}
	//delete sort list key
	if !s.delSortKeyCreated(sa) {
		return false
	}
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
func (s *SoPostWrap) GetAuthor() *prototype.AccountName {
	res := s.getPost()

	if res == nil {
		return nil

	}
	return res.Author
}

func (s *SoPostWrap) MdAuthor(p *prototype.AccountName) bool {
	sa := s.getPost()
	if sa == nil {
		return false
	}

	sa.Author = p
	if !s.update(sa) {
		return false
	}

	return true
}

func (s *SoPostWrap) GetBody() string {
	res := s.getPost()

	if res == nil {
		var tmpValue string
		return tmpValue
	}
	return res.Body
}

func (s *SoPostWrap) MdBody(p string) bool {
	sa := s.getPost()
	if sa == nil {
		return false
	}

	sa.Body = p
	if !s.update(sa) {
		return false
	}

	return true
}

func (s *SoPostWrap) GetCategory() string {
	res := s.getPost()

	if res == nil {
		var tmpValue string
		return tmpValue
	}
	return res.Category
}

func (s *SoPostWrap) MdCategory(p string) bool {
	sa := s.getPost()
	if sa == nil {
		return false
	}

	sa.Category = p
	if !s.update(sa) {
		return false
	}

	return true
}

func (s *SoPostWrap) GetChildren() uint32 {
	res := s.getPost()

	if res == nil {
		var tmpValue uint32
		return tmpValue
	}
	return res.Children
}

func (s *SoPostWrap) MdChildren(p uint32) bool {
	sa := s.getPost()
	if sa == nil {
		return false
	}

	sa.Children = p
	if !s.update(sa) {
		return false
	}

	return true
}

func (s *SoPostWrap) GetCreated() *prototype.TimePointSec {
	res := s.getPost()

	if res == nil {
		return nil

	}
	return res.Created
}

func (s *SoPostWrap) MdCreated(p *prototype.TimePointSec) bool {
	sa := s.getPost()
	if sa == nil {
		return false
	}

	if !s.delSortKeyCreated(sa) {
		return false
	}
	sa.Created = p
	if !s.update(sa) {
		return false
	}

	if !s.insertSortKeyCreated(sa) {
		return false
	}

	return true
}

func (s *SoPostWrap) GetCreatedOrder() *prototype.PostCreatedOrder {
	res := s.getPost()

	if res == nil {
		return nil

	}
	return res.CreatedOrder
}

func (s *SoPostWrap) MdCreatedOrder(p *prototype.PostCreatedOrder) bool {
	sa := s.getPost()
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

func (s *SoPostWrap) GetDepth() uint32 {
	res := s.getPost()

	if res == nil {
		var tmpValue uint32
		return tmpValue
	}
	return res.Depth
}

func (s *SoPostWrap) MdDepth(p uint32) bool {
	sa := s.getPost()
	if sa == nil {
		return false
	}

	sa.Depth = p
	if !s.update(sa) {
		return false
	}

	return true
}

func (s *SoPostWrap) GetLastPayout() *prototype.TimePointSec {
	res := s.getPost()

	if res == nil {
		return nil

	}
	return res.LastPayout
}

func (s *SoPostWrap) MdLastPayout(p *prototype.TimePointSec) bool {
	sa := s.getPost()
	if sa == nil {
		return false
	}

	sa.LastPayout = p
	if !s.update(sa) {
		return false
	}

	return true
}

func (s *SoPostWrap) GetParentId() uint64 {
	res := s.getPost()

	if res == nil {
		var tmpValue uint64
		return tmpValue
	}
	return res.ParentId
}

func (s *SoPostWrap) MdParentId(p uint64) bool {
	sa := s.getPost()
	if sa == nil {
		return false
	}

	sa.ParentId = p
	if !s.update(sa) {
		return false
	}

	return true
}

func (s *SoPostWrap) GetPostId() uint64 {
	res := s.getPost()

	if res == nil {
		var tmpValue uint64
		return tmpValue
	}
	return res.PostId
}

func (s *SoPostWrap) GetReplyOrder() *prototype.PostReplyOrder {
	res := s.getPost()

	if res == nil {
		return nil

	}
	return res.ReplyOrder
}

func (s *SoPostWrap) MdReplyOrder(p *prototype.PostReplyOrder) bool {
	sa := s.getPost()
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

func (s *SoPostWrap) GetRootId() uint64 {
	res := s.getPost()

	if res == nil {
		var tmpValue uint64
		return tmpValue
	}
	return res.RootId
}

func (s *SoPostWrap) MdRootId(p uint64) bool {
	sa := s.getPost()
	if sa == nil {
		return false
	}

	sa.RootId = p
	if !s.update(sa) {
		return false
	}

	return true
}

func (s *SoPostWrap) GetTags() []string {
	res := s.getPost()

	if res == nil {
		var tmpValue []string
		return tmpValue
	}
	return res.Tags
}

func (s *SoPostWrap) MdTags(p []string) bool {
	sa := s.getPost()
	if sa == nil {
		return false
	}

	sa.Tags = p
	if !s.update(sa) {
		return false
	}

	return true
}

func (s *SoPostWrap) GetTitle() string {
	res := s.getPost()

	if res == nil {
		var tmpValue string
		return tmpValue
	}
	return res.Title
}

func (s *SoPostWrap) MdTitle(p string) bool {
	sa := s.getPost()
	if sa == nil {
		return false
	}

	sa.Title = p
	if !s.update(sa) {
		return false
	}

	return true
}

func (s *SoPostWrap) GetVoteCnt() uint64 {
	res := s.getPost()

	if res == nil {
		var tmpValue uint64
		return tmpValue
	}
	return res.VoteCnt
}

func (s *SoPostWrap) MdVoteCnt(p uint64) bool {
	sa := s.getPost()
	if sa == nil {
		return false
	}

	sa.VoteCnt = p
	if !s.update(sa) {
		return false
	}

	return true
}

////////////// SECTION List Keys ///////////////
type SPostCreatedWrap struct {
	Dba iservices.IDatabaseService
}

func (s *SPostCreatedWrap) DelIterater(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SPostCreatedWrap) GetMainVal(iterator iservices.IDatabaseIterator) *uint64 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListPostByCreated{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return &res.PostId

}

func (s *SPostCreatedWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.TimePointSec {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListPostByCreated{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.Created

}

func (m *SoListPostByCreated) OpeEncode() ([]byte, error) {
	pre := PostCreatedTable
	sub := m.Created
	if sub == nil {
		return nil, errors.New("the pro Created is nil")
	}
	sub1 := m.PostId

	kList := []interface{}{pre, sub, sub1}
	kBuf, cErr := encoding.EncodeSlice(kList, false)
	return kBuf, cErr
}

//Query sort by order
//start = nil  end = nil (query the db from start to end)
//start = nil (query from start the db)
//end = nil (query to the end of db)
func (s *SPostCreatedWrap) QueryListByOrder(start *prototype.TimePointSec, end *prototype.TimePointSec) iservices.IDatabaseIterator {
	pre := PostCreatedTable
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

////////////// SECTION List Keys ///////////////
type SPostCreatedOrderWrap struct {
	Dba iservices.IDatabaseService
}

func (s *SPostCreatedOrderWrap) DelIterater(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SPostCreatedOrderWrap) GetMainVal(iterator iservices.IDatabaseIterator) *uint64 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListPostByCreatedOrder{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return &res.PostId

}

func (s *SPostCreatedOrderWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.PostCreatedOrder {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListPostByCreatedOrder{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.CreatedOrder

}

func (m *SoListPostByCreatedOrder) OpeEncode() ([]byte, error) {
	pre := PostCreatedOrderTable
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
func (s *SPostCreatedOrderWrap) QueryListByRevOrder(start *prototype.PostCreatedOrder, end *prototype.PostCreatedOrder) iservices.IDatabaseIterator {

	pre := PostCreatedOrderTable
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
type SPostReplyOrderWrap struct {
	Dba iservices.IDatabaseService
}

func (s *SPostReplyOrderWrap) DelIterater(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SPostReplyOrderWrap) GetMainVal(iterator iservices.IDatabaseIterator) *uint64 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListPostByReplyOrder{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return &res.PostId

}

func (s *SPostReplyOrderWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.PostReplyOrder {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListPostByReplyOrder{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.ReplyOrder

}

func (m *SoListPostByReplyOrder) OpeEncode() ([]byte, error) {
	pre := PostReplyOrderTable
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
func (s *SPostReplyOrderWrap) QueryListByRevOrder(start *prototype.PostReplyOrder, end *prototype.PostReplyOrder) iservices.IDatabaseIterator {

	pre := PostReplyOrderTable
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

func (s *SoPostWrap) update(sa *SoPost) bool {
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

func (s *SoPostWrap) getPost() *SoPost {
	keyBuf, err := s.encodeMainKey()

	if err != nil {
		return nil
	}

	resBuf, err := s.dba.Get(keyBuf)

	if err != nil {
		return nil
	}

	res := &SoPost{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoPostWrap) encodeMainKey() ([]byte, error) {
	pre := PostTable
	sub := s.mainKey
	if sub == nil {
		return nil, errors.New("the mainKey is nil")
	}
	kList := []interface{}{pre, sub}
	kBuf, cErr := encoding.EncodeSlice(kList, false)
	return kBuf, cErr
}

////////////// Unique Query delete/insert/query ///////////////

func (s *SoPostWrap) delUniKeyPostId(sa *SoPost) bool {
	pre := PostPostIdUniTable
	sub := sa.PostId
	kList := []interface{}{pre, sub}
	kBuf, err := encoding.EncodeSlice(kList, false)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoPostWrap) insertUniKeyPostId(sa *SoPost) bool {
	uniWrap := UniPostPostIdWrap{}
	uniWrap.Dba = s.dba
	res := uniWrap.UniQueryPostId(&sa.PostId)

	if res != nil {
		//the unique key is already exist
		return false
	}
	val := SoUniquePostByPostId{}
	val.PostId = sa.PostId

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	pre := PostPostIdUniTable
	sub := sa.PostId
	kList := []interface{}{pre, sub}
	kBuf, err := encoding.EncodeSlice(kList, false)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniPostPostIdWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniPostPostIdWrap) UniQueryPostId(start *uint64) *SoPostWrap {
	pre := PostPostIdUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := encoding.EncodeSlice(kList, false)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniquePostByPostId{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoPostWrap(s.Dba, &res.PostId)
			return wrap
		}
	}
	return nil
}
