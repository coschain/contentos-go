

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
    BlockSummaryObjectBlockIdUniTable = []byte("BlockSummaryObjectBlockIdUniTable")
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

func (s *SoBlockSummaryObjectWrap) CheckExist() bool {
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

func (s *SoBlockSummaryObjectWrap) CreateBlockSummaryObject(sa *SoBlockSummaryObject) bool {

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
    if !s.insertUniKeyBlockId(sa) {
		return false
	}
	if !s.insertUniKeyId(sa) {
		return false
	}
	
    
	return true
}

////////////// SECTION LKeys delete/insert ///////////////

////////////// SECTION LKeys delete/insert //////////////

func (s *SoBlockSummaryObjectWrap) RemoveBlockSummaryObject() bool {
	sa := s.getBlockSummaryObject()
	if sa == nil {
		return false
	}
    //delete sort list key
	
    //delete unique list
    if !s.delUniKeyBlockId(sa) {
		return false
	}
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
func (s *SoBlockSummaryObjectWrap) GetBlockId() *prototype.Sha256 {
	res := s.getBlockSummaryObject()

   if res == nil {
      return nil
      
   }
   return res.BlockId
}



func (s *SoBlockSummaryObjectWrap) MdBlockId(p prototype.Sha256) bool {
	sa := s.getBlockSummaryObject()
	if sa == nil {
		return false
	}
    //judge the unique value if is exist
    uniWrap  := UniBlockSummaryObjectBlockIdWrap{}
   res := uniWrap.UniQueryBlockId(sa.BlockId)
   
	if res != nil {
		//the unique value to be modified is already exist
		return false
	}
	if !s.delUniKeyBlockId(sa) {
		return false
	}
    
	
   
   sa.BlockId = &p
   
	if !s.update(sa) {
		return false
	}
    
    if !s.insertUniKeyBlockId(sa) {
		return false
    }
	return true
}

func (s *SoBlockSummaryObjectWrap) GetId() uint32 {
	res := s.getBlockSummaryObject()

   if res == nil {
      var tmpValue uint32 
      return tmpValue
   }
   return res.Id
}




/////////////// SECTION Private function ////////////////

func (s *SoBlockSummaryObjectWrap) update(sa *SoBlockSummaryObject) bool {
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


func (s *SoBlockSummaryObjectWrap) delUniKeyBlockId(sa *SoBlockSummaryObject) bool {
    pre := BlockSummaryObjectBlockIdUniTable
    sub := sa.BlockId
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}


func (s *SoBlockSummaryObjectWrap) insertUniKeyBlockId(sa *SoBlockSummaryObject) bool {
    uniWrap  := UniBlockSummaryObjectBlockIdWrap{}
     uniWrap.Dba = s.dba
   
   res := uniWrap.UniQueryBlockId(sa.BlockId)
   if res != nil {
		//the unique key is already exist
		return false
	}
    val := SoUniqueBlockSummaryObjectByBlockId{}
    val.Id = sa.Id
    val.BlockId = sa.BlockId
    
	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}
    
    pre := BlockSummaryObjectBlockIdUniTable
    sub := sa.BlockId
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniBlockSummaryObjectBlockIdWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniBlockSummaryObjectBlockIdWrap) UniQueryBlockId(start *prototype.Sha256) *SoBlockSummaryObjectWrap{
    pre := BlockSummaryObjectBlockIdUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueBlockSummaryObjectByBlockId{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoBlockSummaryObjectWrap(s.Dba,&res.Id)
			return wrap
		}
	}
    return nil
}



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
   res := uniWrap.UniQueryId(&sa.Id)
   
   if res != nil {
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

func (s *UniBlockSummaryObjectIdWrap) UniQueryId(start *uint32) *SoBlockSummaryObjectWrap{
    pre := BlockSummaryObjectIdUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueBlockSummaryObjectById{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoBlockSummaryObjectWrap(s.Dba,&res.Id)
			return wrap
		}
	}
    return nil
}



