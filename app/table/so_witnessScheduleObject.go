

package table

import (
     "errors"
     "github.com/coschain/contentos-go/common/encoding"
	 "github.com/gogo/protobuf/proto"
     "github.com/coschain/contentos-go/iservices"
)

////////////// SECTION Prefix Mark ///////////////
var (
	WitnessScheduleObjectTable        = []byte("WitnessScheduleObjectTable")
    WitnessScheduleObjectIdUniTable = []byte("WitnessScheduleObjectIdUniTable")
    )

////////////// SECTION Wrap Define ///////////////
type SoWitnessScheduleObjectWrap struct {
	dba 		iservices.IDatabaseService
	mainKey 	*int32
}

func NewSoWitnessScheduleObjectWrap(dba iservices.IDatabaseService, key *int32) *SoWitnessScheduleObjectWrap{
	result := &SoWitnessScheduleObjectWrap{ dba, key}
	return result
}

func (s *SoWitnessScheduleObjectWrap) CheckExist() bool {
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

func (s *SoWitnessScheduleObjectWrap) CreateWitnessScheduleObject(sa *SoWitnessScheduleObject) bool {

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

func (s *SoWitnessScheduleObjectWrap) RemoveWitnessScheduleObject() bool {
	sa := s.getWitnessScheduleObject()
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
func (s *SoWitnessScheduleObjectWrap) GetCurrentShuffledWitness() []string {
	res := s.getWitnessScheduleObject()

   if res == nil {
      var tmpValue []string 
      return tmpValue
   }
   return res.CurrentShuffledWitness
}



func (s *SoWitnessScheduleObjectWrap) MdCurrentShuffledWitness(p []string) bool {
	sa := s.getWitnessScheduleObject()
	if sa == nil {
		return false
	}
	
    sa.CurrentShuffledWitness = p
	if !s.update(sa) {
		return false
	}
    
	return true
}

func (s *SoWitnessScheduleObjectWrap) GetId() int32 {
	res := s.getWitnessScheduleObject()

   if res == nil {
      var tmpValue int32 
      return tmpValue
   }
   return res.Id
}




/////////////// SECTION Private function ////////////////

func (s *SoWitnessScheduleObjectWrap) update(sa *SoWitnessScheduleObject) bool {
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

func (s *SoWitnessScheduleObjectWrap) getWitnessScheduleObject() *SoWitnessScheduleObject {
	keyBuf, err := s.encodeMainKey()

	if err != nil {
		return nil
	}

	resBuf, err := s.dba.Get(keyBuf)

	if err != nil {
		return nil
	}

	res := &SoWitnessScheduleObject{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoWitnessScheduleObjectWrap) encodeMainKey() ([]byte, error) {
    pre := WitnessScheduleObjectTable
    sub := s.mainKey
    if sub == nil {
       return nil,errors.New("the mainKey is nil")
    }
    kList := []interface{}{pre,sub}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

////////////// Unique Query delete/insert/query ///////////////


func (s *SoWitnessScheduleObjectWrap) delUniKeyId(sa *SoWitnessScheduleObject) bool {
    pre := WitnessScheduleObjectIdUniTable
    sub := sa.Id
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}


func (s *SoWitnessScheduleObjectWrap) insertUniKeyId(sa *SoWitnessScheduleObject) bool {
    uniWrap  := UniWitnessScheduleObjectIdWrap{}
     uniWrap.Dba = s.dba
   res := uniWrap.UniQueryId(&sa.Id)
   
   if res != nil {
		//the unique key is already exist
		return false
	}
    val := SoUniqueWitnessScheduleObjectById{}
    val.Id = sa.Id
    
	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}
    
    pre := WitnessScheduleObjectIdUniTable
    sub := sa.Id
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniWitnessScheduleObjectIdWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniWitnessScheduleObjectIdWrap) UniQueryId(start *int32) *SoWitnessScheduleObjectWrap{
    pre := WitnessScheduleObjectIdUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueWitnessScheduleObjectById{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoWitnessScheduleObjectWrap(s.Dba,&res.Id)
			return wrap
		}
	}
    return nil
}



