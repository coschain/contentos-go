

package table

import (
	"github.com/coschain/contentos-go/common/encoding"
     "github.com/coschain/contentos-go/common/prototype"
	 "github.com/gogo/protobuf/proto"
     "github.com/coschain/contentos-go/iservices"
)

////////////// SECTION Prefix Mark ///////////////
var (
	AccountTable        = []byte("AccountTable")
    AccountCreatedTimeTable = []byte("AccountCreatedTimeTable")
    AccountCreatedTimeRevOrdTable = []byte("AccountCreatedTimeRevOrdTable")
    AccountBalanceTable = []byte("AccountBalanceTable")
    AccountBalanceRevOrdTable = []byte("AccountBalanceRevOrdTable")
    AccountVestingSharesTable = []byte("AccountVestingSharesTable")
    AccountVestingSharesRevOrdTable = []byte("AccountVestingSharesRevOrdTable")
    AccountIdxUniTable = []byte("AccountIdxUniTable")
    AccountNameUniTable = []byte("AccountNameUniTable")
    AccountPubKeyUniTable = []byte("AccountPubKeyUniTable")
    )

////////////// SECTION Wrap Define ///////////////
type SoAccountWrap struct {
	dba 		iservices.IDatabaseService
	mainKey 	*prototype.AccountName
}

func NewSoAccountWrap(dba iservices.IDatabaseService, key *prototype.AccountName) *SoAccountWrap{
	result := &SoAccountWrap{ dba, key}
	return result
}

func (s *SoAccountWrap) CheckExist() bool {
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

func (s *SoAccountWrap) CreateAccount(sa *SoAccount) bool {

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
	
	if !s.insertSortKeyCreatedTime(sa) {
		return false
	}
	
	if !s.insertSortKeyBalance(sa) {
		return false
	}
	
	if !s.insertSortKeyVestingShares(sa) {
		return false
	}
	
  
    //update unique list
    if !s.insertUniKeyIdx(sa) {
		return false
	}
	if !s.insertUniKeyName(sa) {
		return false
	}
	if !s.insertUniKeyPubKey(sa) {
		return false
	}
	
    
	return true
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoAccountWrap) delSortKeyCreatedTime(sa *SoAccount) bool {
	val := SoListAccountByCreatedTime{}
	val.CreatedTime = sa.CreatedTime
	val.Name = sa.Name
    subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
    ordKey := append(AccountCreatedTimeTable, subBuf...)
    ordErr :=  s.dba.Delete(ordKey)
    return ordErr == nil
    
}


func (s *SoAccountWrap) insertSortKeyCreatedTime(sa *SoAccount) bool {
	val := SoListAccountByCreatedTime{}
	val.Name = sa.Name
	val.CreatedTime = sa.CreatedTime
	buf, err := proto.Marshal(&val)
	if err != nil {
		return false
	}
    subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
    ordKey := append(AccountCreatedTimeTable, subBuf...)
    ordErr :=  s.dba.Put(ordKey, buf) 
    return ordErr == nil
    
}


func (s *SoAccountWrap) delSortKeyBalance(sa *SoAccount) bool {
	val := SoListAccountByBalance{}
	val.Balance = sa.Balance
	val.Name = sa.Name
    subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
    ordKey := append(AccountBalanceTable, subBuf...)
    ordErr :=  s.dba.Delete(ordKey)
    return ordErr == nil
    
}


func (s *SoAccountWrap) insertSortKeyBalance(sa *SoAccount) bool {
	val := SoListAccountByBalance{}
	val.Name = sa.Name
	val.Balance = sa.Balance
	buf, err := proto.Marshal(&val)
	if err != nil {
		return false
	}
    subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
    ordKey := append(AccountBalanceTable, subBuf...)
    ordErr :=  s.dba.Put(ordKey, buf) 
    return ordErr == nil
    
}


func (s *SoAccountWrap) delSortKeyVestingShares(sa *SoAccount) bool {
	val := SoListAccountByVestingShares{}
	val.VestingShares = sa.VestingShares
	val.Name = sa.Name
    subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
    ordKey := append(AccountVestingSharesTable, subBuf...)
    ordErr :=  s.dba.Delete(ordKey)
    return ordErr == nil
    
}


func (s *SoAccountWrap) insertSortKeyVestingShares(sa *SoAccount) bool {
	val := SoListAccountByVestingShares{}
	val.Name = sa.Name
	val.VestingShares = sa.VestingShares
	buf, err := proto.Marshal(&val)
	if err != nil {
		return false
	}
    subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
    ordKey := append(AccountVestingSharesTable, subBuf...)
    ordErr :=  s.dba.Put(ordKey, buf) 
    return ordErr == nil
    
}


////////////// SECTION LKeys delete/insert //////////////

func (s *SoAccountWrap) RemoveAccount() bool {
	sa := s.getAccount()
	if sa == nil {
		return false
	}
    //delete sort list key
	if !s.delSortKeyCreatedTime(sa) {
		return false
	}
	if !s.delSortKeyBalance(sa) {
		return false
	}
	if !s.delSortKeyVestingShares(sa) {
		return false
	}
	
    //delete unique list
    if !s.delUniKeyIdx(sa) {
		return false
	}
	if !s.delUniKeyName(sa) {
		return false
	}
	if !s.delUniKeyPubKey(sa) {
		return false
	}
	
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return false
	}
	return s.dba.Delete(keyBuf) == nil
}

////////////// SECTION Members Get/Modify ///////////////
func (s *SoAccountWrap) GetBalance() *prototype.Coin {
	res := s.getAccount()

   if res == nil {
      return nil
      
   }
   return res.Balance
}



func (s *SoAccountWrap) MdBalance(p prototype.Coin) bool {
	sa := s.getAccount()
	if sa == nil {
		return false
	}
	
	if !s.delSortKeyBalance(sa) {
		return false
	}
   
   sa.Balance = &p
   
	if !s.update(sa) {
		return false
	}
    
    if !s.insertSortKeyBalance(sa) {
		return false
    }
       
	return true
}

func (s *SoAccountWrap) GetCreatedTime() *prototype.TimePointSec {
	res := s.getAccount()

   if res == nil {
      return nil
      
   }
   return res.CreatedTime
}



func (s *SoAccountWrap) MdCreatedTime(p prototype.TimePointSec) bool {
	sa := s.getAccount()
	if sa == nil {
		return false
	}
	
	if !s.delSortKeyCreatedTime(sa) {
		return false
	}
   
   sa.CreatedTime = &p
   
	if !s.update(sa) {
		return false
	}
    
    if !s.insertSortKeyCreatedTime(sa) {
		return false
    }
       
	return true
}

func (s *SoAccountWrap) GetCreator() *prototype.AccountName {
	res := s.getAccount()

   if res == nil {
      return nil
      
   }
   return res.Creator
}



func (s *SoAccountWrap) MdCreator(p prototype.AccountName) bool {
	sa := s.getAccount()
	if sa == nil {
		return false
	}
	
   
   sa.Creator = &p
   
	if !s.update(sa) {
		return false
	}
    
	return true
}

func (s *SoAccountWrap) GetIdx() int64 {
	res := s.getAccount()

   if res == nil {
      var tmpValue int64 
      return tmpValue
   }
   return res.Idx
}



func (s *SoAccountWrap) MdIdx(p int64) bool {
	sa := s.getAccount()
	if sa == nil {
		return false
	}
    //judge the unique value if is exist
    uniWrap  := UniAccountIdxWrap{}
   res := uniWrap.UniQueryIdx(&sa.Idx)
	if res != nil {
		//the unique value to be modified is already exist
		return false
	}
	if !s.delUniKeyIdx(sa) {
		return false
	}
    
	
   sa.Idx = p
   
   
	if !s.update(sa) {
		return false
	}
    
    if !s.insertUniKeyIdx(sa) {
		return false
    }
	return true
}

func (s *SoAccountWrap) GetName() *prototype.AccountName {
	res := s.getAccount()

   if res == nil {
      return nil
      
   }
   return res.Name
}


func (s *SoAccountWrap) GetPubKey() *prototype.PublicKeyType {
	res := s.getAccount()

   if res == nil {
      return nil
      
   }
   return res.PubKey
}



func (s *SoAccountWrap) MdPubKey(p prototype.PublicKeyType) bool {
	sa := s.getAccount()
	if sa == nil {
		return false
	}
    //judge the unique value if is exist
    uniWrap  := UniAccountPubKeyWrap{}
   res := uniWrap.UniQueryPubKey(sa.PubKey)
   
	if res != nil {
		//the unique value to be modified is already exist
		return false
	}
	if !s.delUniKeyPubKey(sa) {
		return false
	}
    
	
   
   sa.PubKey = &p
   
	if !s.update(sa) {
		return false
	}
    
    if !s.insertUniKeyPubKey(sa) {
		return false
    }
	return true
}

func (s *SoAccountWrap) GetVestingShares() *prototype.Vest {
	res := s.getAccount()

   if res == nil {
      return nil
      
   }
   return res.VestingShares
}



func (s *SoAccountWrap) MdVestingShares(p prototype.Vest) bool {
	sa := s.getAccount()
	if sa == nil {
		return false
	}
	
	if !s.delSortKeyVestingShares(sa) {
		return false
	}
   
   sa.VestingShares = &p
   
	if !s.update(sa) {
		return false
	}
    
    if !s.insertSortKeyVestingShares(sa) {
		return false
    }
       
	return true
}




////////////// SECTION List Keys ///////////////
type SAccountCreatedTimeWrap struct {
	Dba iservices.IDatabaseService
}

func (s *SAccountCreatedTimeWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListAccountByCreatedTime{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    
   return res.Name
   

}

func (s *SAccountCreatedTimeWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.TimePointSec {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListAccountByCreatedTime{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    
   
    
   return res.CreatedTime
   
}

func (m *SoListAccountByCreatedTime) OpeEncode() ([]byte,error) {
	mainBuf, err := encoding.Encode(m.Name)
	if err != nil {
		return nil,err
	}
	subBuf, err := encoding.Encode(m.CreatedTime)
	if err != nil {
		return nil,err
	}
   ordKey := append(append(AccountCreatedTimeTable, subBuf...), mainBuf...)
   return ordKey,nil
}

func (m *SoListAccountByCreatedTime) EncodeRevSortKey() ([]byte,error) {
     ordKey,err := m.OpeEncode()
     if err != nil {
        return nil,err
     }
     revKey,revRrr := encoding.Complement(ordKey, err) 
     if revRrr != nil {
        return nil,revRrr
     }
     return revKey,nil
}

//Query sort by order 
func (s *SAccountCreatedTimeWrap) QueryListByOrder(start prototype.TimePointSec, end prototype.TimePointSec) iservices.IDatabaseIterator {

	startBuf, err := encoding.Encode(&start)
	if err != nil {
		return nil
	}
	endBuf, err := encoding.Encode(&end)
	if err != nil {
		return nil
	}
    bufStartkey := append(AccountCreatedTimeTable, startBuf...)
	bufEndkey := append(AccountCreatedTimeTable, endBuf...)
    iter := s.Dba.NewIterator(bufStartkey, bufEndkey)
    return iter
    
}

////////////// SECTION List Keys ///////////////
type SAccountBalanceWrap struct {
	Dba iservices.IDatabaseService
}

func (s *SAccountBalanceWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListAccountByBalance{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    
   return res.Name
   

}

func (s *SAccountBalanceWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.Coin {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListAccountByBalance{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    
   
    
   return res.Balance
   
}

func (m *SoListAccountByBalance) OpeEncode() ([]byte,error) {
	mainBuf, err := encoding.Encode(m.Name)
	if err != nil {
		return nil,err
	}
	subBuf, err := encoding.Encode(m.Balance)
	if err != nil {
		return nil,err
	}
   ordKey := append(append(AccountBalanceTable, subBuf...), mainBuf...)
   return ordKey,nil
}

func (m *SoListAccountByBalance) EncodeRevSortKey() ([]byte,error) {
     ordKey,err := m.OpeEncode()
     if err != nil {
        return nil,err
     }
     revKey,revRrr := encoding.Complement(ordKey, err) 
     if revRrr != nil {
        return nil,revRrr
     }
     return revKey,nil
}

//Query sort by order 
func (s *SAccountBalanceWrap) QueryListByOrder(start prototype.Coin, end prototype.Coin) iservices.IDatabaseIterator {

	startBuf, err := encoding.Encode(&start)
	if err != nil {
		return nil
	}
	endBuf, err := encoding.Encode(&end)
	if err != nil {
		return nil
	}
    bufStartkey := append(AccountBalanceTable, startBuf...)
	bufEndkey := append(AccountBalanceTable, endBuf...)
    iter := s.Dba.NewIterator(bufStartkey, bufEndkey)
    return iter
    
}

////////////// SECTION List Keys ///////////////
type SAccountVestingSharesWrap struct {
	Dba iservices.IDatabaseService
}

func (s *SAccountVestingSharesWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListAccountByVestingShares{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    
   return res.Name
   

}

func (s *SAccountVestingSharesWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.Vest {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListAccountByVestingShares{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    
   
    
   return res.VestingShares
   
}

func (m *SoListAccountByVestingShares) OpeEncode() ([]byte,error) {
	mainBuf, err := encoding.Encode(m.Name)
	if err != nil {
		return nil,err
	}
	subBuf, err := encoding.Encode(m.VestingShares)
	if err != nil {
		return nil,err
	}
   ordKey := append(append(AccountVestingSharesTable, subBuf...), mainBuf...)
   return ordKey,nil
}

func (m *SoListAccountByVestingShares) EncodeRevSortKey() ([]byte,error) {
     ordKey,err := m.OpeEncode()
     if err != nil {
        return nil,err
     }
     revKey,revRrr := encoding.Complement(ordKey, err) 
     if revRrr != nil {
        return nil,revRrr
     }
     return revKey,nil
}

//Query sort by order 
func (s *SAccountVestingSharesWrap) QueryListByOrder(start prototype.Vest, end prototype.Vest) iservices.IDatabaseIterator {

	startBuf, err := encoding.Encode(&start)
	if err != nil {
		return nil
	}
	endBuf, err := encoding.Encode(&end)
	if err != nil {
		return nil
	}
    bufStartkey := append(AccountVestingSharesTable, startBuf...)
	bufEndkey := append(AccountVestingSharesTable, endBuf...)
    iter := s.Dba.NewIterator(bufStartkey, bufEndkey)
    return iter
    
}
/////////////// SECTION Private function ////////////////

func (s *SoAccountWrap) update(sa *SoAccount) bool {
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

func (s *SoAccountWrap) getAccount() *SoAccount {
	keyBuf, err := s.encodeMainKey()

	if err != nil {
		return nil
	}

	resBuf, err := s.dba.Get(keyBuf)

	if err != nil {
		return nil
	}

	res := &SoAccount{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoAccountWrap) encodeMainKey() ([]byte, error) {
	res, err := encoding.Encode(s.mainKey)

	if err != nil {
		return nil, err
	}

	return append(AccountTable, res...), nil
}

////////////// Unique Query delete/insert/query ///////////////


func (s *SoAccountWrap) delUniKeyIdx(sa *SoAccount) bool {
	val := SoUniqueAccountByIdx{}

	val.Idx = sa.Idx
	val.Name = sa.Name

	key, err := encoding.Encode(sa.Idx)

	if err != nil {
		return false
	}

	return s.dba.Delete(append(AccountIdxUniTable,key...)) == nil
}


func (s *SoAccountWrap) insertUniKeyIdx(sa *SoAccount) bool {
    uniWrap  := UniAccountIdxWrap{}
     uniWrap.Dba = s.dba
   
    
   	res := uniWrap.UniQueryIdx(&sa.Idx)
   
   
	if res != nil {
		//the unique key is already exist
		return false
	}
 
    val := SoUniqueAccountByIdx{}

    
	val.Name = sa.Name
	val.Idx = sa.Idx
    
	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	key, err := encoding.Encode(sa.Idx)

	if err != nil {
		return false
	}
	return s.dba.Put(append(AccountIdxUniTable,key...), buf) == nil

}

type UniAccountIdxWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniAccountIdxWrap) UniQueryIdx(start *int64) *SoAccountWrap{

   startBuf, err := encoding.Encode(start)
	if err != nil {
		return nil
	}
	bufStartkey := append(AccountIdxUniTable, startBuf...)
	bufEndkey := bufStartkey
	iter := s.Dba.NewIterator(bufStartkey, bufEndkey)
    val, err := iter.Value()
	if err != nil {
		return nil
	}
	res := &SoUniqueAccountByIdx{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
   wrap := NewSoAccountWrap(s.Dba,res.Name)
   
    
	return wrap	
}



func (s *SoAccountWrap) delUniKeyName(sa *SoAccount) bool {
	val := SoUniqueAccountByName{}

	val.Name = sa.Name
	val.Name = sa.Name

	key, err := encoding.Encode(sa.Name)

	if err != nil {
		return false
	}

	return s.dba.Delete(append(AccountNameUniTable,key...)) == nil
}


func (s *SoAccountWrap) insertUniKeyName(sa *SoAccount) bool {
    uniWrap  := UniAccountNameWrap{}
     uniWrap.Dba = s.dba
   
   
    
   	res := uniWrap.UniQueryName(sa.Name)
   
	if res != nil {
		//the unique key is already exist
		return false
	}
 
    val := SoUniqueAccountByName{}

    
	val.Name = sa.Name
	val.Name = sa.Name
    
	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	key, err := encoding.Encode(sa.Name)

	if err != nil {
		return false
	}
	return s.dba.Put(append(AccountNameUniTable,key...), buf) == nil

}

type UniAccountNameWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniAccountNameWrap) UniQueryName(start *prototype.AccountName) *SoAccountWrap{

   startBuf, err := encoding.Encode(start)
	if err != nil {
		return nil
	}
	bufStartkey := append(AccountNameUniTable, startBuf...)
	bufEndkey := bufStartkey
	iter := s.Dba.NewIterator(bufStartkey, bufEndkey)
    val, err := iter.Value()
	if err != nil {
		return nil
	}
	res := &SoUniqueAccountByName{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
   wrap := NewSoAccountWrap(s.Dba,res.Name)
   
    
	return wrap	
}



func (s *SoAccountWrap) delUniKeyPubKey(sa *SoAccount) bool {
	val := SoUniqueAccountByPubKey{}

	val.PubKey = sa.PubKey
	val.Name = sa.Name

	key, err := encoding.Encode(sa.PubKey)

	if err != nil {
		return false
	}

	return s.dba.Delete(append(AccountPubKeyUniTable,key...)) == nil
}


func (s *SoAccountWrap) insertUniKeyPubKey(sa *SoAccount) bool {
    uniWrap  := UniAccountPubKeyWrap{}
     uniWrap.Dba = s.dba
   
   
    
   	res := uniWrap.UniQueryPubKey(sa.PubKey)
   
	if res != nil {
		//the unique key is already exist
		return false
	}
 
    val := SoUniqueAccountByPubKey{}

    
	val.Name = sa.Name
	val.PubKey = sa.PubKey
    
	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	key, err := encoding.Encode(sa.PubKey)

	if err != nil {
		return false
	}
	return s.dba.Put(append(AccountPubKeyUniTable,key...), buf) == nil

}

type UniAccountPubKeyWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniAccountPubKeyWrap) UniQueryPubKey(start *prototype.PublicKeyType) *SoAccountWrap{

   startBuf, err := encoding.Encode(start)
	if err != nil {
		return nil
	}
	bufStartkey := append(AccountPubKeyUniTable, startBuf...)
	bufEndkey := bufStartkey
	iter := s.Dba.NewIterator(bufStartkey, bufEndkey)
    val, err := iter.Value()
	if err != nil {
		return nil
	}
	res := &SoUniqueAccountByPubKey{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
   wrap := NewSoAccountWrap(s.Dba,res.Name)
   
    
	return wrap	
}



