

package table

import (
     "github.com/coschain/contentos-go/common/encoding"
     "github.com/coschain/contentos-go/prototype"
	 "github.com/gogo/protobuf/proto"
     "github.com/coschain/contentos-go/iservices"
)

////////////// SECTION Prefix Mark ///////////////
var (
	FollowCountTable        = []byte("FollowCountTable")
    FollowCountAccountUniTable = []byte("FollowCountAccountUniTable")
    )

////////////// SECTION Wrap Define ///////////////
type SoFollowCountWrap struct {
	dba 		iservices.IDatabaseService
	mainKey 	*prototype.AccountName
}

func NewSoFollowCountWrap(dba iservices.IDatabaseService, key *prototype.AccountName) *SoFollowCountWrap{
	result := &SoFollowCountWrap{ dba, key}
	return result
}

func (s *SoFollowCountWrap) CheckExist() bool {
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

func (s *SoFollowCountWrap) CreateFollowCount(sa *SoFollowCount) bool {

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
	
  
    //update unique list
    if !s.insertUniKeyAccount(sa) {
		return false
	}
	
    
	return true
}

////////////// SECTION LKeys delete/insert ///////////////

////////////// SECTION LKeys delete/insert //////////////

func (s *SoFollowCountWrap) RemoveFollowCount() bool {
	sa := s.getFollowCount()
	if sa == nil {
		return false
	}
    //delete sort list key
	
    //delete unique list
    if !s.delUniKeyAccount(sa) {
		return false
	}
	
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return false
	}
	return s.dba.Delete(keyBuf) == nil
}

////////////// SECTION Members Get/Modify ///////////////
func (s *SoFollowCountWrap) GetAccount() *prototype.AccountName {
	res := s.getFollowCount()

   if res == nil {
      return nil
      
   }
   return res.Account
}


func (s *SoFollowCountWrap) GetFollowerCnt() uint32 {
	res := s.getFollowCount()

   if res == nil {
      var tmpValue uint32 
      return tmpValue
   }
   return res.FollowerCnt
}



func (s *SoFollowCountWrap) MdFollowerCnt(p uint32) bool {
	sa := s.getFollowCount()
	if sa == nil {
		return false
	}
	
   sa.FollowerCnt = p
   
   
	if !s.update(sa) {
		return false
	}
    
	return true
}

func (s *SoFollowCountWrap) GetFollowingCnt() uint32 {
	res := s.getFollowCount()

   if res == nil {
      var tmpValue uint32 
      return tmpValue
   }
   return res.FollowingCnt
}



func (s *SoFollowCountWrap) MdFollowingCnt(p uint32) bool {
	sa := s.getFollowCount()
	if sa == nil {
		return false
	}
	
   sa.FollowingCnt = p
   
   
	if !s.update(sa) {
		return false
	}
    
	return true
}

func (s *SoFollowCountWrap) GetUpdateTime() *prototype.TimePointSec {
	res := s.getFollowCount()

   if res == nil {
      return nil
      
   }
   return res.UpdateTime
}



func (s *SoFollowCountWrap) MdUpdateTime(p prototype.TimePointSec) bool {
	sa := s.getFollowCount()
	if sa == nil {
		return false
	}
	
   
   sa.UpdateTime = &p
   
	if !s.update(sa) {
		return false
	}
    
	return true
}



/////////////// SECTION Private function ////////////////

func (s *SoFollowCountWrap) update(sa *SoFollowCount) bool {
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

func (s *SoFollowCountWrap) getFollowCount() *SoFollowCount {
	keyBuf, err := s.encodeMainKey()

	if err != nil {
		return nil
	}

	resBuf, err := s.dba.Get(keyBuf)

	if err != nil {
		return nil
	}

	res := &SoFollowCount{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoFollowCountWrap) encodeMainKey() ([]byte, error) {
	res, err := encoding.Encode(s.mainKey)

	if err != nil {
		return nil, err
	}

	return append(FollowCountTable, res...), nil
}

////////////// Unique Query delete/insert/query ///////////////


func (s *SoFollowCountWrap) delUniKeyAccount(sa *SoFollowCount) bool {
	val := SoUniqueFollowCountByAccount{}

	val.Account = sa.Account
    key, err := encoding.Encode(sa.Account)

	if err != nil {
		return false
	}

	return s.dba.Delete(append(FollowCountAccountUniTable,key...)) == nil
}


func (s *SoFollowCountWrap) insertUniKeyAccount(sa *SoFollowCount) bool {
    uniWrap  := UniFollowCountAccountWrap{}
     uniWrap.Dba = s.dba
   
   res := uniWrap.UniQueryAccount(sa.Account)
   if res != nil {
		//the unique key is already exist
		return false
	}
    val := SoUniqueFollowCountByAccount{}
    val.Account = sa.Account
    
	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	key, err := encoding.Encode(sa.Account)

	if err != nil {
		return false
	}
	return s.dba.Put(append(FollowCountAccountUniTable,key...), buf) == nil

}

type UniFollowCountAccountWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniFollowCountAccountWrap) UniQueryAccount(start *prototype.AccountName) *SoFollowCountWrap{

   startBuf, err := encoding.Encode(start)
	if err != nil {
		return nil
	}
	bufStartkey := append(FollowCountAccountUniTable, startBuf...)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueFollowCountByAccount{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoFollowCountWrap(s.Dba,res.Account)
            
			return wrap
		}
	}
    return nil
}



