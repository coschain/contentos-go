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
	ExtTrxTrxIdTable       uint32 = 1916120438
	ExtTrxBlockHeightTable uint32 = 3799341326
	ExtTrxBlockTimeTable   uint32 = 1025113122
	ExtTrxTrxIdUniTable    uint32 = 334659987
	ExtTrxBlockHeightCell  uint32 = 2517467390
	ExtTrxBlockTimeCell    uint32 = 2588372818
	ExtTrxTrxIdCell        uint32 = 1776577009
	ExtTrxTrxWrapCell      uint32 = 2374486278
)

////////////// SECTION Wrap Define ///////////////
type SoExtTrxWrap struct {
	dba      iservices.IDatabaseService
	mainKey  *prototype.Sha256
	mKeyFlag int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf  []byte //the buffer after the main key is encoded with prefix
	mBuf     []byte //the value after the main key is encoded
}

func NewSoExtTrxWrap(dba iservices.IDatabaseService, key *prototype.Sha256) *SoExtTrxWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoExtTrxWrap{dba, key, -1, nil, nil}
	return result
}

func (s *SoExtTrxWrap) CheckExist() bool {
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

func (s *SoExtTrxWrap) Create(f func(tInfo *SoExtTrx)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoExtTrx{}
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

func (s *SoExtTrxWrap) getMainKeyBuf() ([]byte, error) {
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

func (s *SoExtTrxWrap) delSortKeyTrxId(sa *SoExtTrx) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListExtTrxByTrxId{}
	if sa == nil {
		key, err := s.encodeMemKey("TrxId")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemExtTrxByTrxId{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		val.TrxId = ori.TrxId
	} else {
		val.TrxId = sa.TrxId
	}

	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoExtTrxWrap) insertSortKeyTrxId(sa *SoExtTrx) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListExtTrxByTrxId{}
	val.TrxId = sa.TrxId
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

func (s *SoExtTrxWrap) delSortKeyBlockHeight(sa *SoExtTrx) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListExtTrxByBlockHeight{}
	if sa == nil {
		key, err := s.encodeMemKey("BlockHeight")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemExtTrxByBlockHeight{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		val.BlockHeight = ori.BlockHeight
		val.TrxId = s.mainKey

	} else {
		val.BlockHeight = sa.BlockHeight
		val.TrxId = sa.TrxId
	}

	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoExtTrxWrap) insertSortKeyBlockHeight(sa *SoExtTrx) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListExtTrxByBlockHeight{}
	val.TrxId = sa.TrxId
	val.BlockHeight = sa.BlockHeight
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

func (s *SoExtTrxWrap) delSortKeyBlockTime(sa *SoExtTrx) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListExtTrxByBlockTime{}
	if sa == nil {
		key, err := s.encodeMemKey("BlockTime")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemExtTrxByBlockTime{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		val.BlockTime = ori.BlockTime
		val.TrxId = s.mainKey

	} else {
		val.BlockTime = sa.BlockTime
		val.TrxId = sa.TrxId
	}

	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoExtTrxWrap) insertSortKeyBlockTime(sa *SoExtTrx) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListExtTrxByBlockTime{}
	val.TrxId = sa.TrxId
	val.BlockTime = sa.BlockTime
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

func (s *SoExtTrxWrap) delAllSortKeys(br bool, val *SoExtTrx) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delSortKeyTrxId(val) {
		if br {
			return false
		} else {
			res = false
		}
	}
	if !s.delSortKeyBlockHeight(val) {
		if br {
			return false
		} else {
			res = false
		}
	}
	if !s.delSortKeyBlockTime(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoExtTrxWrap) insertAllSortKeys(val *SoExtTrx) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoExtTrx fail ")
	}
	if !s.insertSortKeyTrxId(val) {
		return errors.New("insert sort Field TrxId fail while insert table ")
	}
	if !s.insertSortKeyBlockHeight(val) {
		return errors.New("insert sort Field BlockHeight fail while insert table ")
	}
	if !s.insertSortKeyBlockTime(val) {
		return errors.New("insert sort Field BlockTime fail while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoExtTrxWrap) RemoveExtTrx() bool {
	if s.dba == nil {
		return false
	}
	val := &SoExtTrx{}
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
func (s *SoExtTrxWrap) getMemKeyPrefix(fName string) uint32 {
	if fName == "BlockHeight" {
		return ExtTrxBlockHeightCell
	}
	if fName == "BlockTime" {
		return ExtTrxBlockTimeCell
	}
	if fName == "TrxId" {
		return ExtTrxTrxIdCell
	}
	if fName == "TrxWrap" {
		return ExtTrxTrxWrapCell
	}

	return 0
}

func (s *SoExtTrxWrap) encodeMemKey(fName string) ([]byte, error) {
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

func (s *SoExtTrxWrap) saveAllMemKeys(tInfo *SoExtTrx, br bool) error {
	if s.dba == nil {
		return errors.New("save member Field fail , the db is nil")
	}

	if tInfo == nil {
		return errors.New("save member Field fail , the data is nil ")
	}
	var err error = nil
	errDes := ""
	if err = s.saveMemKeyBlockHeight(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "BlockHeight", err)
		}
	}
	if err = s.saveMemKeyBlockTime(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "BlockTime", err)
		}
	}
	if err = s.saveMemKeyTrxId(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "TrxId", err)
		}
	}
	if err = s.saveMemKeyTrxWrap(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "TrxWrap", err)
		}
	}

	if len(errDes) > 0 {
		return errors.New(errDes)
	}
	return err
}

func (s *SoExtTrxWrap) delAllMemKeys(br bool, tInfo *SoExtTrx) error {
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

func (s *SoExtTrxWrap) delMemKey(fName string) error {
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

func (s *SoExtTrxWrap) saveMemKeyBlockHeight(tInfo *SoExtTrx) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemExtTrxByBlockHeight{}
	val.BlockHeight = tInfo.BlockHeight
	key, err := s.encodeMemKey("BlockHeight")
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

func (s *SoExtTrxWrap) GetBlockHeight() uint64 {
	res := true
	msg := &SoMemExtTrxByBlockHeight{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("BlockHeight")
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
				return msg.BlockHeight
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.BlockHeight
}

func (s *SoExtTrxWrap) MdBlockHeight(p uint64) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("BlockHeight")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemExtTrxByBlockHeight{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoExtTrx{}
	sa.TrxId = s.mainKey

	sa.BlockHeight = ori.BlockHeight

	if !s.delSortKeyBlockHeight(sa) {
		return false
	}
	ori.BlockHeight = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.BlockHeight = p

	if !s.insertSortKeyBlockHeight(sa) {
		return false
	}

	return true
}

func (s *SoExtTrxWrap) saveMemKeyBlockTime(tInfo *SoExtTrx) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemExtTrxByBlockTime{}
	val.BlockTime = tInfo.BlockTime
	key, err := s.encodeMemKey("BlockTime")
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

func (s *SoExtTrxWrap) GetBlockTime() *prototype.TimePointSec {
	res := true
	msg := &SoMemExtTrxByBlockTime{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("BlockTime")
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
				return msg.BlockTime
			}
		}
	}
	if !res {
		return nil

	}
	return msg.BlockTime
}

func (s *SoExtTrxWrap) MdBlockTime(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("BlockTime")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemExtTrxByBlockTime{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoExtTrx{}
	sa.TrxId = s.mainKey

	sa.BlockTime = ori.BlockTime

	if !s.delSortKeyBlockTime(sa) {
		return false
	}
	ori.BlockTime = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.BlockTime = p

	if !s.insertSortKeyBlockTime(sa) {
		return false
	}

	return true
}

func (s *SoExtTrxWrap) saveMemKeyTrxId(tInfo *SoExtTrx) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemExtTrxByTrxId{}
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

func (s *SoExtTrxWrap) GetTrxId() *prototype.Sha256 {
	res := true
	msg := &SoMemExtTrxByTrxId{}
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

func (s *SoExtTrxWrap) saveMemKeyTrxWrap(tInfo *SoExtTrx) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemExtTrxByTrxWrap{}
	val.TrxWrap = tInfo.TrxWrap
	key, err := s.encodeMemKey("TrxWrap")
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

func (s *SoExtTrxWrap) GetTrxWrap() *prototype.TransactionWrapper {
	res := true
	msg := &SoMemExtTrxByTrxWrap{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("TrxWrap")
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
				return msg.TrxWrap
			}
		}
	}
	if !res {
		return nil

	}
	return msg.TrxWrap
}

func (s *SoExtTrxWrap) MdTrxWrap(p *prototype.TransactionWrapper) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("TrxWrap")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemExtTrxByTrxWrap{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoExtTrx{}
	sa.TrxId = s.mainKey

	sa.TrxWrap = ori.TrxWrap

	ori.TrxWrap = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.TrxWrap = p

	return true
}

////////////// SECTION List Keys ///////////////
type SExtTrxTrxIdWrap struct {
	Dba iservices.IDatabaseService
}

func NewExtTrxTrxIdWrap(db iservices.IDatabaseService) *SExtTrxTrxIdWrap {
	if db == nil {
		return nil
	}
	wrap := SExtTrxTrxIdWrap{Dba: db}
	return &wrap
}

func (s *SExtTrxTrxIdWrap) DelIterator(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SExtTrxTrxIdWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.Sha256 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListExtTrxByTrxId{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.TrxId

}

func (s *SExtTrxTrxIdWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.Sha256 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListExtTrxByTrxId{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.TrxId

}

func (m *SoListExtTrxByTrxId) OpeEncode() ([]byte, error) {
	pre := ExtTrxTrxIdTable
	sub := m.TrxId
	if sub == nil {
		return nil, errors.New("the pro TrxId is nil")
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
//lastMainKey: the main key of the last one of last page
//lastSubVal: the value  of the last one of last page
//
func (s *SExtTrxTrxIdWrap) ForEachByOrder(start *prototype.Sha256, end *prototype.Sha256, lastMainKey *prototype.Sha256,
	lastSubVal *prototype.Sha256, f func(mVal *prototype.Sha256, sVal *prototype.Sha256, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := ExtTrxTrxIdTable
	skeyList := []interface{}{pre}
	if start != nil {
		skeyList = append(skeyList, start)
		if lastMainKey != nil {
			skeyList = append(skeyList, lastMainKey, kope.MinimalKey)
		}
	} else {
		if lastMainKey != nil && lastSubVal != nil {
			skeyList = append(skeyList, lastSubVal, lastMainKey, kope.MinimalKey)
		}
		skeyList = append(skeyList, kope.MinimalKey)
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

////////////// SECTION List Keys ///////////////
type SExtTrxBlockHeightWrap struct {
	Dba iservices.IDatabaseService
}

func NewExtTrxBlockHeightWrap(db iservices.IDatabaseService) *SExtTrxBlockHeightWrap {
	if db == nil {
		return nil
	}
	wrap := SExtTrxBlockHeightWrap{Dba: db}
	return &wrap
}

func (s *SExtTrxBlockHeightWrap) DelIterator(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SExtTrxBlockHeightWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.Sha256 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListExtTrxByBlockHeight{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.TrxId

}

func (s *SExtTrxBlockHeightWrap) GetSubVal(iterator iservices.IDatabaseIterator) *uint64 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListExtTrxByBlockHeight{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return &res.BlockHeight

}

func (m *SoListExtTrxByBlockHeight) OpeEncode() ([]byte, error) {
	pre := ExtTrxBlockHeightTable
	sub := m.BlockHeight

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
//lastMainKey: the main key of the last one of last page
//lastSubVal: the value  of the last one of last page
//
func (s *SExtTrxBlockHeightWrap) ForEachByOrder(start *uint64, end *uint64, lastMainKey *prototype.Sha256,
	lastSubVal *uint64, f func(mVal *prototype.Sha256, sVal *uint64, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := ExtTrxBlockHeightTable
	skeyList := []interface{}{pre}
	if start != nil {
		skeyList = append(skeyList, start)
		if lastMainKey != nil {
			skeyList = append(skeyList, lastMainKey, kope.MinimalKey)
		}
	} else {
		if lastMainKey != nil && lastSubVal != nil {
			skeyList = append(skeyList, lastSubVal, lastMainKey, kope.MinimalKey)
		}
		skeyList = append(skeyList, kope.MinimalKey)
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

////////////// SECTION List Keys ///////////////
type SExtTrxBlockTimeWrap struct {
	Dba iservices.IDatabaseService
}

func NewExtTrxBlockTimeWrap(db iservices.IDatabaseService) *SExtTrxBlockTimeWrap {
	if db == nil {
		return nil
	}
	wrap := SExtTrxBlockTimeWrap{Dba: db}
	return &wrap
}

func (s *SExtTrxBlockTimeWrap) DelIterator(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SExtTrxBlockTimeWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.Sha256 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListExtTrxByBlockTime{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.TrxId

}

func (s *SExtTrxBlockTimeWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.TimePointSec {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListExtTrxByBlockTime{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.BlockTime

}

func (m *SoListExtTrxByBlockTime) OpeEncode() ([]byte, error) {
	pre := ExtTrxBlockTimeTable
	sub := m.BlockTime
	if sub == nil {
		return nil, errors.New("the pro BlockTime is nil")
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
//lastMainKey: the main key of the last one of last page
//lastSubVal: the value  of the last one of last page
//
func (s *SExtTrxBlockTimeWrap) ForEachByOrder(start *prototype.TimePointSec, end *prototype.TimePointSec, lastMainKey *prototype.Sha256,
	lastSubVal *prototype.TimePointSec, f func(mVal *prototype.Sha256, sVal *prototype.TimePointSec, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := ExtTrxBlockTimeTable
	skeyList := []interface{}{pre}
	if start != nil {
		skeyList = append(skeyList, start)
		if lastMainKey != nil {
			skeyList = append(skeyList, lastMainKey, kope.MinimalKey)
		}
	} else {
		if lastMainKey != nil && lastSubVal != nil {
			skeyList = append(skeyList, lastSubVal, lastMainKey, kope.MinimalKey)
		}
		skeyList = append(skeyList, kope.MinimalKey)
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

//Query srt by reverse order
//
//f: callback for each traversal , primary 、sub key、idx(the number of times it has been iterated)
//as arguments to the callback function
//if the return value of f is true,continue iterating until the end iteration;
//otherwise stop iteration immediately
//
//lastMainKey: the main key of the last one of last page
//lastSubVal: the value  of the last one of last page
//
func (s *SExtTrxBlockTimeWrap) ForEachByRevOrder(start *prototype.TimePointSec, end *prototype.TimePointSec, lastMainKey *prototype.Sha256,
	lastSubVal *prototype.TimePointSec, f func(mVal *prototype.Sha256, sVal *prototype.TimePointSec, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := ExtTrxBlockTimeTable
	skeyList := []interface{}{pre}
	if start != nil {
		skeyList = append(skeyList, start)
		if lastMainKey != nil {
			skeyList = append(skeyList, lastMainKey)
		}
	} else {
		if lastMainKey != nil && lastSubVal != nil {
			skeyList = append(skeyList, lastSubVal, lastMainKey)
		}
		skeyList = append(skeyList, kope.MaximumKey)
	}
	sBuf, cErr := kope.EncodeSlice(skeyList)
	if cErr != nil {
		return cErr
	}
	eKeyList := []interface{}{pre}
	if end != nil {
		eKeyList = append(eKeyList, end)
	}
	eBuf, cErr := kope.EncodeSlice(eKeyList)
	if cErr != nil {
		return cErr
	}
	//reverse the start and end when create ReversedIterator to query by reverse order
	iterator := s.Dba.NewReversedIterator(eBuf, sBuf)
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

func (s *SoExtTrxWrap) update(sa *SoExtTrx) bool {
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

func (s *SoExtTrxWrap) getExtTrx() *SoExtTrx {
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

	res := &SoExtTrx{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoExtTrxWrap) encodeMainKey() ([]byte, error) {
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

func (s *SoExtTrxWrap) delAllUniKeys(br bool, val *SoExtTrx) bool {
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

func (s *SoExtTrxWrap) delUniKeysWithNames(names map[string]string, val *SoExtTrx) bool {
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

func (s *SoExtTrxWrap) insertAllUniKeys(val *SoExtTrx) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoExtTrx fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyTrxId(val) {
		return sucFields, errors.New("insert unique Field TrxId fail while insert table ")
	}
	sucFields["TrxId"] = "TrxId"

	return sucFields, nil
}

func (s *SoExtTrxWrap) delUniKeyTrxId(sa *SoExtTrx) bool {
	if s.dba == nil {
		return false
	}
	pre := ExtTrxTrxIdUniTable
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
		ori := &SoMemExtTrxByTrxId{}
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

func (s *SoExtTrxWrap) insertUniKeyTrxId(sa *SoExtTrx) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	pre := ExtTrxTrxIdUniTable
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
	val := SoUniqueExtTrxByTrxId{}
	val.TrxId = sa.TrxId

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniExtTrxTrxIdWrap struct {
	Dba iservices.IDatabaseService
}

func NewUniExtTrxTrxIdWrap(db iservices.IDatabaseService) *UniExtTrxTrxIdWrap {
	if db == nil {
		return nil
	}
	wrap := UniExtTrxTrxIdWrap{Dba: db}
	return &wrap
}

func (s *UniExtTrxTrxIdWrap) UniQueryTrxId(start *prototype.Sha256) *SoExtTrxWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := ExtTrxTrxIdUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueExtTrxByTrxId{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoExtTrxWrap(s.Dba, res.TrxId)

			return wrap
		}
	}
	return nil
}
