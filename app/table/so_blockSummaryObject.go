

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
	BlockSummaryObjectTable        = []byte("BlockSummaryObjectTable")
    BlockSummaryObjectIdUniTable = []byte("BlockSummaryObjectIdUniTable")
    )

////////////// SECTION Wrap Define ///////////////
type SoBlockSummaryObjectWrap struct {
	dba 		iservices.IDatabaseService
	mainKey 	*uint32
}

func NewSoBlockSummaryObjectWrap(dba iservices.IDatabaseService, key *uint32) *SoBlockSummaryObjectWrap{
	result := &SoBlockSummaryObjectWrap{ dba, key}
	return result
}

func (s *SoBlockSummaryObjectWrap) CheckExist() error {
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

func (s *SoBlockSummaryObjectWrap) CreateBlockSummaryObject(f func(t *SoBlockSummaryObject)) error {

	val := &SoBlockSummaryObject{}
    f(val)
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
	
  
    //update unique list
    if !s.insertUniKeyId(val) {
		return errors.New("insert unique Field uint32 while insert table ")
	}
	
    
	return nil
}

////////////// SECTION LKeys delete/insert ///////////////

////////////// SECTION LKeys delete/insert //////////////

func (s *SoBlockSummaryObjectWrap) RemoveBlockSummaryObject() error {
	sa := s.getBlockSummaryObject()
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
func (s *SoBlockSummaryObjectWrap) GetBlockId(v **prototype.Sha256) error {
	res := s.getBlockSummaryObject()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.BlockId
   return nil
}



func (s *SoBlockSummaryObjectWrap) MdBlockId(p *prototype.Sha256) error {
	sa := s.getBlockSummaryObject()
	if sa == nil {
		return errors.New("initialization data failed")
	}
	
    sa.BlockId = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    
	return nil
}

func (s *SoBlockSummaryObjectWrap) GetId(v *uint32) error {
	res := s.getBlockSummaryObject()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.Id
   return nil
}




/////////////// SECTION Private function ////////////////

func (s *SoBlockSummaryObjectWrap) update(sa *SoBlockSummaryObject) error {
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

func (s *SoBlockSummaryObjectWrap) getBlockSummaryObject() *SoBlockSummaryObject {
	keyBuf, err := s.encodeMainKey()

	if err != nil {
		return nil
	}

	resBuf, err := s.dba.Get(keyBuf)

	if err != nil {
		return nil
	}

	res := &SoBlockSummaryObject{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoBlockSummaryObjectWrap) encodeMainKey() ([]byte, error) {
    pre := BlockSummaryObjectTable
    sub := s.mainKey
    if sub == nil {
       return nil,errors.New("the mainKey is nil")
    }
    kList := []interface{}{pre,sub}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

////////////// Unique Query delete/insert/query ///////////////


func (s *SoBlockSummaryObjectWrap) delUniKeyId(sa *SoBlockSummaryObject) bool {
    pre := BlockSummaryObjectIdUniTable
    sub := sa.Id
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}


func (s *SoBlockSummaryObjectWrap) insertUniKeyId(sa *SoBlockSummaryObject) bool {
    uniWrap  := UniBlockSummaryObjectIdWrap{}
     uniWrap.Dba = s.dba
   res := uniWrap.UniQueryId(&sa.Id,nil)
   
   if res == nil {
		//the unique key is already exist
		return false
	}
    val := SoUniqueBlockSummaryObjectById{}
    val.Id = sa.Id
    
	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}
    
    pre := BlockSummaryObjectIdUniTable
    sub := sa.Id
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniBlockSummaryObjectIdWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniBlockSummaryObjectIdWrap) UniQueryId(start *uint32,wrap *SoBlockSummaryObjectWrap) error{
    pre := BlockSummaryObjectIdUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueBlockSummaryObjectById{}
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



