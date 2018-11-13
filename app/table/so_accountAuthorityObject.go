

package table

import (
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

func (s *SoAccountAuthorityObjectWrap) CreateAccountAuthorityObject(sa *SoAccountAuthorityObject) bool {

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

func (s *SoAccountAuthorityObjectWrap) RemoveAccountAuthorityObject() bool {
	sa := s.getAccountAuthorityObject()
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
func (s *SoAccountAuthorityObjectWrap) GetAccount() *prototype.AccountName {
	res := s.getAccountAuthorityObject()

   if res == nil {
      return nil
      
   }
   return res.Account
}


func (s *SoAccountAuthorityObjectWrap) GetActive() *prototype.Authority {
	res := s.getAccountAuthorityObject()

   if res == nil {
      return nil
      
   }
   return res.Active
}



func (s *SoAccountAuthorityObjectWrap) MdActive(p prototype.Authority) bool {
	sa := s.getAccountAuthorityObject()
	if sa == nil {
		return false
	}
	
   
   sa.Active = &p
   
	if !s.update(sa) {
		return false
	}
    
	return true
}

func (s *SoAccountAuthorityObjectWrap) GetLastOwnerUpdate() *prototype.TimePointSec {
	res := s.getAccountAuthorityObject()

   if res == nil {
      return nil
      
   }
   return res.LastOwnerUpdate
}



func (s *SoAccountAuthorityObjectWrap) MdLastOwnerUpdate(p prototype.TimePointSec) bool {
	sa := s.getAccountAuthorityObject()
	if sa == nil {
		return false
	}
	
   
   sa.LastOwnerUpdate = &p
   
	if !s.update(sa) {
		return false
	}
    
	return true
}

func (s *SoAccountAuthorityObjectWrap) GetOwner() *prototype.Authority {
	res := s.getAccountAuthorityObject()

   if res == nil {
      return nil
      
   }
   return res.Owner
}



func (s *SoAccountAuthorityObjectWrap) MdOwner(p prototype.Authority) bool {
	sa := s.getAccountAuthorityObject()
	if sa == nil {
		return false
	}
	
   
   sa.Owner = &p
   
	if !s.update(sa) {
		return false
	}
    
	return true
}

func (s *SoAccountAuthorityObjectWrap) GetPosting() *prototype.Authority {
	res := s.getAccountAuthorityObject()

   if res == nil {
      return nil
      
   }
   return res.Posting
}



func (s *SoAccountAuthorityObjectWrap) MdPosting(p prototype.Authority) bool {
	sa := s.getAccountAuthorityObject()
	if sa == nil {
		return false
	}
	
   
   sa.Posting = &p
   
	if !s.update(sa) {
		return false
	}
    
	return true
}



/////////////// SECTION Private function ////////////////

func (s *SoAccountAuthorityObjectWrap) update(sa *SoAccountAuthorityObject) bool {
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
	res, err := encoding.Encode(s.mainKey)

	if err != nil {
		return nil, err
	}

	return append(AccountAuthorityObjectTable, res...), nil
}

////////////// Unique Query delete/insert/query ///////////////


func (s *SoAccountAuthorityObjectWrap) delUniKeyAccount(sa *SoAccountAuthorityObject) bool {
	val := SoUniqueAccountAuthorityObjectByAccount{}

	val.Account = sa.Account
	val.Account = sa.Account

	key, err := encoding.Encode(sa.Account)

	if err != nil {
		return false
	}

	return s.dba.Delete(append(AccountAuthorityObjectAccountUniTable,key...)) == nil
}


func (s *SoAccountAuthorityObjectWrap) insertUniKeyAccount(sa *SoAccountAuthorityObject) bool {
    uniWrap  := UniAccountAuthorityObjectAccountWrap{}
     uniWrap.Dba = s.dba
   
   
    
   	res := uniWrap.UniQueryAccount(sa.Account)
   
	if res != nil {
		//the unique key is already exist
		return false
	}
 
    val := SoUniqueAccountAuthorityObjectByAccount{}

    
	val.Account = sa.Account
	val.Account = sa.Account
    
	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	key, err := encoding.Encode(sa.Account)

	if err != nil {
		return false
	}
	return s.dba.Put(append(AccountAuthorityObjectAccountUniTable,key...), buf) == nil

}

type UniAccountAuthorityObjectAccountWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniAccountAuthorityObjectAccountWrap) UniQueryAccount(start *prototype.AccountName) *SoAccountAuthorityObjectWrap{

   startBuf, err := encoding.Encode(start)
	if err != nil {
		return nil
	}
	bufStartkey := append(AccountAuthorityObjectAccountUniTable, startBuf...)
	bufEndkey := bufStartkey
	iter := s.Dba.NewIterator(bufStartkey, bufEndkey)
    val, err := iter.Value()
	if err != nil {
		return nil
	}
	res := &SoUniqueAccountAuthorityObjectByAccount{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
   wrap := NewSoAccountAuthorityObjectWrap(s.Dba,res.Account)
   
    
	return wrap	
}



