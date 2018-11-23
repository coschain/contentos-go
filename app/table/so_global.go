package table

import (
	"errors"

	"github.com/coschain/contentos-go/common/encoding/kope"
	"github.com/coschain/contentos-go/iservices"
	prototype "github.com/coschain/contentos-go/prototype"
	proto "github.com/golang/protobuf/proto"
)

////////////// SECTION Prefix Mark ///////////////
var (
	GlobalTable      = []byte("GlobalTable")
	GlobalIdUniTable = []byte("GlobalIdUniTable")
)

////////////// SECTION Wrap Define ///////////////
type SoGlobalWrap struct {
	dba     iservices.IDatabaseService
	mainKey *int32
}

func NewSoGlobalWrap(dba iservices.IDatabaseService, key *int32) *SoGlobalWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoGlobalWrap{dba, key}
	return result
}

func (s *SoGlobalWrap) CheckExist() bool {
	if s.dba == nil {
		return false
	}
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

func (s *SoGlobalWrap) Create(f func(tInfo *SoGlobal)) error {
	val := &SoGlobal{}
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
	if err = s.insertAllSortKeys(val); err != nil {
		s.delAllSortKeys(false, val)
		s.dba.Delete(keyBuf)
		return err
	}

	//update unique list
	if err = s.insertAllUniKeys(val); err != nil {
		s.delAllSortKeys(false, val)
		s.delAllUniKeys(false, val)
		s.dba.Delete(keyBuf)
		return err
	}

	return nil
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoGlobalWrap) delAllSortKeys(br bool, val *SoGlobal) bool {
	if s.dba == nil {
		return false
	}
	if val == nil {
		return false
	}
	res := true

	return res
}

func (s *SoGlobalWrap) insertAllSortKeys(val *SoGlobal) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoGlobal fail ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoGlobalWrap) RemoveGlobal() bool {
	if s.dba == nil {
		return false
	}
	val := s.getGlobal()
	if val == nil {
		return false
	}
	//delete sort list key
	if res := s.delAllSortKeys(true, val); !res {
		return false
	}

	//delete unique list
	if res := s.delAllUniKeys(true, val); !res {
		return false
	}

	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return false
	}
	return s.dba.Delete(keyBuf) == nil
}

////////////// SECTION Members Get/Modify ///////////////
func (s *SoGlobalWrap) GetId() int32 {
	res := s.getGlobal()

	if res == nil {
		var tmpValue int32
		return tmpValue
	}
	return res.Id
}

func (s *SoGlobalWrap) GetProps() *prototype.DynamicProperties {
	res := s.getGlobal()

	if res == nil {
		return nil

	}
	return res.Props
}

func (s *SoGlobalWrap) MdProps(p *prototype.DynamicProperties) bool {
	if s.dba == nil {
		return false
	}
	sa := s.getGlobal()
	if sa == nil {
		return false
	}

	sa.Props = p
	if !s.update(sa) {
		return false
	}

	return true
}

/////////////// SECTION Private function ////////////////

func (s *SoGlobalWrap) update(sa *SoGlobal) bool {
	if s.dba == nil {
		return false
	}
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

func (s *SoGlobalWrap) getGlobal() *SoGlobal {
	if s.dba == nil {
		return nil
	}
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return nil
	}
	resBuf, err := s.dba.Get(keyBuf)

	if err != nil {
		return nil
	}

	res := &SoGlobal{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoGlobalWrap) encodeMainKey() ([]byte, error) {
	pre := GlobalTable
	sub := s.mainKey
	if sub == nil {
		return nil, errors.New("the mainKey is nil")
	}
	kList := []interface{}{pre, sub}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
}

////////////// Unique Query delete/insert/query ///////////////

func (s *SoGlobalWrap) delAllUniKeys(br bool, val *SoGlobal) bool {
	if s.dba == nil {
		return false
	}
	if val == nil {
		return false
	}
	res := true
	if !s.delUniKeyId(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoGlobalWrap) insertAllUniKeys(val *SoGlobal) error {
	if s.dba == nil {
		return errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert uniuqe Field fail,get the SoGlobal fail ")
	}
	if !s.insertUniKeyId(val) {
		return errors.New("insert unique Field int32 while insert table ")
	}

	return nil
}

func (s *SoGlobalWrap) delUniKeyId(sa *SoGlobal) bool {
	if s.dba == nil {
		return false
	}
	pre := GlobalIdUniTable
	sub := sa.Id
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoGlobalWrap) insertUniKeyId(sa *SoGlobal) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	uniWrap := UniGlobalIdWrap{}
	uniWrap.Dba = s.dba
	res := uniWrap.UniQueryId(&sa.Id)

	if res != nil {
		//the unique key is already exist
		return false
	}
	val := SoUniqueGlobalById{}
	val.Id = sa.Id

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	pre := GlobalIdUniTable
	sub := sa.Id
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniGlobalIdWrap struct {
	Dba iservices.IDatabaseService
}

func NewUniGlobalIdWrap(db iservices.IDatabaseService) *UniGlobalIdWrap {
	if db == nil {
		return nil
	}
	wrap := UniGlobalIdWrap{Dba: db}
	return &wrap
}

func (s *UniGlobalIdWrap) UniQueryId(start *int32) *SoGlobalWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := GlobalIdUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueGlobalById{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoGlobalWrap(s.Dba, &res.Id)
			return wrap
		}
	}
	return nil
}
