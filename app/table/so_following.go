

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
	FollowingTable        = []byte("FollowingTable")
    FollowingFollowingInfoTable = []byte("FollowingFollowingInfoTable")
    FollowingFollowingInfoRevOrdTable = []byte("FollowingFollowingInfoRevOrdTable")
    FollowingFollowingInfoUniTable = []byte("FollowingFollowingInfoUniTable")
    )

////////////// SECTION Wrap Define ///////////////
type SoFollowingWrap struct {
	dba 		iservices.IDatabaseService
	mainKey 	*prototype.FollowingRelation
}

func NewSoFollowingWrap(dba iservices.IDatabaseService, key *prototype.FollowingRelation) *SoFollowingWrap{
	result := &SoFollowingWrap{ dba, key}
	return result
}

func (s *SoFollowingWrap) CheckExist() bool {
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

func (s *SoFollowingWrap) CreateFollowing(sa *SoFollowing) bool {

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
	
	if !s.insertSortKeyFollowingInfo(sa) {
		return false
	}
	
  
    //update unique list
    if !s.insertUniKeyFollowingInfo(sa) {
		return false
	}
	
    
	return true
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoFollowingWrap) delSortKeyFollowingInfo(sa *SoFollowing) bool {
	val := SoListFollowingByFollowingInfo{}
	val.FollowingInfo = sa.FollowingInfo
	val.FollowingInfo = sa.FollowingInfo
    subRevBuf, err := val.EncodeRevSortKey()
	if err != nil {
		return false
	}
    revOrdErr :=  s.dba.Delete(subRevBuf) 
    return revOrdErr == nil
    
}


func (s *SoFollowingWrap) insertSortKeyFollowingInfo(sa *SoFollowing) bool {
	val := SoListFollowingByFollowingInfo{}
	val.FollowingInfo = sa.FollowingInfo
	val.FollowingInfo = sa.FollowingInfo
	buf, err := proto.Marshal(&val)
	if err != nil {
		return false
	}
    subRevBuf, err := val.EncodeRevSortKey()
	if err != nil {
		return false
	}
    revOrdErr :=  s.dba.Put(subRevBuf, buf) 
    return revOrdErr == nil
    
}


////////////// SECTION LKeys delete/insert //////////////

func (s *SoFollowingWrap) RemoveFollowing() bool {
	sa := s.getFollowing()
	if sa == nil {
		return false
	}
    //delete sort list key
	if !s.delSortKeyFollowingInfo(sa) {
		return false
	}
	
    //delete unique list
    if !s.delUniKeyFollowingInfo(sa) {
		return false
	}
	
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return false
	}
	return s.dba.Delete(keyBuf) == nil
}

////////////// SECTION Members Get/Modify ///////////////
func (s *SoFollowingWrap) GetFollowingInfo() *prototype.FollowingRelation {
	res := s.getFollowing()

   if res == nil {
      return nil
      
   }
   return res.FollowingInfo
}





////////////// SECTION List Keys ///////////////
type SFollowingFollowingInfoWrap struct {
	Dba iservices.IDatabaseService
}

func (s *SFollowingFollowingInfoWrap)DelIterater(iterator iservices.IDatabaseIterator){
   if iterator == nil || !iterator.Valid() {
		return 
	}
   s.Dba.DeleteIterator(iterator)
}

func (s *SFollowingFollowingInfoWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.FollowingRelation {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListFollowingByFollowingInfo{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    
   return res.FollowingInfo
   

}

func (s *SFollowingFollowingInfoWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.FollowingRelation {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListFollowingByFollowingInfo{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    
   
    
   return res.FollowingInfo
   
}

func (m *SoListFollowingByFollowingInfo) OpeEncode() ([]byte,error) {
    pre := FollowingFollowingInfoTable
    sub := m.FollowingInfo
    if sub == nil {
       return nil,errors.New("the pro FollowingInfo is nil")
    }
    sub1 := m.FollowingInfo
    if sub1 == nil {
       return nil,errors.New("the mainKey FollowingInfo is nil")
    }
    kList := []interface{}{pre,sub,sub1}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

func (m *SoListFollowingByFollowingInfo) EncodeRevSortKey() ([]byte,error) {
    pre := FollowingFollowingInfoRevOrdTable
    sub := m.FollowingInfo
    if sub == nil {
       return nil,errors.New("the pro FollowingInfo is nil")
    }
    sub1 := m.FollowingInfo
    if sub1 == nil {
       return nil,errors.New("the mainKey FollowingInfo is nil")
    }
    kList := []interface{}{pre,sub,sub1}
    ordKey,cErr := encoding.EncodeSlice(kList,false)
    if cErr != nil {
       return nil,cErr
    }
    revKey,revRrr := encoding.Complement(ordKey, cErr)
    return revKey,revRrr
}


//Query sort by reverse order 
func (s *SFollowingFollowingInfoWrap) QueryListByRevOrder(start *prototype.FollowingRelation, end *prototype.FollowingRelation) iservices.IDatabaseIterator {

    var sBuf,eBuf,rBufStart,rBufEnd []byte
    pre := FollowingFollowingInfoRevOrdTable
    if start != nil {
       skeyList := []interface{}{pre,start}
       buf,cErr := encoding.EncodeSlice(skeyList,false)
       if cErr != nil {
         return nil
       }
       sBuf = buf
    }
    
    if end != nil {
       eKeyList := []interface{}{pre,end}
       buf,err := encoding.EncodeSlice(eKeyList,false)
       if err != nil {
          return nil
       }
       eBuf = buf

    }

    if sBuf != nil && eBuf != nil {
       res := bytes.Compare(sBuf,eBuf)
       if res == -1 {
          // order
          return nil
       }
       if sBuf != nil {
       rBuf,rErr := encoding.Complement(sBuf, nil)
       if rErr != nil {
          return nil
       }
       rBufStart = rBuf
    }
       if eBuf != nil {
          rBuf,rErr := encoding.Complement(eBuf, nil)
          if rErr != nil { 
            return nil
          }
          rBufEnd = rBuf
       }
    }
     
       if sBuf != nil && eBuf != nil {
          res := bytes.Compare(sBuf,eBuf)
          if res == -1 {
            // order
            return nil
        }
    }
    iter := s.Dba.NewIterator(rBufStart, rBufEnd)
    return iter
}
/////////////// SECTION Private function ////////////////

func (s *SoFollowingWrap) update(sa *SoFollowing) bool {
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

func (s *SoFollowingWrap) getFollowing() *SoFollowing {
	keyBuf, err := s.encodeMainKey()

	if err != nil {
		return nil
	}

	resBuf, err := s.dba.Get(keyBuf)

	if err != nil {
		return nil
	}

	res := &SoFollowing{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoFollowingWrap) encodeMainKey() ([]byte, error) {
    pre := FollowingTable
    sub := s.mainKey
    if sub == nil {
       return nil,errors.New("the mainKey is nil")
    }
    kList := []interface{}{pre,sub}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

////////////// Unique Query delete/insert/query ///////////////


func (s *SoFollowingWrap) delUniKeyFollowingInfo(sa *SoFollowing) bool {
    pre := FollowingFollowingInfoUniTable
    sub := sa.FollowingInfo
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}


func (s *SoFollowingWrap) insertUniKeyFollowingInfo(sa *SoFollowing) bool {
    uniWrap  := UniFollowingFollowingInfoWrap{}
     uniWrap.Dba = s.dba
   
   res := uniWrap.UniQueryFollowingInfo(sa.FollowingInfo)
   if res != nil {
		//the unique key is already exist
		return false
	}
    val := SoUniqueFollowingByFollowingInfo{}
    val.FollowingInfo = sa.FollowingInfo
    
	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}
    
    pre := FollowingFollowingInfoUniTable
    sub := sa.FollowingInfo
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniFollowingFollowingInfoWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniFollowingFollowingInfoWrap) UniQueryFollowingInfo(start *prototype.FollowingRelation) *SoFollowingWrap{
    pre := FollowingFollowingInfoUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueFollowingByFollowingInfo{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoFollowingWrap(s.Dba,res.FollowingInfo)
            
			return wrap
		}
	}
    return nil
}



