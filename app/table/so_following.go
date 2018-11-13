

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
	FollowingTable        = []byte("FollowingTable")
    FollowingCreateTimeTable = []byte("FollowingCreateTimeTable")
    FollowingCreateTimeRevOrdTable = []byte("FollowingCreateTimeRevOrdTable")
    FollowingAccountUniTable = []byte("FollowingAccountUniTable")
    FollowingFollowingUniTable = []byte("FollowingFollowingUniTable")
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
	if !s.insertUniKeyFollowing(sa) {
		return false
	}
	
    
	return true
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoFollowingWrap) delSortKeyCreateTime(sa *SoFollowing) bool {
	val := SoListFollowingByCreateTime{}
	val.CreateTime = sa.CreateTime
	val.Account = sa.Account
    subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
    ordKey := append(FollowingCreateTimeTable, subBuf...)
    ordErr :=  s.dba.Delete(ordKey)
    return ordErr == nil
    
}


func (s *SoFollowingWrap) insertSortKeyCreateTime(sa *SoFollowing) bool {
	val := SoListFollowingByCreateTime{}
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

func (s *SoFollowingWrap) RemoveFollowing() bool {
	sa := s.getFollowing()
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
	if !s.delUniKeyFollowing(sa) {
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


func (s *SoFollowingWrap) GetCreateTime() *prototype.TimePointSec {
	res := s.getFollowing()

   if res == nil {
      return nil
      
   }
   return res.CreateTime
}



func (s *SoFollowingWrap) MdCreateTime(p prototype.TimePointSec) bool {
	sa := s.getFollowing()
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




////////////// SECTION List Keys ///////////////
type SFollowingCreateTimeWrap struct {
	Dba iservices.IDatabaseService
}

func (s *SFollowingCreateTimeWrap)DelIterater(iterator iservices.IDatabaseIterator){
   if iterator == nil || !iterator.Valid() {
		return 
	}
   s.Dba.DeleteIterator(iterator)
}

func (s *SFollowingCreateTimeWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListFollowingByCreateTime{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    
   return res.Account
   

}

func (s *SFollowingCreateTimeWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.TimePointSec {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListFollowingByCreateTime{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    
   
    
   return res.CreateTime
   
}

func (m *SoListFollowingByCreateTime) OpeEncode() ([]byte,error) {
	mainBuf, err := encoding.Encode(m.Account)
	if err != nil {
		return nil,err
	}
	subBuf, err := encoding.Encode(m.CreateTime)
	if err != nil {
		return nil,err
	}
   ordKey := append(append(FollowingCreateTimeTable, subBuf...), mainBuf...)
   return ordKey,nil
}

func (m *SoListFollowingByCreateTime) EncodeRevSortKey() ([]byte,error) {
    mainBuf, err := encoding.Encode(m.Account)
	if err != nil {
		return nil,err
	}
	subBuf, err := encoding.Encode(m.CreateTime)
	if err != nil {
		return nil,err
	}
    ordKey := append(append(FollowingCreateTimeRevOrdTable, subBuf...), mainBuf...)
    revKey,revRrr := encoding.Complement(ordKey, err)
	if revRrr != nil {
        return nil,revRrr
	}
    return revKey,nil
}

//Query sort by order 
func (s *SFollowingCreateTimeWrap) QueryListByOrder(start prototype.TimePointSec, end prototype.TimePointSec) iservices.IDatabaseIterator {

	startBuf, err := encoding.Encode(&start)
	if err != nil {
		return nil
	}
	endBuf, err := encoding.Encode(&end)
	if err != nil {
		return nil
	}
    bufStartkey := append(FollowingCreateTimeTable, startBuf...)
	bufEndkey := append(FollowingCreateTimeTable, endBuf...)
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
	res, err := encoding.Encode(s.mainKey)

	if err != nil {
		return nil, err
	}

	return append(FollowingTable, res...), nil
}

////////////// Unique Query delete/insert/query ///////////////


func (s *SoFollowingWrap) delUniKeyAccount(sa *SoFollowing) bool {
	val := SoUniqueFollowingByAccount{}

	val.Account = sa.Account
    key, err := encoding.Encode(sa.Account)

	if err != nil {
		return false
	}

	return s.dba.Delete(append(FollowingAccountUniTable,key...)) == nil
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

	key, err := encoding.Encode(sa.Account)

	if err != nil {
		return false
	}
	return s.dba.Put(append(FollowingAccountUniTable,key...), buf) == nil

}

type UniFollowingAccountWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniFollowingAccountWrap) UniQueryAccount(start *prototype.AccountName) *SoFollowingWrap{

   startBuf, err := encoding.Encode(start)
	if err != nil {
		return nil
	}
	bufStartkey := append(FollowingAccountUniTable, startBuf...)
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
	val := SoUniqueFollowingByFollowing{}

	val.Following = sa.Following
    val.Account = sa.Account
    key, err := encoding.Encode(sa.Following)

	if err != nil {
		return false
	}

	return s.dba.Delete(append(FollowingFollowingUniTable,key...)) == nil
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

	key, err := encoding.Encode(sa.Following)

	if err != nil {
		return false
	}
	return s.dba.Put(append(FollowingFollowingUniTable,key...), buf) == nil

}

type UniFollowingFollowingWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniFollowingFollowingWrap) UniQueryFollowing(start *prototype.AccountName) *SoFollowingWrap{

   startBuf, err := encoding.Encode(start)
	if err != nil {
		return nil
	}
	bufStartkey := append(FollowingFollowingUniTable, startBuf...)
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



