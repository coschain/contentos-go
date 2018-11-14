

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
	FollowerTable        = []byte("FollowerTable")
    FollowerCreateTimeTable = []byte("FollowerCreateTimeTable")
    FollowerCreateTimeRevOrdTable = []byte("FollowerCreateTimeRevOrdTable")
    FollowerAccountUniTable = []byte("FollowerAccountUniTable")
    FollowerFollowerUniTable = []byte("FollowerFollowerUniTable")
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
	
	if !s.insertSortKeyCreateTime(sa) {
		return false
	}
	
  
    //update unique list
    if !s.insertUniKeyAccount(sa) {
		return false
	}
	if !s.insertUniKeyFollower(sa) {
		return false
	}
	
    
	return true
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoFollowerWrap) delSortKeyCreateTime(sa *SoFollower) bool {
	val := SoListFollowerByCreateTime{}
	val.CreateTime = sa.CreateTime
	val.Account = sa.Account
    subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
    ordKey := append(FollowerCreateTimeTable, subBuf...)
    ordErr :=  s.dba.Delete(ordKey)
    return ordErr == nil
    
}


func (s *SoFollowerWrap) insertSortKeyCreateTime(sa *SoFollower) bool {
	val := SoListFollowerByCreateTime{}
	val.Account = sa.Account
	val.CreateTime = sa.CreateTime
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

func (s *SoFollowerWrap) RemoveFollower() bool {
	sa := s.getFollower()
	if sa == nil {
		return false
	}
    //delete sort list key
	if !s.delSortKeyCreateTime(sa) {
		return false
	}
	
    //delete unique list
    if !s.delUniKeyAccount(sa) {
		return false
	}
	if !s.delUniKeyFollower(sa) {
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


func (s *SoFollowerWrap) GetCreateTime() *prototype.TimePointSec {
	res := s.getFollower()

   if res == nil {
      return nil
      
   }
   return res.CreateTime
}



func (s *SoFollowerWrap) MdCreateTime(p prototype.TimePointSec) bool {
	sa := s.getFollower()
	if sa == nil {
		return false
	}
	
	if !s.delSortKeyCreateTime(sa) {
		return false
	}
   
   sa.CreateTime = &p
   
	if !s.update(sa) {
		return false
	}
    
    if !s.insertSortKeyCreateTime(sa) {
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




////////////// SECTION List Keys ///////////////
type SFollowerCreateTimeWrap struct {
	Dba iservices.IDatabaseService
}

func (s *SFollowerCreateTimeWrap)DelIterater(iterator iservices.IDatabaseIterator){
   if iterator == nil || !iterator.Valid() {
		return 
	}
   s.Dba.DeleteIterator(iterator)
}

func (s *SFollowerCreateTimeWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListFollowerByCreateTime{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    
   return res.Account
   

}

func (s *SFollowerCreateTimeWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.TimePointSec {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListFollowerByCreateTime{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    
   
    
   return res.CreateTime
   
}

func (m *SoListFollowerByCreateTime) OpeEncode() ([]byte,error) {
    pre := FollowerCreateTimeTable
    sub := m.CreateTime
    sub1 := m.Account
    kList := []interface{}{pre,sub,sub1}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

func (m *SoListFollowerByCreateTime) EncodeRevSortKey() ([]byte,error) {
    pre := FollowerCreateTimeRevOrdTable
    sub := m.CreateTime
    sub1 := m.Account
    kList := []interface{}{pre,sub,sub1}
    ordKey,cErr := encoding.EncodeSlice(kList,false)
    revKey,revRrr := encoding.Complement(ordKey, cErr)
    return revKey,revRrr
}

//Query sort by order 
//start = nil  end = nil (query the db from start to end)
//start = nil (query from start the db)
//end = nil (query to the end of db)
func (s *SFollowerCreateTimeWrap) QueryListByOrder(start *prototype.TimePointSec, end *prototype.TimePointSec) iservices.IDatabaseIterator {
    pre := FollowerCreateTimeRevOrdTable
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



