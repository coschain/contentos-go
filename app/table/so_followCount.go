

package table

import (
     "errors"
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

func (s *SoFollowCountWrap) CreateFollowCount(f func(t *SoFollowCount)) error {

	val := &SoFollowCount{}
    f(val)
    if val.Account == nil {
       return errors.New("the mainkey is nil")
    }
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
	
  
    //update unique list
    if !s.insertUniKeyAccount(val) {
		return err
	}
	
    
	return nil
}

////////////// SECTION LKeys delete/insert ///////////////

////////////// SECTION LKeys delete/insert //////////////

func (s *SoFollowCountWrap) RemoveFollowCount() error {
	sa := s.getFollowCount()
	if sa == nil {
		return errors.New("delete data fail ")
	}
    //delete sort list key
	
    //delete unique list
    if !s.delUniKeyAccount(sa) {
		return errors.New("delete the unique key Account fail")
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
func (s *SoFollowCountWrap) GetAccount(v **prototype.AccountName) error {
	res := s.getFollowCount()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Account
   return nil
}


func (s *SoFollowCountWrap) GetFollowerCnt(v *uint32) error {
	res := s.getFollowCount()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.FollowerCnt
   return nil
}



func (s *SoFollowCountWrap) MdFollowerCnt(p uint32) error {
	sa := s.getFollowCount()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
   sa.FollowerCnt = p
   
   
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoFollowCountWrap) GetFollowingCnt(v *uint32) error {
	res := s.getFollowCount()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.FollowingCnt
   return nil
}



func (s *SoFollowCountWrap) MdFollowingCnt(p uint32) error {
	sa := s.getFollowCount()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
   sa.FollowingCnt = p
   
   
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoFollowCountWrap) GetUpdateTime(v **prototype.TimePointSec) error {
	res := s.getFollowCount()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.UpdateTime
   return nil
}



func (s *SoFollowCountWrap) MdUpdateTime(p prototype.TimePointSec) error {
	sa := s.getFollowCount()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
   
   sa.UpdateTime = &p
   
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}



/////////////// SECTION Private function ////////////////

func (s *SoFollowCountWrap) update(sa *SoFollowCount) error {
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
    pre := FollowCountTable
    sub := s.mainKey
    if sub == nil {
       return nil,errors.New("the mainKey is nil")
    }
    kList := []interface{}{pre,sub}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

////////////// Unique Query delete/insert/query ///////////////


func (s *SoFollowCountWrap) delUniKeyAccount(sa *SoFollowCount) bool {
    pre := FollowCountAccountUniTable
    sub := sa.Account
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}


func (s *SoFollowCountWrap) insertUniKeyAccount(sa *SoFollowCount) bool {
    uniWrap  := UniFollowCountAccountWrap{}
     uniWrap.Dba = s.dba
   
   res := uniWrap.UniQueryAccount(sa.Account,nil)
   if res == nil {
		//the unique key is already exist
		return false
	}
    val := SoUniqueFollowCountByAccount{}
    val.Account = sa.Account
    
	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}
    
    pre := FollowCountAccountUniTable
    sub := sa.Account
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniFollowCountAccountWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniFollowCountAccountWrap) UniQueryAccount(start *prototype.AccountName,wrap *SoFollowCountWrap) error{
    pre := FollowCountAccountUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueFollowCountByAccount{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap.mainKey = res.Account
            
            wrap.dba = s.Dba
			return nil  
		}
        return rErr
	}
    return err
}



