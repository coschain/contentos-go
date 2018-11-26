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
	TransactionObjectTable           = []byte("TransactionObjectTable")
	TransactionObjectExpirationTable = []byte("TransactionObjectExpirationTable")
	TransactionObjectTrxIdUniTable   = []byte("TransactionObjectTrxIdUniTable")
)

////////////// SECTION Wrap Define ///////////////
type SoTransactionObjectWrap struct {
	dba     iservices.IDatabaseService
	mainKey *prototype.Sha256
}

func NewSoTransactionObjectWrap(dba iservices.IDatabaseService, key *prototype.Sha256) *SoTransactionObjectWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoTransactionObjectWrap{dba, key}
	return result
}

func (s *SoTransactionObjectWrap) CheckExist() bool {
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

func (s *SoTransactionObjectWrap) Create(f func(tInfo *SoTransactionObject)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoTransactionObject{}
	f(val)
	if val.TrxId == nil {
		val.TrxId = s.mainKey
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

func (s *SoTransactionObjectWrap) delSortKeyExpiration(sa *SoTransactionObject) bool {
	if s.dba == nil {
		return false
	}
	val := SoListTransactionObjectByExpiration{}
	val.Expiration = sa.Expiration
	val.TrxId = sa.TrxId
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoTransactionObjectWrap) insertSortKeyExpiration(sa *SoTransactionObject) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListTransactionObjectByExpiration{}
	val.TrxId = sa.TrxId
	val.Expiration = sa.Expiration
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

func (s *SoTransactionObjectWrap) delAllSortKeys(br bool, val *SoTransactionObject) bool {
	if s.dba == nil || val == nil {
		return false
	}
	res := true
	if !s.delSortKeyExpiration(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoTransactionObjectWrap) insertAllSortKeys(val *SoTransactionObject) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoTransactionObject fail ")
	}
	if !s.insertSortKeyExpiration(val) {
		return errors.New("insert sort Field Expiration fail while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoTransactionObjectWrap) RemoveTransactionObject() bool {
	if s.dba == nil {
		return false
	}
	val := s.getTransactionObject()
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
func (s *SoTransactionObjectWrap) GetExpiration() *prototype.TimePointSec {
	res := s.getTransactionObject()

	if res == nil {
		return nil

	}
	return res.Expiration
}

func (s *SoTransactionObjectWrap) MdExpiration(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}
	sa := s.getTransactionObject()
	if sa == nil {
		return false
	}

	if !s.delSortKeyExpiration(sa) {
		return false
	}
	sa.Expiration = p
	if !s.update(sa) {
		return false
	}

	if !s.insertSortKeyExpiration(sa) {
		return false
	}

	return true
}

func (s *SoTransactionObjectWrap) GetTrxId() *prototype.Sha256 {
	res := s.getTransactionObject()

	if res == nil {
		return nil

	}
	return res.TrxId
}

////////////// SECTION List Keys ///////////////
type STransactionObjectExpirationWrap struct {
	Dba iservices.IDatabaseService
}

func NewTransactionObjectExpirationWrap(db iservices.IDatabaseService) *STransactionObjectExpirationWrap {
	if db == nil {
		return nil
	}
	wrap := STransactionObjectExpirationWrap{Dba: db}
	return &wrap
}

func (s *STransactionObjectExpirationWrap) DelIterater(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *STransactionObjectExpirationWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.Sha256 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListTransactionObjectByExpiration{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.TrxId

}

func (s *STransactionObjectExpirationWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.TimePointSec {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListTransactionObjectByExpiration{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.Expiration

}

func (m *SoListTransactionObjectByExpiration) OpeEncode() ([]byte, error) {
	pre := TransactionObjectExpirationTable
	sub := m.Expiration
	if sub == nil {
		return nil, errors.New("the pro Expiration is nil")
	}
	sub1 := m.TrxId
	if sub1 == nil {
		return nil, errors.New("the mainkey TrxId is nil")
	}
	kList := []interface{}{pre, sub, sub1}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
}

//Query sort by order
//start = nil  end = nil (query the db from start to end)
//start = nil (query from start the db)
//end = nil (query to the end of db)
func (s *STransactionObjectExpirationWrap) QueryListByOrder(start *prototype.TimePointSec, end *prototype.TimePointSec) iservices.IDatabaseIterator {
	if s.Dba == nil {
		return nil
	}
	pre := TransactionObjectExpirationTable
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

func (s *SoTransactionObjectWrap) update(sa *SoTransactionObject) bool {
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

func (s *SoTransactionObjectWrap) getTransactionObject() *SoTransactionObject {
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

	res := &SoTransactionObject{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoTransactionObjectWrap) encodeMainKey() ([]byte, error) {
	pre := TransactionObjectTable
	sub := s.mainKey
	if sub == nil {
		return nil, errors.New("the mainKey is nil")
	}
	kList := []interface{}{pre, sub}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
}

////////////// Unique Query delete/insert/query ///////////////

func (s *SoTransactionObjectWrap) delAllUniKeys(br bool, val *SoTransactionObject) bool {
	if s.dba == nil || val == nil {
		return false
	}
	res := true
	if !s.delUniKeyTrxId(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoTransactionObjectWrap) insertAllUniKeys(val *SoTransactionObject) error {
	if s.dba == nil {
		return errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert uniuqe Field fail,get the SoTransactionObject fail ")
	}
	if !s.insertUniKeyTrxId(val) {
		return errors.New("insert unique Field TrxId fail while insert table ")
	}

	return nil
}

func (s *SoTransactionObjectWrap) delUniKeyTrxId(sa *SoTransactionObject) bool {
	if s.dba == nil {
		return false
	}

	if sa.TrxId == nil {
		return false
	}

	pre := TransactionObjectTrxIdUniTable
	sub := sa.TrxId
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoTransactionObjectWrap) insertUniKeyTrxId(sa *SoTransactionObject) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	uniWrap := UniTransactionObjectTrxIdWrap{}
	uniWrap.Dba = s.dba

	res := uniWrap.UniQueryTrxId(sa.TrxId)
	if res != nil {
		//the unique key is already exist
		return false
	}
	val := SoUniqueTransactionObjectByTrxId{}
	val.TrxId = sa.TrxId

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	pre := TransactionObjectTrxIdUniTable
	sub := sa.TrxId
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniTransactionObjectTrxIdWrap struct {
	Dba iservices.IDatabaseService
}

func NewUniTransactionObjectTrxIdWrap(db iservices.IDatabaseService) *UniTransactionObjectTrxIdWrap {
	if db == nil {
		return nil
	}
	wrap := UniTransactionObjectTrxIdWrap{Dba: db}
	return &wrap
}

func (s *UniTransactionObjectTrxIdWrap) UniQueryTrxId(start *prototype.Sha256) *SoTransactionObjectWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := TransactionObjectTrxIdUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueTransactionObjectByTrxId{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoTransactionObjectWrap(s.Dba, res.TrxId)

			return wrap
		}
	}
	return nil
}
