

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
	FollowingTable        = []byte("FollowingTable")
    FollowingFollowingOrderTable = []byte("FollowingFollowingOrderTable")
    FollowingFollowingOrderRevOrdTable = []byte("FollowingFollowingOrderRevOrdTable")
    FollowingAccountUniTable = []byte("FollowingAccountUniTable")
    FollowingFollowingUniTable = []byte("FollowingFollowingUniTable")
    FollowingFollowingOrderUniTable = []byte("FollowingFollowingOrderUniTable")
    )

////////////// SECTION Wrap Define ///////////////
type SoFollowingWrap struct {
	dba 		iservices.IDatabaseService
	mainKey 	*prototype.AccountName
}

func NewSoFollowingWrap(dba iservices.IDatabaseService, key *prototype.AccountName) *SoFollowingWrap{
	result := &SoFollowingWrap{ dba, key}
	return result
}

func (s *SoFollowingWrap) CheckExist() bool {
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

func (s *SoFollowingWrap) CreateFollowing(sa *SoFollowing) bool {

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
	
	if !s.insertSortKeyFollowingOrder(sa) {
		return false
	}
	
  
    //update unique list
    if !s.insertUniKeyAccount(sa) {
		return false
	}
	if !s.insertUniKeyFollowing(sa) {
		return false
	}
	if !s.insertUniKeyFollowingOrder(sa) {
		return false
	}
	
    
	return true
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoFollowingWrap) delSortKeyFollowingOrder(sa *SoFollowing) bool {
	val := SoListFollowingByFollowingOrder{}
	val.FollowingOrder = sa.FollowingOrder
	val.Account = sa.Account
    subRevBuf, err := val.EncodeRevSortKey()
	if err != nil {
		return false
	}
    revOrdKey := append(FollowingFollowingOrderRevOrdTable, subRevBuf...)
    revOrdErr :=  s.dba.Delete(revOrdKey) 
    return revOrdErr == nil
    
}


func (s *SoFollowingWrap) insertSortKeyFollowingOrder(sa *SoFollowing) bool {
	val := SoListFollowingByFollowingOrder{}
	val.Account = sa.Account
	val.FollowingOrder = sa.FollowingOrder
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

func (s *SoFollowingWrap) RemoveFollowing() bool {
	sa := s.getFollowing()
	if sa == nil {
		return false
	}
    //delete sort list key
	if !s.delSortKeyFollowingOrder(sa) {
		return false
	}
	
    //delete unique list
    if !s.delUniKeyAccount(sa) {
		return false
	}
	if !s.delUniKeyFollowing(sa) {
		return false
	}
	if !s.delUniKeyFollowingOrder(sa) {
		return false
	}
	
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return false
	}
	return s.dba.Delete(keyBuf) == nil
}

////////////// SECTION Members Get/Modify ///////////////
func (s *SoFollowingWrap) GetAccount() *prototype.AccountName {
	res := s.getFollowing()

   if res == nil {
      return nil
      
   }
   return res.Account
}


func (s *SoFollowingWrap) GetCreatedTime() *prototype.TimePointSec {
	res := s.getFollowing()

   if res == nil {
      return nil
      
   }
   return res.CreatedTime
}



func (s *SoFollowingWrap) MdCreatedTime(p prototype.TimePointSec) bool {
	sa := s.getFollowing()
	if sa == nil {
		return false
	}
	
   
   sa.CreatedTime = &p
   
	if !s.update(sa) {
		return false
	}
    
	return true
}

func (s *SoFollowingWrap) GetFollowing() *prototype.AccountName {
	res := s.getFollowing()

   if res == nil {
      return nil
      
   }
   return res.Following
}



func (s *SoFollowingWrap) MdFollowing(p prototype.AccountName) bool {
	sa := s.getFollowing()
	if sa == nil {
		return false
	}
    //judge the unique value if is exist
    uniWrap  := UniFollowingFollowingWrap{}
   res := uniWrap.UniQueryFollowing(sa.Following)
   
	if res != nil {
		//the unique value to be modified is already exist
		return false
	}
	if !s.delUniKeyFollowing(sa) {
		return false
	}
    
	
   
   sa.Following = &p
   
	if !s.update(sa) {
		return false
	}
    
    if !s.insertUniKeyFollowing(sa) {
		return false
    }
	return true
}

func (s *SoFollowingWrap) GetFollowingOrder() *prototype.FollowingOrder {
	res := s.getFollowing()

   if res == nil {
      return nil
      
   }
   return res.FollowingOrder
}



func (s *SoFollowingWrap) MdFollowingOrder(p prototype.FollowingOrder) bool {
	sa := s.getFollowing()
	if sa == nil {
		return false
	}
    //judge the unique value if is exist
    uniWrap  := UniFollowingFollowingOrderWrap{}
   res := uniWrap.UniQueryFollowingOrder(sa.FollowingOrder)
   
	if res != nil {
		//the unique value to be modified is already exist
		return false
	}
	if !s.delUniKeyFollowingOrder(sa) {
		return false
	}
    
	
	if !s.delSortKeyFollowingOrder(sa) {
		return false
	}
   
   sa.FollowingOrder = &p
   
	if !s.update(sa) {
		return false
	}
    
    if !s.insertSortKeyFollowingOrder(sa) {
		return false
    }
       
    if !s.insertUniKeyFollowingOrder(sa) {
		return false
    }
	return true
}




////////////// SECTION List Keys ///////////////
type SFollowingFollowingOrderWrap struct {
	Dba iservices.IDatabaseService
}

func (s *SFollowingFollowingOrderWrap)DelIterater(iterator iservices.IDatabaseIterator){
   if iterator == nil || !iterator.Valid() {
		return 
	}
   s.Dba.DeleteIterator(iterator)
}

func (s *SFollowingFollowingOrderWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListFollowingByFollowingOrder{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    
   return res.Account
   

}

func (s *SFollowingFollowingOrderWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.FollowingOrder {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListFollowingByFollowingOrder{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    
   
    
   return res.FollowingOrder
   
}

func (m *SoListFollowingByFollowingOrder) OpeEncode() ([]byte,error) {
    pre := FollowingFollowingOrderTable
    sub := m.FollowingOrder
    if sub == nil {
       return nil,errors.New("the pro FollowingOrder is nil")
    }
    sub1 := m.Account
    if sub1 == nil {
       return nil,errors.New("the mainKey FollowingOrder is nil")
    }
    kList := []interface{}{pre,sub,sub1}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

func (m *SoListFollowingByFollowingOrder) EncodeRevSortKey() ([]byte,error) {
    pre := FollowingFollowingOrderRevOrdTable
    sub := m.FollowingOrder
    if sub == nil {
       return nil,errors.New("the pro FollowingOrder is nil")
    }
    sub1 := m.Account
    if sub1 == nil {
       return nil,errors.New("the mainKey FollowingOrder is nil")
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
func (s *SFollowingFollowingOrderWrap) QueryListByRevOrder(start *prototype.FollowingOrder, end *prototype.FollowingOrder) iservices.IDatabaseIterator {

    pre := FollowingFollowingOrderTable
    skeyList := []interface{}{pre}
    if start != nil {
       skeyList = append(skeyList,start)
    }
    sBuf,cErr := encoding.EncodeSlice(skeyList,false)
    if cErr != nil {
         return nil
    }
    
    eKeyList := []interface{}{pre}
    if end != nil {
       eKeyList = append(eKeyList,end)
    }
    eBuf,cErr := encoding.EncodeSlice(eKeyList,false)
    if cErr != nil {
       return nil
    }
    rBufStart,rErr := encoding.Complement(sBuf, cErr)
    if rErr != nil {
       return nil
    }
    rBufEnd,rErr := encoding.Complement(eBuf, cErr)
    if rErr != nil { 
        return nil
    }
    res := bytes.Compare(sBuf,eBuf)
    if res == 0 {
		rBufEnd = nil
	}else if res == -1 {
       // order
       return nil
    }
    iter := s.Dba.NewIterator(rBufStart, rBufEnd)
    return iter
}
/////////////// SECTION Private function ////////////////

func (s *SoFollowingWrap) update(sa *SoFollowing) bool {
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

func (s *SoFollowingWrap) getFollowing() *SoFollowing {
	keyBuf, err := s.encodeMainKey()

	if err != nil {
		return nil
	}

	resBuf, err := s.dba.Get(keyBuf)

	if err != nil {
		return nil
	}

	res := &SoFollowing{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoFollowingWrap) encodeMainKey() ([]byte, error) {
    pre := FollowingTable
    sub := s.mainKey
    if sub == nil {
       return nil,errors.New("the mainKey is nil")
    }
    kList := []interface{}{pre,sub}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

////////////// Unique Query delete/insert/query ///////////////


func (s *SoFollowingWrap) delUniKeyAccount(sa *SoFollowing) bool {
    pre := FollowingAccountUniTable
    sub := sa.Account
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}


func (s *SoFollowingWrap) insertUniKeyAccount(sa *SoFollowing) bool {
    uniWrap  := UniFollowingAccountWrap{}
     uniWrap.Dba = s.dba
   
   res := uniWrap.UniQueryAccount(sa.Account)
   if res != nil {
		//the unique key is already exist
		return false
	}
    val := SoUniqueFollowingByAccount{}
    val.Account = sa.Account
    
	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}
    
    pre := FollowingAccountUniTable
    sub := sa.Account
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniFollowingAccountWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniFollowingAccountWrap) UniQueryAccount(start *prototype.AccountName) *SoFollowingWrap{
    pre := FollowingAccountUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueFollowingByAccount{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoFollowingWrap(s.Dba,res.Account)
            
			return wrap
		}
	}
    return nil
}



func (s *SoFollowingWrap) delUniKeyFollowing(sa *SoFollowing) bool {
    pre := FollowingFollowingUniTable
    sub := sa.Following
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}


func (s *SoFollowingWrap) insertUniKeyFollowing(sa *SoFollowing) bool {
    uniWrap  := UniFollowingFollowingWrap{}
     uniWrap.Dba = s.dba
   
   res := uniWrap.UniQueryFollowing(sa.Following)
   if res != nil {
		//the unique key is already exist
		return false
	}
    val := SoUniqueFollowingByFollowing{}
    val.Account = sa.Account
    val.Following = sa.Following
    
	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}
    
    pre := FollowingFollowingUniTable
    sub := sa.Following
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniFollowingFollowingWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniFollowingFollowingWrap) UniQueryFollowing(start *prototype.AccountName) *SoFollowingWrap{
    pre := FollowingFollowingUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueFollowingByFollowing{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoFollowingWrap(s.Dba,res.Account)
            
			return wrap
		}
	}
    return nil
}



func (s *SoFollowingWrap) delUniKeyFollowingOrder(sa *SoFollowing) bool {
    pre := FollowingFollowingOrderUniTable
    sub := sa.FollowingOrder
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}


func (s *SoFollowingWrap) insertUniKeyFollowingOrder(sa *SoFollowing) bool {
    uniWrap  := UniFollowingFollowingOrderWrap{}
     uniWrap.Dba = s.dba
   
   res := uniWrap.UniQueryFollowingOrder(sa.FollowingOrder)
   if res != nil {
		//the unique key is already exist
		return false
	}
    val := SoUniqueFollowingByFollowingOrder{}
    val.Account = sa.Account
    val.FollowingOrder = sa.FollowingOrder
    
	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}
    
    pre := FollowingFollowingOrderUniTable
    sub := sa.FollowingOrder
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniFollowingFollowingOrderWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniFollowingFollowingOrderWrap) UniQueryFollowingOrder(start *prototype.FollowingOrder) *SoFollowingWrap{
    pre := FollowingFollowingOrderUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueFollowingByFollowingOrder{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoFollowingWrap(s.Dba,res.Account)
            
			return wrap
		}
	}
    return nil
}



