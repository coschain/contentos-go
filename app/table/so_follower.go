

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

func (s *SoFollowerWrap) CheckExist(exi *bool) error {
	keyBuf, err := s.encodeMainKey()
	if err != nil {
        *exi = false
		return errors.New("encode the mainKey fail")
	}

	res, err := s.dba.Has(keyBuf)
	if err != nil {
        *exi = false
		return errors.New("check the db fail")
	}
    *exi = res
	return nil
}

func (s *SoFollowerWrap) CreateFollower(f func(t *SoFollower)) error {

	val := &SoFollower{}
    f(val)
    if val.FollowerInfo == nil {
       return errors.New("the mainkey is nil")
    }
    res := false
    if s.CheckExist(&res) == nil && res {
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
	
	if !s.insertSortKeyFollowerInfo(val) {
		return errors.New("insert sort Field FollowerInfo while insert table ")
	}
	
  
    //update unique list
    if !s.insertUniKeyFollowerInfo(val) {
		return errors.New("insert unique Field prototype.FollowerRelation while insert table ")
	}
	
    
	return nil
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

func (s *SoFollowerWrap) RemoveFollower() error {
	sa := s.getFollower()
	if sa == nil {
		return errors.New("delete data fail ")
	}
    //delete sort list key
	if !s.delSortKeyFollowerInfo(sa) {
		return errors.New("delete the sort key FollowerInfo fail")
	}
	
    //delete unique list
    if !s.delUniKeyFollowerInfo(sa) {
		return errors.New("delete the unique key FollowerInfo fail")
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
func (s *SoFollowerWrap) GetFollowerInfo(v **prototype.FollowerRelation) error {
	res := s.getFollower()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.FollowerInfo
   return nil
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

func (s *SFollowerFollowerInfoWrap) GetMainVal(iterator iservices.IDatabaseIterator,mKey **prototype.FollowerRelation) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}
	val, err := iterator.Value()

	if err != nil {
		return err
	}

	res := &SoListFollowerByFollowerInfo{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return err
	}
    *mKey = res.FollowerInfo
    return nil
}

func (s *SFollowerFollowerInfoWrap) GetSubVal(iterator iservices.IDatabaseIterator, sub **prototype.FollowerRelation) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}

	val, err := iterator.Value()

	if err != nil {
		return errors.New("the value of iterator is nil")
	}
	res := &SoListFollowerByFollowerInfo{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return err
	}
    *sub = res.FollowerInfo
    return nil
}

func (m *SoListFollowerByFollowerInfo) OpeEncode() ([]byte,error) {
    pre := FollowerFollowerInfoTable
    sub := m.FollowerInfo
    if sub == nil {
       return nil,errors.New("the pro FollowerInfo is nil")
    }
    sub1 := m.FollowerInfo
    if sub1 == nil {
       return nil,errors.New("the mainkey FollowerInfo is nil")
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
       return nil,errors.New("the mainkey FollowerInfo is nil")
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
func (s *SFollowerFollowerInfoWrap) QueryListByRevOrder(start *prototype.FollowerRelation, end *prototype.FollowerRelation,iter *iservices.IDatabaseIterator) error {

    var sBuf,eBuf,rBufStart,rBufEnd []byte
    pre := FollowerFollowerInfoRevOrdTable
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

func (s *SoFollowerWrap) update(sa *SoFollower) error {
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
   
   res := uniWrap.UniQueryFollowerInfo(sa.FollowerInfo,nil)
   if res == nil {
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

func (s *UniFollowerFollowerInfoWrap) UniQueryFollowerInfo(start *prototype.FollowerRelation,wrap *SoFollowerWrap) error{
    pre := FollowerFollowerInfoUniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueFollowerByFollowerInfo{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap.mainKey = res.FollowerInfo
            
            wrap.dba = s.Dba
			return nil  
		}
        return rErr
	}
    return err
}



