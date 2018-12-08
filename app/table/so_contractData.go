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
	ContractDataTable      = []byte("ContractDataTable")
	ContractDataIdUniTable = []byte("ContractDataIdUniTable")
)

////////////// SECTION Wrap Define ///////////////
type SoContractDataWrap struct {
	dba     iservices.IDatabaseService
	mainKey *prototype.ContractDataId
}

func NewSoContractDataWrap(dba iservices.IDatabaseService, key *prototype.ContractDataId) *SoContractDataWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoContractDataWrap{dba, key}
	return result
}

func (s *SoContractDataWrap) CheckExist() bool {
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

func (s *SoContractDataWrap) Create(f func(tInfo *SoContractData)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoContractData{}
	f(val)
	if val.Id == nil {
		val.Id = s.mainKey
	}
	if s.CheckExist() {
		return errors.New("the main key is already exist")
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

func (s *SoContractDataWrap) delAllSortKeys(br bool, val *SoContractData) bool {
	if s.dba == nil || val == nil {
		return false
	}
	res := true

	return res
}

func (s *SoContractDataWrap) insertAllSortKeys(val *SoContractData) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoContractData fail ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoContractDataWrap) RemoveContractData() bool {
	if s.dba == nil {
		return false
	}
	val := s.getContractData()
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
func (s *SoContractDataWrap) GetId() *prototype.ContractDataId {
	res := s.getContractData()

	if res == nil {
		return nil

	}
	return res.Id
}

func (s *SoContractDataWrap) GetValue() []byte {
	res := s.getContractData()

	if res == nil {
		var tmpValue []byte
		return tmpValue
	}
	return res.Value
}

func (s *SoContractDataWrap) MdValue(p []byte) bool {
	if s.dba == nil {
		return false
	}
	sa := s.getContractData()
	if sa == nil {
		return false
	}

	sa.Value = p
	if !s.update(sa) {
		return false
	}

	return true
}

/////////////// SECTION Private function ////////////////

func (s *SoContractDataWrap) update(sa *SoContractData) bool {
	if s.dba == nil || sa == nil {
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

func (s *SoContractDataWrap) getContractData() *SoContractData {
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

	res := &SoContractData{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoContractDataWrap) encodeMainKey() ([]byte, error) {
	pre := ContractDataTable
	sub := s.mainKey
	if sub == nil {
		return nil, errors.New("the mainKey is nil")
	}
	kList := []interface{}{pre, sub}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
}

////////////// Unique Query delete/insert/query ///////////////

func (s *SoContractDataWrap) delAllUniKeys(br bool, val *SoContractData) bool {
	if s.dba == nil || val == nil {
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

func (s *SoContractDataWrap) insertAllUniKeys(val *SoContractData) error {
	if s.dba == nil {
		return errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert uniuqe Field fail,get the SoContractData fail ")
	}
	if !s.insertUniKeyId(val) {
		return errors.New("insert unique Field Id fail while insert table ")
	}

	return nil
}

func (s *SoContractDataWrap) delUniKeyId(sa *SoContractData) bool {
	if s.dba == nil {
		return false
	}

	if sa.Id == nil {
		return false
	}

	pre := ContractDataIdUniTable
	sub := sa.Id
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoContractDataWrap) insertUniKeyId(sa *SoContractData) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	uniWrap := UniContractDataIdWrap{}
	uniWrap.Dba = s.dba

	res := uniWrap.UniQueryId(sa.Id)
	if res != nil {
		//the unique key is already exist
		return false
	}
	val := SoUniqueContractDataById{}
	val.Id = sa.Id

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	pre := ContractDataIdUniTable
	sub := sa.Id
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniContractDataIdWrap struct {
	Dba iservices.IDatabaseService
}

func NewUniContractDataIdWrap(db iservices.IDatabaseService) *UniContractDataIdWrap {
	if db == nil {
		return nil
	}
	wrap := UniContractDataIdWrap{Dba: db}
	return &wrap
}

func (s *UniContractDataIdWrap) UniQueryId(start *prototype.ContractDataId) *SoContractDataWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := ContractDataIdUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueContractDataById{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoContractDataWrap(s.Dba, res.Id)

			return wrap
		}
	}
	return nil
}
