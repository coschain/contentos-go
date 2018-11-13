

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
	mainBuf, err := encoding.Encode(m.TrxId)
	if err != nil {
		return nil,err
	}
	subBuf, err := encoding.Encode(m.Expiration)
	if err != nil {
		return nil,err
	}
   ordKey := append(append(TransactionObjectExpirationTable, subBuf...), mainBuf...)
   return ordKey,nil
}

func (m *SoListTransactionObjectByExpiration) EncodeRevSortKey() ([]byte,error) {
    mainBuf, err := encoding.Encode(m.TrxId)
	if err != nil {
		return nil,err
	}
	subBuf, err := encoding.Encode(m.Expiration)
	if err != nil {
		return nil,err
	}
    ordKey := append(append(TransactionObjectExpirationRevOrdTable, subBuf...), mainBuf...)
    revKey,revRrr := encoding.Complement(ordKey, err)
	if revRrr != nil {
        return nil,revRrr
	}
    return revKey,nil
}

//Query sort by order 
func (s *STransactionObjectExpirationWrap) QueryListByOrder(start prototype.TimePointSec, end prototype.TimePointSec) iservices.IDatabaseIterator {

	startBuf, err := encoding.Encode(&start)
	if err != nil {
		return nil
	}
	endBuf, err := encoding.Encode(&end)
	if err != nil {
		return nil
	}
    bufStartkey := append(TransactionObjectExpirationTable, startBuf...)
	bufEndkey := append(TransactionObjectExpirationTable, endBuf...)
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
	res, err := encoding.Encode(s.mainKey)

	if err != nil {
		return nil, err
	}

	return append(TransactionObjectTable, res...), nil
}

////////////// Unique Query delete/insert/query ///////////////


func (s *SoTransactionObjectWrap) delUniKeyTrxId(sa *SoTransactionObject) bool {
	val := SoUniqueTransactionObjectByTrxId{}

	val.TrxId = sa.TrxId
    key, err := encoding.Encode(sa.TrxId)

	if err != nil {
		return false
	}

	return s.dba.Delete(append(TransactionObjectTrxIdUniTable,key...)) == nil
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

	key, err := encoding.Encode(sa.TrxId)

	if err != nil {
		return false
	}
	return s.dba.Put(append(TransactionObjectTrxIdUniTable,key...), buf) == nil

}

type UniTransactionObjectTrxIdWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniTransactionObjectTrxIdWrap) UniQueryTrxId(start *prototype.Sha256) *SoTransactionObjectWrap{

   startBuf, err := encoding.Encode(start)
	if err != nil {
		return nil
	}
	bufStartkey := append(TransactionObjectTrxIdUniTable, startBuf...)
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



