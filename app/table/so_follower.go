

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

func (s *SoFollowerWrap) Create(f func(tInfo *SoFollower)) error {
    val := &SoFollower{}
    f(val)
    if val.FollowerInfo == nil {
       return errors.New("the mainkey is nil")
    }
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
    subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
    ordErr :=  s.dba.Delete(subBuf)
    return ordErr == nil
}


func (s *SoFollowerWrap) insertSortKeyFollowerInfo(sa *SoFollower) bool {
	val := SoListFollowerByFollowerInfo{}
    val.FollowerInfo = sa.FollowerInfo
	buf, err := proto.Marshal(&val)
	if err != nil {
		return false
	}
    subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
    ordErr :=  s.dba.Put(subBuf, buf) 
    return ordErr == nil
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
       return nil,errors.New("the mainkey FollowerInfo is nil")
    }
    kList := []interface{}{pre,sub,sub1}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

//Query sort by order 
//start = nil  end = nil (query the db from start to end)
//start = nil (query from start the db)
//end = nil (query to the end of db)
func (s *SFollowerFollowerInfoWrap) QueryListByOrder(start *prototype.FollowerRelation, end *prototype.FollowerRelation) iservices.IDatabaseIterator {
    pre := FollowerFollowerInfoTable
    skeyList := []interface{}{pre}
    if start != nil {
       skeyList = append(skeyList,start)
    }
    sBuf,cErr := encoding.EncodeSlice(skeyList,false)
    if cErr != nil {
         return nil
    }
    if start != nil && end == nil {
		iter := s.Dba.NewIterator(sBuf, nil)
		return iter
	}
    eKeyList := []interface{}{pre}
    if end != nil {
       eKeyList = append(eKeyList,end)
    }
    eBuf,cErr := encoding.EncodeSlice(eKeyList,false)
    if cErr != nil {
       return nil
    }
    
    res := bytes.Compare(sBuf,eBuf)
    if res == 0 {
		eBuf = nil
	}else if res == 1 {
       //reverse order
       return nil
    }
    iter := s.Dba.NewIterator(sBuf, eBuf)
    
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



