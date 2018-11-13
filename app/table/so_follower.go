

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
	mainBuf, err := encoding.Encode(m.Account)
	if err != nil {
		return nil,err
	}
	subBuf, err := encoding.Encode(m.CreateTime)
	if err != nil {
		return nil,err
	}
   ordKey := append(append(FollowerCreateTimeTable, subBuf...), mainBuf...)
   return ordKey,nil
}

func (m *SoListFollowerByCreateTime) EncodeRevSortKey() ([]byte,error) {
    mainBuf, err := encoding.Encode(m.Account)
	if err != nil {
		return nil,err
	}
	subBuf, err := encoding.Encode(m.CreateTime)
	if err != nil {
		return nil,err
	}
    ordKey := append(append(FollowerCreateTimeRevOrdTable, subBuf...), mainBuf...)
    revKey,revRrr := encoding.Complement(ordKey, err)
	if revRrr != nil {
        return nil,revRrr
	}
    return revKey,nil
}

//Query sort by order 
func (s *SFollowerCreateTimeWrap) QueryListByOrder(start prototype.TimePointSec, end prototype.TimePointSec) iservices.IDatabaseIterator {

	startBuf, err := encoding.Encode(&start)
	if err != nil {
		return nil
	}
	endBuf, err := encoding.Encode(&end)
	if err != nil {
		return nil
	}
    bufStartkey := append(FollowerCreateTimeTable, startBuf...)
	bufEndkey := append(FollowerCreateTimeTable, endBuf...)
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
	res, err := encoding.Encode(s.mainKey)

	if err != nil {
		return nil, err
	}

	return append(FollowerTable, res...), nil
}

////////////// Unique Query delete/insert/query ///////////////


func (s *SoFollowerWrap) delUniKeyAccount(sa *SoFollower) bool {
	val := SoUniqueFollowerByAccount{}

	val.Account = sa.Account
    key, err := encoding.Encode(sa.Account)

	if err != nil {
		return false
	}

	return s.dba.Delete(append(FollowerAccountUniTable,key...)) == nil
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

	key, err := encoding.Encode(sa.Account)

	if err != nil {
		return false
	}
	return s.dba.Put(append(FollowerAccountUniTable,key...), buf) == nil

}

type UniFollowerAccountWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniFollowerAccountWrap) UniQueryAccount(start *prototype.AccountName) *SoFollowerWrap{

   startBuf, err := encoding.Encode(start)
	if err != nil {
		return nil
	}
	bufStartkey := append(FollowerAccountUniTable, startBuf...)
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
	val := SoUniqueFollowerByFollower{}

	val.Follower = sa.Follower
    val.Account = sa.Account
    key, err := encoding.Encode(sa.Follower)

	if err != nil {
		return false
	}

	return s.dba.Delete(append(FollowerFollowerUniTable,key...)) == nil
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

	key, err := encoding.Encode(sa.Follower)

	if err != nil {
		return false
	}
	return s.dba.Put(append(FollowerFollowerUniTable,key...), buf) == nil

}

type UniFollowerFollowerWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniFollowerFollowerWrap) UniQueryFollower(start *prototype.AccountName) *SoFollowerWrap{

   startBuf, err := encoding.Encode(start)
	if err != nil {
		return nil
	}
	bufStartkey := append(FollowerFollowerUniTable, startBuf...)
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



