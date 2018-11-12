

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
    
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return false
	}
	return s.dba.Delete(keyBuf) == nil
}

////////////// SECTION Members Get/Modify ///////////////
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
	res, err := encoding.Encode(s.mainKey)

	if err != nil {
		return nil, err
	}

	return append(DynamicGlobalPropertiesTable, res...), nil
}

////////////// Unique Query delete/insert/query ///////////////


