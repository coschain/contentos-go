

package table

import (
     "bytes"
     "errors"
     "github.com/coschain/contentos-go/common/encoding"
     "github.com/coschain/contentos-go/prototype"
	 "github.com/gogo/protobuf/proto"
     "github.com/coschain/contentos-go/iservices"
)

////////////// SECTION Prefix Mark ///////////////
var (
	PostTable        = []byte("PostTable")
    PostCreatedOrderTable = []byte("PostCreatedOrderTable")
    PostCreatedOrderRevOrdTable = []byte("PostCreatedOrderRevOrdTable")
    PostReplyOrderTable = []byte("PostReplyOrderTable")
    PostReplyOrderRevOrdTable = []byte("PostReplyOrderRevOrdTable")
    PostPostIdUniTable = []byte("PostPostIdUniTable")
    )

////////////// SECTION Wrap Define ///////////////
type SoPostWrap struct {
	dba 		iservices.IDatabaseService
	mainKey 	*uint64
}

func NewSoPostWrap(dba iservices.IDatabaseService, key *uint64) *SoPostWrap{
	result := &SoPostWrap{ dba, key}
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

func (s *SoPostWrap) CreatePost(f func(t *SoPost)) error {

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
	
	if !s.insertSortKeyCreatedOrder(val) {
		return err
	}
	
	if !s.insertSortKeyReplyOrder(val) {
		return err
	}
	
  
    //update unique list
    if !s.insertUniKeyPostId(val) {
		return err
	}
	
    
	return nil
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoPostWrap) delSortKeyCreatedOrder(sa *SoPost) bool {
	val := SoListPostByCreatedOrder{}
	val.CreatedOrder = sa.CreatedOrder
	val.PostId = sa.PostId
    subRevBuf, err := val.EncodeRevSortKey()
	if err != nil {
		return false
	}
    revOrdErr :=  s.dba.Delete(subRevBuf) 
    return revOrdErr == nil
    
}


func (s *SoPostWrap) insertSortKeyCreatedOrder(sa *SoPost) bool {
	val := SoListPostByCreatedOrder{}
    val.PostId = sa.PostId
    val.CreatedOrder = sa.CreatedOrder
	buf, err := proto.Marshal(&val)
	if err != nil {
		return false
	}
    subRevBuf, err := val.EncodeRevSortKey()
	if err != nil {
		return false
	}
    revOrdErr :=  s.dba.Put(subRevBuf, buf) 
    return revOrdErr == nil
    
}


func (s *SoPostWrap) delSortKeyReplyOrder(sa *SoPost) bool {
	val := SoListPostByReplyOrder{}
	val.ReplyOrder = sa.ReplyOrder
	val.PostId = sa.PostId
    subRevBuf, err := val.EncodeRevSortKey()
	if err != nil {
		return false
	}
    revOrdErr :=  s.dba.Delete(subRevBuf) 
    return revOrdErr == nil
    
}


func (s *SoPostWrap) insertSortKeyReplyOrder(sa *SoPost) bool {
	val := SoListPostByReplyOrder{}
    val.PostId = sa.PostId
    val.ReplyOrder = sa.ReplyOrder
	buf, err := proto.Marshal(&val)
	if err != nil {
		return false
	}
    subRevBuf, err := val.EncodeRevSortKey()
	if err != nil {
		return false
	}
    revOrdErr :=  s.dba.Put(subRevBuf, buf) 
    return revOrdErr == nil
    
}


////////////// SECTION LKeys delete/insert //////////////

func (s *SoPostWrap) RemovePost() error {
	sa := s.getPost()
	if sa == nil {
		return errors.New("delete data fail ")
	}
    //delete sort list key
	if !s.delSortKeyCreatedOrder(sa) {
		return errors.New("delete the sort key CreatedOrder fail")
	}
	if !s.delSortKeyReplyOrder(sa) {
		return errors.New("delete the sort key ReplyOrder fail")
	}
	
    //delete unique list
    if !s.delUniKeyPostId(sa) {
		return errors.New("delete the unique key PostId fail")
	}
	
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return err
	}
    if err := s.dba.Delete(keyBuf); err != nil {
       return err
    }
	return nil
}

////////////// SECTION Members Get/Modify ///////////////
func (s *SoPostWrap) GetActive(v **prototype.TimePointSec) error {
	res := s.getPost()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Active
   return nil
}



func (s *SoPostWrap) MdActive(p *prototype.TimePointSec) error {
	sa := s.getPost()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.Active = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoPostWrap) GetAllowReplies(v *bool) error {
	res := s.getPost()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.AllowReplies
   return nil
}



func (s *SoPostWrap) MdAllowReplies(p bool) error {
	sa := s.getPost()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.AllowReplies = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoPostWrap) GetAllowVotes(v *bool) error {
	res := s.getPost()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.AllowVotes
   return nil
}



func (s *SoPostWrap) MdAllowVotes(p bool) error {
	sa := s.getPost()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.AllowVotes = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoPostWrap) GetAuthor(v **prototype.AccountName) error {
	res := s.getPost()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Author
   return nil
}



func (s *SoPostWrap) MdAuthor(p *prototype.AccountName) error {
	sa := s.getPost()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.Author = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoPostWrap) GetBody(v *string) error {
	res := s.getPost()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Body
   return nil
}



func (s *SoPostWrap) MdBody(p string) error {
	sa := s.getPost()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.Body = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoPostWrap) GetCategory(v *string) error {
	res := s.getPost()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Category
   return nil
}



func (s *SoPostWrap) MdCategory(p string) error {
	sa := s.getPost()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.Category = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoPostWrap) GetChildren(v *uint32) error {
	res := s.getPost()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Children
   return nil
}



func (s *SoPostWrap) MdChildren(p uint32) error {
	sa := s.getPost()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.Children = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoPostWrap) GetCreated(v **prototype.TimePointSec) error {
	res := s.getPost()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Created
   return nil
}



func (s *SoPostWrap) MdCreated(p *prototype.TimePointSec) error {
	sa := s.getPost()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.Created = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoPostWrap) GetCreatedOrder(v **prototype.PostCreatedOrder) error {
	res := s.getPost()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.CreatedOrder
   return nil
}



func (s *SoPostWrap) MdCreatedOrder(p *prototype.PostCreatedOrder) error {
	sa := s.getPost()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
	if !s.delSortKeyCreatedOrder(sa) {
		return errors.New("delete the sort key CreatedOrder fail")
	}
    sa.CreatedOrder = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
    if !s.insertSortKeyCreatedOrder(sa) {
		return errors.New("reinsert sort key CreatedOrder fail")
    }
       
	return nil
}

func (s *SoPostWrap) GetDepth(v *uint32) error {
	res := s.getPost()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Depth
   return nil
}



func (s *SoPostWrap) MdDepth(p uint32) error {
	sa := s.getPost()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.Depth = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoPostWrap) GetJsonMetadata(v *string) error {
	res := s.getPost()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.JsonMetadata
   return nil
}



func (s *SoPostWrap) MdJsonMetadata(p string) error {
	sa := s.getPost()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.JsonMetadata = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoPostWrap) GetLastPayout(v **prototype.TimePointSec) error {
	res := s.getPost()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.LastPayout
   return nil
}



func (s *SoPostWrap) MdLastPayout(p *prototype.TimePointSec) error {
	sa := s.getPost()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.LastPayout = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoPostWrap) GetLastUpdate(v **prototype.TimePointSec) error {
	res := s.getPost()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.LastUpdate
   return nil
}



func (s *SoPostWrap) MdLastUpdate(p *prototype.TimePointSec) error {
	sa := s.getPost()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.LastUpdate = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoPostWrap) GetParentAuthor(v **prototype.AccountName) error {
	res := s.getPost()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.ParentAuthor
   return nil
}



func (s *SoPostWrap) MdParentAuthor(p *prototype.AccountName) error {
	sa := s.getPost()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.ParentAuthor = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoPostWrap) GetParentId(v *uint64) error {
	res := s.getPost()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.ParentId
   return nil
}



func (s *SoPostWrap) MdParentId(p uint64) error {
	sa := s.getPost()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.ParentId = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoPostWrap) GetParentPermlink(v *string) error {
	res := s.getPost()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.ParentPermlink
   return nil
}



func (s *SoPostWrap) MdParentPermlink(p string) error {
	sa := s.getPost()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.ParentPermlink = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoPostWrap) GetPermlink(v *string) error {
	res := s.getPost()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Permlink
   return nil
}



func (s *SoPostWrap) MdPermlink(p string) error {
	sa := s.getPost()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.Permlink = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoPostWrap) GetPostId(v *uint64) error {
	res := s.getPost()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.PostId
   return nil
}


func (s *SoPostWrap) GetReplyOrder(v **prototype.PostReplyOrder) error {
	res := s.getPost()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.ReplyOrder
   return nil
}



func (s *SoPostWrap) MdReplyOrder(p *prototype.PostReplyOrder) error {
	sa := s.getPost()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
	if !s.delSortKeyReplyOrder(sa) {
		return errors.New("delete the sort key ReplyOrder fail")
	}
    sa.ReplyOrder = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
    if !s.insertSortKeyReplyOrder(sa) {
		return errors.New("reinsert sort key ReplyOrder fail")
    }
       
	return nil
}

func (s *SoPostWrap) GetRootId(v *uint64) error {
	res := s.getPost()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.RootId
   return nil
}



func (s *SoPostWrap) MdRootId(p uint64) error {
	sa := s.getPost()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.RootId = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoPostWrap) GetTitle(v *string) error {
	res := s.getPost()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Title
   return nil
}



func (s *SoPostWrap) MdTitle(p string) error {
	sa := s.getPost()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.Title = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}




////////////// SECTION List Keys ///////////////
type SPostCreatedOrderWrap struct {
	Dba iservices.IDatabaseService
}

func (s *SPostCreatedOrderWrap)DelIterater(iterator iservices.IDatabaseIterator){
   if iterator == nil || !iterator.Valid() {
		return 
	}
   s.Dba.DeleteIterator(iterator)
}

func (s *SPostCreatedOrderWrap) GetMainVal(iterator iservices.IDatabaseIterator,mKey *uint64) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}
	val, err := iterator.Value()

	if err != nil {
		return errors.New("the value of iterator is nil")
	}

	res := &SoListPostByCreatedOrder{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return err
	}
    *mKey = res.PostId
    return nil
}

func (s *SPostCreatedOrderWrap) GetSubVal(iterator iservices.IDatabaseIterator, sub **prototype.PostCreatedOrder) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}

	val, err := iterator.Value()

	if err != nil {
		return errors.New("the value of iterator is nil")
	}
	res := &SoListPostByCreatedOrder{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return err
	}
    *sub = res.CreatedOrder
    return nil
}

func (m *SoListPostByCreatedOrder) OpeEncode() ([]byte,error) {
    pre := PostCreatedOrderTable
    sub := m.CreatedOrder
    if sub == nil {
       return nil,errors.New("the pro CreatedOrder is nil")
    }
    sub1 := m.PostId
    
    kList := []interface{}{pre,sub,sub1}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

func (m *SoListPostByCreatedOrder) EncodeRevSortKey() ([]byte,error) {
    pre := PostCreatedOrderRevOrdTable
    sub := m.CreatedOrder
    if sub == nil {
       return nil,errors.New("the pro CreatedOrder is nil")
    }
    sub1 := m.PostId
    
    kList := []interface{}{pre,sub,sub1}
    ordKey,cErr := encoding.EncodeSlice(kList,false)
    if cErr != nil {
       return nil,cErr
    }
    revKey,revRrr := encoding.Complement(ordKey, cErr)
    return revKey,revRrr
}


//Query sort by reverse order 
func (s *SPostCreatedOrderWrap) QueryListByRevOrder(start *prototype.PostCreatedOrder, end *prototype.PostCreatedOrder,iter *iservices.IDatabaseIterator) error {

    var sBuf,eBuf,rBufStart,rBufEnd []byte
    pre := PostCreatedOrderRevOrdTable
    if start != nil {
       skeyList := []interface{}{pre,start}
       buf,cErr := encoding.EncodeSlice(skeyList,false)
       if cErr != nil {
         return cErr
       }
       sBuf = buf
    }
    
    if end != nil {
       eKeyList := []interface{}{pre,end}
       buf,err := encoding.EncodeSlice(eKeyList,false)
       if err != nil {
          return err
       }
       eBuf = buf

    }

    if sBuf != nil && eBuf != nil {
       res := bytes.Compare(sBuf,eBuf)
       if res == -1 {
          // order
          return errors.New("the start and end are not reverse order")
       }
       if sBuf != nil {
       rBuf,rErr := encoding.Complement(sBuf, nil)
       if rErr != nil {
          return rErr
       }
       rBufStart = rBuf
    }
    if eBuf != nil {
          rBuf,rErr := encoding.Complement(eBuf, nil)
          if rErr != nil { 
            return rErr
          }
          rBufEnd = rBuf
       }
    }
    *iter = s.Dba.NewIterator(rBufStart, rBufEnd)
    return nil
}

////////////// SECTION List Keys ///////////////
type SPostReplyOrderWrap struct {
	Dba iservices.IDatabaseService
}

func (s *SPostReplyOrderWrap)DelIterater(iterator iservices.IDatabaseIterator){
   if iterator == nil || !iterator.Valid() {
		return 
	}
   s.Dba.DeleteIterator(iterator)
}

func (s *SPostReplyOrderWrap) GetMainVal(iterator iservices.IDatabaseIterator,mKey *uint64) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}
	val, err := iterator.Value()

	if err != nil {
		return errors.New("the value of iterator is nil")
	}

	res := &SoListPostByReplyOrder{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return err
	}
    *mKey = res.PostId
    return nil
}

func (s *SPostReplyOrderWrap) GetSubVal(iterator iservices.IDatabaseIterator, sub **prototype.PostReplyOrder) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}

	val, err := iterator.Value()

	if err != nil {
		return errors.New("the value of iterator is nil")
	}
	res := &SoListPostByReplyOrder{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return err
	}
    *sub = res.ReplyOrder
    return nil
}

func (m *SoListPostByReplyOrder) OpeEncode() ([]byte,error) {
    pre := PostReplyOrderTable
    sub := m.ReplyOrder
    if sub == nil {
       return nil,errors.New("the pro ReplyOrder is nil")
    }
    sub1 := m.PostId
    
    kList := []interface{}{pre,sub,sub1}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

func (m *SoListPostByReplyOrder) EncodeRevSortKey() ([]byte,error) {
    pre := PostReplyOrderRevOrdTable
    sub := m.ReplyOrder
    if sub == nil {
       return nil,errors.New("the pro ReplyOrder is nil")
    }
    sub1 := m.PostId
    
    kList := []interface{}{pre,sub,sub1}
    ordKey,cErr := encoding.EncodeSlice(kList,false)
    if cErr != nil {
       return nil,cErr
    }
    revKey,revRrr := encoding.Complement(ordKey, cErr)
    return revKey,revRrr
}


//Query sort by reverse order 
func (s *SPostReplyOrderWrap) QueryListByRevOrder(start *prototype.PostReplyOrder, end *prototype.PostReplyOrder,iter *iservices.IDatabaseIterator) error {

    var sBuf,eBuf,rBufStart,rBufEnd []byte
    pre := PostReplyOrderRevOrdTable
    if start != nil {
       skeyList := []interface{}{pre,start}
       buf,cErr := encoding.EncodeSlice(skeyList,false)
       if cErr != nil {
         return cErr
       }
       sBuf = buf
    }
    
    if end != nil {
       eKeyList := []interface{}{pre,end}
       buf,err := encoding.EncodeSlice(eKeyList,false)
       if err != nil {
          return err
       }
       eBuf = buf

    }

    if sBuf != nil && eBuf != nil {
       res := bytes.Compare(sBuf,eBuf)
       if res == -1 {
          // order
          return errors.New("the start and end are not reverse order")
       }
       if sBuf != nil {
       rBuf,rErr := encoding.Complement(sBuf, nil)
       if rErr != nil {
          return rErr
       }
       rBufStart = rBuf
    }
    if eBuf != nil {
          rBuf,rErr := encoding.Complement(eBuf, nil)
          if rErr != nil { 
            return rErr
          }
          rBufEnd = rBuf
       }
    }
    *iter = s.Dba.NewIterator(rBufStart, rBufEnd)
    return nil
}
/////////////// SECTION Private function ////////////////

func (s *SoPostWrap) update(sa *SoPost) error {
	buf, err := proto.Marshal(sa)
	if err != nil {
		return errors.New("initialization data failed")
	}

	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return err
	}
    pErr := s.dba.Put(keyBuf, buf)
    if pErr != nil {
       return pErr
    }
	return nil
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
       return nil,errors.New("the mainKey is nil")
    }
    kList := []interface{}{pre,sub}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

////////////// Unique Query delete/insert/query ///////////////


func (s *SoPostWrap) delUniKeyPostId(sa *SoPost) bool {
    pre := PostPostIdUniTable
    sub := sa.PostId
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}


func (s *SoPostWrap) insertUniKeyPostId(sa *SoPost) bool {
    uniWrap  := UniPostPostIdWrap{}
     uniWrap.Dba = s.dba
   res := uniWrap.UniQueryPostId(&sa.PostId,nil)
   
   if res == nil {
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
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniPostPostIdWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniPostPostIdWrap) UniQueryPostId(start *uint64,wrap *SoPostWrap) error{
    pre := PostPostIdUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniquePostByPostId{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap.mainKey = &res.PostId
            wrap.dba = s.Dba
			return nil  
		}
        return rErr
	}
    return err
}



