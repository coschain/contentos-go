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
	ContractTable            = []byte("ContractTable")
	ContractCreatedTimeTable = []byte("ContractCreatedTimeTable")
	ContractIdUniTable       = []byte("ContractIdUniTable")
)

////////////// SECTION Wrap Define ///////////////
type SoContractWrap struct {
	dba     iservices.IDatabaseService
	mainKey *prototype.ContractId
}

func NewSoContractWrap(dba iservices.IDatabaseService, key *prototype.ContractId) *SoContractWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoContractWrap{dba, key}
	return result
}

func (s *SoContractWrap) CheckExist() bool {
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

func (s *SoContractWrap) Create(f func(tInfo *SoContract)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoContract{}
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

func (s *SoContractWrap) delSortKeyCreatedTime(sa *SoContract) bool {
	if s.dba == nil {
		return false
	}
	val := SoListContractByCreatedTime{}
	val.CreatedTime = sa.CreatedTime
	val.Id = sa.Id
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoContractWrap) insertSortKeyCreatedTime(sa *SoContract) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListContractByCreatedTime{}
	val.Id = sa.Id
	val.CreatedTime = sa.CreatedTime
	buf, err := proto.Marshal(&val)
	if err != nil {
		return false
	}
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Put(subBuf, buf)
	return ordErr == nil
}

func (s *SoContractWrap) delAllSortKeys(br bool, val *SoContract) bool {
	if s.dba == nil || val == nil {
		return false
	}
	res := true
	if !s.delSortKeyCreatedTime(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoContractWrap) insertAllSortKeys(val *SoContract) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoContract fail ")
	}
	if !s.insertSortKeyCreatedTime(val) {
		return errors.New("insert sort Field CreatedTime fail while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoContractWrap) RemoveContract() bool {
	if s.dba == nil {
		return false
	}
	val := s.getContract()
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
func (s *SoContractWrap) GetAbi() string {
	res := s.getContract()

	if res == nil {
		var tmpValue string
		return tmpValue
	}
	return res.Abi
}

func (s *SoContractWrap) MdAbi(p string) bool {
	if s.dba == nil {
		return false
	}
	sa := s.getContract()
	if sa == nil {
		return false
	}

	sa.Abi = p
	if !s.update(sa) {
		return false
	}

	return true
}

func (s *SoContractWrap) GetBalance() *prototype.Coin {
	res := s.getContract()

	if res == nil {
		return nil

	}
	return res.Balance
}

func (s *SoContractWrap) MdBalance(p *prototype.Coin) bool {
	if s.dba == nil {
		return false
	}
	sa := s.getContract()
	if sa == nil {
		return false
	}

	sa.Balance = p
	if !s.update(sa) {
		return false
	}

	return true
}

func (s *SoContractWrap) GetCode() []byte {
	res := s.getContract()

	if res == nil {
		var tmpValue []byte
		return tmpValue
	}
	return res.Code
}

func (s *SoContractWrap) MdCode(p []byte) bool {
	if s.dba == nil {
		return false
	}
	sa := s.getContract()
	if sa == nil {
		return false
	}

	sa.Code = p
	if !s.update(sa) {
		return false
	}

	return true
}

func (s *SoContractWrap) GetCreatedTime() *prototype.TimePointSec {
	res := s.getContract()

	if res == nil {
		return nil

	}
	return res.CreatedTime
}

func (s *SoContractWrap) MdCreatedTime(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}
	sa := s.getContract()
	if sa == nil {
		return false
	}

	if !s.delSortKeyCreatedTime(sa) {
		return false
	}
	sa.CreatedTime = p
	if !s.update(sa) {
		return false
	}

	if !s.insertSortKeyCreatedTime(sa) {
		return false
	}

	return true
}

func (s *SoContractWrap) GetId() *prototype.ContractId {
	res := s.getContract()

	if res == nil {
		return nil

	}
	return res.Id
}

////////////// SECTION List Keys ///////////////
type SContractCreatedTimeWrap struct {
	Dba iservices.IDatabaseService
}

func NewContractCreatedTimeWrap(db iservices.IDatabaseService) *SContractCreatedTimeWrap {
	if db == nil {
		return nil
	}
	wrap := SContractCreatedTimeWrap{Dba: db}
	return &wrap
}

func (s *SContractCreatedTimeWrap) DelIterater(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SContractCreatedTimeWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.ContractId {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListContractByCreatedTime{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Id

}

func (s *SContractCreatedTimeWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.TimePointSec {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListContractByCreatedTime{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.CreatedTime

}

func (m *SoListContractByCreatedTime) OpeEncode() ([]byte, error) {
	pre := ContractCreatedTimeTable
	sub := m.CreatedTime
	if sub == nil {
		return nil, errors.New("the pro CreatedTime is nil")
	}
	sub1 := m.Id
	if sub1 == nil {
		return nil, errors.New("the mainkey Id is nil")
	}
	kList := []interface{}{pre, sub, sub1}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
}

//Query sort by order
//start = nil  end = nil (query the db from start to end)
//start = nil (query from start the db)
//end = nil (query to the end of db)
func (s *SContractCreatedTimeWrap) QueryListByOrder(start *prototype.TimePointSec, end *prototype.TimePointSec) iservices.IDatabaseIterator {
	if s.Dba == nil {
		return nil
	}
	pre := ContractCreatedTimeTable
	skeyList := []interface{}{pre}
	if start != nil {
		skeyList = append(skeyList, start)
	}
	sBuf, cErr := kope.EncodeSlice(skeyList)
	if cErr != nil {
		return nil
	}
	eKeyList := []interface{}{pre}
	if end != nil {
		eKeyList = append(eKeyList, end)
	} else {
		eKeyList = append(eKeyList, kope.MaximumKey)
	}
	eBuf, cErr := kope.EncodeSlice(eKeyList)
	if cErr != nil {
		return nil
	}
	return s.Dba.NewIterator(sBuf, eBuf)
}

/////////////// SECTION Private function ////////////////

func (s *SoContractWrap) update(sa *SoContract) bool {
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

func (s *SoContractWrap) getContract() *SoContract {
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

	res := &SoContract{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoContractWrap) encodeMainKey() ([]byte, error) {
	pre := ContractTable
	sub := s.mainKey
	if sub == nil {
		return nil, errors.New("the mainKey is nil")
	}
	kList := []interface{}{pre, sub}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
}

////////////// Unique Query delete/insert/query ///////////////

func (s *SoContractWrap) delAllUniKeys(br bool, val *SoContract) bool {
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

func (s *SoContractWrap) insertAllUniKeys(val *SoContract) error {
	if s.dba == nil {
		return errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert uniuqe Field fail,get the SoContract fail ")
	}
	if !s.insertUniKeyId(val) {
		return errors.New("insert unique Field Id fail while insert table ")
	}

	return nil
}

func (s *SoContractWrap) delUniKeyId(sa *SoContract) bool {
	if s.dba == nil {
		return false
	}

	if sa.Id == nil {
		return false
	}

	pre := ContractIdUniTable
	sub := sa.Id
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoContractWrap) insertUniKeyId(sa *SoContract) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	uniWrap := UniContractIdWrap{}
	uniWrap.Dba = s.dba

	res := uniWrap.UniQueryId(sa.Id)
	if res != nil {
		//the unique key is already exist
		return false
	}
	val := SoUniqueContractById{}
	val.Id = sa.Id

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	pre := ContractIdUniTable
	sub := sa.Id
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniContractIdWrap struct {
	Dba iservices.IDatabaseService
}

func NewUniContractIdWrap(db iservices.IDatabaseService) *UniContractIdWrap {
	if db == nil {
		return nil
	}
	wrap := UniContractIdWrap{Dba: db}
	return &wrap
}

func (s *UniContractIdWrap) UniQueryId(start *prototype.ContractId) *SoContractWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := ContractIdUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueContractById{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoContractWrap(s.Dba, res.Id)

			return wrap
		}
	}
	return nil
}
