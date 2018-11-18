

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

func (s *SoWitnessWrap) CreateWitness(sa *SoWitness) bool {

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
	
	if !s.insertSortKeyOwner(sa) {
		return false
	}
	
  
    //update unique list
    if !s.insertUniKeyOwner(sa) {
		return false
	}
	
    
	return true
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

func (s *SoWitnessWrap) RemoveWitness() bool {
	sa := s.getWitness()
	if sa == nil {
		return false
	}
    //delete sort list key
	if !s.delSortKeyOwner(sa) {
		return false
	}
	
    //delete unique list
    if !s.delUniKeyOwner(sa) {
		return false
	}
	
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return false
	}
	return s.dba.Delete(keyBuf) == nil
}

////////////// SECTION Members Get/Modify ///////////////
func (s *SoWitnessWrap) GetCreatedTime() *prototype.TimePointSec {
	res := s.getWitness()

   if res == nil {
      return nil
      
   }
   return res.CreatedTime
}



func (s *SoWitnessWrap) MdCreatedTime(p *prototype.TimePointSec) bool {
	sa := s.getWitness()
	if sa == nil {
		return false
	}
	
    sa.CreatedTime = p
	if !s.update(sa) {
		return false
	}
    
	return true
}

func (s *SoWitnessWrap) GetLastConfirmedBlockNum() uint32 {
	res := s.getWitness()

   if res == nil {
      var tmpValue uint32 
      return tmpValue
   }
   return res.LastConfirmedBlockNum
}



func (s *SoWitnessWrap) MdLastConfirmedBlockNum(p uint32) bool {
	sa := s.getWitness()
	if sa == nil {
		return false
	}
	
    sa.LastConfirmedBlockNum = p
	if !s.update(sa) {
		return false
	}
    
	return true
}

func (s *SoWitnessWrap) GetLastWork() *prototype.Sha256 {
	res := s.getWitness()

   if res == nil {
      return nil
      
   }
   return res.LastWork
}



func (s *SoWitnessWrap) MdLastWork(p *prototype.Sha256) bool {
	sa := s.getWitness()
	if sa == nil {
		return false
	}
	
    sa.LastWork = p
	if !s.update(sa) {
		return false
	}
    
	return true
}

func (s *SoWitnessWrap) GetOwner() *prototype.AccountName {
	res := s.getWitness()

   if res == nil {
      return nil
      
   }
   return res.Owner
}


func (s *SoWitnessWrap) GetPowWorker() uint32 {
	res := s.getWitness()

   if res == nil {
      var tmpValue uint32 
      return tmpValue
   }
   return res.PowWorker
}



func (s *SoWitnessWrap) MdPowWorker(p uint32) bool {
	sa := s.getWitness()
	if sa == nil {
		return false
	}
	
    sa.PowWorker = p
	if !s.update(sa) {
		return false
	}
    
	return true
}

func (s *SoWitnessWrap) GetRunningVersion() uint32 {
	res := s.getWitness()

   if res == nil {
      var tmpValue uint32 
      return tmpValue
   }
   return res.RunningVersion
}



func (s *SoWitnessWrap) MdRunningVersion(p uint32) bool {
	sa := s.getWitness()
	if sa == nil {
		return false
	}
	
    sa.RunningVersion = p
	if !s.update(sa) {
		return false
	}
    
	return true
}

func (s *SoWitnessWrap) GetSigningKey() *prototype.PublicKeyType {
	res := s.getWitness()

   if res == nil {
      return nil
      
   }
   return res.SigningKey
}



func (s *SoWitnessWrap) MdSigningKey(p *prototype.PublicKeyType) bool {
	sa := s.getWitness()
	if sa == nil {
		return false
	}
	
    sa.SigningKey = p
	if !s.update(sa) {
		return false
	}
    
	return true
}

func (s *SoWitnessWrap) GetTotalMissed() uint32 {
	res := s.getWitness()

   if res == nil {
      var tmpValue uint32 
      return tmpValue
   }
   return res.TotalMissed
}



func (s *SoWitnessWrap) MdTotalMissed(p uint32) bool {
	sa := s.getWitness()
	if sa == nil {
		return false
	}
	
    sa.TotalMissed = p
	if !s.update(sa) {
		return false
	}
    
	return true
}

func (s *SoWitnessWrap) GetUrl() string {
	res := s.getWitness()

   if res == nil {
      var tmpValue string 
      return tmpValue
   }
   return res.Url
}



func (s *SoWitnessWrap) MdUrl(p string) bool {
	sa := s.getWitness()
	if sa == nil {
		return false
	}
	
    sa.Url = p
	if !s.update(sa) {
		return false
	}
    
	return true
}

func (s *SoWitnessWrap) GetWitnessScheduleType() *prototype.WitnessScheduleType {
	res := s.getWitness()

   if res == nil {
      return nil
      
   }
   return res.WitnessScheduleType
}



func (s *SoWitnessWrap) MdWitnessScheduleType(p *prototype.WitnessScheduleType) bool {
	sa := s.getWitness()
	if sa == nil {
		return false
	}
	
    sa.WitnessScheduleType = p
	if !s.update(sa) {
		return false
	}
    
	return true
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

func (s *SWitnessOwnerWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListWitnessByOwner{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    return res.Owner
   
}

func (s *SWitnessOwnerWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListWitnessByOwner{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
    return res.Owner
   
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
func (s *SWitnessOwnerWrap) QueryListByOrder(start *prototype.AccountName, end *prototype.AccountName) iservices.IDatabaseIterator {
    pre := WitnessOwnerTable
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

func (s *SoWitnessWrap) update(sa *SoWitness) bool {
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
   
   res := uniWrap.UniQueryOwner(sa.Owner)
   if res != nil {
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

func (s *UniWitnessOwnerWrap) UniQueryOwner(start *prototype.AccountName) *SoWitnessWrap{
    pre := WitnessOwnerUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueWitnessByOwner{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoWitnessWrap(s.Dba,res.Owner)
            
			return wrap
		}
	}
    return nil
}



