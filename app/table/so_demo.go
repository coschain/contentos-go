

package table

import (
     "bytes"
     "github.com/coschain/contentos-go/common/encoding"
     "github.com/coschain/contentos-go/prototype"
	 "github.com/gogo/protobuf/proto"
     "github.com/coschain/contentos-go/iservices"
)

////////////// SECTION Prefix Mark ///////////////
var (
	DemoTable        = []byte("DemoTable")
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
    revOrdKey := append(DemoLikeCountRevOrdTable, subRevBuf...)
    revOrdErr :=  s.dba.Delete(revOrdKey) 
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
    revOrdKey := append(DemoIdxRevOrdTable, subRevBuf...)
    revOrdErr :=  s.dba.Delete(revOrdKey) 
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
    ordKey := append(DemoReplayCountTable, subBuf...)
    ordErr :=  s.dba.Delete(ordKey)
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
	mainBuf, err := encoding.Encode(m.Owner)
	if err != nil {
		return nil,err
	}
	subBuf, err := encoding.Encode(m.PostTime)
	if err != nil {
		return nil,err
	}
   ordKey := append(append(DemoPostTimeTable, subBuf...), mainBuf...)
   return ordKey,nil
}

func (m *SoListDemoByPostTime) EncodeRevSortKey() ([]byte,error) {
    mainBuf, err := encoding.Encode(m.Owner)
	if err != nil {
		return nil,err
	}
	subBuf, err := encoding.Encode(m.PostTime)
	if err != nil {
		return nil,err
	}
    ordKey := append(append(DemoPostTimeRevOrdTable, subBuf...), mainBuf...)
    revKey,revRrr := encoding.Complement(ordKey, err)
	if revRrr != nil {
        return nil,revRrr
	}
    return revKey,nil
}

//Query sort by order 
func (s *SDemoPostTimeWrap) QueryListByOrder(start prototype.TimePointSec, end prototype.TimePointSec) iservices.IDatabaseIterator {

	startBuf, err := encoding.Encode(&start)
	if err != nil {
		return nil
	}
	endBuf, err := encoding.Encode(&end)
	if err != nil {
		return nil
	}
    bufStartkey := append(DemoPostTimeTable, startBuf...)
	bufEndkey := append(DemoPostTimeTable, endBuf...)
    res := bytes.Compare(bufStartkey,bufEndkey)
    if res == 0 {
		bufEndkey = nil
	}else if res == 1 {
       //reverse order
       return nil
    }
    iter := s.Dba.NewIterator(bufStartkey, bufEndkey)
    
    return iter
}
//Query sort by reverse order 
func (s *SDemoPostTimeWrap) QueryListByRevOrder(start prototype.TimePointSec, end prototype.TimePointSec) iservices.IDatabaseIterator {

	startBuf, err := encoding.Encode(&start)
	if err != nil {
		return nil
	}
	endBuf, err := encoding.Encode(&end)
	if err != nil {
		return nil
	}
    bufStartkey := append(DemoPostTimeRevOrdTable, startBuf...)
	bufEndkey := append(DemoPostTimeRevOrdTable, endBuf...)

    rBufStart,rErr := encoding.Complement(bufStartkey, err)
    if rErr != nil {
       return nil
    }
    rBufEnd,rErr := encoding.Complement(bufEndkey, err)
    if rErr != nil { 
        return nil
    }
    res := bytes.Compare(rBufStart,rBufEnd)
    if res == 0 {
		rBufEnd = nil
	}else if res == -1 {
       // order
       return nil
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
	mainBuf, err := encoding.Encode(m.Owner)
	if err != nil {
		return nil,err
	}
	subBuf, err := encoding.Encode(m.LikeCount)
	if err != nil {
		return nil,err
	}
   ordKey := append(append(DemoLikeCountTable, subBuf...), mainBuf...)
   return ordKey,nil
}

func (m *SoListDemoByLikeCount) EncodeRevSortKey() ([]byte,error) {
    mainBuf, err := encoding.Encode(m.Owner)
	if err != nil {
		return nil,err
	}
	subBuf, err := encoding.Encode(m.LikeCount)
	if err != nil {
		return nil,err
	}
    ordKey := append(append(DemoLikeCountRevOrdTable, subBuf...), mainBuf...)
    revKey,revRrr := encoding.Complement(ordKey, err)
	if revRrr != nil {
        return nil,revRrr
	}
    return revKey,nil
}

//Query sort by order 
func (s *SDemoLikeCountWrap) QueryListByOrder(start int64, end int64) iservices.IDatabaseIterator {

	startBuf, err := encoding.Encode(&start)
	if err != nil {
		return nil
	}
	endBuf, err := encoding.Encode(&end)
	if err != nil {
		return nil
	}
    bufStartkey := append(DemoLikeCountTable, startBuf...)
	bufEndkey := append(DemoLikeCountTable, endBuf...)
    res := bytes.Compare(bufStartkey,bufEndkey)
    if res == 0 {
		bufEndkey = nil
	}else if res == 1 {
       //reverse order
       return nil
    }
    iter := s.Dba.NewIterator(bufStartkey, bufEndkey)
    
    return iter
}
//Query sort by reverse order 
func (s *SDemoLikeCountWrap) QueryListByRevOrder(start int64, end int64) iservices.IDatabaseIterator {

	startBuf, err := encoding.Encode(&start)
	if err != nil {
		return nil
	}
	endBuf, err := encoding.Encode(&end)
	if err != nil {
		return nil
	}
    bufStartkey := append(DemoLikeCountRevOrdTable, startBuf...)
	bufEndkey := append(DemoLikeCountRevOrdTable, endBuf...)

    rBufStart,rErr := encoding.Complement(bufStartkey, err)
    if rErr != nil {
       return nil
    }
    rBufEnd,rErr := encoding.Complement(bufEndkey, err)
    if rErr != nil { 
        return nil
    }
    res := bytes.Compare(rBufStart,rBufEnd)
    if res == 0 {
		rBufEnd = nil
	}else if res == -1 {
       // order
       return nil
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
	mainBuf, err := encoding.Encode(m.Owner)
	if err != nil {
		return nil,err
	}
	subBuf, err := encoding.Encode(m.Idx)
	if err != nil {
		return nil,err
	}
   ordKey := append(append(DemoIdxTable, subBuf...), mainBuf...)
   return ordKey,nil
}

func (m *SoListDemoByIdx) EncodeRevSortKey() ([]byte,error) {
    mainBuf, err := encoding.Encode(m.Owner)
	if err != nil {
		return nil,err
	}
	subBuf, err := encoding.Encode(m.Idx)
	if err != nil {
		return nil,err
	}
    ordKey := append(append(DemoIdxRevOrdTable, subBuf...), mainBuf...)
    revKey,revRrr := encoding.Complement(ordKey, err)
	if revRrr != nil {
        return nil,revRrr
	}
    return revKey,nil
}

//Query sort by order 
func (s *SDemoIdxWrap) QueryListByOrder(start int64, end int64) iservices.IDatabaseIterator {

	startBuf, err := encoding.Encode(&start)
	if err != nil {
		return nil
	}
	endBuf, err := encoding.Encode(&end)
	if err != nil {
		return nil
	}
    bufStartkey := append(DemoIdxTable, startBuf...)
	bufEndkey := append(DemoIdxTable, endBuf...)
    res := bytes.Compare(bufStartkey,bufEndkey)
    if res == 0 {
		bufEndkey = nil
	}else if res == 1 {
       //reverse order
       return nil
    }
    iter := s.Dba.NewIterator(bufStartkey, bufEndkey)
    
    return iter
}
//Query sort by reverse order 
func (s *SDemoIdxWrap) QueryListByRevOrder(start int64, end int64) iservices.IDatabaseIterator {

	startBuf, err := encoding.Encode(&start)
	if err != nil {
		return nil
	}
	endBuf, err := encoding.Encode(&end)
	if err != nil {
		return nil
	}
    bufStartkey := append(DemoIdxRevOrdTable, startBuf...)
	bufEndkey := append(DemoIdxRevOrdTable, endBuf...)

    rBufStart,rErr := encoding.Complement(bufStartkey, err)
    if rErr != nil {
       return nil
    }
    rBufEnd,rErr := encoding.Complement(bufEndkey, err)
    if rErr != nil { 
        return nil
    }
    res := bytes.Compare(rBufStart,rBufEnd)
    if res == 0 {
		rBufEnd = nil
	}else if res == -1 {
       // order
       return nil
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
	mainBuf, err := encoding.Encode(m.Owner)
	if err != nil {
		return nil,err
	}
	subBuf, err := encoding.Encode(m.ReplayCount)
	if err != nil {
		return nil,err
	}
   ordKey := append(append(DemoReplayCountTable, subBuf...), mainBuf...)
   return ordKey,nil
}

func (m *SoListDemoByReplayCount) EncodeRevSortKey() ([]byte,error) {
    mainBuf, err := encoding.Encode(m.Owner)
	if err != nil {
		return nil,err
	}
	subBuf, err := encoding.Encode(m.ReplayCount)
	if err != nil {
		return nil,err
	}
    ordKey := append(append(DemoReplayCountRevOrdTable, subBuf...), mainBuf...)
    revKey,revRrr := encoding.Complement(ordKey, err)
	if revRrr != nil {
        return nil,revRrr
	}
    return revKey,nil
}

//Query sort by order 
func (s *SDemoReplayCountWrap) QueryListByOrder(start int64, end int64) iservices.IDatabaseIterator {

	startBuf, err := encoding.Encode(&start)
	if err != nil {
		return nil
	}
	endBuf, err := encoding.Encode(&end)
	if err != nil {
		return nil
	}
    bufStartkey := append(DemoReplayCountTable, startBuf...)
	bufEndkey := append(DemoReplayCountTable, endBuf...)
    res := bytes.Compare(bufStartkey,bufEndkey)
    if res == 0 {
		bufEndkey = nil
	}else if res == 1 {
       //reverse order
       return nil
    }
    iter := s.Dba.NewIterator(bufStartkey, bufEndkey)
    
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
	res, err := encoding.Encode(s.mainKey)

	if err != nil {
		return nil, err
	}

	return append(DemoTable, res...), nil
}

////////////// Unique Query delete/insert/query ///////////////


func (s *SoDemoWrap) delUniKeyIdx(sa *SoDemo) bool {
	val := SoUniqueDemoByIdx{}

	val.Idx = sa.Idx
    val.Owner = sa.Owner
    key, err := encoding.Encode(sa.Idx)

	if err != nil {
		return false
	}

	return s.dba.Delete(append(DemoIdxUniTable,key...)) == nil
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

	key, err := encoding.Encode(sa.Idx)

	if err != nil {
		return false
	}
	return s.dba.Put(append(DemoIdxUniTable,key...), buf) == nil

}

type UniDemoIdxWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniDemoIdxWrap) UniQueryIdx(start *int64) *SoDemoWrap{

   startBuf, err := encoding.Encode(start)
	if err != nil {
		return nil
	}
	bufStartkey := append(DemoIdxUniTable, startBuf...)
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
	val := SoUniqueDemoByLikeCount{}

	val.LikeCount = sa.LikeCount
    val.Owner = sa.Owner
    key, err := encoding.Encode(sa.LikeCount)

	if err != nil {
		return false
	}

	return s.dba.Delete(append(DemoLikeCountUniTable,key...)) == nil
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

	key, err := encoding.Encode(sa.LikeCount)

	if err != nil {
		return false
	}
	return s.dba.Put(append(DemoLikeCountUniTable,key...), buf) == nil

}

type UniDemoLikeCountWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniDemoLikeCountWrap) UniQueryLikeCount(start *int64) *SoDemoWrap{

   startBuf, err := encoding.Encode(start)
	if err != nil {
		return nil
	}
	bufStartkey := append(DemoLikeCountUniTable, startBuf...)
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
	val := SoUniqueDemoByOwner{}

	val.Owner = sa.Owner
    key, err := encoding.Encode(sa.Owner)

	if err != nil {
		return false
	}

	return s.dba.Delete(append(DemoOwnerUniTable,key...)) == nil
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

	key, err := encoding.Encode(sa.Owner)

	if err != nil {
		return false
	}
	return s.dba.Put(append(DemoOwnerUniTable,key...), buf) == nil

}

type UniDemoOwnerWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniDemoOwnerWrap) UniQueryOwner(start *prototype.AccountName) *SoDemoWrap{

   startBuf, err := encoding.Encode(start)
	if err != nil {
		return nil
	}
	bufStartkey := append(DemoOwnerUniTable, startBuf...)
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



