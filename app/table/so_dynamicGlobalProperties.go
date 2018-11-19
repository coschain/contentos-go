

package table

import (
     "errors"
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

func (s *SoDynamicGlobalPropertiesWrap) CreateDynamicGlobalProperties(f func(t *SoDynamicGlobalProperties)) error {

	val := &SoDynamicGlobalProperties{}
    f(val)
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
	
  
    //update unique list
    if !s.insertUniKeyId(val) {
		return err
	}
	
    
	return nil
}

////////////// SECTION LKeys delete/insert ///////////////

////////////// SECTION LKeys delete/insert //////////////

func (s *SoDynamicGlobalPropertiesWrap) RemoveDynamicGlobalProperties() error {
	sa := s.getDynamicGlobalProperties()
	if sa == nil {
		return errors.New("delete data fail ")
	}
    //delete sort list key
	
    //delete unique list
    if !s.delUniKeyId(sa) {
		return errors.New("delete the unique key Id fail")
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
func (s *SoDynamicGlobalPropertiesWrap) GetCurrentSupply(v **prototype.Coin) error {
	res := s.getDynamicGlobalProperties()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.CurrentSupply
   return nil
}



func (s *SoDynamicGlobalPropertiesWrap) MdCurrentSupply(p *prototype.Coin) error {
	sa := s.getDynamicGlobalProperties()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.CurrentSupply = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoDynamicGlobalPropertiesWrap) GetCurrentWitness(v **prototype.AccountName) error {
	res := s.getDynamicGlobalProperties()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.CurrentWitness
   return nil
}



func (s *SoDynamicGlobalPropertiesWrap) MdCurrentWitness(p *prototype.AccountName) error {
	sa := s.getDynamicGlobalProperties()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.CurrentWitness = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoDynamicGlobalPropertiesWrap) GetHeadBlockId(v **prototype.Sha256) error {
	res := s.getDynamicGlobalProperties()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.HeadBlockId
   return nil
}



func (s *SoDynamicGlobalPropertiesWrap) MdHeadBlockId(p *prototype.Sha256) error {
	sa := s.getDynamicGlobalProperties()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.HeadBlockId = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoDynamicGlobalPropertiesWrap) GetHeadBlockNumber(v *uint32) error {
	res := s.getDynamicGlobalProperties()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.HeadBlockNumber
   return nil
}



func (s *SoDynamicGlobalPropertiesWrap) MdHeadBlockNumber(p uint32) error {
	sa := s.getDynamicGlobalProperties()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.HeadBlockNumber = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoDynamicGlobalPropertiesWrap) GetId(v *int32) error {
	res := s.getDynamicGlobalProperties()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Id
   return nil
}


func (s *SoDynamicGlobalPropertiesWrap) GetIrreversibleBlockNum(v *uint32) error {
	res := s.getDynamicGlobalProperties()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.IrreversibleBlockNum
   return nil
}



func (s *SoDynamicGlobalPropertiesWrap) MdIrreversibleBlockNum(p uint32) error {
	sa := s.getDynamicGlobalProperties()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.IrreversibleBlockNum = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoDynamicGlobalPropertiesWrap) GetMaximumBlockSize(v *uint32) error {
	res := s.getDynamicGlobalProperties()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.MaximumBlockSize
   return nil
}



func (s *SoDynamicGlobalPropertiesWrap) MdMaximumBlockSize(p uint32) error {
	sa := s.getDynamicGlobalProperties()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.MaximumBlockSize = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoDynamicGlobalPropertiesWrap) GetTime(v **prototype.TimePointSec) error {
	res := s.getDynamicGlobalProperties()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Time
   return nil
}



func (s *SoDynamicGlobalPropertiesWrap) MdTime(p *prototype.TimePointSec) error {
	sa := s.getDynamicGlobalProperties()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.Time = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoDynamicGlobalPropertiesWrap) GetTotalCos(v **prototype.Coin) error {
	res := s.getDynamicGlobalProperties()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.TotalCos
   return nil
}



func (s *SoDynamicGlobalPropertiesWrap) MdTotalCos(p *prototype.Coin) error {
	sa := s.getDynamicGlobalProperties()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.TotalCos = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoDynamicGlobalPropertiesWrap) GetTotalVestingShares(v **prototype.Vest) error {
	res := s.getDynamicGlobalProperties()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.TotalVestingShares
   return nil
}



func (s *SoDynamicGlobalPropertiesWrap) MdTotalVestingShares(p *prototype.Vest) error {
	sa := s.getDynamicGlobalProperties()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.TotalVestingShares = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoDynamicGlobalPropertiesWrap) GetTps(v *uint32) error {
	res := s.getDynamicGlobalProperties()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Tps
   return nil
}



func (s *SoDynamicGlobalPropertiesWrap) MdTps(p uint32) error {
	sa := s.getDynamicGlobalProperties()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.Tps = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}



/////////////// SECTION Private function ////////////////

func (s *SoDynamicGlobalPropertiesWrap) update(sa *SoDynamicGlobalProperties) error {
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
    if sub == nil {
       return nil,errors.New("the mainKey is nil")
    }
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
   res := uniWrap.UniQueryId(&sa.Id,nil)
   
   if res == nil {
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

func (s *UniDynamicGlobalPropertiesIdWrap) UniQueryId(start *int32,wrap *SoDynamicGlobalPropertiesWrap) error{
    pre := DynamicGlobalPropertiesIdUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueDynamicGlobalPropertiesById{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap.mainKey = &res.Id
            wrap.dba = s.Dba
			return nil  
		}
        return rErr
	}
    return err
}



