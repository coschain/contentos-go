

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
	FollowerTable        = []byte("FollowerTable")
    FollowerFollowerOrderTable = []byte("FollowerFollowerOrderTable")
    FollowerFollowerOrderRevOrdTable = []byte("FollowerFollowerOrderRevOrdTable")
    FollowerAccountUniTable = []byte("FollowerAccountUniTable")
    FollowerFollowerUniTable = []byte("FollowerFollowerUniTable")
    FollowerFollowerOrderUniTable = []byte("FollowerFollowerOrderUniTable")
    )

////////////// SECTION Wrap Define ///////////////
type SoFollowerWrap struct {
	dba 		iservices.IDatabaseService
	mainKey 	*prototype.AccountName
}

func NewSoFollowerWrap(dba iservices.IDatabaseService, key *prototype.AccountName) *SoFollowerWrap{
	result := &SoFollowerWrap{ dba, key}
	return result
}

func (s *SoFollowerWrap) CheckExist() bool {
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

func (s *SoFollowerWrap) CreateFollower(sa *SoFollower) bool {

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
	
	if !s.insertSortKeyFollowerOrder(sa) {
		return false
	}
	
  
    //update unique list
    if !s.insertUniKeyAccount(sa) {
		return false
	}
	if !s.insertUniKeyFollower(sa) {
		return false
	}
	if !s.insertUniKeyFollowerOrder(sa) {
		return false
	}
	
    
	return true
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoFollowerWrap) delSortKeyFollowerOrder(sa *SoFollower) bool {
	val := SoListFollowerByFollowerOrder{}
	val.FollowerOrder = sa.FollowerOrder
	val.Account = sa.Account
    subRevBuf, err := val.EncodeRevSortKey()
	if err != nil {
		return false
	}
    revOrdKey := append(FollowerFollowerOrderRevOrdTable, subRevBuf...)
    revOrdErr :=  s.dba.Delete(revOrdKey) 
    return revOrdErr == nil
    
}


func (s *SoFollowerWrap) insertSortKeyFollowerOrder(sa *SoFollower) bool {
	val := SoListFollowerByFollowerOrder{}
	val.Account = sa.Account
	val.FollowerOrder = sa.FollowerOrder
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

func (s *SoFollowerWrap) RemoveFollower() bool {
	sa := s.getFollower()
	if sa == nil {
		return false
	}
    //delete sort list key
	if !s.delSortKeyFollowerOrder(sa) {
		return false
	}
	
    //delete unique list
    if !s.delUniKeyAccount(sa) {
		return false
	}
	if !s.delUniKeyFollower(sa) {
		return false
	}
	if !s.delUniKeyFollowerOrder(sa) {
		return false
	}
	
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return false
	}
	return s.dba.Delete(keyBuf) == nil
}

////////////// SECTION Members Get/Modify ///////////////
func (s *SoFollowerWrap) GetAccount() *prototype.AccountName {
	res := s.getFollower()

   if res == nil {
      return nil
      
   }
   return res.Account
}


func (s *SoFollowerWrap) GetCreatedTime() *prototype.TimePointSec {
	res := s.getFollower()

   if res == nil {
      return nil
      
   }
   return res.CreatedTime
}



func (s *SoFollowerWrap) MdCreatedTime(p prototype.TimePointSec) bool {
	sa := s.getFollower()
	if sa == nil {
		return false
	}
	
   
   sa.CreatedTime = &p
   
	if !s.update(sa) {
		return false
	}
    
	return true
}

func (s *SoFollowerWrap) GetFollower() *prototype.AccountName {
	res := s.getFollower()

   if res == nil {
      return nil
      
   }
   return res.Follower
}



func (s *SoFollowerWrap) MdFollower(p prototype.AccountName) bool {
	sa := s.getFollower()
	if sa == nil {
		return false
	}
    //judge the unique value if is exist
    uniWrap  := UniFollowerFollowerWrap{}
   res := uniWrap.UniQueryFollower(sa.Follower)
   
	if res != nil {
		//the unique value to be modified is already exist
		return false
	}
	if !s.delUniKeyFollower(sa) {
		return false
	}
    
	
   
   sa.Follower = &p
   
	if !s.update(sa) {
		return false
	}
    
    if !s.insertUniKeyFollower(sa) {
		return false
    }
	return true
}

func (s *SoFollowerWrap) GetFollowerOrder() *prototype.FollowerOrder {
	res := s.getFollower()

   if res == nil {
      return nil
      
   }
   return res.FollowerOrder
}



func (s *SoFollowerWrap) MdFollowerOrder(p prototype.FollowerOrder) bool {
	sa := s.getFollower()
	if sa == nil {
		return false
	}
    //judge the unique value if is exist
    uniWrap  := UniFollowerFollowerOrderWrap{}
   res := uniWrap.UniQueryFollowerOrder(sa.FollowerOrder)
   
	if res != nil {
		//the unique value to be modified is already exist
		return false
	}
	if !s.delUniKeyFollowerOrder(sa) {
		return false
	}
    
	
	if !s.delSortKeyFollowerOrder(sa) {
		return false
	}
   
   sa.FollowerOrder = &p
   
	if !s.update(sa) {
		return false
	}
    
    if !s.insertSortKeyFollowerOrder(sa) {
		return false
    }
       
    if !s.insertUniKeyFollowerOrder(sa) {
		return false
    }
	return true
}




////////////// SECTION List Keys ///////////////
type SFollowerFollowerOrderWrap struct {
	Dba iservices.IDatabaseService
}

func (s *SFollowerFollowerOrderWrap)DelIterater(iterator iservices.IDatabaseIterator){
   if iterator == nil || !iterator.Valid() {
		return 
	}
   s.Dba.DeleteIterator(iterator)
}

func (s *SFollowerFollowerOrderWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListFollowerByFollowerOrder{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    
   return res.Account
   

}

func (s *SFollowerFollowerOrderWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.FollowerOrder {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListFollowerByFollowerOrder{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    
   
    
   return res.FollowerOrder
   
}

func (m *SoListFollowerByFollowerOrder) OpeEncode() ([]byte,error) {
    pre := FollowerFollowerOrderTable
    sub := m.FollowerOrder
    if sub == nil {
       return nil,errors.New("the pro FollowerOrder is nil")
    }
    sub1 := m.Account
    if sub1 == nil {
       return nil,errors.New("the mainKey FollowerOrder is nil")
    }
    kList := []interface{}{pre,sub,sub1}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

func (m *SoListFollowerByFollowerOrder) EncodeRevSortKey() ([]byte,error) {
    pre := FollowerFollowerOrderRevOrdTable
    sub := m.FollowerOrder
    if sub == nil {
       return nil,errors.New("the pro FollowerOrder is nil")
    }
    sub1 := m.Account
    if sub1 == nil {
       return nil,errors.New("the mainKey FollowerOrder is nil")
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
func (s *SFollowerFollowerOrderWrap) QueryListByRevOrder(start *prototype.FollowerOrder, end *prototype.FollowerOrder) iservices.IDatabaseIterator {

    pre := FollowerFollowerOrderTable
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

func (s *SoFollowerWrap) update(sa *SoFollower) bool {
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

func (s *SoFollowerWrap) getFollower() *SoFollower {
	keyBuf, err := s.encodeMainKey()

	if err != nil {
		return nil
	}

	resBuf, err := s.dba.Get(keyBuf)

	if err != nil {
		return nil
	}

	res := &SoFollower{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoFollowerWrap) encodeMainKey() ([]byte, error) {
    pre := FollowerTable
    sub := s.mainKey
    if sub == nil {
       return nil,errors.New("the mainKey is nil")
    }
    kList := []interface{}{pre,sub}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

////////////// Unique Query delete/insert/query ///////////////


func (s *SoFollowerWrap) delUniKeyAccount(sa *SoFollower) bool {
    pre := FollowerAccountUniTable
    sub := sa.Account
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}


func (s *SoFollowerWrap) insertUniKeyAccount(sa *SoFollower) bool {
    uniWrap  := UniFollowerAccountWrap{}
     uniWrap.Dba = s.dba
   
   res := uniWrap.UniQueryAccount(sa.Account)
   if res != nil {
		//the unique key is already exist
		return false
	}
    val := SoUniqueFollowerByAccount{}
    val.Account = sa.Account
    
	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}
    
    pre := FollowerAccountUniTable
    sub := sa.Account
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniFollowerAccountWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniFollowerAccountWrap) UniQueryAccount(start *prototype.AccountName) *SoFollowerWrap{
    pre := FollowerAccountUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueFollowerByAccount{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoFollowerWrap(s.Dba,res.Account)
            
			return wrap
		}
	}
    return nil
}



func (s *SoFollowerWrap) delUniKeyFollower(sa *SoFollower) bool {
    pre := FollowerFollowerUniTable
    sub := sa.Follower
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}


func (s *SoFollowerWrap) insertUniKeyFollower(sa *SoFollower) bool {
    uniWrap  := UniFollowerFollowerWrap{}
     uniWrap.Dba = s.dba
   
   res := uniWrap.UniQueryFollower(sa.Follower)
   if res != nil {
		//the unique key is already exist
		return false
	}
    val := SoUniqueFollowerByFollower{}
    val.Account = sa.Account
    val.Follower = sa.Follower
    
	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}
    
    pre := FollowerFollowerUniTable
    sub := sa.Follower
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniFollowerFollowerWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniFollowerFollowerWrap) UniQueryFollower(start *prototype.AccountName) *SoFollowerWrap{
    pre := FollowerFollowerUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueFollowerByFollower{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoFollowerWrap(s.Dba,res.Account)
            
			return wrap
		}
	}
    return nil
}



func (s *SoFollowerWrap) delUniKeyFollowerOrder(sa *SoFollower) bool {
    pre := FollowerFollowerOrderUniTable
    sub := sa.FollowerOrder
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}


func (s *SoFollowerWrap) insertUniKeyFollowerOrder(sa *SoFollower) bool {
    uniWrap  := UniFollowerFollowerOrderWrap{}
     uniWrap.Dba = s.dba
   
   res := uniWrap.UniQueryFollowerOrder(sa.FollowerOrder)
   if res != nil {
		//the unique key is already exist
		return false
	}
    val := SoUniqueFollowerByFollowerOrder{}
    val.Account = sa.Account
    val.FollowerOrder = sa.FollowerOrder
    
	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}
    
    pre := FollowerFollowerOrderUniTable
    sub := sa.FollowerOrder
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniFollowerFollowerOrderWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniFollowerFollowerOrderWrap) UniQueryFollowerOrder(start *prototype.FollowerOrder) *SoFollowerWrap{
    pre := FollowerFollowerOrderUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueFollowerByFollowerOrder{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoFollowerWrap(s.Dba,res.Account)
            
			return wrap
		}
	}
    return nil
}



