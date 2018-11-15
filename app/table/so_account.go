

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
	AccountTable        = []byte("AccountTable")
    AccountCreatedTimeTable = []byte("AccountCreatedTimeTable")
    AccountCreatedTimeRevOrdTable = []byte("AccountCreatedTimeRevOrdTable")
    AccountBalanceTable = []byte("AccountBalanceTable")
    AccountBalanceRevOrdTable = []byte("AccountBalanceRevOrdTable")
    AccountVestingSharesTable = []byte("AccountVestingSharesTable")
    AccountVestingSharesRevOrdTable = []byte("AccountVestingSharesRevOrdTable")
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
    ordErr :=  s.dba.Delete(subBuf)
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
    ordErr :=  s.dba.Put(subBuf, buf) 
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
    ordErr :=  s.dba.Delete(subBuf)
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
    ordErr :=  s.dba.Put(subBuf, buf) 
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
    ordErr :=  s.dba.Delete(subBuf)
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
    ordErr :=  s.dba.Put(subBuf, buf) 
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

func (s *SAccountCreatedTimeWrap)DelIterater(iterator iservices.IDatabaseIterator){
   if iterator == nil || !iterator.Valid() {
		return 
	}
   s.Dba.DeleteIterator(iterator)
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
    pre := AccountCreatedTimeTable
    sub := m.CreatedTime
    if sub == nil {
       return nil,errors.New("the pro CreatedTime is nil")
    }
    sub1 := m.Name
    if sub1 == nil {
       return nil,errors.New("the mainKey CreatedTime is nil")
    }
    kList := []interface{}{pre,sub,sub1}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

func (m *SoListAccountByCreatedTime) EncodeRevSortKey() ([]byte,error) {
    pre := AccountCreatedTimeRevOrdTable
    sub := m.CreatedTime
    if sub == nil {
       return nil,errors.New("the pro CreatedTime is nil")
    }
    sub1 := m.Name
    if sub1 == nil {
       return nil,errors.New("the mainKey CreatedTime is nil")
    }
    kList := []interface{}{pre,sub,sub1}
    ordKey,cErr := encoding.EncodeSlice(kList,false)
    if cErr != nil {
       return nil,cErr
    }
    revKey,revRrr := encoding.Complement(ordKey, cErr)
    return revKey,revRrr
}

//Query sort by order 
//start = nil  end = nil (query the db from start to end)
//start = nil (query from start the db)
//end = nil (query to the end of db)
func (s *SAccountCreatedTimeWrap) QueryListByOrder(start *prototype.TimePointSec, end *prototype.TimePointSec) iservices.IDatabaseIterator {
    pre := AccountCreatedTimeTable
    skeyList := []interface{}{pre}
    if start != nil {
       skeyList = append(skeyList,start)
    }
    sBuf,cErr := encoding.EncodeSlice(skeyList,false)
    if cErr != nil {
         return nil
    }
    if start != nil && end == nil {
		iter := s.Dba.NewIterator(sBuf, nil)
		return iter
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


////////////// SECTION List Keys ///////////////
type SAccountBalanceWrap struct {
	Dba iservices.IDatabaseService
}

func (s *SAccountBalanceWrap)DelIterater(iterator iservices.IDatabaseIterator){
   if iterator == nil || !iterator.Valid() {
		return 
	}
   s.Dba.DeleteIterator(iterator)
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
    pre := AccountBalanceTable
    sub := m.Balance
    if sub == nil {
       return nil,errors.New("the pro Balance is nil")
    }
    sub1 := m.Name
    if sub1 == nil {
       return nil,errors.New("the mainKey Balance is nil")
    }
    kList := []interface{}{pre,sub,sub1}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

func (m *SoListAccountByBalance) EncodeRevSortKey() ([]byte,error) {
    pre := AccountBalanceRevOrdTable
    sub := m.Balance
    if sub == nil {
       return nil,errors.New("the pro Balance is nil")
    }
    sub1 := m.Name
    if sub1 == nil {
       return nil,errors.New("the mainKey Balance is nil")
    }
    kList := []interface{}{pre,sub,sub1}
    ordKey,cErr := encoding.EncodeSlice(kList,false)
    if cErr != nil {
       return nil,cErr
    }
    revKey,revRrr := encoding.Complement(ordKey, cErr)
    return revKey,revRrr
}

//Query sort by order 
//start = nil  end = nil (query the db from start to end)
//start = nil (query from start the db)
//end = nil (query to the end of db)
func (s *SAccountBalanceWrap) QueryListByOrder(start *prototype.Coin, end *prototype.Coin) iservices.IDatabaseIterator {
    pre := AccountBalanceTable
    skeyList := []interface{}{pre}
    if start != nil {
       skeyList = append(skeyList,start)
    }
    sBuf,cErr := encoding.EncodeSlice(skeyList,false)
    if cErr != nil {
         return nil
    }
    if start != nil && end == nil {
		iter := s.Dba.NewIterator(sBuf, nil)
		return iter
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


////////////// SECTION List Keys ///////////////
type SAccountVestingSharesWrap struct {
	Dba iservices.IDatabaseService
}

func (s *SAccountVestingSharesWrap)DelIterater(iterator iservices.IDatabaseIterator){
   if iterator == nil || !iterator.Valid() {
		return 
	}
   s.Dba.DeleteIterator(iterator)
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
    pre := AccountVestingSharesTable
    sub := m.VestingShares
    if sub == nil {
       return nil,errors.New("the pro VestingShares is nil")
    }
    sub1 := m.Name
    if sub1 == nil {
       return nil,errors.New("the mainKey VestingShares is nil")
    }
    kList := []interface{}{pre,sub,sub1}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

func (m *SoListAccountByVestingShares) EncodeRevSortKey() ([]byte,error) {
    pre := AccountVestingSharesRevOrdTable
    sub := m.VestingShares
    if sub == nil {
       return nil,errors.New("the pro VestingShares is nil")
    }
    sub1 := m.Name
    if sub1 == nil {
       return nil,errors.New("the mainKey VestingShares is nil")
    }
    kList := []interface{}{pre,sub,sub1}
    ordKey,cErr := encoding.EncodeSlice(kList,false)
    if cErr != nil {
       return nil,cErr
    }
    revKey,revRrr := encoding.Complement(ordKey, cErr)
    return revKey,revRrr
}

//Query sort by order 
//start = nil  end = nil (query the db from start to end)
//start = nil (query from start the db)
//end = nil (query to the end of db)
func (s *SAccountVestingSharesWrap) QueryListByOrder(start *prototype.Vest, end *prototype.Vest) iservices.IDatabaseIterator {
    pre := AccountVestingSharesTable
    skeyList := []interface{}{pre}
    if start != nil {
       skeyList = append(skeyList,start)
    }
    sBuf,cErr := encoding.EncodeSlice(skeyList,false)
    if cErr != nil {
         return nil
    }
    if start != nil && end == nil {
		iter := s.Dba.NewIterator(sBuf, nil)
		return iter
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
    pre := AccountTable
    sub := s.mainKey
    if sub == nil {
       return nil,errors.New("the mainKey is nil")
    }
    kList := []interface{}{pre,sub}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

////////////// Unique Query delete/insert/query ///////////////


func (s *SoAccountWrap) delUniKeyName(sa *SoAccount) bool {
    pre := AccountNameUniTable
    sub := sa.Name
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
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
    
	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}
    
    pre := AccountNameUniTable
    sub := sa.Name
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniAccountNameWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniAccountNameWrap) UniQueryName(start *prototype.AccountName) *SoAccountWrap{
    pre := AccountNameUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueAccountByName{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoAccountWrap(s.Dba,res.Name)
            
			return wrap
		}
	}
    return nil
}



func (s *SoAccountWrap) delUniKeyPubKey(sa *SoAccount) bool {
    pre := AccountPubKeyUniTable
    sub := sa.PubKey
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
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
    
    pre := AccountPubKeyUniTable
    sub := sa.PubKey
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniAccountPubKeyWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniAccountPubKeyWrap) UniQueryPubKey(start *prototype.PublicKeyType) *SoAccountWrap{
    pre := AccountPubKeyUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueAccountByPubKey{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoAccountWrap(s.Dba,res.Name)
            
			return wrap
		}
	}
    return nil
}



