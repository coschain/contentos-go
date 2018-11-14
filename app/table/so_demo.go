

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
	DemoTable        = []byte("DemoTable")
    DemoOwnerTable = []byte("DemoOwnerTable")
    DemoOwnerRevOrdTable = []byte("DemoOwnerRevOrdTable")
    DemoPostTimeTable = []byte("DemoPostTimeTable")
    DemoPostTimeRevOrdTable = []byte("DemoPostTimeRevOrdTable")
    DemoLikeCountTable = []byte("DemoLikeCountTable")
    DemoLikeCountRevOrdTable = []byte("DemoLikeCountRevOrdTable")
    DemoIdxTable = []byte("DemoIdxTable")
    DemoIdxRevOrdTable = []byte("DemoIdxRevOrdTable")
    DemoReplayCountTable = []byte("DemoReplayCountTable")
    DemoReplayCountRevOrdTable = []byte("DemoReplayCountRevOrdTable")
    DemoIdxUniTable = []byte("DemoIdxUniTable")
    DemoLikeCountUniTable = []byte("DemoLikeCountUniTable")
    DemoOwnerUniTable = []byte("DemoOwnerUniTable")
    )

////////////// SECTION Wrap Define ///////////////
type SoDemoWrap struct {
	dba 		iservices.IDatabaseService
	mainKey 	*prototype.AccountName
}

func NewSoDemoWrap(dba iservices.IDatabaseService, key *prototype.AccountName) *SoDemoWrap{
	result := &SoDemoWrap{ dba, key}
	return result
}

func (s *SoDemoWrap) CheckExist() bool {
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

func (s *SoDemoWrap) CreateDemo(sa *SoDemo) bool {

	if sa == nil {
		return false
	}
    if s.CheckExist() {
       return false
    }
	keyBuf, err := s.encodeMainKey()

	if err != nil {
		return false
	}
	resBuf, err := proto.Marshal(sa)
	if err != nil {
		return false
	}
	err = s.dba.Put(keyBuf, resBuf)
	if err != nil {
		return false
	}

	// update sort list keys
	
	if !s.insertSortKeyOwner(sa) {
		return false
	}
	
	if !s.insertSortKeyPostTime(sa) {
		return false
	}
	
	if !s.insertSortKeyLikeCount(sa) {
		return false
	}
	
	if !s.insertSortKeyIdx(sa) {
		return false
	}
	
	if !s.insertSortKeyReplayCount(sa) {
		return false
	}
	
  
    //update unique list
    if !s.insertUniKeyIdx(sa) {
		return false
	}
	if !s.insertUniKeyLikeCount(sa) {
		return false
	}
	if !s.insertUniKeyOwner(sa) {
		return false
	}
	
    
	return true
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoDemoWrap) delSortKeyOwner(sa *SoDemo) bool {
	val := SoListDemoByOwner{}
	val.Owner = sa.Owner
	val.Owner = sa.Owner
    subRevBuf, err := val.EncodeRevSortKey()
	if err != nil {
		return false
	}
    revOrdErr :=  s.dba.Delete(subRevBuf) 
    return revOrdErr == nil
    
}


func (s *SoDemoWrap) insertSortKeyOwner(sa *SoDemo) bool {
	val := SoListDemoByOwner{}
	val.Owner = sa.Owner
	val.Owner = sa.Owner
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


func (s *SoDemoWrap) delSortKeyPostTime(sa *SoDemo) bool {
	val := SoListDemoByPostTime{}
	val.PostTime = sa.PostTime
	val.Owner = sa.Owner
    subBuf, err := val.OpeEncode()
    var ordErr,revOrdErr error
	if err == nil {
       ordKey := append(DemoPostTimeTable, subBuf...)
       ordErr =  s.dba.Delete(ordKey) 
	}
    subRevBuf, err := val.EncodeRevSortKey()
	if err == nil {
		revOrdKey := append(DemoPostTimeRevOrdTable, subRevBuf...)
        revOrdErr =  s.dba.Delete(revOrdKey) 
	}
    if ordErr == nil && revOrdErr == nil {
       return true
    }
    return false
    
}


func (s *SoDemoWrap) insertSortKeyPostTime(sa *SoDemo) bool {
	val := SoListDemoByPostTime{}
	val.Owner = sa.Owner
	val.PostTime = sa.PostTime
	buf, err := proto.Marshal(&val)
	if err != nil {
		return false
	}
    subBuf, err := val.OpeEncode()
    var ordErr,revOrdErr error
	if err == nil {
       ordErr =  s.dba.Put(subBuf, buf) 
	}
    subRevBuf, err := val.EncodeRevSortKey()
	if err == nil {
        revOrdErr =  s.dba.Put(subRevBuf, buf) 
	}
    if ordErr == nil && revOrdErr == nil {
       return true
    }
    return false
    
}


func (s *SoDemoWrap) delSortKeyLikeCount(sa *SoDemo) bool {
	val := SoListDemoByLikeCount{}
	val.LikeCount = sa.LikeCount
	val.Owner = sa.Owner
    subRevBuf, err := val.EncodeRevSortKey()
	if err != nil {
		return false
	}
    revOrdErr :=  s.dba.Delete(subRevBuf) 
    return revOrdErr == nil
    
}


func (s *SoDemoWrap) insertSortKeyLikeCount(sa *SoDemo) bool {
	val := SoListDemoByLikeCount{}
	val.Owner = sa.Owner
	val.LikeCount = sa.LikeCount
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


func (s *SoDemoWrap) delSortKeyIdx(sa *SoDemo) bool {
	val := SoListDemoByIdx{}
	val.Idx = sa.Idx
	val.Owner = sa.Owner
    subRevBuf, err := val.EncodeRevSortKey()
	if err != nil {
		return false
	}
    revOrdErr :=  s.dba.Delete(subRevBuf) 
    return revOrdErr == nil
    
}


func (s *SoDemoWrap) insertSortKeyIdx(sa *SoDemo) bool {
	val := SoListDemoByIdx{}
	val.Owner = sa.Owner
	val.Idx = sa.Idx
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


func (s *SoDemoWrap) delSortKeyReplayCount(sa *SoDemo) bool {
	val := SoListDemoByReplayCount{}
	val.ReplayCount = sa.ReplayCount
	val.Owner = sa.Owner
    subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
    ordErr :=  s.dba.Delete(subBuf)
    return ordErr == nil
    
}


func (s *SoDemoWrap) insertSortKeyReplayCount(sa *SoDemo) bool {
	val := SoListDemoByReplayCount{}
	val.Owner = sa.Owner
	val.ReplayCount = sa.ReplayCount
	buf, err := proto.Marshal(&val)
	if err != nil {
		return false
	}
    subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
    ordErr :=  s.dba.Put(subBuf, buf) 
    return ordErr == nil
    
}


////////////// SECTION LKeys delete/insert //////////////

func (s *SoDemoWrap) RemoveDemo() bool {
	sa := s.getDemo()
	if sa == nil {
		return false
	}
    //delete sort list key
	if !s.delSortKeyOwner(sa) {
		return false
	}
	if !s.delSortKeyPostTime(sa) {
		return false
	}
	if !s.delSortKeyLikeCount(sa) {
		return false
	}
	if !s.delSortKeyIdx(sa) {
		return false
	}
	if !s.delSortKeyReplayCount(sa) {
		return false
	}
	
    //delete unique list
    if !s.delUniKeyIdx(sa) {
		return false
	}
	if !s.delUniKeyLikeCount(sa) {
		return false
	}
	if !s.delUniKeyOwner(sa) {
		return false
	}
	
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return false
	}
	return s.dba.Delete(keyBuf) == nil
}

////////////// SECTION Members Get/Modify ///////////////
func (s *SoDemoWrap) GetContent() string {
	res := s.getDemo()

   if res == nil {
      var tmpValue string 
      return tmpValue
   }
   return res.Content
}



func (s *SoDemoWrap) MdContent(p string) bool {
	sa := s.getDemo()
	if sa == nil {
		return false
	}
	
   sa.Content = p
   
   
	if !s.update(sa) {
		return false
	}
    
	return true
}

func (s *SoDemoWrap) GetIdx() int64 {
	res := s.getDemo()

   if res == nil {
      var tmpValue int64 
      return tmpValue
   }
   return res.Idx
}



func (s *SoDemoWrap) MdIdx(p int64) bool {
	sa := s.getDemo()
	if sa == nil {
		return false
	}
    //judge the unique value if is exist
    uniWrap  := UniDemoIdxWrap{}
   res := uniWrap.UniQueryIdx(&sa.Idx)
	if res != nil {
		//the unique value to be modified is already exist
		return false
	}
	if !s.delUniKeyIdx(sa) {
		return false
	}
    
	
	if !s.delSortKeyIdx(sa) {
		return false
	}
   sa.Idx = p
   
   
	if !s.update(sa) {
		return false
	}
    
    if !s.insertSortKeyIdx(sa) {
		return false
    }
       
    if !s.insertUniKeyIdx(sa) {
		return false
    }
	return true
}

func (s *SoDemoWrap) GetLikeCount() int64 {
	res := s.getDemo()

   if res == nil {
      var tmpValue int64 
      return tmpValue
   }
   return res.LikeCount
}



func (s *SoDemoWrap) MdLikeCount(p int64) bool {
	sa := s.getDemo()
	if sa == nil {
		return false
	}
    //judge the unique value if is exist
    uniWrap  := UniDemoLikeCountWrap{}
   res := uniWrap.UniQueryLikeCount(&sa.LikeCount)
	if res != nil {
		//the unique value to be modified is already exist
		return false
	}
	if !s.delUniKeyLikeCount(sa) {
		return false
	}
    
	
	if !s.delSortKeyLikeCount(sa) {
		return false
	}
   sa.LikeCount = p
   
   
	if !s.update(sa) {
		return false
	}
    
    if !s.insertSortKeyLikeCount(sa) {
		return false
    }
       
    if !s.insertUniKeyLikeCount(sa) {
		return false
    }
	return true
}

func (s *SoDemoWrap) GetOwner() *prototype.AccountName {
	res := s.getDemo()

   if res == nil {
      return nil
      
   }
   return res.Owner
}


func (s *SoDemoWrap) GetPostTime() *prototype.TimePointSec {
	res := s.getDemo()

   if res == nil {
      return nil
      
   }
   return res.PostTime
}



func (s *SoDemoWrap) MdPostTime(p prototype.TimePointSec) bool {
	sa := s.getDemo()
	if sa == nil {
		return false
	}
	
	if !s.delSortKeyPostTime(sa) {
		return false
	}
   
   sa.PostTime = &p
   
	if !s.update(sa) {
		return false
	}
    
    if !s.insertSortKeyPostTime(sa) {
		return false
    }
       
	return true
}

func (s *SoDemoWrap) GetReplayCount() int64 {
	res := s.getDemo()

   if res == nil {
      var tmpValue int64 
      return tmpValue
   }
   return res.ReplayCount
}



func (s *SoDemoWrap) MdReplayCount(p int64) bool {
	sa := s.getDemo()
	if sa == nil {
		return false
	}
	
	if !s.delSortKeyReplayCount(sa) {
		return false
	}
   sa.ReplayCount = p
   
   
	if !s.update(sa) {
		return false
	}
    
    if !s.insertSortKeyReplayCount(sa) {
		return false
    }
       
	return true
}

func (s *SoDemoWrap) GetTaglist() string {
	res := s.getDemo()

   if res == nil {
      var tmpValue string 
      return tmpValue
   }
   return res.Taglist
}



func (s *SoDemoWrap) MdTaglist(p string) bool {
	sa := s.getDemo()
	if sa == nil {
		return false
	}
	
   sa.Taglist = p
   
   
	if !s.update(sa) {
		return false
	}
    
	return true
}

func (s *SoDemoWrap) GetTitle() string {
	res := s.getDemo()

   if res == nil {
      var tmpValue string 
      return tmpValue
   }
   return res.Title
}



func (s *SoDemoWrap) MdTitle(p string) bool {
	sa := s.getDemo()
	if sa == nil {
		return false
	}
	
   sa.Title = p
   
   
	if !s.update(sa) {
		return false
	}
    
	return true
}




////////////// SECTION List Keys ///////////////
type SDemoOwnerWrap struct {
	Dba iservices.IDatabaseService
}

func (s *SDemoOwnerWrap)DelIterater(iterator iservices.IDatabaseIterator){
   if iterator == nil || !iterator.Valid() {
		return 
	}
   s.Dba.DeleteIterator(iterator)
}

func (s *SDemoOwnerWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListDemoByOwner{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    
   return res.Owner
   

}

func (s *SDemoOwnerWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListDemoByOwner{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    
   
    
   return res.Owner
   
}

func (m *SoListDemoByOwner) OpeEncode() ([]byte,error) {
    pre := DemoOwnerTable
    sub := m.Owner
    if sub == nil {
       return nil,errors.New("the pro Owner is nil")
    }
    sub1 := m.Owner
    if sub1 == nil {
       return nil,errors.New("the mainKey Owner is nil")
    }
    kList := []interface{}{pre,sub,sub1}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

func (m *SoListDemoByOwner) EncodeRevSortKey() ([]byte,error) {
    pre := DemoOwnerRevOrdTable
    sub := m.Owner
    if sub == nil {
       return nil,errors.New("the pro Owner is nil")
    }
    sub1 := m.Owner
    if sub1 == nil {
       return nil,errors.New("the mainKey Owner is nil")
    }
    kList := []interface{}{pre,sub,sub1}
    ordKey,cErr := encoding.EncodeSlice(kList,false)
    if cErr != nil {
       return nil,cErr
    }
    revKey,revRrr := encoding.Complement(ordKey, cErr)
    return revKey,revRrr
}


//Query sort by reverse order 
func (s *SDemoOwnerWrap) QueryListByRevOrder(start *prototype.AccountName, end *prototype.AccountName) iservices.IDatabaseIterator {

    var sBuf,eBuf,rBufStart,rBufEnd []byte
    pre := DemoOwnerRevOrdTable
    if start != nil {
       skeyList := []interface{}{pre,start}
       buf,cErr := encoding.EncodeSlice(skeyList,false)
       if cErr != nil {
         return nil
       }
       sBuf = buf
    }
    
    if end != nil {
       eKeyList := []interface{}{pre,end}
       buf,err := encoding.EncodeSlice(eKeyList,false)
       if err != nil {
          return nil
       }
       eBuf = buf

    }

    if sBuf != nil && eBuf != nil {
       res := bytes.Compare(sBuf,eBuf)
       if res == -1 {
          // order
          return nil
       }
       if sBuf != nil {
       rBuf,rErr := encoding.Complement(sBuf, nil)
       if rErr != nil {
          return nil
       }
       rBufStart = rBuf
    }
       if eBuf != nil {
          rBuf,rErr := encoding.Complement(eBuf, nil)
          if rErr != nil { 
            return nil
          }
          rBufEnd = rBuf
       }
    }
     
       if sBuf != nil && eBuf != nil {
          res := bytes.Compare(sBuf,eBuf)
          if res == -1 {
            // order
            return nil
        }
    }
    iter := s.Dba.NewIterator(rBufStart, rBufEnd)
    return iter
}

////////////// SECTION List Keys ///////////////
type SDemoPostTimeWrap struct {
	Dba iservices.IDatabaseService
}

func (s *SDemoPostTimeWrap)DelIterater(iterator iservices.IDatabaseIterator){
   if iterator == nil || !iterator.Valid() {
		return 
	}
   s.Dba.DeleteIterator(iterator)
}

func (s *SDemoPostTimeWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListDemoByPostTime{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    
   return res.Owner
   

}

func (s *SDemoPostTimeWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.TimePointSec {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListDemoByPostTime{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    
   
    
   return res.PostTime
   
}

func (m *SoListDemoByPostTime) OpeEncode() ([]byte,error) {
    pre := DemoPostTimeTable
    sub := m.PostTime
    if sub == nil {
       return nil,errors.New("the pro PostTime is nil")
    }
    sub1 := m.Owner
    if sub1 == nil {
       return nil,errors.New("the mainKey PostTime is nil")
    }
    kList := []interface{}{pre,sub,sub1}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

func (m *SoListDemoByPostTime) EncodeRevSortKey() ([]byte,error) {
    pre := DemoPostTimeRevOrdTable
    sub := m.PostTime
    if sub == nil {
       return nil,errors.New("the pro PostTime is nil")
    }
    sub1 := m.Owner
    if sub1 == nil {
       return nil,errors.New("the mainKey PostTime is nil")
    }
    kList := []interface{}{pre,sub,sub1}
    ordKey,cErr := encoding.EncodeSlice(kList,false)
    if cErr != nil {
       return nil,cErr
    }
    revKey,revRrr := encoding.Complement(ordKey, cErr)
    return revKey,revRrr
}

//Query sort by order 
//start = nil  end = nil (query the db from start to end)
//start = nil (query from start the db)
//end = nil (query to the end of db)
func (s *SDemoPostTimeWrap) QueryListByOrder(start *prototype.TimePointSec, end *prototype.TimePointSec) iservices.IDatabaseIterator {
    pre := DemoPostTimeTable
    skeyList := []interface{}{pre}
    if start != nil {
       skeyList = append(skeyList,start)
    }
    sBuf,cErr := encoding.EncodeSlice(skeyList,false)
    if cErr != nil {
         return nil
    }
    if start != nil && end == nil {
		iter := s.Dba.NewIterator(sBuf, nil)
		return iter
	}
    eKeyList := []interface{}{pre}
    if end != nil {
       eKeyList = append(eKeyList,end)
    }
    eBuf,cErr := encoding.EncodeSlice(eKeyList,false)
    if cErr != nil {
       return nil
    }
    
    res := bytes.Compare(sBuf,eBuf)
    if res == 0 {
		eBuf = nil
	}else if res == 1 {
       //reverse order
       return nil
    }
    iter := s.Dba.NewIterator(sBuf, eBuf)
    
    return iter
}

//Query sort by reverse order 
func (s *SDemoPostTimeWrap) QueryListByRevOrder(start *prototype.TimePointSec, end *prototype.TimePointSec) iservices.IDatabaseIterator {

    var sBuf,eBuf,rBufStart,rBufEnd []byte
    pre := DemoPostTimeRevOrdTable
    if start != nil {
       skeyList := []interface{}{pre,start}
       buf,cErr := encoding.EncodeSlice(skeyList,false)
       if cErr != nil {
         return nil
       }
       sBuf = buf
    }
    
    if end != nil {
       eKeyList := []interface{}{pre,end}
       buf,err := encoding.EncodeSlice(eKeyList,false)
       if err != nil {
          return nil
       }
       eBuf = buf

    }

    if sBuf != nil && eBuf != nil {
       res := bytes.Compare(sBuf,eBuf)
       if res == -1 {
          // order
          return nil
       }
       if sBuf != nil {
       rBuf,rErr := encoding.Complement(sBuf, nil)
       if rErr != nil {
          return nil
       }
       rBufStart = rBuf
    }
       if eBuf != nil {
          rBuf,rErr := encoding.Complement(eBuf, nil)
          if rErr != nil { 
            return nil
          }
          rBufEnd = rBuf
       }
    }
     
       if sBuf != nil && eBuf != nil {
          res := bytes.Compare(sBuf,eBuf)
          if res == -1 {
            // order
            return nil
        }
    }
    iter := s.Dba.NewIterator(rBufStart, rBufEnd)
    return iter
}

////////////// SECTION List Keys ///////////////
type SDemoLikeCountWrap struct {
	Dba iservices.IDatabaseService
}

func (s *SDemoLikeCountWrap)DelIterater(iterator iservices.IDatabaseIterator){
   if iterator == nil || !iterator.Valid() {
		return 
	}
   s.Dba.DeleteIterator(iterator)
}

func (s *SDemoLikeCountWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListDemoByLikeCount{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    
   return res.Owner
   

}

func (s *SDemoLikeCountWrap) GetSubVal(iterator iservices.IDatabaseIterator) *int64 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListDemoByLikeCount{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    
    
     return &res.LikeCount
   
   
}

func (m *SoListDemoByLikeCount) OpeEncode() ([]byte,error) {
    pre := DemoLikeCountTable
    sub := m.LikeCount
    
    sub1 := m.Owner
    if sub1 == nil {
       return nil,errors.New("the mainKey LikeCount is nil")
    }
    kList := []interface{}{pre,sub,sub1}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

func (m *SoListDemoByLikeCount) EncodeRevSortKey() ([]byte,error) {
    pre := DemoLikeCountRevOrdTable
    sub := m.LikeCount
    
    sub1 := m.Owner
    if sub1 == nil {
       return nil,errors.New("the mainKey LikeCount is nil")
    }
    kList := []interface{}{pre,sub,sub1}
    ordKey,cErr := encoding.EncodeSlice(kList,false)
    if cErr != nil {
       return nil,cErr
    }
    revKey,revRrr := encoding.Complement(ordKey, cErr)
    return revKey,revRrr
}


//Query sort by reverse order 
func (s *SDemoLikeCountWrap) QueryListByRevOrder(start *int64, end *int64) iservices.IDatabaseIterator {

    var sBuf,eBuf,rBufStart,rBufEnd []byte
    pre := DemoLikeCountRevOrdTable
    if start != nil {
       skeyList := []interface{}{pre,start}
       buf,cErr := encoding.EncodeSlice(skeyList,false)
       if cErr != nil {
         return nil
       }
       sBuf = buf
    }
    
    if end != nil {
       eKeyList := []interface{}{pre,end}
       buf,err := encoding.EncodeSlice(eKeyList,false)
       if err != nil {
          return nil
       }
       eBuf = buf

    }

    if sBuf != nil && eBuf != nil {
       res := bytes.Compare(sBuf,eBuf)
       if res == -1 {
          // order
          return nil
       }
       if sBuf != nil {
       rBuf,rErr := encoding.Complement(sBuf, nil)
       if rErr != nil {
          return nil
       }
       rBufStart = rBuf
    }
       if eBuf != nil {
          rBuf,rErr := encoding.Complement(eBuf, nil)
          if rErr != nil { 
            return nil
          }
          rBufEnd = rBuf
       }
    }
     
       if sBuf != nil && eBuf != nil {
          res := bytes.Compare(sBuf,eBuf)
          if res == -1 {
            // order
            return nil
        }
    }
    iter := s.Dba.NewIterator(rBufStart, rBufEnd)
    return iter
}

////////////// SECTION List Keys ///////////////
type SDemoIdxWrap struct {
	Dba iservices.IDatabaseService
}

func (s *SDemoIdxWrap)DelIterater(iterator iservices.IDatabaseIterator){
   if iterator == nil || !iterator.Valid() {
		return 
	}
   s.Dba.DeleteIterator(iterator)
}

func (s *SDemoIdxWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListDemoByIdx{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    
   return res.Owner
   

}

func (s *SDemoIdxWrap) GetSubVal(iterator iservices.IDatabaseIterator) *int64 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListDemoByIdx{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    
    
     return &res.Idx
   
   
}

func (m *SoListDemoByIdx) OpeEncode() ([]byte,error) {
    pre := DemoIdxTable
    sub := m.Idx
    
    sub1 := m.Owner
    if sub1 == nil {
       return nil,errors.New("the mainKey Idx is nil")
    }
    kList := []interface{}{pre,sub,sub1}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

func (m *SoListDemoByIdx) EncodeRevSortKey() ([]byte,error) {
    pre := DemoIdxRevOrdTable
    sub := m.Idx
    
    sub1 := m.Owner
    if sub1 == nil {
       return nil,errors.New("the mainKey Idx is nil")
    }
    kList := []interface{}{pre,sub,sub1}
    ordKey,cErr := encoding.EncodeSlice(kList,false)
    if cErr != nil {
       return nil,cErr
    }
    revKey,revRrr := encoding.Complement(ordKey, cErr)
    return revKey,revRrr
}


//Query sort by reverse order 
func (s *SDemoIdxWrap) QueryListByRevOrder(start *int64, end *int64) iservices.IDatabaseIterator {

    var sBuf,eBuf,rBufStart,rBufEnd []byte
    pre := DemoIdxRevOrdTable
    if start != nil {
       skeyList := []interface{}{pre,start}
       buf,cErr := encoding.EncodeSlice(skeyList,false)
       if cErr != nil {
         return nil
       }
       sBuf = buf
    }
    
    if end != nil {
       eKeyList := []interface{}{pre,end}
       buf,err := encoding.EncodeSlice(eKeyList,false)
       if err != nil {
          return nil
       }
       eBuf = buf

    }

    if sBuf != nil && eBuf != nil {
       res := bytes.Compare(sBuf,eBuf)
       if res == -1 {
          // order
          return nil
       }
       if sBuf != nil {
       rBuf,rErr := encoding.Complement(sBuf, nil)
       if rErr != nil {
          return nil
       }
       rBufStart = rBuf
    }
       if eBuf != nil {
          rBuf,rErr := encoding.Complement(eBuf, nil)
          if rErr != nil { 
            return nil
          }
          rBufEnd = rBuf
       }
    }
     
       if sBuf != nil && eBuf != nil {
          res := bytes.Compare(sBuf,eBuf)
          if res == -1 {
            // order
            return nil
        }
    }
    iter := s.Dba.NewIterator(rBufStart, rBufEnd)
    return iter
}

////////////// SECTION List Keys ///////////////
type SDemoReplayCountWrap struct {
	Dba iservices.IDatabaseService
}

func (s *SDemoReplayCountWrap)DelIterater(iterator iservices.IDatabaseIterator){
   if iterator == nil || !iterator.Valid() {
		return 
	}
   s.Dba.DeleteIterator(iterator)
}

func (s *SDemoReplayCountWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListDemoByReplayCount{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    
   return res.Owner
   

}

func (s *SDemoReplayCountWrap) GetSubVal(iterator iservices.IDatabaseIterator) *int64 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListDemoByReplayCount{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    
    
     return &res.ReplayCount
   
   
}

func (m *SoListDemoByReplayCount) OpeEncode() ([]byte,error) {
    pre := DemoReplayCountTable
    sub := m.ReplayCount
    
    sub1 := m.Owner
    if sub1 == nil {
       return nil,errors.New("the mainKey ReplayCount is nil")
    }
    kList := []interface{}{pre,sub,sub1}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

func (m *SoListDemoByReplayCount) EncodeRevSortKey() ([]byte,error) {
    pre := DemoReplayCountRevOrdTable
    sub := m.ReplayCount
    
    sub1 := m.Owner
    if sub1 == nil {
       return nil,errors.New("the mainKey ReplayCount is nil")
    }
    kList := []interface{}{pre,sub,sub1}
    ordKey,cErr := encoding.EncodeSlice(kList,false)
    if cErr != nil {
       return nil,cErr
    }
    revKey,revRrr := encoding.Complement(ordKey, cErr)
    return revKey,revRrr
}

//Query sort by order 
//start = nil  end = nil (query the db from start to end)
//start = nil (query from start the db)
//end = nil (query to the end of db)
func (s *SDemoReplayCountWrap) QueryListByOrder(start *int64, end *int64) iservices.IDatabaseIterator {
    pre := DemoReplayCountTable
    skeyList := []interface{}{pre}
    if start != nil {
       skeyList = append(skeyList,start)
    }
    sBuf,cErr := encoding.EncodeSlice(skeyList,false)
    if cErr != nil {
         return nil
    }
    if start != nil && end == nil {
		iter := s.Dba.NewIterator(sBuf, nil)
		return iter
	}
    eKeyList := []interface{}{pre}
    if end != nil {
       eKeyList = append(eKeyList,end)
    }
    eBuf,cErr := encoding.EncodeSlice(eKeyList,false)
    if cErr != nil {
       return nil
    }
    
    res := bytes.Compare(sBuf,eBuf)
    if res == 0 {
		eBuf = nil
	}else if res == 1 {
       //reverse order
       return nil
    }
    iter := s.Dba.NewIterator(sBuf, eBuf)
    
    return iter
}

/////////////// SECTION Private function ////////////////

func (s *SoDemoWrap) update(sa *SoDemo) bool {
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

func (s *SoDemoWrap) getDemo() *SoDemo {
	keyBuf, err := s.encodeMainKey()

	if err != nil {
		return nil
	}

	resBuf, err := s.dba.Get(keyBuf)

	if err != nil {
		return nil
	}

	res := &SoDemo{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoDemoWrap) encodeMainKey() ([]byte, error) {
    pre := DemoTable
    sub := s.mainKey
    if sub == nil {
       return nil,errors.New("the mainKey is nil")
    }
    kList := []interface{}{pre,sub}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

////////////// Unique Query delete/insert/query ///////////////


func (s *SoDemoWrap) delUniKeyIdx(sa *SoDemo) bool {
    pre := DemoIdxUniTable
    sub := sa.Idx
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}


func (s *SoDemoWrap) insertUniKeyIdx(sa *SoDemo) bool {
    uniWrap  := UniDemoIdxWrap{}
     uniWrap.Dba = s.dba
   res := uniWrap.UniQueryIdx(&sa.Idx)
   
   if res != nil {
		//the unique key is already exist
		return false
	}
    val := SoUniqueDemoByIdx{}
    val.Owner = sa.Owner
    val.Idx = sa.Idx
    
	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}
    
    pre := DemoIdxUniTable
    sub := sa.Idx
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniDemoIdxWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniDemoIdxWrap) UniQueryIdx(start *int64) *SoDemoWrap{
    pre := DemoIdxUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueDemoByIdx{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoDemoWrap(s.Dba,res.Owner)
            
			return wrap
		}
	}
    return nil
}



func (s *SoDemoWrap) delUniKeyLikeCount(sa *SoDemo) bool {
    pre := DemoLikeCountUniTable
    sub := sa.LikeCount
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}


func (s *SoDemoWrap) insertUniKeyLikeCount(sa *SoDemo) bool {
    uniWrap  := UniDemoLikeCountWrap{}
     uniWrap.Dba = s.dba
   res := uniWrap.UniQueryLikeCount(&sa.LikeCount)
   
   if res != nil {
		//the unique key is already exist
		return false
	}
    val := SoUniqueDemoByLikeCount{}
    val.Owner = sa.Owner
    val.LikeCount = sa.LikeCount
    
	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}
    
    pre := DemoLikeCountUniTable
    sub := sa.LikeCount
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniDemoLikeCountWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniDemoLikeCountWrap) UniQueryLikeCount(start *int64) *SoDemoWrap{
    pre := DemoLikeCountUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueDemoByLikeCount{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoDemoWrap(s.Dba,res.Owner)
            
			return wrap
		}
	}
    return nil
}



func (s *SoDemoWrap) delUniKeyOwner(sa *SoDemo) bool {
    pre := DemoOwnerUniTable
    sub := sa.Owner
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}


func (s *SoDemoWrap) insertUniKeyOwner(sa *SoDemo) bool {
    uniWrap  := UniDemoOwnerWrap{}
     uniWrap.Dba = s.dba
   
   res := uniWrap.UniQueryOwner(sa.Owner)
   if res != nil {
		//the unique key is already exist
		return false
	}
    val := SoUniqueDemoByOwner{}
    val.Owner = sa.Owner
    
	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}
    
    pre := DemoOwnerUniTable
    sub := sa.Owner
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniDemoOwnerWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniDemoOwnerWrap) UniQueryOwner(start *prototype.AccountName) *SoDemoWrap{
    pre := DemoOwnerUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueDemoByOwner{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoDemoWrap(s.Dba,res.Owner)
            
			return wrap
		}
	}
    return nil
}



