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
	ExtDailyTrxDateTable    uint32 = 4241530934
	ExtDailyTrxCountTable   uint32 = 1672192129
	ExtDailyTrxDateUniTable uint32 = 111567975
	ExtDailyTrxCountCell    uint32 = 4142055978
	ExtDailyTrxDateCell     uint32 = 839307372
)

////////////// SECTION Wrap Define ///////////////
type SoExtDailyTrxWrap struct {
	dba      iservices.IDatabaseRW
	mainKey  *prototype.TimePointSec
	mKeyFlag int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf  []byte //the buffer after the main key is encoded with prefix
	mBuf     []byte //the value after the main key is encoded
}

func NewSoExtDailyTrxWrap(dba iservices.IDatabaseRW, key *prototype.TimePointSec) *SoExtDailyTrxWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoExtDailyTrxWrap{dba, key, -1, nil, nil}
	return result
}

func (s *SoExtDailyTrxWrap) CheckExist() bool {
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

func (s *SoExtDailyTrxWrap) Create(f func(tInfo *SoExtDailyTrx)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoExtDailyTrx{}
	f(val)
	if val.Date == nil {
		val.Date = s.mainKey
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

func (s *SoExtDailyTrxWrap) getMainKeyBuf() ([]byte, error) {
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

func (s *SoExtDailyTrxWrap) delSortKeyDate(sa *SoExtDailyTrx) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListExtDailyTrxByDate{}
	if sa == nil {
		key, err := s.encodeMemKey("Date")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemExtDailyTrxByDate{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		val.Date = ori.Date
	} else {
		val.Date = sa.Date
	}

	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoExtDailyTrxWrap) insertSortKeyDate(sa *SoExtDailyTrx) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListExtDailyTrxByDate{}
	val.Date = sa.Date
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

func (s *SoExtDailyTrxWrap) delSortKeyCount(sa *SoExtDailyTrx) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListExtDailyTrxByCount{}
	if sa == nil {
		key, err := s.encodeMemKey("Count")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemExtDailyTrxByCount{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		val.Count = ori.Count
		val.Date = s.mainKey

	} else {
		val.Count = sa.Count
		val.Date = sa.Date
	}

	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoExtDailyTrxWrap) insertSortKeyCount(sa *SoExtDailyTrx) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListExtDailyTrxByCount{}
	val.Date = sa.Date
	val.Count = sa.Count
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

func (s *SoExtDailyTrxWrap) delAllSortKeys(br bool, val *SoExtDailyTrx) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delSortKeyDate(val) {
		if br {
			return false
		} else {
			res = false
		}
	}
	if !s.delSortKeyCount(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoExtDailyTrxWrap) insertAllSortKeys(val *SoExtDailyTrx) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoExtDailyTrx fail ")
	}
	if !s.insertSortKeyDate(val) {
		return errors.New("insert sort Field Date fail while insert table ")
	}
	if !s.insertSortKeyCount(val) {
		return errors.New("insert sort Field Count fail while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoExtDailyTrxWrap) RemoveExtDailyTrx() bool {
	if s.dba == nil {
		return false
	}
	val := &SoExtDailyTrx{}
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
func (s *SoExtDailyTrxWrap) getMemKeyPrefix(fName string) uint32 {
	if fName == "Count" {
		return ExtDailyTrxCountCell
	}
	if fName == "Date" {
		return ExtDailyTrxDateCell
	}

	return 0
}

func (s *SoExtDailyTrxWrap) encodeMemKey(fName string) ([]byte, error) {
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

func (s *SoExtDailyTrxWrap) saveAllMemKeys(tInfo *SoExtDailyTrx, br bool) error {
	if s.dba == nil {
		return errors.New("save member Field fail , the db is nil")
	}

	if tInfo == nil {
		return errors.New("save member Field fail , the data is nil ")
	}
	var err error = nil
	errDes := ""
	if err = s.saveMemKeyCount(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Count", err)
		}
	}
	if err = s.saveMemKeyDate(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Date", err)
		}
	}

	if len(errDes) > 0 {
		return errors.New(errDes)
	}
	return err
}

func (s *SoExtDailyTrxWrap) delAllMemKeys(br bool, tInfo *SoExtDailyTrx) error {
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

func (s *SoExtDailyTrxWrap) delMemKey(fName string) error {
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

func (s *SoExtDailyTrxWrap) saveMemKeyCount(tInfo *SoExtDailyTrx) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemExtDailyTrxByCount{}
	val.Count = tInfo.Count
	key, err := s.encodeMemKey("Count")
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

func (s *SoExtDailyTrxWrap) GetCount() uint64 {
	res := true
	msg := &SoMemExtDailyTrxByCount{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Count")
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
				return msg.Count
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.Count
}

func (s *SoExtDailyTrxWrap) MdCount(p uint64) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Count")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemExtDailyTrxByCount{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoExtDailyTrx{}
	sa.Date = s.mainKey

	sa.Count = ori.Count

	if !s.delSortKeyCount(sa) {
		return false
	}
	ori.Count = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Count = p

	if !s.insertSortKeyCount(sa) {
		return false
	}

	return true
}

func (s *SoExtDailyTrxWrap) saveMemKeyDate(tInfo *SoExtDailyTrx) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemExtDailyTrxByDate{}
	val.Date = tInfo.Date
	key, err := s.encodeMemKey("Date")
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

func (s *SoExtDailyTrxWrap) GetDate() *prototype.TimePointSec {
	res := true
	msg := &SoMemExtDailyTrxByDate{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Date")
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
				return msg.Date
			}
		}
	}
	if !res {
		return nil

	}
	return msg.Date
}

////////////// SECTION List Keys ///////////////
type SExtDailyTrxDateWrap struct {
	Dba iservices.IDatabaseRW
}

func NewExtDailyTrxDateWrap(db iservices.IDatabaseRW) *SExtDailyTrxDateWrap {
	if db == nil {
		return nil
	}
	wrap := SExtDailyTrxDateWrap{Dba: db}
	return &wrap
}

func (s *SExtDailyTrxDateWrap) DelIterator(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SExtDailyTrxDateWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.TimePointSec {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListExtDailyTrxByDate{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Date

}

func (s *SExtDailyTrxDateWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.TimePointSec {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListExtDailyTrxByDate{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.Date

}

func (m *SoListExtDailyTrxByDate) OpeEncode() ([]byte, error) {
	pre := ExtDailyTrxDateTable
	sub := m.Date
	if sub == nil {
		return nil, errors.New("the pro Date is nil")
	}
	sub1 := m.Date
	if sub1 == nil {
		return nil, errors.New("the mainkey Date is nil")
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
func (s *SExtDailyTrxDateWrap) ForEachByOrder(start *prototype.TimePointSec, end *prototype.TimePointSec, lastMainKey *prototype.TimePointSec,
	lastSubVal *prototype.TimePointSec, f func(mVal *prototype.TimePointSec, sVal *prototype.TimePointSec, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := ExtDailyTrxDateTable
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
type SExtDailyTrxCountWrap struct {
	Dba iservices.IDatabaseRW
}

func NewExtDailyTrxCountWrap(db iservices.IDatabaseRW) *SExtDailyTrxCountWrap {
	if db == nil {
		return nil
	}
	wrap := SExtDailyTrxCountWrap{Dba: db}
	return &wrap
}

func (s *SExtDailyTrxCountWrap) DelIterator(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SExtDailyTrxCountWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.TimePointSec {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListExtDailyTrxByCount{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Date

}

func (s *SExtDailyTrxCountWrap) GetSubVal(iterator iservices.IDatabaseIterator) *uint64 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListExtDailyTrxByCount{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return &res.Count

}

func (m *SoListExtDailyTrxByCount) OpeEncode() ([]byte, error) {
	pre := ExtDailyTrxCountTable
	sub := m.Count

	sub1 := m.Date
	if sub1 == nil {
		return nil, errors.New("the mainkey Date is nil")
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
func (s *SExtDailyTrxCountWrap) ForEachByOrder(start *uint64, end *uint64, lastMainKey *prototype.TimePointSec,
	lastSubVal *uint64, f func(mVal *prototype.TimePointSec, sVal *uint64, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := ExtDailyTrxCountTable
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

/////////////// SECTION Private function ////////////////

func (s *SoExtDailyTrxWrap) update(sa *SoExtDailyTrx) bool {
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

func (s *SoExtDailyTrxWrap) getExtDailyTrx() *SoExtDailyTrx {
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

	res := &SoExtDailyTrx{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoExtDailyTrxWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := s.getMemKeyPrefix("Date")
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

func (s *SoExtDailyTrxWrap) delAllUniKeys(br bool, val *SoExtDailyTrx) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delUniKeyDate(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoExtDailyTrxWrap) delUniKeysWithNames(names map[string]string, val *SoExtDailyTrx) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if len(names["Date"]) > 0 {
		if !s.delUniKeyDate(val) {
			res = false
		}
	}

	return res
}

func (s *SoExtDailyTrxWrap) insertAllUniKeys(val *SoExtDailyTrx) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoExtDailyTrx fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyDate(val) {
		return sucFields, errors.New("insert unique Field Date fail while insert table ")
	}
	sucFields["Date"] = "Date"

	return sucFields, nil
}

func (s *SoExtDailyTrxWrap) delUniKeyDate(sa *SoExtDailyTrx) bool {
	if s.dba == nil {
		return false
	}
	pre := ExtDailyTrxDateUniTable
	kList := []interface{}{pre}
	if sa != nil {

		if sa.Date == nil {
			return false
		}

		sub := sa.Date
		kList = append(kList, sub)
	} else {
		key, err := s.encodeMemKey("Date")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemExtDailyTrxByDate{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		sub := ori.Date
		kList = append(kList, sub)

	}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoExtDailyTrxWrap) insertUniKeyDate(sa *SoExtDailyTrx) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	pre := ExtDailyTrxDateUniTable
	sub := sa.Date
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
	val := SoUniqueExtDailyTrxByDate{}
	val.Date = sa.Date

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniExtDailyTrxDateWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniExtDailyTrxDateWrap(db iservices.IDatabaseRW) *UniExtDailyTrxDateWrap {
	if db == nil {
		return nil
	}
	wrap := UniExtDailyTrxDateWrap{Dba: db}
	return &wrap
}

func (s *UniExtDailyTrxDateWrap) UniQueryDate(start *prototype.TimePointSec) *SoExtDailyTrxWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := ExtDailyTrxDateUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueExtDailyTrxByDate{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoExtDailyTrxWrap(s.Dba, res.Date)

			return wrap
		}
	}
	return nil
}
