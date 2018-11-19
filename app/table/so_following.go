

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

func (s *SoFollowingWrap) CheckExist() error {
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

func (s *SoFollowingWrap) CreateFollowing(f func(t *SoFollowing)) error {

	val := &SoFollowing{}
    f(val)
    if val.FollowingInfo == nil {
       return errors.New("the mainkey is nil")
    }
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
	
	if !s.insertSortKeyFollowingInfo(val) {
		return errors.New("insert sort Field FollowingInfo while insert table ")
	}
	
  
    //update unique list
    if !s.insertUniKeyFollowingInfo(val) {
		return errors.New("insert unique Field prototype.FollowingRelation while insert table ")
	}
	
    
	return nil
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

func (s *SoFollowingWrap) RemoveFollowing() error {
	sa := s.getFollowing()
	if sa == nil {
		return errors.New("delete data fail ")
	}
    //delete sort list key
	if !s.delSortKeyFollowingInfo(sa) {
		return errors.New("delete the sort key FollowingInfo fail")
	}
	
    //delete unique list
    if !s.delUniKeyFollowingInfo(sa) {
		return errors.New("delete the unique key FollowingInfo fail")
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
func (s *SoFollowingWrap) GetFollowingInfo(v **prototype.FollowingRelation) error {
	res := s.getFollowing()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.FollowingInfo
   return nil
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

func (s *SFollowingFollowingInfoWrap) GetMainVal(iterator iservices.IDatabaseIterator,mKey **prototype.FollowingRelation) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}
	val, err := iterator.Value()

	if err != nil {
		return err
	}

	res := &SoListFollowingByFollowingInfo{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return err
	}
    *mKey = res.FollowingInfo
    return nil
}

func (s *SFollowingFollowingInfoWrap) GetSubVal(iterator iservices.IDatabaseIterator, sub **prototype.FollowingRelation) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}

	val, err := iterator.Value()

	if err != nil {
		return errors.New("the value of iterator is nil")
	}
	res := &SoListFollowingByFollowingInfo{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return err
	}
    *sub = res.FollowingInfo
    return nil
}

func (m *SoListFollowingByFollowingInfo) OpeEncode() ([]byte,error) {
    pre := FollowingFollowingInfoTable
    sub := m.FollowingInfo
    if sub == nil {
       return nil,errors.New("the pro FollowingInfo is nil")
    }
    sub1 := m.FollowingInfo
    if sub1 == nil {
       return nil,errors.New("the mainkey FollowingInfo is nil")
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
       return nil,errors.New("the mainkey FollowingInfo is nil")
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
func (s *SFollowingFollowingInfoWrap) QueryListByRevOrder(start *prototype.FollowingRelation, end *prototype.FollowingRelation,iter *iservices.IDatabaseIterator) error {

    var sBuf,eBuf,rBufStart,rBufEnd []byte
    pre := FollowingFollowingInfoRevOrdTable
    if start != nil {
       skeyList := []interface{}{pre,start}
       buf,cErr := encoding.EncodeSlice(skeyList,false)
       if cErr != nil {
         return cErr
       }
       sBuf = buf
    }
    
    if end != nil {
       eKeyList := []interface{}{pre,end}
       buf,err := encoding.EncodeSlice(eKeyList,false)
       if err != nil {
          return err
       }
       eBuf = buf

    }

    if sBuf != nil && eBuf != nil {
       res := bytes.Compare(sBuf,eBuf)
       if res == -1 {
          // order
          return errors.New("the start and end are not reverse order")
       }
       if sBuf != nil {
       rBuf,rErr := encoding.Complement(sBuf, nil)
       if rErr != nil {
          return rErr
       }
       rBufStart = rBuf
    }
    if eBuf != nil {
          rBuf,rErr := encoding.Complement(eBuf, nil)
          if rErr != nil { 
            return rErr
          }
          rBufEnd = rBuf
       }
    }
    *iter = s.Dba.NewIterator(rBufStart, rBufEnd)
    return nil
}
/////////////// SECTION Private function ////////////////

func (s *SoFollowingWrap) update(sa *SoFollowing) error {
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
   
   res := uniWrap.UniQueryFollowingInfo(sa.FollowingInfo,nil)
   if res == nil {
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

func (s *UniFollowingFollowingInfoWrap) UniQueryFollowingInfo(start *prototype.FollowingRelation,wrap *SoFollowingWrap) error{
    pre := FollowingFollowingInfoUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueFollowingByFollowingInfo{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap.mainKey = res.FollowingInfo
            
            wrap.dba = s.Dba
			return nil  
		}
        return rErr
	}
    return err
}



