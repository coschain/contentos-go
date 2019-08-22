package table

import (
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/coschain/contentos-go/common/encoding/kope"
	"github.com/coschain/contentos-go/iservices"
	proto "github.com/golang/protobuf/proto"
)

////////////// SECTION Prefix Mark ///////////////
var (
	ReportListReportedTimesTable uint32 = 4124045745
	ReportListUuidUniTable       uint32 = 4051252686

	ReportListUuidRow uint32 = 1111682916
)

////////////// SECTION Wrap Define ///////////////
type SoReportListWrap struct {
	dba         iservices.IDatabaseRW
	mainKey     *uint64
	watcherFlag *ReportListWatcherFlag
	mKeyFlag    int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf     []byte //the buffer after the main key is encoded with prefix
	mBuf        []byte //the value after the main key is encoded
	mdFuncMap   map[string]interface{}
}

func NewSoReportListWrap(dba iservices.IDatabaseRW, key *uint64) *SoReportListWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoReportListWrap{dba, key, nil, -1, nil, nil, nil}
	result.initWatcherFlag()
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

func (s *SoReportListWrap) MustExist(errMsgs ...interface{}) *SoReportListWrap {
	if !s.CheckExist() {
		panic(bindErrorInfo(fmt.Sprintf("SoReportListWrap.MustExist: %v not found", s.mainKey), errMsgs...))
	}
	return s
}

func (s *SoReportListWrap) MustNotExist(errMsgs ...interface{}) *SoReportListWrap {
	if s.CheckExist() {
		panic(bindErrorInfo(fmt.Sprintf("SoReportListWrap.MustNotExist: %v already exists", s.mainKey), errMsgs...))
	}
	return s
}

func (s *SoReportListWrap) initWatcherFlag() {
	if s.watcherFlag == nil {
		s.watcherFlag = new(ReportListWatcherFlag)
		*(s.watcherFlag) = ReportListWatcherFlagOfDb(s.dba.ServiceId())
	}
}

func (s *SoReportListWrap) create(f func(tInfo *SoReportList)) error {
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

	buf, err := proto.Marshal(val)
	if err != nil {
		return err
	}
	err = s.dba.Put(keyBuf, buf)
	if err != nil {
		return err
	}

	// update srt list keys
	if err = s.insertAllSortKeys(val); err != nil {
		s.delAllSortKeys(false, val)
		s.dba.Delete(keyBuf)
		return err
	}

	//update unique list
	if sucNames, err := s.insertAllUniKeys(val); err != nil {
		s.delAllSortKeys(false, val)
		s.delUniKeysWithNames(sucNames, val)
		s.dba.Delete(keyBuf)
		return err
	}

	s.mKeyFlag = 1

	// call watchers
	s.initWatcherFlag()
	if s.watcherFlag.AnyWatcher {
		ReportTableRecordInsert(s.dba.ServiceId(), s.dba.BranchId(), s.mainKey, val)
	}

	return nil
}

func (s *SoReportListWrap) Create(f func(tInfo *SoReportList), errArgs ...interface{}) *SoReportListWrap {
	err := s.create(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Errorf("SoReportListWrap.Create failed: %s", err.Error()), errArgs...))
	}
	return s
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

func (s *SoReportListWrap) modify(f func(tInfo *SoReportList)) error {
	if !s.CheckExist() {
		return errors.New("the SoReportList table does not exist. Please create a table first")
	}
	oriTable := s.getReportList()
	if oriTable == nil {
		return errors.New("fail to get origin table SoReportList")
	}

	curTable := s.getReportList()
	if curTable == nil {
		return errors.New("fail to create current table SoReportList")
	}
	f(curTable)

	//the main key is not support modify
	if !reflect.DeepEqual(curTable.Uuid, oriTable.Uuid) {
		return errors.New("primary key does not support modification")
	}

	s.initWatcherFlag()
	fieldSli, hasWatcher, err := s.getModifiedFields(oriTable, curTable)
	if err != nil {
		return err
	}

	if fieldSli == nil || len(fieldSli) < 1 {
		return nil
	}

	//check whether modify sort and unique field to nil
	err = s.checkSortAndUniFieldValidity(curTable, fieldSli)
	if err != nil {
		return err
	}

	//check unique
	err = s.handleFieldMd(FieldMdHandleTypeCheck, curTable, fieldSli)
	if err != nil {
		return err
	}

	//delete sort and unique key
	err = s.handleFieldMd(FieldMdHandleTypeDel, oriTable, fieldSli)
	if err != nil {
		return err
	}

	//update table
	err = s.updateReportList(curTable)
	if err != nil {
		return err
	}

	//insert sort and unique key
	err = s.handleFieldMd(FieldMdHandleTypeInsert, curTable, fieldSli)
	if err != nil {
		return err
	}

	// call watchers
	if hasWatcher {
		ReportTableRecordUpdate(s.dba.ServiceId(), s.dba.BranchId(), s.mainKey, oriTable, curTable)
	}

	return nil

}

func (s *SoReportListWrap) Modify(f func(tInfo *SoReportList), errArgs ...interface{}) *SoReportListWrap {
	err := s.modify(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoReportListWrap.Modify failed: %s", err.Error()), errArgs...))
	}
	return s
}

func (s *SoReportListWrap) SetIsArbitrated(p bool, errArgs ...interface{}) *SoReportListWrap {
	err := s.modify(func(r *SoReportList) {
		r.IsArbitrated = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoReportListWrap.SetIsArbitrated( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoReportListWrap) SetReportedTimes(p uint32, errArgs ...interface{}) *SoReportListWrap {
	err := s.modify(func(r *SoReportList) {
		r.ReportedTimes = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoReportListWrap.SetReportedTimes( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoReportListWrap) SetTags(p []int32, errArgs ...interface{}) *SoReportListWrap {
	err := s.modify(func(r *SoReportList) {
		r.Tags = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoReportListWrap.SetTags( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoReportListWrap) checkSortAndUniFieldValidity(curTable *SoReportList, fields map[string]bool) error {
	if curTable != nil && fields != nil && len(fields) > 0 {

	}
	return nil
}

//Get all the modified fields in the table
func (s *SoReportListWrap) getModifiedFields(oriTable *SoReportList, curTable *SoReportList) (map[string]bool, bool, error) {
	if oriTable == nil {
		return nil, false, errors.New("table info is nil, can't get modified fields")
	}
	hasWatcher := false
	fields := make(map[string]bool)

	if !reflect.DeepEqual(oriTable.IsArbitrated, curTable.IsArbitrated) {
		fields["IsArbitrated"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasIsArbitratedWatcher
	}

	if !reflect.DeepEqual(oriTable.ReportedTimes, curTable.ReportedTimes) {
		fields["ReportedTimes"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasReportedTimesWatcher
	}

	if !reflect.DeepEqual(oriTable.Tags, curTable.Tags) {
		fields["Tags"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasTagsWatcher
	}

	hasWatcher = hasWatcher || s.watcherFlag.WholeWatcher
	return fields, hasWatcher, nil
}

func (s *SoReportListWrap) handleFieldMd(t FieldMdHandleType, so *SoReportList, fields map[string]bool) error {
	if so == nil {
		return errors.New("fail to modify empty table")
	}

	//there is no field need to modify
	if fields == nil || len(fields) < 1 {
		return nil
	}

	errStr := ""

	if fields["IsArbitrated"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldIsArbitrated(so.IsArbitrated, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "IsArbitrated")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldIsArbitrated(so.IsArbitrated, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "IsArbitrated")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldIsArbitrated(so.IsArbitrated, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "IsArbitrated")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["ReportedTimes"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldReportedTimes(so.ReportedTimes, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "ReportedTimes")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldReportedTimes(so.ReportedTimes, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "ReportedTimes")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldReportedTimes(so.ReportedTimes, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "ReportedTimes")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["Tags"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldTags(so.Tags, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "Tags")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldTags(so.Tags, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "Tags")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldTags(so.Tags, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "Tags")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	return nil
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoReportListWrap) delSortKeyReportedTimes(sa *SoReportList) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListReportListByReportedTimes{}
	if sa == nil {
		val.ReportedTimes = s.GetReportedTimes()
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

func (s *SoReportListWrap) removeReportList() error {
	if s.dba == nil {
		return errors.New("database is nil")
	}

	s.initWatcherFlag()

	var oldVal *SoReportList
	if s.watcherFlag.AnyWatcher {
		oldVal = s.getReportList()
	}

	//delete sort list key
	if res := s.delAllSortKeys(true, nil); !res {
		return errors.New("delAllSortKeys failed")
	}

	//delete unique list
	if res := s.delAllUniKeys(true, nil); !res {
		return errors.New("delAllUniKeys failed")
	}

	//delete table
	key, err := s.encodeMainKey()
	if err != nil {
		return fmt.Errorf("encodeMainKey failed: %s", err.Error())
	}
	err = s.dba.Delete(key)
	if err == nil {
		s.mKeyBuf = nil
		s.mKeyFlag = -1

		// call watchers
		if s.watcherFlag.AnyWatcher && oldVal != nil {
			ReportTableRecordDelete(s.dba.ServiceId(), s.dba.BranchId(), s.mainKey, oldVal)
		}
		return nil
	} else {
		return fmt.Errorf("database.Delete failed: %s", err.Error())
	}
}

func (s *SoReportListWrap) RemoveReportList(errMsgs ...interface{}) *SoReportListWrap {
	err := s.removeReportList()
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoReportListWrap.RemoveReportList failed: %s", err.Error()), errMsgs...))
	}
	return s
}

////////////// SECTION Members Get/Modify ///////////////

func (s *SoReportListWrap) GetIsArbitrated() bool {
	res := true
	msg := &SoReportList{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMainKey()
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

func (s *SoReportListWrap) mdFieldIsArbitrated(p bool, isCheck bool, isDel bool, isInsert bool,
	so *SoReportList) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkIsArbitratedIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldIsArbitrated(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldIsArbitrated(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoReportListWrap) delFieldIsArbitrated(so *SoReportList) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoReportListWrap) insertFieldIsArbitrated(so *SoReportList) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoReportListWrap) checkIsArbitratedIsMetMdCondition(p bool) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoReportListWrap) GetReportedTimes() uint32 {
	res := true
	msg := &SoReportList{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMainKey()
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

func (s *SoReportListWrap) mdFieldReportedTimes(p uint32, isCheck bool, isDel bool, isInsert bool,
	so *SoReportList) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkReportedTimesIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldReportedTimes(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldReportedTimes(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoReportListWrap) delFieldReportedTimes(so *SoReportList) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyReportedTimes(so) {
		return false
	}

	return true
}

func (s *SoReportListWrap) insertFieldReportedTimes(so *SoReportList) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyReportedTimes(so) {
		return false
	}

	return true
}

func (s *SoReportListWrap) checkReportedTimesIsMetMdCondition(p uint32) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoReportListWrap) GetTags() []int32 {
	res := true
	msg := &SoReportList{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMainKey()
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

func (s *SoReportListWrap) mdFieldTags(p []int32, isCheck bool, isDel bool, isInsert bool,
	so *SoReportList) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkTagsIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldTags(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldTags(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoReportListWrap) delFieldTags(so *SoReportList) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoReportListWrap) insertFieldTags(so *SoReportList) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoReportListWrap) checkTagsIsMetMdCondition(p []int32) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoReportListWrap) GetUuid() uint64 {
	res := true
	msg := &SoReportList{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMainKey()
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

func (s *SReportListReportedTimesWrap) GetMainVal(val []byte) *uint64 {
	res := &SoListReportListByReportedTimes{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return &res.Uuid

}

func (s *SReportListReportedTimesWrap) GetSubVal(val []byte) *uint32 {
	res := &SoListReportListByReportedTimes{}
	err := proto.Unmarshal(val, res)
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
	var idx uint32 = 0
	s.Dba.Iterate(sBuf, eBuf, false, func(key, value []byte) bool {
		idx++
		return f(s.GetMainVal(value), s.GetSubVal(value), idx)
	})
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

func (s *SoReportListWrap) updateReportList(so *SoReportList) error {
	if s.dba == nil {
		return errors.New("update fail:the db is nil")
	}

	if so == nil {
		return errors.New("update fail: the SoReportList is nil")
	}

	key, err := s.encodeMainKey()
	if err != nil {
		return nil
	}

	buf, err := proto.Marshal(so)
	if err != nil {
		return err
	}

	err = s.dba.Put(key, buf)
	if err != nil {
		return err
	}

	return nil
}

func (s *SoReportListWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := ReportListUuidRow
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
		sub := s.GetUuid()

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

////////////// SECTION Watchers ///////////////

type ReportListWatcherFlag struct {
	HasIsArbitratedWatcher bool

	HasReportedTimesWatcher bool

	HasTagsWatcher bool

	WholeWatcher bool
	AnyWatcher   bool
}

var (
	ReportListRecordType       = reflect.TypeOf((*SoReportList)(nil)).Elem()
	ReportListWatcherFlags     = make(map[uint32]ReportListWatcherFlag)
	ReportListWatcherFlagsLock sync.RWMutex
)

func ReportListWatcherFlagOfDb(dbSvcId uint32) ReportListWatcherFlag {
	ReportListWatcherFlagsLock.RLock()
	defer ReportListWatcherFlagsLock.RUnlock()
	return ReportListWatcherFlags[dbSvcId]
}

func ReportListRecordWatcherChanged(dbSvcId uint32) {
	var flag ReportListWatcherFlag
	flag.WholeWatcher = HasTableRecordWatcher(dbSvcId, ReportListRecordType, "")
	flag.AnyWatcher = flag.WholeWatcher

	flag.HasIsArbitratedWatcher = HasTableRecordWatcher(dbSvcId, ReportListRecordType, "IsArbitrated")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasIsArbitratedWatcher

	flag.HasReportedTimesWatcher = HasTableRecordWatcher(dbSvcId, ReportListRecordType, "ReportedTimes")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasReportedTimesWatcher

	flag.HasTagsWatcher = HasTableRecordWatcher(dbSvcId, ReportListRecordType, "Tags")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasTagsWatcher

	ReportListWatcherFlagsLock.Lock()
	ReportListWatcherFlags[dbSvcId] = flag
	ReportListWatcherFlagsLock.Unlock()
}

func init() {
	RegisterTableWatcherChangedCallback(ReportListRecordType, ReportListRecordWatcherChanged)
}
