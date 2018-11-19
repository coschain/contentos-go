

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

func (s *SoTransactionObjectWrap) CheckExist() error {
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return errors.New("encode the mainKey fail")
	}

	res, err := s.dba.Has(keyBuf)
	if err != nil {
		return err
	}
    if !res {
       return errors.New("the table is already exist")
    }
	return nil
}

func (s *SoTransactionObjectWrap) CreateTransactionObject(f func(t *SoTransactionObject)) error {

	val := &SoTransactionObject{}
    f(val)
    if val.TrxId == nil {
       return errors.New("the mainkey is nil")
    }
    if s.CheckExist() == nil {
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
	
	if !s.insertSortKeyExpiration(val) {
		return errors.New("insert sort Field Expiration while insert table ")
	}
	
  
    //update unique list
    if !s.insertUniKeyTrxId(val) {
		return errors.New("insert unique Field prototype.Sha256 while insert table ")
	}
	
    
	return nil
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
    ordErr :=  s.dba.Delete(subBuf)
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

func (s *SoTransactionObjectWrap) RemoveTransactionObject() error {
	sa := s.getTransactionObject()
	if sa == nil {
		return errors.New("delete data fail ")
	}
    //delete sort list key
	if !s.delSortKeyExpiration(sa) {
		return errors.New("delete the sort key Expiration fail")
	}
	
    //delete unique list
    if !s.delUniKeyTrxId(sa) {
		return errors.New("delete the unique key TrxId fail")
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
func (s *SoTransactionObjectWrap) GetExpiration(v **prototype.TimePointSec) error {
	res := s.getTransactionObject()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Expiration
   return nil
}



func (s *SoTransactionObjectWrap) MdExpiration(p *prototype.TimePointSec) error {
	sa := s.getTransactionObject()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
	if !s.delSortKeyExpiration(sa) {
		return errors.New("delete the sort key Expiration fail")
	}
    sa.Expiration = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
    if !s.insertSortKeyExpiration(sa) {
		return errors.New("reinsert sort key Expiration fail")
    }
       
	return nil
}

func (s *SoTransactionObjectWrap) GetTrxId(v **prototype.Sha256) error {
	res := s.getTransactionObject()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.TrxId
   return nil
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

func (s *STransactionObjectExpirationWrap) GetMainVal(iterator iservices.IDatabaseIterator,mKey **prototype.Sha256) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}
	val, err := iterator.Value()

	if err != nil {
		return err
	}

	res := &SoListTransactionObjectByExpiration{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return err
	}
    *mKey = res.TrxId
    return nil
}

func (s *STransactionObjectExpirationWrap) GetSubVal(iterator iservices.IDatabaseIterator, sub **prototype.TimePointSec) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}

	val, err := iterator.Value()

	if err != nil {
		return errors.New("the value of iterator is nil")
	}
	res := &SoListTransactionObjectByExpiration{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return err
	}
    *sub = res.Expiration
    return nil
}

func (m *SoListTransactionObjectByExpiration) OpeEncode() ([]byte,error) {
    pre := TransactionObjectExpirationTable
    sub := m.Expiration
    if sub == nil {
       return nil,errors.New("the pro Expiration is nil")
    }
    sub1 := m.TrxId
    if sub1 == nil {
       return nil,errors.New("the mainkey TrxId is nil")
    }
    kList := []interface{}{pre,sub,sub1}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

func (m *SoListTransactionObjectByExpiration) EncodeRevSortKey() ([]byte,error) {
    pre := TransactionObjectExpirationRevOrdTable
    sub := m.Expiration
    if sub == nil {
       return nil,errors.New("the pro Expiration is nil")
    }
    sub1 := m.TrxId
    if sub1 == nil {
       return nil,errors.New("the mainkey TrxId is nil")
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
func (s *STransactionObjectExpirationWrap) QueryListByOrder(start *prototype.TimePointSec, end *prototype.TimePointSec,iter *iservices.IDatabaseIterator) error {
    pre := TransactionObjectExpirationTable
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

func (s *SoTransactionObjectWrap) update(sa *SoTransactionObject) error {
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
    if sub == nil {
       return nil,errors.New("the mainKey is nil")
    }
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
   
   res := uniWrap.UniQueryTrxId(sa.TrxId,nil)
   if res == nil {
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

func (s *UniTransactionObjectTrxIdWrap) UniQueryTrxId(start *prototype.Sha256,wrap *SoTransactionObjectWrap) error{
    pre := TransactionObjectTrxIdUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueTransactionObjectByTrxId{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap.mainKey = res.TrxId
            
            wrap.dba = s.Dba
			return nil  
		}
        return rErr
	}
    return err
}



