

package table

import (
     "github.com/coschain/contentos-go/common/encoding"
     "github.com/coschain/contentos-go/prototype"
	 "github.com/gogo/protobuf/proto"
     "github.com/coschain/contentos-go/iservices"
)

////////////// SECTION Prefix Mark ///////////////
var (
	DynamicGlobalPropertiesTable        = []byte("DynamicGlobalPropertiesTable")
    DynamicGlobalPropertiesIdUniTable = []byte("DynamicGlobalPropertiesIdUniTable")
    )

////////////// SECTION Wrap Define ///////////////
type SoDynamicGlobalPropertiesWrap struct {
	dba 		iservices.IDatabaseService
	mainKey 	*int32
}

func NewSoDynamicGlobalPropertiesWrap(dba iservices.IDatabaseService, key *int32) *SoDynamicGlobalPropertiesWrap{
	result := &SoDynamicGlobalPropertiesWrap{ dba, key}
	return result
}

func (s *SoDynamicGlobalPropertiesWrap) CheckExist() bool {
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

func (s *SoDynamicGlobalPropertiesWrap) CreateDynamicGlobalProperties(sa *SoDynamicGlobalProperties) bool {

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
	
  
    //update unique list
    if !s.insertUniKeyId(sa) {
		return false
	}
	
    
	return true
}

////////////// SECTION LKeys delete/insert ///////////////

////////////// SECTION LKeys delete/insert //////////////

func (s *SoDynamicGlobalPropertiesWrap) RemoveDynamicGlobalProperties() bool {
	sa := s.getDynamicGlobalProperties()
	if sa == nil {
		return false
	}
    //delete sort list key
	
    //delete unique list
    if !s.delUniKeyId(sa) {
		return false
	}
	
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return false
	}
	return s.dba.Delete(keyBuf) == nil
}

////////////// SECTION Members Get/Modify ///////////////
func (s *SoDynamicGlobalPropertiesWrap) GetCurrentSupply() *prototype.Coin {
	res := s.getDynamicGlobalProperties()

   if res == nil {
      return nil
      
   }
   return res.CurrentSupply
}



func (s *SoDynamicGlobalPropertiesWrap) MdCurrentSupply(p prototype.Coin) bool {
	sa := s.getDynamicGlobalProperties()
	if sa == nil {
		return false
	}
	
   
   sa.CurrentSupply = &p
   
	if !s.update(sa) {
		return false
	}
    
	return true
}

func (s *SoDynamicGlobalPropertiesWrap) GetCurrentWitness() *prototype.AccountName {
	res := s.getDynamicGlobalProperties()

   if res == nil {
      return nil
      
   }
   return res.CurrentWitness
}



func (s *SoDynamicGlobalPropertiesWrap) MdCurrentWitness(p prototype.AccountName) bool {
	sa := s.getDynamicGlobalProperties()
	if sa == nil {
		return false
	}
	
   
   sa.CurrentWitness = &p
   
	if !s.update(sa) {
		return false
	}
    
	return true
}

func (s *SoDynamicGlobalPropertiesWrap) GetHeadBlockId() *prototype.Sha256 {
	res := s.getDynamicGlobalProperties()

   if res == nil {
      return nil
      
   }
   return res.HeadBlockId
}



func (s *SoDynamicGlobalPropertiesWrap) MdHeadBlockId(p prototype.Sha256) bool {
	sa := s.getDynamicGlobalProperties()
	if sa == nil {
		return false
	}
	
   
   sa.HeadBlockId = &p
   
	if !s.update(sa) {
		return false
	}
    
	return true
}

func (s *SoDynamicGlobalPropertiesWrap) GetHeadBlockNumber() uint32 {
	res := s.getDynamicGlobalProperties()

   if res == nil {
      var tmpValue uint32 
      return tmpValue
   }
   return res.HeadBlockNumber
}



func (s *SoDynamicGlobalPropertiesWrap) MdHeadBlockNumber(p uint32) bool {
	sa := s.getDynamicGlobalProperties()
	if sa == nil {
		return false
	}
	
   sa.HeadBlockNumber = p
   
   
	if !s.update(sa) {
		return false
	}
    
	return true
}

func (s *SoDynamicGlobalPropertiesWrap) GetId() int32 {
	res := s.getDynamicGlobalProperties()

   if res == nil {
      var tmpValue int32 
      return tmpValue
   }
   return res.Id
}


func (s *SoDynamicGlobalPropertiesWrap) GetIrreversibleBlockNum() uint32 {
	res := s.getDynamicGlobalProperties()

   if res == nil {
      var tmpValue uint32 
      return tmpValue
   }
   return res.IrreversibleBlockNum
}



func (s *SoDynamicGlobalPropertiesWrap) MdIrreversibleBlockNum(p uint32) bool {
	sa := s.getDynamicGlobalProperties()
	if sa == nil {
		return false
	}
	
   sa.IrreversibleBlockNum = p
   
   
	if !s.update(sa) {
		return false
	}
    
	return true
}

func (s *SoDynamicGlobalPropertiesWrap) GetMaximumBlockSize() uint32 {
	res := s.getDynamicGlobalProperties()

   if res == nil {
      var tmpValue uint32 
      return tmpValue
   }
   return res.MaximumBlockSize
}



func (s *SoDynamicGlobalPropertiesWrap) MdMaximumBlockSize(p uint32) bool {
	sa := s.getDynamicGlobalProperties()
	if sa == nil {
		return false
	}
	
   sa.MaximumBlockSize = p
   
   
	if !s.update(sa) {
		return false
	}
    
	return true
}

func (s *SoDynamicGlobalPropertiesWrap) GetTime() *prototype.TimePointSec {
	res := s.getDynamicGlobalProperties()

   if res == nil {
      return nil
      
   }
   return res.Time
}



func (s *SoDynamicGlobalPropertiesWrap) MdTime(p prototype.TimePointSec) bool {
	sa := s.getDynamicGlobalProperties()
	if sa == nil {
		return false
	}
	
   
   sa.Time = &p
   
	if !s.update(sa) {
		return false
	}
    
	return true
}

func (s *SoDynamicGlobalPropertiesWrap) GetTotalCos() *prototype.Coin {
	res := s.getDynamicGlobalProperties()

   if res == nil {
      return nil
      
   }
   return res.TotalCos
}



func (s *SoDynamicGlobalPropertiesWrap) MdTotalCos(p prototype.Coin) bool {
	sa := s.getDynamicGlobalProperties()
	if sa == nil {
		return false
	}
	
   
   sa.TotalCos = &p
   
	if !s.update(sa) {
		return false
	}
    
	return true
}

func (s *SoDynamicGlobalPropertiesWrap) GetTotalVestingShares() *prototype.Vest {
	res := s.getDynamicGlobalProperties()

   if res == nil {
      return nil
      
   }
   return res.TotalVestingShares
}



func (s *SoDynamicGlobalPropertiesWrap) MdTotalVestingShares(p prototype.Vest) bool {
	sa := s.getDynamicGlobalProperties()
	if sa == nil {
		return false
	}
	
   
   sa.TotalVestingShares = &p
   
	if !s.update(sa) {
		return false
	}
    
	return true
}

func (s *SoDynamicGlobalPropertiesWrap) GetTps() uint32 {
	res := s.getDynamicGlobalProperties()

   if res == nil {
      var tmpValue uint32 
      return tmpValue
   }
   return res.Tps
}



func (s *SoDynamicGlobalPropertiesWrap) MdTps(p uint32) bool {
	sa := s.getDynamicGlobalProperties()
	if sa == nil {
		return false
	}
	
   sa.Tps = p
   
   
	if !s.update(sa) {
		return false
	}
    
	return true
}



/////////////// SECTION Private function ////////////////

func (s *SoDynamicGlobalPropertiesWrap) update(sa *SoDynamicGlobalProperties) bool {
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

func (s *SoDynamicGlobalPropertiesWrap) getDynamicGlobalProperties() *SoDynamicGlobalProperties {
	keyBuf, err := s.encodeMainKey()

	if err != nil {
		return nil
	}

	resBuf, err := s.dba.Get(keyBuf)

	if err != nil {
		return nil
	}

	res := &SoDynamicGlobalProperties{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoDynamicGlobalPropertiesWrap) encodeMainKey() ([]byte, error) {
    pre := DynamicGlobalPropertiesTable
    sub := s.mainKey
    kList := []interface{}{pre,sub}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

////////////// Unique Query delete/insert/query ///////////////


func (s *SoDynamicGlobalPropertiesWrap) delUniKeyId(sa *SoDynamicGlobalProperties) bool {
    pre := DynamicGlobalPropertiesIdUniTable
    sub := sa.Id
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}


func (s *SoDynamicGlobalPropertiesWrap) insertUniKeyId(sa *SoDynamicGlobalProperties) bool {
    uniWrap  := UniDynamicGlobalPropertiesIdWrap{}
     uniWrap.Dba = s.dba
   res := uniWrap.UniQueryId(&sa.Id)
   
   if res != nil {
		//the unique key is already exist
		return false
	}
    val := SoUniqueDynamicGlobalPropertiesById{}
    val.Id = sa.Id
    
	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}
    
    pre := DynamicGlobalPropertiesIdUniTable
    sub := sa.Id
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniDynamicGlobalPropertiesIdWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniDynamicGlobalPropertiesIdWrap) UniQueryId(start *int32) *SoDynamicGlobalPropertiesWrap{
    pre := DynamicGlobalPropertiesIdUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueDynamicGlobalPropertiesById{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoDynamicGlobalPropertiesWrap(s.Dba,&res.Id)
			return wrap
		}
	}
    return nil
}



