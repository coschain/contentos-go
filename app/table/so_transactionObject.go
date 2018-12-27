package table

import (
	"errors"
	fmt "fmt"
	"reflect"
	"strings"

	"github.com/coschain/contentos-go/common/encoding/kope"
	"github.com/coschain/contentos-go/iservices"
	prototype "github.com/coschain/contentos-go/prototype"
	proto "github.com/golang/protobuf/proto"
)

////////////// SECTION Prefix Mark ///////////////
var (
	TransactionObjectExpirationTable = uint32(3273070683)
	TransactionObjectTrxIdUniTable   = uint32(482982412)
	TransactionObjectExpirationCell  = uint32(1085825226)
	TransactionObjectTrxIdCell       = uint32(3384783386)
)

////////////// SECTION Wrap Define ///////////////
type SoTransactionObjectWrap struct {
	dba      iservices.IDatabaseService
	mainKey  *prototype.Sha256
	mKeyFlag int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf  []byte //the buffer after the main key is encoded with prefix
	mBuf     []byte //the value after the main key is encoded
}

func NewSoTransactionObjectWrap(dba iservices.IDatabaseService, key *prototype.Sha256) *SoTransactionObjectWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoTransactionObjectWrap{dba, key, -1, nil, nil}
	return result
}

func (s *SoTransactionObjectWrap) CheckExist() bool {
	if s.dba == nil {
		return false
	}
	if s.mKeyFlag != -1 {
		//if you have already obtained the existence status of the primary key, use it directly
		if s.mKeyFlag == 0 {
			return false
		}
		return true
	}
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return false
	}

	res, err := s.dba.Has(keyBuf)
	if err != nil {
		return false
	}
	if res == false {
		s.mKeyFlag = 0
	} else {
		s.mKeyFlag = 1
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
	err = s.saveAllMemKeys(val, true)
	if err != nil {
		s.delAllMemKeys(false, val)
		return err
	}

	// update srt list keys
	if err = s.insertAllSortKeys(val); err != nil {
		s.delAllSortKeys(false, val)
		s.dba.Delete(keyBuf)
		s.delAllMemKeys(false, val)
		return err
	}

	//update unique list
	if sucNames, err := s.insertAllUniKeys(val); err != nil {
		s.delAllSortKeys(false, val)
		s.delUniKeysWithNames(sucNames, val)
		s.dba.Delete(keyBuf)
		s.delAllMemKeys(false, val)
		return err
	}

	return nil
}

func (s *SoTransactionObjectWrap) getMainKeyBuf() ([]byte, error) {
	if s.mainKey == nil {
		return nil, errors.New("the main key is nil")
	}
	if s.mBuf == nil {
		var err error = nil
		s.mBuf, err = kope.Encode(s.mainKey)
		if err != nil {
			return nil, err
		}
	}
	return s.mBuf, nil
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoTransactionObjectWrap) delSortKeyExpiration(sa *SoTransactionObject) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListTransactionObjectByExpiration{}
	if sa == nil {
		key, err := s.encodeMemKey("Expiration")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemTransactionObjectByExpiration{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		val.Expiration = ori.Expiration
		val.TrxId = s.mainKey

	} else {
		val.Expiration = sa.Expiration
		val.TrxId = sa.TrxId
	}

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
	if s.dba == nil {
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
	val := &SoTransactionObject{}
	//delete sort list key
	if res := s.delAllSortKeys(true, nil); !res {
		return false
	}

	//delete unique list
	if res := s.delAllUniKeys(true, nil); !res {
		return false
	}

	err := s.delAllMemKeys(true, val)
	if err == nil {
		s.mKeyBuf = nil
		s.mKeyFlag = -1
		return true
	} else {
		return false
	}
}

////////////// SECTION Members Get/Modify ///////////////
func (s *SoTransactionObjectWrap) getMemKeyPrefix(fName string) uint32 {
	if fName == "Expiration" {
		return TransactionObjectExpirationCell
	}
	if fName == "TrxId" {
		return TransactionObjectTrxIdCell
	}

	return 0
}

func (s *SoTransactionObjectWrap) encodeMemKey(fName string) ([]byte, error) {
	if len(fName) < 1 || s.mainKey == nil {
		return nil, errors.New("field name or main key is empty")
	}
	pre := s.getMemKeyPrefix(fName)
	preBuf, err := kope.Encode(pre)
	if err != nil {
		return nil, err
	}
	mBuf, err := s.getMainKeyBuf()
	if err != nil {
		return nil, err
	}
	list := make([][]byte, 2)
	list[0] = preBuf
	list[1] = mBuf
	return kope.PackList(list), nil
}

func (s *SoTransactionObjectWrap) saveAllMemKeys(tInfo *SoTransactionObject, br bool) error {
	if s.dba == nil {
		return errors.New("save member Field fail , the db is nil")
	}

	if tInfo == nil {
		return errors.New("save member Field fail , the data is nil ")
	}
	var err error = nil
	errDes := ""
	if err = s.saveMemKeyExpiration(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Expiration", err)
		}
	}
	if err = s.saveMemKeyTrxId(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "TrxId", err)
		}
	}

	if len(errDes) > 0 {
		return errors.New(errDes)
	}
	return err
}

func (s *SoTransactionObjectWrap) delAllMemKeys(br bool, tInfo *SoTransactionObject) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	t := reflect.TypeOf(*tInfo)
	errDesc := ""
	for k := 0; k < t.NumField(); k++ {
		name := t.Field(k).Name
		if len(name) > 0 && !strings.HasPrefix(name, "XXX_") {
			err := s.delMemKey(name)
			if err != nil {
				if br {
					return err
				}
				errDesc += fmt.Sprintf("delete the Field %s fail,error is %s;\n", name, err)
			}
		}
	}
	if len(errDesc) > 0 {
		return errors.New(errDesc)
	}
	return nil
}

func (s *SoTransactionObjectWrap) delMemKey(fName string) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if len(fName) <= 0 {
		return errors.New("the field name is empty ")
	}
	key, err := s.encodeMemKey(fName)
	if err != nil {
		return err
	}
	err = s.dba.Delete(key)
	return err
}

func (s *SoTransactionObjectWrap) saveMemKeyExpiration(tInfo *SoTransactionObject) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemTransactionObjectByExpiration{}
	val.Expiration = tInfo.Expiration
	key, err := s.encodeMemKey("Expiration")
	if err != nil {
		return err
	}
	buf, err := proto.Marshal(&val)
	if err != nil {
		return err
	}
	err = s.dba.Put(key, buf)
	return err
}

func (s *SoTransactionObjectWrap) GetExpiration() *prototype.TimePointSec {
	res := true
	msg := &SoMemTransactionObjectByExpiration{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Expiration")
		if err != nil {
			res = false
		} else {
			buf, err := s.dba.Get(key)
			if err != nil {
				res = false
			}
			err = proto.Unmarshal(buf, msg)
			if err != nil {
				res = false
			} else {
				return msg.Expiration
			}
		}
	}
	if !res {
		return nil

	}
	return msg.Expiration
}

func (s *SoTransactionObjectWrap) MdExpiration(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Expiration")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemTransactionObjectByExpiration{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoTransactionObject{}
	sa.TrxId = s.mainKey

	sa.Expiration = ori.Expiration

	if !s.delSortKeyExpiration(sa) {
		return false
	}
	ori.Expiration = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Expiration = p

	if !s.insertSortKeyExpiration(sa) {
		return false
	}

	return true
}

func (s *SoTransactionObjectWrap) saveMemKeyTrxId(tInfo *SoTransactionObject) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemTransactionObjectByTrxId{}
	val.TrxId = tInfo.TrxId
	key, err := s.encodeMemKey("TrxId")
	if err != nil {
		return err
	}
	buf, err := proto.Marshal(&val)
	if err != nil {
		return err
	}
	err = s.dba.Put(key, buf)
	return err
}

func (s *SoTransactionObjectWrap) GetTrxId() *prototype.Sha256 {
	res := true
	msg := &SoMemTransactionObjectByTrxId{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("TrxId")
		if err != nil {
			res = false
		} else {
			buf, err := s.dba.Get(key)
			if err != nil {
				res = false
			}
			err = proto.Unmarshal(buf, msg)
			if err != nil {
				res = false
			} else {
				return msg.TrxId
			}
		}
	}
	if !res {
		return nil

	}
	return msg.TrxId
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

func (s *STransactionObjectExpirationWrap) DelIterator(iterator iservices.IDatabaseIterator) {
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

//Query srt by order
//
//start = nil  end = nil (query the db from start to end)
//start = nil (query from start the db)
//end = nil (query to the end of db)
//
//f: callback for each traversal , primary 、sub key、idx(the number of times it has been iterated)
//as arguments to the callback function
//if the return value of f is true,continue iterating until the end iteration;
//otherwise stop iteration immediately
//
func (s *STransactionObjectExpirationWrap) ForEachByOrder(start *prototype.TimePointSec, end *prototype.TimePointSec,
	f func(mVal *prototype.Sha256, sVal *prototype.TimePointSec, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if f == nil {
		return nil
	}
	pre := TransactionObjectExpirationTable
	skeyList := []interface{}{pre}
	if start != nil {
		skeyList = append(skeyList, start)
	}
	sBuf, cErr := kope.EncodeSlice(skeyList)
	if cErr != nil {
		return cErr
	}
	eKeyList := []interface{}{pre}
	if end != nil {
		eKeyList = append(eKeyList, end)
	} else {
		eKeyList = append(eKeyList, kope.MaximumKey)
	}
	eBuf, cErr := kope.EncodeSlice(eKeyList)
	if cErr != nil {
		return cErr
	}
	iterator := s.Dba.NewIterator(sBuf, eBuf)
	if iterator == nil {
		return errors.New("there is no data in range")
	}
	var idx uint32 = 0
	for iterator.Next() {
		idx++
		if isContinue := f(s.GetMainVal(iterator), s.GetSubVal(iterator), idx); !isContinue {
			break
		}
	}
	s.DelIterator(iterator)
	return nil
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
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := s.getMemKeyPrefix("TrxId")
	sub := s.mainKey
	if sub == nil {
		return nil, errors.New("the mainKey is nil")
	}
	preBuf, err := kope.Encode(pre)
	if err != nil {
		return nil, err
	}
	mBuf, err := s.getMainKeyBuf()
	if err != nil {
		return nil, err
	}
	list := make([][]byte, 2)
	list[0] = preBuf
	list[1] = mBuf
	s.mKeyBuf = kope.PackList(list)
	return s.mKeyBuf, nil
}

////////////// Unique Query delete/insert/query ///////////////

func (s *SoTransactionObjectWrap) delAllUniKeys(br bool, val *SoTransactionObject) bool {
	if s.dba == nil {
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

func (s *SoTransactionObjectWrap) delUniKeysWithNames(names map[string]string, val *SoTransactionObject) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if len(names["TrxId"]) > 0 {
		if !s.delUniKeyTrxId(val) {
			res = false
		}
	}

	return res
}

func (s *SoTransactionObjectWrap) insertAllUniKeys(val *SoTransactionObject) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoTransactionObject fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyTrxId(val) {
		return sucFields, errors.New("insert unique Field TrxId fail while insert table ")
	}
	sucFields["TrxId"] = "TrxId"

	return sucFields, nil
}

func (s *SoTransactionObjectWrap) delUniKeyTrxId(sa *SoTransactionObject) bool {
	if s.dba == nil {
		return false
	}
	pre := TransactionObjectTrxIdUniTable
	kList := []interface{}{pre}
	if sa != nil {

		if sa.TrxId == nil {
			return false
		}

		sub := sa.TrxId
		kList = append(kList, sub)
	} else {
		key, err := s.encodeMemKey("TrxId")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemTransactionObjectByTrxId{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		sub := ori.TrxId
		kList = append(kList, sub)

	}
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
	pre := TransactionObjectTrxIdUniTable
	sub := sa.TrxId
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	res, err := s.dba.Has(kBuf)
	if err == nil && res == true {
		//the unique key is already exist
		return false
	}
	val := SoUniqueTransactionObjectByTrxId{}
	val.TrxId = sa.TrxId

	buf, err := proto.Marshal(&val)

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
