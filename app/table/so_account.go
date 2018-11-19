

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

func (s *SoAccountWrap) CreateAccount(f func(t *SoAccount)) error {

	val := &SoAccount{}
    f(val)
    if val.Name == nil {
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
	
	if !s.insertSortKeyCreatedTime(val) {
		return err
	}
	
	if !s.insertSortKeyBalance(val) {
		return err
	}
	
	if !s.insertSortKeyVestingShares(val) {
		return err
	}
	
  
    //update unique list
    if !s.insertUniKeyName(val) {
		return err
	}
	if !s.insertUniKeyPubKey(val) {
		return err
	}
	
    
	return nil
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

func (s *SoAccountWrap) RemoveAccount() error {
	sa := s.getAccount()
	if sa == nil {
		return errors.New("delete data fail ")
	}
    //delete sort list key
	if !s.delSortKeyCreatedTime(sa) {
		return errors.New("delete the sort key CreatedTime fail")
	}
	if !s.delSortKeyBalance(sa) {
		return errors.New("delete the sort key Balance fail")
	}
	if !s.delSortKeyVestingShares(sa) {
		return errors.New("delete the sort key VestingShares fail")
	}
	
    //delete unique list
    if !s.delUniKeyName(sa) {
		return errors.New("delete the unique key Name fail")
	}
	if !s.delUniKeyPubKey(sa) {
		return errors.New("delete the unique key PubKey fail")
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
func (s *SoAccountWrap) GetBalance(v **prototype.Coin) error {
	res := s.getAccount()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Balance
   return nil
}



func (s *SoAccountWrap) MdBalance(p *prototype.Coin) error {
	sa := s.getAccount()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
	if !s.delSortKeyBalance(sa) {
		return errors.New("delete the sort key Balance fail")
	}
    sa.Balance = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
    if !s.insertSortKeyBalance(sa) {
		return errors.New("reinsert sort key Balance fail")
    }
       
	return nil
}

func (s *SoAccountWrap) GetCreatedTime(v **prototype.TimePointSec) error {
	res := s.getAccount()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.CreatedTime
   return nil
}



func (s *SoAccountWrap) MdCreatedTime(p *prototype.TimePointSec) error {
	sa := s.getAccount()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
	if !s.delSortKeyCreatedTime(sa) {
		return errors.New("delete the sort key CreatedTime fail")
	}
    sa.CreatedTime = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
    if !s.insertSortKeyCreatedTime(sa) {
		return errors.New("reinsert sort key CreatedTime fail")
    }
       
	return nil
}

func (s *SoAccountWrap) GetCreator(v **prototype.AccountName) error {
	res := s.getAccount()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Creator
   return nil
}



func (s *SoAccountWrap) MdCreator(p *prototype.AccountName) error {
	sa := s.getAccount()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.Creator = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoAccountWrap) GetName(v **prototype.AccountName) error {
	res := s.getAccount()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Name
   return nil
}


func (s *SoAccountWrap) GetPubKey(v **prototype.PublicKeyType) error {
	res := s.getAccount()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.PubKey
   return nil
}



func (s *SoAccountWrap) MdPubKey(p *prototype.PublicKeyType) error {
	sa := s.getAccount()
	if sa == nil {
		return errors.New("initialization data failed")
	}
    //judge the unique value if is exist
    uniWrap  := UniAccountPubKeyWrap{}
   err := uniWrap.UniQueryPubKey(sa.PubKey,nil)
   
	if err != nil {
		//the unique value to be modified is already exist
		return errors.New("the unique value to be modified is already exist")
	}
	if !s.delUniKeyPubKey(sa) {
		return errors.New("delete the unique key PubKey fail")
	}
    
	
    sa.PubKey = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
    if !s.insertUniKeyPubKey(sa) {
		return errors.New("reinsert unique key PubKey fail")
    }
	return nil
}

func (s *SoAccountWrap) GetVestingShares(v **prototype.Vest) error {
	res := s.getAccount()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.VestingShares
   return nil
}



func (s *SoAccountWrap) MdVestingShares(p *prototype.Vest) error {
	sa := s.getAccount()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
	if !s.delSortKeyVestingShares(sa) {
		return errors.New("delete the sort key VestingShares fail")
	}
    sa.VestingShares = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
    if !s.insertSortKeyVestingShares(sa) {
		return errors.New("reinsert sort key VestingShares fail")
    }
       
	return nil
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

func (s *SAccountCreatedTimeWrap) GetMainVal(iterator iservices.IDatabaseIterator,mKey **prototype.AccountName) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}
	val, err := iterator.Value()

	if err != nil {
		return errors.New("the value of iterator is nil")
	}

	res := &SoListAccountByCreatedTime{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return err
	}
    *mKey = res.Name
    return nil
}

func (s *SAccountCreatedTimeWrap) GetSubVal(iterator iservices.IDatabaseIterator, sub **prototype.TimePointSec) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}

	val, err := iterator.Value()

	if err != nil {
		return errors.New("the value of iterator is nil")
	}
	res := &SoListAccountByCreatedTime{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return err
	}
    *sub = res.CreatedTime
    return nil
}

func (m *SoListAccountByCreatedTime) OpeEncode() ([]byte,error) {
    pre := AccountCreatedTimeTable
    sub := m.CreatedTime
    if sub == nil {
       return nil,errors.New("the pro CreatedTime is nil")
    }
    sub1 := m.Name
    if sub1 == nil {
       return nil,errors.New("the mainkey Name is nil")
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
       return nil,errors.New("the mainkey Name is nil")
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
func (s *SAccountCreatedTimeWrap) QueryListByOrder(start *prototype.TimePointSec, end *prototype.TimePointSec,iter *iservices.IDatabaseIterator) error {
    pre := AccountCreatedTimeTable
    skeyList := []interface{}{pre}
    if start != nil {
       skeyList = append(skeyList,start)
    }
    sBuf,cErr := encoding.EncodeSlice(skeyList,false)
    if cErr != nil {
         return cErr
    }
    if start != nil && end == nil {
		*iter = s.Dba.NewIterator(sBuf, nil)
		return nil
	}
    eKeyList := []interface{}{pre}
    if end != nil {
       eKeyList = append(eKeyList,end)
    }
    eBuf,cErr := encoding.EncodeSlice(eKeyList,false)
    if cErr != nil {
       return cErr
    }
    
    res := bytes.Compare(sBuf,eBuf)
    if res == 0 {
		eBuf = nil
	}else if res == 1 {
       //reverse order
       return errors.New("the start and end are not order")
    }
    *iter = s.Dba.NewIterator(sBuf, eBuf)
    
    return nil
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

func (s *SAccountBalanceWrap) GetMainVal(iterator iservices.IDatabaseIterator,mKey **prototype.AccountName) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}
	val, err := iterator.Value()

	if err != nil {
		return errors.New("the value of iterator is nil")
	}

	res := &SoListAccountByBalance{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return err
	}
    *mKey = res.Name
    return nil
}

func (s *SAccountBalanceWrap) GetSubVal(iterator iservices.IDatabaseIterator, sub **prototype.Coin) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}

	val, err := iterator.Value()

	if err != nil {
		return errors.New("the value of iterator is nil")
	}
	res := &SoListAccountByBalance{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return err
	}
    *sub = res.Balance
    return nil
}

func (m *SoListAccountByBalance) OpeEncode() ([]byte,error) {
    pre := AccountBalanceTable
    sub := m.Balance
    if sub == nil {
       return nil,errors.New("the pro Balance is nil")
    }
    sub1 := m.Name
    if sub1 == nil {
       return nil,errors.New("the mainkey Name is nil")
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
       return nil,errors.New("the mainkey Name is nil")
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
func (s *SAccountBalanceWrap) QueryListByOrder(start *prototype.Coin, end *prototype.Coin,iter *iservices.IDatabaseIterator) error {
    pre := AccountBalanceTable
    skeyList := []interface{}{pre}
    if start != nil {
       skeyList = append(skeyList,start)
    }
    sBuf,cErr := encoding.EncodeSlice(skeyList,false)
    if cErr != nil {
         return cErr
    }
    if start != nil && end == nil {
		*iter = s.Dba.NewIterator(sBuf, nil)
		return nil
	}
    eKeyList := []interface{}{pre}
    if end != nil {
       eKeyList = append(eKeyList,end)
    }
    eBuf,cErr := encoding.EncodeSlice(eKeyList,false)
    if cErr != nil {
       return cErr
    }
    
    res := bytes.Compare(sBuf,eBuf)
    if res == 0 {
		eBuf = nil
	}else if res == 1 {
       //reverse order
       return errors.New("the start and end are not order")
    }
    *iter = s.Dba.NewIterator(sBuf, eBuf)
    
    return nil
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

func (s *SAccountVestingSharesWrap) GetMainVal(iterator iservices.IDatabaseIterator,mKey **prototype.AccountName) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}
	val, err := iterator.Value()

	if err != nil {
		return errors.New("the value of iterator is nil")
	}

	res := &SoListAccountByVestingShares{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return err
	}
    *mKey = res.Name
    return nil
}

func (s *SAccountVestingSharesWrap) GetSubVal(iterator iservices.IDatabaseIterator, sub **prototype.Vest) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}

	val, err := iterator.Value()

	if err != nil {
		return errors.New("the value of iterator is nil")
	}
	res := &SoListAccountByVestingShares{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return err
	}
    *sub = res.VestingShares
    return nil
}

func (m *SoListAccountByVestingShares) OpeEncode() ([]byte,error) {
    pre := AccountVestingSharesTable
    sub := m.VestingShares
    if sub == nil {
       return nil,errors.New("the pro VestingShares is nil")
    }
    sub1 := m.Name
    if sub1 == nil {
       return nil,errors.New("the mainkey Name is nil")
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
       return nil,errors.New("the mainkey Name is nil")
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
func (s *SAccountVestingSharesWrap) QueryListByOrder(start *prototype.Vest, end *prototype.Vest,iter *iservices.IDatabaseIterator) error {
    pre := AccountVestingSharesTable
    skeyList := []interface{}{pre}
    if start != nil {
       skeyList = append(skeyList,start)
    }
    sBuf,cErr := encoding.EncodeSlice(skeyList,false)
    if cErr != nil {
         return cErr
    }
    if start != nil && end == nil {
		*iter = s.Dba.NewIterator(sBuf, nil)
		return nil
	}
    eKeyList := []interface{}{pre}
    if end != nil {
       eKeyList = append(eKeyList,end)
    }
    eBuf,cErr := encoding.EncodeSlice(eKeyList,false)
    if cErr != nil {
       return cErr
    }
    
    res := bytes.Compare(sBuf,eBuf)
    if res == 0 {
		eBuf = nil
	}else if res == 1 {
       //reverse order
       return errors.New("the start and end are not order")
    }
    *iter = s.Dba.NewIterator(sBuf, eBuf)
    
    return nil
}

/////////////// SECTION Private function ////////////////

func (s *SoAccountWrap) update(sa *SoAccount) error {
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
   
   res := uniWrap.UniQueryName(sa.Name,nil)
   if res == nil {
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

func (s *UniAccountNameWrap) UniQueryName(start *prototype.AccountName,wrap *SoAccountWrap) error{
    pre := AccountNameUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueAccountByName{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap.mainKey = res.Name
            
            wrap.dba = s.Dba
			return nil  
		}
        return rErr
	}
    return err
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
   
   res := uniWrap.UniQueryPubKey(sa.PubKey,nil)
   if res == nil {
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

func (s *UniAccountPubKeyWrap) UniQueryPubKey(start *prototype.PublicKeyType,wrap *SoAccountWrap) error{
    pre := AccountPubKeyUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueAccountByPubKey{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap.mainKey = res.Name
            
            wrap.dba = s.Dba
			return nil  
		}
        return rErr
	}
    return err
}



