

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
	WitnessTable        = []byte("WitnessTable")
    WitnessOwnerTable = []byte("WitnessOwnerTable")
    WitnessOwnerRevOrdTable = []byte("WitnessOwnerRevOrdTable")
    WitnessOwnerUniTable = []byte("WitnessOwnerUniTable")
    )

////////////// SECTION Wrap Define ///////////////
type SoWitnessWrap struct {
	dba 		iservices.IDatabaseService
	mainKey 	*prototype.AccountName
}

func NewSoWitnessWrap(dba iservices.IDatabaseService, key *prototype.AccountName) *SoWitnessWrap{
	result := &SoWitnessWrap{ dba, key}
	return result
}

func (s *SoWitnessWrap) CheckExist() bool {
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

func (s *SoWitnessWrap) CreateWitness(f func(t *SoWitness)) error {

	val := &SoWitness{}
    f(val)
    if val.Owner == nil {
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
	
	if !s.insertSortKeyOwner(val) {
		return err
	}
	
  
    //update unique list
    if !s.insertUniKeyOwner(val) {
		return err
	}
	
    
	return nil
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoWitnessWrap) delSortKeyOwner(sa *SoWitness) bool {
	val := SoListWitnessByOwner{}
	val.Owner = sa.Owner
	val.Owner = sa.Owner
    subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
    ordErr :=  s.dba.Delete(subBuf)
    return ordErr == nil
    
}


func (s *SoWitnessWrap) insertSortKeyOwner(sa *SoWitness) bool {
	val := SoListWitnessByOwner{}
    val.Owner = sa.Owner
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

func (s *SoWitnessWrap) RemoveWitness() error {
	sa := s.getWitness()
	if sa == nil {
		return errors.New("delete data fail ")
	}
    //delete sort list key
	if !s.delSortKeyOwner(sa) {
		return errors.New("delete the sort key Owner fail")
	}
	
    //delete unique list
    if !s.delUniKeyOwner(sa) {
		return errors.New("delete the unique key Owner fail")
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
func (s *SoWitnessWrap) GetCreatedTime(v **prototype.TimePointSec) error {
	res := s.getWitness()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.CreatedTime
   return nil
}



func (s *SoWitnessWrap) MdCreatedTime(p prototype.TimePointSec) error {
	sa := s.getWitness()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
   
   sa.CreatedTime = &p
   
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoWitnessWrap) GetLastConfirmedBlockNum(v *uint32) error {
	res := s.getWitness()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.LastConfirmedBlockNum
   return nil
}



func (s *SoWitnessWrap) MdLastConfirmedBlockNum(p uint32) error {
	sa := s.getWitness()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
   sa.LastConfirmedBlockNum = p
   
   
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoWitnessWrap) GetLastWork(v **prototype.Sha256) error {
	res := s.getWitness()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.LastWork
   return nil
}



func (s *SoWitnessWrap) MdLastWork(p prototype.Sha256) error {
	sa := s.getWitness()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
   
   sa.LastWork = &p
   
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoWitnessWrap) GetOwner(v **prototype.AccountName) error {
	res := s.getWitness()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Owner
   return nil
}


func (s *SoWitnessWrap) GetPowWorker(v *uint32) error {
	res := s.getWitness()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.PowWorker
   return nil
}



func (s *SoWitnessWrap) MdPowWorker(p uint32) error {
	sa := s.getWitness()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
   sa.PowWorker = p
   
   
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoWitnessWrap) GetRunningVersion(v *uint32) error {
	res := s.getWitness()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.RunningVersion
   return nil
}



func (s *SoWitnessWrap) MdRunningVersion(p uint32) error {
	sa := s.getWitness()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
   sa.RunningVersion = p
   
   
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoWitnessWrap) GetSigningKey(v **prototype.PublicKeyType) error {
	res := s.getWitness()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.SigningKey
   return nil
}



func (s *SoWitnessWrap) MdSigningKey(p prototype.PublicKeyType) error {
	sa := s.getWitness()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
   
   sa.SigningKey = &p
   
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoWitnessWrap) GetTotalMissed(v *uint32) error {
	res := s.getWitness()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.TotalMissed
   return nil
}



func (s *SoWitnessWrap) MdTotalMissed(p uint32) error {
	sa := s.getWitness()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
   sa.TotalMissed = p
   
   
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoWitnessWrap) GetUrl(v *string) error {
	res := s.getWitness()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Url
   return nil
}



func (s *SoWitnessWrap) MdUrl(p string) error {
	sa := s.getWitness()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
   sa.Url = p
   
   
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoWitnessWrap) GetWitnessScheduleType(v **prototype.WitnessScheduleType) error {
	res := s.getWitness()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.WitnessScheduleType
   return nil
}



func (s *SoWitnessWrap) MdWitnessScheduleType(p prototype.WitnessScheduleType) error {
	sa := s.getWitness()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
   
   sa.WitnessScheduleType = &p
   
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}




////////////// SECTION List Keys ///////////////
type SWitnessOwnerWrap struct {
	Dba iservices.IDatabaseService
}

func (s *SWitnessOwnerWrap)DelIterater(iterator iservices.IDatabaseIterator){
   if iterator == nil || !iterator.Valid() {
		return 
	}
   s.Dba.DeleteIterator(iterator)
}

func (s *SWitnessOwnerWrap) GetMainVal(iterator iservices.IDatabaseIterator,mKey **prototype.AccountName) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}
	val, err := iterator.Value()

	if err != nil {
		return errors.New("the value of iterator is nil")
	}

	res := &SoListWitnessByOwner{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return err
	}
    *mKey = res.Owner
    return nil
}

func (s *SWitnessOwnerWrap) GetSubVal(iterator iservices.IDatabaseIterator, sub **prototype.AccountName) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}

	val, err := iterator.Value()

	if err != nil {
		return errors.New("the value of iterator is nil")
	}
	res := &SoListWitnessByOwner{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return err
	}
    *sub = res.Owner
    return nil
}

func (m *SoListWitnessByOwner) OpeEncode() ([]byte,error) {
    pre := WitnessOwnerTable
    sub := m.Owner
    if sub == nil {
       return nil,errors.New("the pro Owner is nil")
    }
    sub1 := m.Owner
    if sub1 == nil {
       return nil,errors.New("the mainkey Owner is nil")
    }
    kList := []interface{}{pre,sub,sub1}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

func (m *SoListWitnessByOwner) EncodeRevSortKey() ([]byte,error) {
    pre := WitnessOwnerRevOrdTable
    sub := m.Owner
    if sub == nil {
       return nil,errors.New("the pro Owner is nil")
    }
    sub1 := m.Owner
    if sub1 == nil {
       return nil,errors.New("the mainkey Owner is nil")
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
func (s *SWitnessOwnerWrap) QueryListByOrder(start *prototype.AccountName, end *prototype.AccountName,iter *iservices.IDatabaseIterator) error {
    pre := WitnessOwnerTable
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

func (s *SoWitnessWrap) update(sa *SoWitness) error {
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

func (s *SoWitnessWrap) getWitness() *SoWitness {
	keyBuf, err := s.encodeMainKey()

	if err != nil {
		return nil
	}

	resBuf, err := s.dba.Get(keyBuf)

	if err != nil {
		return nil
	}

	res := &SoWitness{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoWitnessWrap) encodeMainKey() ([]byte, error) {
    pre := WitnessTable
    sub := s.mainKey
    if sub == nil {
       return nil,errors.New("the mainKey is nil")
    }
    kList := []interface{}{pre,sub}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

////////////// Unique Query delete/insert/query ///////////////


func (s *SoWitnessWrap) delUniKeyOwner(sa *SoWitness) bool {
    pre := WitnessOwnerUniTable
    sub := sa.Owner
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}


func (s *SoWitnessWrap) insertUniKeyOwner(sa *SoWitness) bool {
    uniWrap  := UniWitnessOwnerWrap{}
     uniWrap.Dba = s.dba
   
   res := uniWrap.UniQueryOwner(sa.Owner,nil)
   if res == nil {
		//the unique key is already exist
		return false
	}
    val := SoUniqueWitnessByOwner{}
    val.Owner = sa.Owner
    
	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}
    
    pre := WitnessOwnerUniTable
    sub := sa.Owner
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniWitnessOwnerWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniWitnessOwnerWrap) UniQueryOwner(start *prototype.AccountName,wrap *SoWitnessWrap) error{
    pre := WitnessOwnerUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueWitnessByOwner{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap.mainKey = res.Owner
            
            wrap.dba = s.Dba
			return nil  
		}
        return rErr
	}
    return err
}



