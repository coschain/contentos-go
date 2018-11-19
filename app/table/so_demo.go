

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

func (s *SoDemoWrap) CheckExist() error {
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return errors.New("encode the mainKey fail")
	}

	res, err := s.dba.Has(keyBuf)
	if err != nil {
		return err
	}
    if !res {
       return errors.New("the table is already exist")
    }
	return nil
}

func (s *SoDemoWrap) CreateDemo(f func(t *SoDemo)) error {

	val := &SoDemo{}
    f(val)
    if val.Owner == nil {
       return errors.New("the mainkey is nil")
    }
    if s.CheckExist() == nil {
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
	
	if !s.insertSortKeyOwner(val) {
		return errors.New("insert sort Field Owner while insert table ")
	}
	
	if !s.insertSortKeyPostTime(val) {
		return errors.New("insert sort Field PostTime while insert table ")
	}
	
	if !s.insertSortKeyLikeCount(val) {
		return errors.New("insert sort Field LikeCount while insert table ")
	}
	
	if !s.insertSortKeyIdx(val) {
		return errors.New("insert sort Field Idx while insert table ")
	}
	
	if !s.insertSortKeyReplayCount(val) {
		return errors.New("insert sort Field ReplayCount while insert table ")
	}
	
  
    //update unique list
    if !s.insertUniKeyIdx(val) {
		return errors.New("insert unique Field int64 while insert table ")
	}
	if !s.insertUniKeyLikeCount(val) {
		return errors.New("insert unique Field int64 while insert table ")
	}
	if !s.insertUniKeyOwner(val) {
		return errors.New("insert unique Field prototype.AccountName while insert table ")
	}
	
    
	return nil
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

func (s *SoDemoWrap) RemoveDemo() error {
	sa := s.getDemo()
	if sa == nil {
		return errors.New("delete data fail ")
	}
    //delete sort list key
	if !s.delSortKeyOwner(sa) {
		return errors.New("delete the sort key Owner fail")
	}
	if !s.delSortKeyPostTime(sa) {
		return errors.New("delete the sort key PostTime fail")
	}
	if !s.delSortKeyLikeCount(sa) {
		return errors.New("delete the sort key LikeCount fail")
	}
	if !s.delSortKeyIdx(sa) {
		return errors.New("delete the sort key Idx fail")
	}
	if !s.delSortKeyReplayCount(sa) {
		return errors.New("delete the sort key ReplayCount fail")
	}
	
    //delete unique list
    if !s.delUniKeyIdx(sa) {
		return errors.New("delete the unique key Idx fail")
	}
	if !s.delUniKeyLikeCount(sa) {
		return errors.New("delete the unique key LikeCount fail")
	}
	if !s.delUniKeyOwner(sa) {
		return errors.New("delete the unique key Owner fail")
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
func (s *SoDemoWrap) GetContent(v *string) error {
	res := s.getDemo()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Content
   return nil
}



func (s *SoDemoWrap) MdContent(p string) error {
	sa := s.getDemo()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.Content = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoDemoWrap) GetIdx(v *int64) error {
	res := s.getDemo()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Idx
   return nil
}



func (s *SoDemoWrap) MdIdx(p int64) error {
	sa := s.getDemo()
	if sa == nil {
		return errors.New("initialization data failed")
	}
    //judge the unique value if is exist
    uniWrap  := UniDemoIdxWrap{}
   err := uniWrap.UniQueryIdx(&sa.Idx,nil)
	if err != nil {
		//the unique value to be modified is already exist
		return errors.New("the unique value to be modified is already exist")
	}
	if !s.delUniKeyIdx(sa) {
		return errors.New("delete the unique key Idx fail")
	}
    
	
	if !s.delSortKeyIdx(sa) {
		return errors.New("delete the sort key Idx fail")
	}
    sa.Idx = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
    if !s.insertSortKeyIdx(sa) {
		return errors.New("reinsert sort key Idx fail")
    }
       
    if !s.insertUniKeyIdx(sa) {
		return errors.New("reinsert unique key Idx fail")
    }
	return nil
}

func (s *SoDemoWrap) GetLikeCount(v *int64) error {
	res := s.getDemo()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.LikeCount
   return nil
}



func (s *SoDemoWrap) MdLikeCount(p int64) error {
	sa := s.getDemo()
	if sa == nil {
		return errors.New("initialization data failed")
	}
    //judge the unique value if is exist
    uniWrap  := UniDemoLikeCountWrap{}
   err := uniWrap.UniQueryLikeCount(&sa.LikeCount,nil)
	if err != nil {
		//the unique value to be modified is already exist
		return errors.New("the unique value to be modified is already exist")
	}
	if !s.delUniKeyLikeCount(sa) {
		return errors.New("delete the unique key LikeCount fail")
	}
    
	
	if !s.delSortKeyLikeCount(sa) {
		return errors.New("delete the sort key LikeCount fail")
	}
    sa.LikeCount = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
    if !s.insertSortKeyLikeCount(sa) {
		return errors.New("reinsert sort key LikeCount fail")
    }
       
    if !s.insertUniKeyLikeCount(sa) {
		return errors.New("reinsert unique key LikeCount fail")
    }
	return nil
}

func (s *SoDemoWrap) GetOwner(v **prototype.AccountName) error {
	res := s.getDemo()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Owner
   return nil
}


func (s *SoDemoWrap) GetPostTime(v **prototype.TimePointSec) error {
	res := s.getDemo()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.PostTime
   return nil
}



func (s *SoDemoWrap) MdPostTime(p *prototype.TimePointSec) error {
	sa := s.getDemo()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
	if !s.delSortKeyPostTime(sa) {
		return errors.New("delete the sort key PostTime fail")
	}
    sa.PostTime = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
    if !s.insertSortKeyPostTime(sa) {
		return errors.New("reinsert sort key PostTime fail")
    }
       
	return nil
}

func (s *SoDemoWrap) GetReplayCount(v *int64) error {
	res := s.getDemo()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.ReplayCount
   return nil
}



func (s *SoDemoWrap) MdReplayCount(p int64) error {
	sa := s.getDemo()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
	if !s.delSortKeyReplayCount(sa) {
		return errors.New("delete the sort key ReplayCount fail")
	}
    sa.ReplayCount = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
    if !s.insertSortKeyReplayCount(sa) {
		return errors.New("reinsert sort key ReplayCount fail")
    }
       
	return nil
}

func (s *SoDemoWrap) GetTaglist(v *string) error {
	res := s.getDemo()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Taglist
   return nil
}



func (s *SoDemoWrap) MdTaglist(p string) error {
	sa := s.getDemo()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.Taglist = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoDemoWrap) GetTitle(v *string) error {
	res := s.getDemo()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Title
   return nil
}



func (s *SoDemoWrap) MdTitle(p string) error {
	sa := s.getDemo()
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
type SDemoOwnerWrap struct {
	Dba iservices.IDatabaseService
}

func (s *SDemoOwnerWrap)DelIterater(iterator iservices.IDatabaseIterator){
   if iterator == nil || !iterator.Valid() {
		return 
	}
   s.Dba.DeleteIterator(iterator)
}

func (s *SDemoOwnerWrap) GetMainVal(iterator iservices.IDatabaseIterator,mKey **prototype.AccountName) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}
	val, err := iterator.Value()

	if err != nil {
		return err
	}

	res := &SoListDemoByOwner{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return err
	}
    *mKey = res.Owner
    return nil
}

func (s *SDemoOwnerWrap) GetSubVal(iterator iservices.IDatabaseIterator, sub **prototype.AccountName) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}

	val, err := iterator.Value()

	if err != nil {
		return errors.New("the value of iterator is nil")
	}
	res := &SoListDemoByOwner{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return err
	}
    *sub = res.Owner
    return nil
}

func (m *SoListDemoByOwner) OpeEncode() ([]byte,error) {
    pre := DemoOwnerTable
    sub := m.Owner
    if sub == nil {
       return nil,errors.New("the pro Owner is nil")
    }
    sub1 := m.Owner
    if sub1 == nil {
       return nil,errors.New("the mainkey Owner is nil")
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
       return nil,errors.New("the mainkey Owner is nil")
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
func (s *SDemoOwnerWrap) QueryListByRevOrder(start *prototype.AccountName, end *prototype.AccountName,iter *iservices.IDatabaseIterator) error {

    var sBuf,eBuf,rBufStart,rBufEnd []byte
    pre := DemoOwnerRevOrdTable
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
type SDemoPostTimeWrap struct {
	Dba iservices.IDatabaseService
}

func (s *SDemoPostTimeWrap)DelIterater(iterator iservices.IDatabaseIterator){
   if iterator == nil || !iterator.Valid() {
		return 
	}
   s.Dba.DeleteIterator(iterator)
}

func (s *SDemoPostTimeWrap) GetMainVal(iterator iservices.IDatabaseIterator,mKey **prototype.AccountName) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}
	val, err := iterator.Value()

	if err != nil {
		return err
	}

	res := &SoListDemoByPostTime{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return err
	}
    *mKey = res.Owner
    return nil
}

func (s *SDemoPostTimeWrap) GetSubVal(iterator iservices.IDatabaseIterator, sub **prototype.TimePointSec) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}

	val, err := iterator.Value()

	if err != nil {
		return errors.New("the value of iterator is nil")
	}
	res := &SoListDemoByPostTime{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return err
	}
    *sub = res.PostTime
    return nil
}

func (m *SoListDemoByPostTime) OpeEncode() ([]byte,error) {
    pre := DemoPostTimeTable
    sub := m.PostTime
    if sub == nil {
       return nil,errors.New("the pro PostTime is nil")
    }
    sub1 := m.Owner
    if sub1 == nil {
       return nil,errors.New("the mainkey Owner is nil")
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
       return nil,errors.New("the mainkey Owner is nil")
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
func (s *SDemoPostTimeWrap) QueryListByOrder(start *prototype.TimePointSec, end *prototype.TimePointSec,iter *iservices.IDatabaseIterator) error {
    pre := DemoPostTimeTable
    skeyList := []interface{}{pre}
    if start != nil {
       skeyList = append(skeyList,start)
    }
    sBuf,cErr := encoding.EncodeSlice(skeyList,false)
    if cErr != nil {
         return cErr
    }
    if start != nil && end == nil {
		*iter = s.Dba.NewIterator(sBuf, nil)
		return nil
	}
    eKeyList := []interface{}{pre}
    if end != nil {
       eKeyList = append(eKeyList,end)
    }
    eBuf,cErr := encoding.EncodeSlice(eKeyList,false)
    if cErr != nil {
       return cErr
    }
    
    res := bytes.Compare(sBuf,eBuf)
    if res == 0 {
		eBuf = nil
	}else if res == 1 {
       //reverse order
       return errors.New("the start and end are not order")
    }
    *iter = s.Dba.NewIterator(sBuf, eBuf)
    
    return nil
}

//Query sort by reverse order 
func (s *SDemoPostTimeWrap) QueryListByRevOrder(start *prototype.TimePointSec, end *prototype.TimePointSec,iter *iservices.IDatabaseIterator) error {

    var sBuf,eBuf,rBufStart,rBufEnd []byte
    pre := DemoPostTimeRevOrdTable
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
type SDemoLikeCountWrap struct {
	Dba iservices.IDatabaseService
}

func (s *SDemoLikeCountWrap)DelIterater(iterator iservices.IDatabaseIterator){
   if iterator == nil || !iterator.Valid() {
		return 
	}
   s.Dba.DeleteIterator(iterator)
}

func (s *SDemoLikeCountWrap) GetMainVal(iterator iservices.IDatabaseIterator,mKey **prototype.AccountName) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}
	val, err := iterator.Value()

	if err != nil {
		return err
	}

	res := &SoListDemoByLikeCount{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return err
	}
    *mKey = res.Owner
    return nil
}

func (s *SDemoLikeCountWrap) GetSubVal(iterator iservices.IDatabaseIterator, sub *int64) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}

	val, err := iterator.Value()

	if err != nil {
		return errors.New("the value of iterator is nil")
	}
	res := &SoListDemoByLikeCount{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return err
	}
    *sub = res.LikeCount
    return nil
}

func (m *SoListDemoByLikeCount) OpeEncode() ([]byte,error) {
    pre := DemoLikeCountTable
    sub := m.LikeCount
    
    sub1 := m.Owner
    if sub1 == nil {
       return nil,errors.New("the mainkey Owner is nil")
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
       return nil,errors.New("the mainkey Owner is nil")
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
func (s *SDemoLikeCountWrap) QueryListByRevOrder(start *int64, end *int64,iter *iservices.IDatabaseIterator) error {

    var sBuf,eBuf,rBufStart,rBufEnd []byte
    pre := DemoLikeCountRevOrdTable
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
type SDemoIdxWrap struct {
	Dba iservices.IDatabaseService
}

func (s *SDemoIdxWrap)DelIterater(iterator iservices.IDatabaseIterator){
   if iterator == nil || !iterator.Valid() {
		return 
	}
   s.Dba.DeleteIterator(iterator)
}

func (s *SDemoIdxWrap) GetMainVal(iterator iservices.IDatabaseIterator,mKey **prototype.AccountName) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}
	val, err := iterator.Value()

	if err != nil {
		return err
	}

	res := &SoListDemoByIdx{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return err
	}
    *mKey = res.Owner
    return nil
}

func (s *SDemoIdxWrap) GetSubVal(iterator iservices.IDatabaseIterator, sub *int64) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}

	val, err := iterator.Value()

	if err != nil {
		return errors.New("the value of iterator is nil")
	}
	res := &SoListDemoByIdx{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return err
	}
    *sub = res.Idx
    return nil
}

func (m *SoListDemoByIdx) OpeEncode() ([]byte,error) {
    pre := DemoIdxTable
    sub := m.Idx
    
    sub1 := m.Owner
    if sub1 == nil {
       return nil,errors.New("the mainkey Owner is nil")
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
       return nil,errors.New("the mainkey Owner is nil")
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
func (s *SDemoIdxWrap) QueryListByRevOrder(start *int64, end *int64,iter *iservices.IDatabaseIterator) error {

    var sBuf,eBuf,rBufStart,rBufEnd []byte
    pre := DemoIdxRevOrdTable
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
type SDemoReplayCountWrap struct {
	Dba iservices.IDatabaseService
}

func (s *SDemoReplayCountWrap)DelIterater(iterator iservices.IDatabaseIterator){
   if iterator == nil || !iterator.Valid() {
		return 
	}
   s.Dba.DeleteIterator(iterator)
}

func (s *SDemoReplayCountWrap) GetMainVal(iterator iservices.IDatabaseIterator,mKey **prototype.AccountName) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}
	val, err := iterator.Value()

	if err != nil {
		return err
	}

	res := &SoListDemoByReplayCount{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return err
	}
    *mKey = res.Owner
    return nil
}

func (s *SDemoReplayCountWrap) GetSubVal(iterator iservices.IDatabaseIterator, sub *int64) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}

	val, err := iterator.Value()

	if err != nil {
		return errors.New("the value of iterator is nil")
	}
	res := &SoListDemoByReplayCount{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return err
	}
    *sub = res.ReplayCount
    return nil
}

func (m *SoListDemoByReplayCount) OpeEncode() ([]byte,error) {
    pre := DemoReplayCountTable
    sub := m.ReplayCount
    
    sub1 := m.Owner
    if sub1 == nil {
       return nil,errors.New("the mainkey Owner is nil")
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
       return nil,errors.New("the mainkey Owner is nil")
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
func (s *SDemoReplayCountWrap) QueryListByOrder(start *int64, end *int64,iter *iservices.IDatabaseIterator) error {
    pre := DemoReplayCountTable
    skeyList := []interface{}{pre}
    if start != nil {
       skeyList = append(skeyList,start)
    }
    sBuf,cErr := encoding.EncodeSlice(skeyList,false)
    if cErr != nil {
         return cErr
    }
    if start != nil && end == nil {
		*iter = s.Dba.NewIterator(sBuf, nil)
		return nil
	}
    eKeyList := []interface{}{pre}
    if end != nil {
       eKeyList = append(eKeyList,end)
    }
    eBuf,cErr := encoding.EncodeSlice(eKeyList,false)
    if cErr != nil {
       return cErr
    }
    
    res := bytes.Compare(sBuf,eBuf)
    if res == 0 {
		eBuf = nil
	}else if res == 1 {
       //reverse order
       return errors.New("the start and end are not order")
    }
    *iter = s.Dba.NewIterator(sBuf, eBuf)
    
    return nil
}

/////////////// SECTION Private function ////////////////

func (s *SoDemoWrap) update(sa *SoDemo) error {
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
   res := uniWrap.UniQueryIdx(&sa.Idx,nil)
   
   if res == nil {
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

func (s *UniDemoIdxWrap) UniQueryIdx(start *int64,wrap *SoDemoWrap) error{
    pre := DemoIdxUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueDemoByIdx{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap.mainKey = res.Owner
            
            wrap.dba = s.Dba
			return nil  
		}
        return rErr
	}
    return err
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
   res := uniWrap.UniQueryLikeCount(&sa.LikeCount,nil)
   
   if res == nil {
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

func (s *UniDemoLikeCountWrap) UniQueryLikeCount(start *int64,wrap *SoDemoWrap) error{
    pre := DemoLikeCountUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueDemoByLikeCount{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap.mainKey = res.Owner
            
            wrap.dba = s.Dba
			return nil  
		}
        return rErr
	}
    return err
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
   
   res := uniWrap.UniQueryOwner(sa.Owner,nil)
   if res == nil {
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

func (s *UniDemoOwnerWrap) UniQueryOwner(start *prototype.AccountName,wrap *SoDemoWrap) error{
    pre := DemoOwnerUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueDemoByOwner{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap.mainKey = res.Owner
            
            wrap.dba = s.Dba
			return nil  
		}
        return rErr
	}
    return err
}



