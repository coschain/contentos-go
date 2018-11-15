

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
	FollowerTable        = []byte("FollowerTable")
    FollowerFollowerInfoTable = []byte("FollowerFollowerInfoTable")
    FollowerFollowerInfoRevOrdTable = []byte("FollowerFollowerInfoRevOrdTable")
    FollowerFollowerInfoUniTable = []byte("FollowerFollowerInfoUniTable")
    )

////////////// SECTION Wrap Define ///////////////
type SoFollowerWrap struct {
	dba 		iservices.IDatabaseService
	mainKey 	*prototype.FollowerRelation
}

func NewSoFollowerWrap(dba iservices.IDatabaseService, key *prototype.FollowerRelation) *SoFollowerWrap{
	result := &SoFollowerWrap{ dba, key}
	return result
}

func (s *SoFollowerWrap) CheckExist() bool {
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

func (s *SoFollowerWrap) CreateFollower(sa *SoFollower) bool {

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
	
	if !s.insertSortKeyFollowerInfo(sa) {
		return false
	}
	
  
    //update unique list
    if !s.insertUniKeyFollowerInfo(sa) {
		return false
	}
	
    
	return true
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoFollowerWrap) delSortKeyFollowerInfo(sa *SoFollower) bool {
	val := SoListFollowerByFollowerInfo{}
	val.FollowerInfo = sa.FollowerInfo
	val.FollowerInfo = sa.FollowerInfo
    subRevBuf, err := val.EncodeRevSortKey()
	if err != nil {
		return false
	}
    revOrdErr :=  s.dba.Delete(subRevBuf) 
    return revOrdErr == nil
    
}


func (s *SoFollowerWrap) insertSortKeyFollowerInfo(sa *SoFollower) bool {
	val := SoListFollowerByFollowerInfo{}
	val.FollowerInfo = sa.FollowerInfo
	val.FollowerInfo = sa.FollowerInfo
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

func (s *SoFollowerWrap) RemoveFollower() bool {
	sa := s.getFollower()
	if sa == nil {
		return false
	}
    //delete sort list key
	if !s.delSortKeyFollowerInfo(sa) {
		return false
	}
	
    //delete unique list
    if !s.delUniKeyFollowerInfo(sa) {
		return false
	}
	
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return false
	}
	return s.dba.Delete(keyBuf) == nil
}

////////////// SECTION Members Get/Modify ///////////////
func (s *SoFollowerWrap) GetFollowerInfo() *prototype.FollowerRelation {
	res := s.getFollower()

   if res == nil {
      return nil
      
   }
   return res.FollowerInfo
}





////////////// SECTION List Keys ///////////////
type SFollowerFollowerInfoWrap struct {
	Dba iservices.IDatabaseService
}

func (s *SFollowerFollowerInfoWrap)DelIterater(iterator iservices.IDatabaseIterator){
   if iterator == nil || !iterator.Valid() {
		return 
	}
   s.Dba.DeleteIterator(iterator)
}

func (s *SFollowerFollowerInfoWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.FollowerRelation {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListFollowerByFollowerInfo{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    return res.FollowerInfo
   
}

func (s *SFollowerFollowerInfoWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.FollowerRelation {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListFollowerByFollowerInfo{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
    return res.FollowerInfo
   
}

func (m *SoListFollowerByFollowerInfo) OpeEncode() ([]byte,error) {
    pre := FollowerFollowerInfoTable
    sub := m.FollowerInfo
    if sub == nil {
       return nil,errors.New("the pro FollowerInfo is nil")
    }
    sub1 := m.FollowerInfo
    if sub1 == nil {
       return nil,errors.New("the mainKey FollowerInfo is nil")
    }
    kList := []interface{}{pre,sub,sub1}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

func (m *SoListFollowerByFollowerInfo) EncodeRevSortKey() ([]byte,error) {
    pre := FollowerFollowerInfoRevOrdTable
    sub := m.FollowerInfo
    if sub == nil {
       return nil,errors.New("the pro FollowerInfo is nil")
    }
    sub1 := m.FollowerInfo
    if sub1 == nil {
       return nil,errors.New("the mainKey FollowerInfo is nil")
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
func (s *SFollowerFollowerInfoWrap) QueryListByRevOrder(start *prototype.FollowerRelation, end *prototype.FollowerRelation) iservices.IDatabaseIterator {

    var sBuf,eBuf,rBufStart,rBufEnd []byte
    pre := FollowerFollowerInfoRevOrdTable
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

func (s *SoFollowerWrap) update(sa *SoFollower) bool {
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

func (s *SoFollowerWrap) getFollower() *SoFollower {
	keyBuf, err := s.encodeMainKey()

	if err != nil {
		return nil
	}

	resBuf, err := s.dba.Get(keyBuf)

	if err != nil {
		return nil
	}

	res := &SoFollower{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoFollowerWrap) encodeMainKey() ([]byte, error) {
    pre := FollowerTable
    sub := s.mainKey
    if sub == nil {
       return nil,errors.New("the mainKey is nil")
    }
    kList := []interface{}{pre,sub}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

////////////// Unique Query delete/insert/query ///////////////


func (s *SoFollowerWrap) delUniKeyFollowerInfo(sa *SoFollower) bool {
    pre := FollowerFollowerInfoUniTable
    sub := sa.FollowerInfo
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}


func (s *SoFollowerWrap) insertUniKeyFollowerInfo(sa *SoFollower) bool {
    uniWrap  := UniFollowerFollowerInfoWrap{}
     uniWrap.Dba = s.dba
   
   res := uniWrap.UniQueryFollowerInfo(sa.FollowerInfo)
   if res != nil {
		//the unique key is already exist
		return false
	}
    val := SoUniqueFollowerByFollowerInfo{}
    val.FollowerInfo = sa.FollowerInfo
    
	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}
    
    pre := FollowerFollowerInfoUniTable
    sub := sa.FollowerInfo
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniFollowerFollowerInfoWrap struct {
	Dba iservices.IDatabaseService
}

func (s *UniFollowerFollowerInfoWrap) UniQueryFollowerInfo(start *prototype.FollowerRelation) *SoFollowerWrap{
    pre := FollowerFollowerInfoUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueFollowerByFollowerInfo{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoFollowerWrap(s.Dba,res.FollowerInfo)
            
			return wrap
		}
	}
    return nil
}



