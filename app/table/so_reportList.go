package table

import (
	"errors"
	fmt "fmt"
	"reflect"
	"strings"

	"github.com/coschain/contentos-go/common/encoding/kope"
	"github.com/coschain/contentos-go/iservices"
	proto "github.com/golang/protobuf/proto"
)

////////////// SECTION Prefix Mark ///////////////
var (
	ReportListReportedTimesTable uint32 = 4124045745
	ReportListUuidUniTable       uint32 = 4051252686
	ReportListIsArbitratedCell   uint32 = 380656159
	ReportListReportedTimesCell  uint32 = 2602269792
	ReportListTagsCell           uint32 = 661571911
	ReportListUuidCell           uint32 = 2426362854
)

////////////// SECTION Wrap Define ///////////////
type SoReportListWrap struct {
	dba      iservices.IDatabaseRW
	mainKey  *uint64
	mKeyFlag int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf  []byte //the buffer after the main key is encoded with prefix
	mBuf     []byte //the value after the main key is encoded
}

func NewSoReportListWrap(dba iservices.IDatabaseRW, key *uint64) *SoReportListWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoReportListWrap{dba, key, -1, nil, nil}
	return result
}

func (s *SoReportListWrap) CheckExist() bool {
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

func (s *SoReportListWrap) Create(f func(tInfo *SoReportList)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoReportList{}
	f(val)
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

func (s *SoReportListWrap) getMainKeyBuf() ([]byte, error) {
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

func (s *SoReportListWrap) delSortKeyReportedTimes(sa *SoReportList) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListReportListByReportedTimes{}
	if sa == nil {
		key, err := s.encodeMemKey("ReportedTimes")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemReportListByReportedTimes{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		val.ReportedTimes = ori.ReportedTimes
		val.Uuid = *s.mainKey
	} else {
		val.ReportedTimes = sa.ReportedTimes
		val.Uuid = sa.Uuid
	}

	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoReportListWrap) insertSortKeyReportedTimes(sa *SoReportList) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListReportListByReportedTimes{}
	val.Uuid = sa.Uuid
	val.ReportedTimes = sa.ReportedTimes
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

func (s *SoReportListWrap) delAllSortKeys(br bool, val *SoReportList) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delSortKeyReportedTimes(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoReportListWrap) insertAllSortKeys(val *SoReportList) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoReportList fail ")
	}
	if !s.insertSortKeyReportedTimes(val) {
		return errors.New("insert sort Field ReportedTimes fail while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoReportListWrap) RemoveReportList() bool {
	if s.dba == nil {
		return false
	}
	val := &SoReportList{}
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
func (s *SoReportListWrap) getMemKeyPrefix(fName string) uint32 {
	if fName == "IsArbitrated" {
		return ReportListIsArbitratedCell
	}
	if fName == "ReportedTimes" {
		return ReportListReportedTimesCell
	}
	if fName == "Tags" {
		return ReportListTagsCell
	}
	if fName == "Uuid" {
		return ReportListUuidCell
	}

	return 0
}

func (s *SoReportListWrap) encodeMemKey(fName string) ([]byte, error) {
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

func (s *SoReportListWrap) saveAllMemKeys(tInfo *SoReportList, br bool) error {
	if s.dba == nil {
		return errors.New("save member Field fail , the db is nil")
	}

	if tInfo == nil {
		return errors.New("save member Field fail , the data is nil ")
	}
	var err error = nil
	errDes := ""
	if err = s.saveMemKeyIsArbitrated(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "IsArbitrated", err)
		}
	}
	if err = s.saveMemKeyReportedTimes(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "ReportedTimes", err)
		}
	}
	if err = s.saveMemKeyTags(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Tags", err)
		}
	}
	if err = s.saveMemKeyUuid(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Uuid", err)
		}
	}

	if len(errDes) > 0 {
		return errors.New(errDes)
	}
	return err
}

func (s *SoReportListWrap) delAllMemKeys(br bool, tInfo *SoReportList) error {
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

func (s *SoReportListWrap) delMemKey(fName string) error {
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

func (s *SoReportListWrap) saveMemKeyIsArbitrated(tInfo *SoReportList) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemReportListByIsArbitrated{}
	val.IsArbitrated = tInfo.IsArbitrated
	key, err := s.encodeMemKey("IsArbitrated")
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

func (s *SoReportListWrap) GetIsArbitrated() bool {
	res := true
	msg := &SoMemReportListByIsArbitrated{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("IsArbitrated")
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
				return msg.IsArbitrated
			}
		}
	}
	if !res {
		var tmpValue bool
		return tmpValue
	}
	return msg.IsArbitrated
}

func (s *SoReportListWrap) MdIsArbitrated(p bool) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("IsArbitrated")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemReportListByIsArbitrated{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoReportList{}
	sa.Uuid = *s.mainKey
	sa.IsArbitrated = ori.IsArbitrated

	ori.IsArbitrated = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.IsArbitrated = p

	return true
}

func (s *SoReportListWrap) saveMemKeyReportedTimes(tInfo *SoReportList) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemReportListByReportedTimes{}
	val.ReportedTimes = tInfo.ReportedTimes
	key, err := s.encodeMemKey("ReportedTimes")
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

func (s *SoReportListWrap) GetReportedTimes() uint32 {
	res := true
	msg := &SoMemReportListByReportedTimes{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("ReportedTimes")
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
				return msg.ReportedTimes
			}
		}
	}
	if !res {
		var tmpValue uint32
		return tmpValue
	}
	return msg.ReportedTimes
}

func (s *SoReportListWrap) MdReportedTimes(p uint32) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("ReportedTimes")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemReportListByReportedTimes{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoReportList{}
	sa.Uuid = *s.mainKey
	sa.ReportedTimes = ori.ReportedTimes

	if !s.delSortKeyReportedTimes(sa) {
		return false
	}
	ori.ReportedTimes = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.ReportedTimes = p

	if !s.insertSortKeyReportedTimes(sa) {
		return false
	}

	return true
}

func (s *SoReportListWrap) saveMemKeyTags(tInfo *SoReportList) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemReportListByTags{}
	val.Tags = tInfo.Tags
	key, err := s.encodeMemKey("Tags")
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

func (s *SoReportListWrap) GetTags() []int32 {
	res := true
	msg := &SoMemReportListByTags{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Tags")
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
				return msg.Tags
			}
		}
	}
	if !res {
		var tmpValue []int32
		return tmpValue
	}
	return msg.Tags
}

func (s *SoReportListWrap) MdTags(p []int32) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Tags")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemReportListByTags{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoReportList{}
	sa.Uuid = *s.mainKey
	sa.Tags = ori.Tags

	ori.Tags = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Tags = p

	return true
}

func (s *SoReportListWrap) saveMemKeyUuid(tInfo *SoReportList) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemReportListByUuid{}
	val.Uuid = tInfo.Uuid
	key, err := s.encodeMemKey("Uuid")
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

func (s *SoReportListWrap) GetUuid() uint64 {
	res := true
	msg := &SoMemReportListByUuid{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Uuid")
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
				return msg.Uuid
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.Uuid
}

////////////// SECTION List Keys ///////////////
type SReportListReportedTimesWrap struct {
	Dba iservices.IDatabaseRW
}

func NewReportListReportedTimesWrap(db iservices.IDatabaseRW) *SReportListReportedTimesWrap {
	if db == nil {
		return nil
	}
	wrap := SReportListReportedTimesWrap{Dba: db}
	return &wrap
}

func (s *SReportListReportedTimesWrap) DelIterator(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SReportListReportedTimesWrap) GetMainVal(iterator iservices.IDatabaseIterator) *uint64 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListReportListByReportedTimes{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return &res.Uuid

}

func (s *SReportListReportedTimesWrap) GetSubVal(iterator iservices.IDatabaseIterator) *uint32 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListReportListByReportedTimes{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return &res.ReportedTimes

}

func (m *SoListReportListByReportedTimes) OpeEncode() ([]byte, error) {
	pre := ReportListReportedTimesTable
	sub := m.ReportedTimes

	sub1 := m.Uuid

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
func (s *SReportListReportedTimesWrap) ForEachByOrder(start *uint32, end *uint32, lastMainKey *uint64,
	lastSubVal *uint32, f func(mVal *uint64, sVal *uint32, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := ReportListReportedTimesTable
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

func (s *SoReportListWrap) update(sa *SoReportList) bool {
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

func (s *SoReportListWrap) getReportList() *SoReportList {
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

	res := &SoReportList{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoReportListWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := s.getMemKeyPrefix("Uuid")
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

func (s *SoReportListWrap) delAllUniKeys(br bool, val *SoReportList) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delUniKeyUuid(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoReportListWrap) delUniKeysWithNames(names map[string]string, val *SoReportList) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if len(names["Uuid"]) > 0 {
		if !s.delUniKeyUuid(val) {
			res = false
		}
	}

	return res
}

func (s *SoReportListWrap) insertAllUniKeys(val *SoReportList) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoReportList fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyUuid(val) {
		return sucFields, errors.New("insert unique Field Uuid fail while insert table ")
	}
	sucFields["Uuid"] = "Uuid"

	return sucFields, nil
}

func (s *SoReportListWrap) delUniKeyUuid(sa *SoReportList) bool {
	if s.dba == nil {
		return false
	}
	pre := ReportListUuidUniTable
	kList := []interface{}{pre}
	if sa != nil {

		sub := sa.Uuid
		kList = append(kList, sub)
	} else {
		key, err := s.encodeMemKey("Uuid")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemReportListByUuid{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		sub := ori.Uuid
		kList = append(kList, sub)

	}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoReportListWrap) insertUniKeyUuid(sa *SoReportList) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	pre := ReportListUuidUniTable
	sub := sa.Uuid
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
	val := SoUniqueReportListByUuid{}
	val.Uuid = sa.Uuid

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniReportListUuidWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniReportListUuidWrap(db iservices.IDatabaseRW) *UniReportListUuidWrap {
	if db == nil {
		return nil
	}
	wrap := UniReportListUuidWrap{Dba: db}
	return &wrap
}

func (s *UniReportListUuidWrap) UniQueryUuid(start *uint64) *SoReportListWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := ReportListUuidUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueReportListByUuid{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoReportListWrap(s.Dba, &res.Uuid)
			return wrap
		}
	}
	return nil
}
