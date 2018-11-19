

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
	AccountAuthorityObjectTable        = []byte("AccountAuthorityObjectTable")
    AccountAuthorityObjectAccountUniTable = []byte("AccountAuthorityObjectAccountUniTable")
    )

////////////// SECTION Wrap Define ///////////////
type SoAccountAuthorityObjectWrap struct {
	dba 		iservices.IDatabaseService
	mainKey 	*prototype.AccountName
}

func NewSoAccountAuthorityObjectWrap(dba iservices.IDatabaseService, key *prototype.AccountName) *SoAccountAuthorityObjectWrap{
	result := &SoAccountAuthorityObjectWrap{ dba, key}
	return result
}

func (s *SoAccountAuthorityObjectWrap) CheckExist() bool {
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

func (s *SoAccountAuthorityObjectWrap) CreateAccountAuthorityObject(f func(t *SoAccountAuthorityObject)) error {

	val := &SoAccountAuthorityObject{}
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

func (s *SoAccountAuthorityObjectWrap) RemoveAccountAuthorityObject() error {
	sa := s.getAccountAuthorityObject()
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
func (s *SoAccountAuthorityObjectWrap) GetAccount(v **prototype.AccountName) error {
	res := s.getAccountAuthorityObject()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Account
   return nil
}


func (s *SoAccountAuthorityObjectWrap) GetActive(v **prototype.Authority) error {
	res := s.getAccountAuthorityObject()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Active
   return nil
}



func (s *SoAccountAuthorityObjectWrap) MdActive(p *prototype.Authority) error {
	sa := s.getAccountAuthorityObject()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.Active = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoAccountAuthorityObjectWrap) GetLastOwnerUpdate(v **prototype.TimePointSec) error {
	res := s.getAccountAuthorityObject()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.LastOwnerUpdate
   return nil
}



func (s *SoAccountAuthorityObjectWrap) MdLastOwnerUpdate(p *prototype.TimePointSec) error {
	sa := s.getAccountAuthorityObject()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.LastOwnerUpdate = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoAccountAuthorityObjectWrap) GetOwner(v **prototype.Authority) error {
	res := s.getAccountAuthorityObject()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Owner
   return nil
}



func (s *SoAccountAuthorityObjectWrap) MdOwner(p *prototype.Authority) error {
	sa := s.getAccountAuthorityObject()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.Owner = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoAccountAuthorityObjectWrap) GetPosting(v **prototype.Authority) error {
	res := s.getAccountAuthorityObject()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Posting
   return nil
}



func (s *SoAccountAuthorityObjectWrap) MdPosting(p *prototype.Authority) error {
	sa := s.getAccountAuthorityObject()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.Posting = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}



/////////////// SECTION Private function ////////////////

func (s *SoAccountAuthorityObjectWrap) update(sa *SoAccountAuthorityObject) error {
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

func (s *SoAccountAuthorityObjectWrap) getAccountAuthorityObject() *SoAccountAuthorityObject {
	keyBuf, err := s.encodeMainKey()

	if err != nil {
		return nil
	}

	resBuf, err := s.dba.Get(keyBuf)

	if err != nil {
		return nil
	}

	res := &SoAccountAuthorityObject{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoAccountAuthorityObjectWrap) encodeMainKey() ([]byte, error) {
    pre := AccountAuthorityObjectTable
    sub := s.mainKey
    if sub == nil {
       return nil,errors.New("the mainKey is nil")
    }
    kList := []interface{}{pre,sub}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

////////////// Unique Query delete/insert/query ///////////////


func (s *SoAccountAuthorityObjectWrap) delUniKeyAccount(sa *SoAccountAuthorityObject) bool {
    pre := AccountAuthorityObjectAccountUniTable
    sub := sa.Account
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}


func (s *SoAccountAuthorityObjectWrap) insertUniKeyAccount(sa *SoAccountAuthorityObject) bool {
    uniWrap  := UniAccountAuthorityObjectAccountWrap{}
     uniWrap.Dba = s.dba
   
   res := uniWrap.UniQueryAccount(sa.Account,nil)
   if res == nil {
		//the unique key is already exist
		return false
	}
    val := SoUniqueAccountAuthorityObjectByAccount{}
    val.Account = sa.Account
    
	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}
    
    pre := AccountAuthorityObjectAccountUniTable
    sub := sa.Account
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniAccountAuthorityObjectAccountWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniAccountAuthorityObjectAccountWrap) UniQueryAccount(start *prototype.AccountName,wrap *SoAccountAuthorityObjectWrap) error{
    pre := AccountAuthorityObjectAccountUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueAccountAuthorityObjectByAccount{}
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



