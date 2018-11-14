

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
	TransactionObjectTable        = []byte("TransactionObjectTable")
    TransactionObjectExpirationTable = []byte("TransactionObjectExpirationTable")
    TransactionObjectExpirationRevOrdTable = []byte("TransactionObjectExpirationRevOrdTable")
    TransactionObjectTrxIdUniTable = []byte("TransactionObjectTrxIdUniTable")
    )

////////////// SECTION Wrap Define ///////////////
type SoTransactionObjectWrap struct {
	dba 		iservices.IDatabaseService
	mainKey 	*prototype.Sha256
}

func NewSoTransactionObjectWrap(dba iservices.IDatabaseService, key *prototype.Sha256) *SoTransactionObjectWrap{
	result := &SoTransactionObjectWrap{ dba, key}
	return result
}

func (s *SoTransactionObjectWrap) CheckExist() bool {
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

func (s *SoTransactionObjectWrap) CreateTransactionObject(sa *SoTransactionObject) bool {

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
	
	if !s.insertSortKeyExpiration(sa) {
		return false
	}
	
  
    //update unique list
    if !s.insertUniKeyTrxId(sa) {
		return false
	}
	
    
	return true
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoTransactionObjectWrap) delSortKeyExpiration(sa *SoTransactionObject) bool {
	val := SoListTransactionObjectByExpiration{}
	val.Expiration = sa.Expiration
	val.TrxId = sa.TrxId
    subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
    ordKey := append(TransactionObjectExpirationTable, subBuf...)
    ordErr :=  s.dba.Delete(ordKey)
    return ordErr == nil
    
}


func (s *SoTransactionObjectWrap) insertSortKeyExpiration(sa *SoTransactionObject) bool {
	val := SoListTransactionObjectByExpiration{}
	val.TrxId = sa.TrxId
	val.Expiration = sa.Expiration
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

func (s *SoTransactionObjectWrap) RemoveTransactionObject() bool {
	sa := s.getTransactionObject()
	if sa == nil {
		return false
	}
    //delete sort list key
	if !s.delSortKeyExpiration(sa) {
		return false
	}
	
    //delete unique list
    if !s.delUniKeyTrxId(sa) {
		return false
	}
	
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return false
	}
	return s.dba.Delete(keyBuf) == nil
}

////////////// SECTION Members Get/Modify ///////////////
func (s *SoTransactionObjectWrap) GetExpiration() *prototype.TimePointSec {
	res := s.getTransactionObject()

   if res == nil {
      return nil
      
   }
   return res.Expiration
}



func (s *SoTransactionObjectWrap) MdExpiration(p prototype.TimePointSec) bool {
	sa := s.getTransactionObject()
	if sa == nil {
		return false
	}
	
	if !s.delSortKeyExpiration(sa) {
		return false
	}
   
   sa.Expiration = &p
   
	if !s.update(sa) {
		return false
	}
    
    if !s.insertSortKeyExpiration(sa) {
		return false
    }
       
	return true
}

func (s *SoTransactionObjectWrap) GetTrxId() *prototype.Sha256 {
	res := s.getTransactionObject()

   if res == nil {
      return nil
      
   }
   return res.TrxId
}





////////////// SECTION List Keys ///////////////
type STransactionObjectExpirationWrap struct {
	Dba iservices.IDatabaseService
}

func (s *STransactionObjectExpirationWrap)DelIterater(iterator iservices.IDatabaseIterator){
   if iterator == nil || !iterator.Valid() {
		return 
	}
   s.Dba.DeleteIterator(iterator)
}

func (s *STransactionObjectExpirationWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.Sha256 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListTransactionObjectByExpiration{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    
   return res.TrxId
   

}

func (s *STransactionObjectExpirationWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.TimePointSec {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListTransactionObjectByExpiration{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    
   
    
   return res.Expiration
   
}

func (m *SoListTransactionObjectByExpiration) OpeEncode() ([]byte,error) {
    pre := TransactionObjectExpirationTable
    sub := m.Expiration
    sub1 := m.TrxId
    kList := []interface{}{pre,sub,sub1}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

func (m *SoListTransactionObjectByExpiration) EncodeRevSortKey() ([]byte,error) {
    pre := TransactionObjectExpirationRevOrdTable
    sub := m.Expiration
    sub1 := m.TrxId
    kList := []interface{}{pre,sub,sub1}
    ordKey,cErr := encoding.EncodeSlice(kList,false)
    revKey,revRrr := encoding.Complement(ordKey, cErr)
    return revKey,revRrr
}

//Query sort by order 
//start = nil  end = nil (query the db from start to end)
//start = nil (query from start the db)
//end = nil (query to the end of db)
func (s *STransactionObjectExpirationWrap) QueryListByOrder(start *prototype.TimePointSec, end *prototype.TimePointSec) iservices.IDatabaseIterator {
    pre := TransactionObjectExpirationRevOrdTable
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

func (s *SoTransactionObjectWrap) update(sa *SoTransactionObject) bool {
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

func (s *SoTransactionObjectWrap) getTransactionObject() *SoTransactionObject {
	keyBuf, err := s.encodeMainKey()

	if err != nil {
		return nil
	}

	resBuf, err := s.dba.Get(keyBuf)

	if err != nil {
		return nil
	}

	res := &SoTransactionObject{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoTransactionObjectWrap) encodeMainKey() ([]byte, error) {
    pre := TransactionObjectTable
    sub := s.mainKey
    kList := []interface{}{pre,sub}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

////////////// Unique Query delete/insert/query ///////////////


func (s *SoTransactionObjectWrap) delUniKeyTrxId(sa *SoTransactionObject) bool {
    pre := TransactionObjectTrxIdUniTable
    sub := sa.TrxId
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}


func (s *SoTransactionObjectWrap) insertUniKeyTrxId(sa *SoTransactionObject) bool {
    uniWrap  := UniTransactionObjectTrxIdWrap{}
     uniWrap.Dba = s.dba
   
   res := uniWrap.UniQueryTrxId(sa.TrxId)
   if res != nil {
		//the unique key is already exist
		return false
	}
    val := SoUniqueTransactionObjectByTrxId{}
    val.TrxId = sa.TrxId
    
	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}
    
    pre := TransactionObjectTrxIdUniTable
    sub := sa.TrxId
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniTransactionObjectTrxIdWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniTransactionObjectTrxIdWrap) UniQueryTrxId(start *prototype.Sha256) *SoTransactionObjectWrap{
    pre := TransactionObjectTrxIdUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueTransactionObjectByTrxId{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoTransactionObjectWrap(s.Dba,res.TrxId)
            
			return wrap
		}
	}
    return nil
}



