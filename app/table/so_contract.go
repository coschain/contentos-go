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
	ContractCreatedTimeTable = uint32(1292005739)
	ContractIdUniTable       = uint32(4175408872)
	ContractAbiCell          = uint32(562884560)
	ContractBalanceCell      = uint32(1230027001)
	ContractCodeCell         = uint32(1267857519)
	ContractCreatedTimeCell  = uint32(3946752343)
	ContractIdCell           = uint32(1995418866)
)

////////////// SECTION Wrap Define ///////////////
type SoContractWrap struct {
	dba      iservices.IDatabaseService
	mainKey  *prototype.ContractId
	mKeyFlag int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf  []byte //the buffer after the main key is encoded with prefix
	mBuf     []byte //the value after the main key is encoded
}

func NewSoContractWrap(dba iservices.IDatabaseService, key *prototype.ContractId) *SoContractWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoContractWrap{dba, key, -1, nil, nil}
	return result
}

func (s *SoContractWrap) CheckExist() bool {
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

func (s *SoContractWrap) getMainKeyBuf() ([]byte, error) {
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

func (s *SoContractWrap) delSortKeyCreatedTime(sa *SoContract) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListContractByCreatedTime{}
	if sa == nil {
		key, err := s.encodeMemKey("CreatedTime")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemContractByCreatedTime{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		val.CreatedTime = ori.CreatedTime
		val.Id = s.mainKey

	} else {
		val.CreatedTime = sa.CreatedTime
		val.Id = sa.Id
	}

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
	if s.dba == nil {
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
	val := &SoContract{}
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
func (s *SoContractWrap) getMemKeyPrefix(fName string) uint32 {
	if fName == "Abi" {
		return ContractAbiCell
	}
	if fName == "Balance" {
		return ContractBalanceCell
	}
	if fName == "Code" {
		return ContractCodeCell
	}
	if fName == "CreatedTime" {
		return ContractCreatedTimeCell
	}
	if fName == "Id" {
		return ContractIdCell
	}

	return 0
}

func (s *SoContractWrap) encodeMemKey(fName string) ([]byte, error) {
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

func (s *SoContractWrap) saveAllMemKeys(tInfo *SoContract, br bool) error {
	if s.dba == nil {
		return errors.New("save member Field fail , the db is nil")
	}

	if tInfo == nil {
		return errors.New("save member Field fail , the data is nil ")
	}
	var err error = nil
	errDes := ""
	if err = s.saveMemKeyAbi(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Abi", err)
		}
	}
	if err = s.saveMemKeyBalance(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Balance", err)
		}
	}
	if err = s.saveMemKeyCode(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Code", err)
		}
	}
	if err = s.saveMemKeyCreatedTime(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "CreatedTime", err)
		}
	}
	if err = s.saveMemKeyId(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Id", err)
		}
	}

	if len(errDes) > 0 {
		return errors.New(errDes)
	}
	return err
}

func (s *SoContractWrap) delAllMemKeys(br bool, tInfo *SoContract) error {
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

func (s *SoContractWrap) delMemKey(fName string) error {
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

func (s *SoContractWrap) saveMemKeyAbi(tInfo *SoContract) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemContractByAbi{}
	val.Abi = tInfo.Abi
	key, err := s.encodeMemKey("Abi")
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

func (s *SoContractWrap) GetAbi() string {
	res := true
	msg := &SoMemContractByAbi{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Abi")
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
				return msg.Abi
			}
		}
	}
	if !res {
		var tmpValue string
		return tmpValue
	}
	return msg.Abi
}

func (s *SoContractWrap) MdAbi(p string) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Abi")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemContractByAbi{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoContract{}
	sa.Id = s.mainKey

	sa.Abi = ori.Abi

	ori.Abi = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Abi = p

	return true
}

func (s *SoContractWrap) saveMemKeyBalance(tInfo *SoContract) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemContractByBalance{}
	val.Balance = tInfo.Balance
	key, err := s.encodeMemKey("Balance")
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

func (s *SoContractWrap) GetBalance() *prototype.Coin {
	res := true
	msg := &SoMemContractByBalance{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Balance")
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
				return msg.Balance
			}
		}
	}
	if !res {
		return nil

	}
	return msg.Balance
}

func (s *SoContractWrap) MdBalance(p *prototype.Coin) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Balance")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemContractByBalance{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoContract{}
	sa.Id = s.mainKey

	sa.Balance = ori.Balance

	ori.Balance = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Balance = p

	return true
}

func (s *SoContractWrap) saveMemKeyCode(tInfo *SoContract) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemContractByCode{}
	val.Code = tInfo.Code
	key, err := s.encodeMemKey("Code")
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

func (s *SoContractWrap) GetCode() []byte {
	res := true
	msg := &SoMemContractByCode{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Code")
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
				return msg.Code
			}
		}
	}
	if !res {
		var tmpValue []byte
		return tmpValue
	}
	return msg.Code
}

func (s *SoContractWrap) MdCode(p []byte) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Code")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemContractByCode{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoContract{}
	sa.Id = s.mainKey

	sa.Code = ori.Code

	ori.Code = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Code = p

	return true
}

func (s *SoContractWrap) saveMemKeyCreatedTime(tInfo *SoContract) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemContractByCreatedTime{}
	val.CreatedTime = tInfo.CreatedTime
	key, err := s.encodeMemKey("CreatedTime")
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

func (s *SoContractWrap) GetCreatedTime() *prototype.TimePointSec {
	res := true
	msg := &SoMemContractByCreatedTime{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("CreatedTime")
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
				return msg.CreatedTime
			}
		}
	}
	if !res {
		return nil

	}
	return msg.CreatedTime
}

func (s *SoContractWrap) MdCreatedTime(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("CreatedTime")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemContractByCreatedTime{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoContract{}
	sa.Id = s.mainKey

	sa.CreatedTime = ori.CreatedTime

	if !s.delSortKeyCreatedTime(sa) {
		return false
	}
	ori.CreatedTime = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.CreatedTime = p

	if !s.insertSortKeyCreatedTime(sa) {
		return false
	}

	return true
}

func (s *SoContractWrap) saveMemKeyId(tInfo *SoContract) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemContractById{}
	val.Id = tInfo.Id
	key, err := s.encodeMemKey("Id")
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

func (s *SoContractWrap) GetId() *prototype.ContractId {
	res := true
	msg := &SoMemContractById{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Id")
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
				return msg.Id
			}
		}
	}
	if !res {
		return nil

	}
	return msg.Id
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

func (s *SContractCreatedTimeWrap) DelIterator(iterator iservices.IDatabaseIterator) {
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
func (s *SContractCreatedTimeWrap) ForEachByOrder(start *prototype.TimePointSec, end *prototype.TimePointSec,
	f func(mVal *prototype.ContractId, sVal *prototype.TimePointSec, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if f == nil {
		return nil
	}
	pre := ContractCreatedTimeTable
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
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := s.getMemKeyPrefix("Id")
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

func (s *SoContractWrap) delAllUniKeys(br bool, val *SoContract) bool {
	if s.dba == nil {
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

func (s *SoContractWrap) delUniKeysWithNames(names map[string]string, val *SoContract) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if len(names["Id"]) > 0 {
		if !s.delUniKeyId(val) {
			res = false
		}
	}

	return res
}

func (s *SoContractWrap) insertAllUniKeys(val *SoContract) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoContract fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyId(val) {
		return sucFields, errors.New("insert unique Field Id fail while insert table ")
	}
	sucFields["Id"] = "Id"

	return sucFields, nil
}

func (s *SoContractWrap) delUniKeyId(sa *SoContract) bool {
	if s.dba == nil {
		return false
	}
	pre := ContractIdUniTable
	kList := []interface{}{pre}
	if sa != nil {

		if sa.Id == nil {
			return false
		}

		sub := sa.Id
		kList = append(kList, sub)
	} else {
		key, err := s.encodeMemKey("Id")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemContractById{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		sub := ori.Id
		kList = append(kList, sub)

	}
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
	pre := ContractIdUniTable
	sub := sa.Id
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
	val := SoUniqueContractById{}
	val.Id = sa.Id

	buf, err := proto.Marshal(&val)

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
