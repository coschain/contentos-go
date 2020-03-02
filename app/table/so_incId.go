package table

import (
	"bytes"
	"encoding/json"
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
	IncIdIdUniTable uint32 = 561605276

	IncIdIdRow uint32 = 3838899201
)

////////////// SECTION Wrap Define ///////////////
type SoIncIdWrap struct {
	dba         iservices.IDatabaseRW
	mainKey     *int32
	watcherFlag *IncIdWatcherFlag
	mKeyFlag    int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf     []byte //the buffer after the main key is encoded with prefix
	mBuf        []byte //the value after the main key is encoded
	mdFuncMap   map[string]interface{}
}

func NewSoIncIdWrap(dba iservices.IDatabaseRW, key *int32) *SoIncIdWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoIncIdWrap{dba, key, nil, -1, nil, nil, nil}
	result.initWatcherFlag()
	return result
}

func (s *SoIncIdWrap) CheckExist() bool {
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

func (s *SoIncIdWrap) MustExist(errMsgs ...interface{}) *SoIncIdWrap {
	if !s.CheckExist() {
		panic(bindErrorInfo(fmt.Sprintf("SoIncIdWrap.MustExist: %v not found", s.mainKey), errMsgs...))
	}
	return s
}

func (s *SoIncIdWrap) MustNotExist(errMsgs ...interface{}) *SoIncIdWrap {
	if s.CheckExist() {
		panic(bindErrorInfo(fmt.Sprintf("SoIncIdWrap.MustNotExist: %v already exists", s.mainKey), errMsgs...))
	}
	return s
}

func (s *SoIncIdWrap) initWatcherFlag() {
	if s.watcherFlag == nil {
		s.watcherFlag = new(IncIdWatcherFlag)
		*(s.watcherFlag) = IncIdWatcherFlagOfDb(s.dba.ServiceId())
	}
}

func (s *SoIncIdWrap) create(f func(tInfo *SoIncId)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoIncId{}
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

func (s *SoIncIdWrap) Create(f func(tInfo *SoIncId), errArgs ...interface{}) *SoIncIdWrap {
	err := s.create(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Errorf("SoIncIdWrap.Create failed: %s", err.Error()), errArgs...))
	}
	return s
}

func (s *SoIncIdWrap) getMainKeyBuf() ([]byte, error) {
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

func (s *SoIncIdWrap) modify(f func(tInfo *SoIncId)) error {
	if !s.CheckExist() {
		return errors.New("the SoIncId table does not exist. Please create a table first")
	}
	oriTable := s.getIncId()
	if oriTable == nil {
		return errors.New("fail to get origin table SoIncId")
	}

	curTable := s.getIncId()
	if curTable == nil {
		return errors.New("fail to create current table SoIncId")
	}
	f(curTable)

	//the main key is not support modify
	if !reflect.DeepEqual(curTable.Id, oriTable.Id) {
		return errors.New("primary key does not support modification")
	}

	s.initWatcherFlag()
	modifiedFields, hasWatcher, err := s.getModifiedFields(oriTable, curTable)
	if err != nil {
		return err
	}

	if modifiedFields == nil || len(modifiedFields) < 1 {
		return nil
	}

	//check whether modify sort and unique field to nil
	err = s.checkSortAndUniFieldValidity(curTable, modifiedFields)
	if err != nil {
		return err
	}

	//check unique
	err = s.handleFieldMd(FieldMdHandleTypeCheck, curTable, modifiedFields)
	if err != nil {
		return err
	}

	//delete sort and unique key
	err = s.handleFieldMd(FieldMdHandleTypeDel, oriTable, modifiedFields)
	if err != nil {
		return err
	}

	//update table
	err = s.updateIncId(curTable)
	if err != nil {
		return err
	}

	//insert sort and unique key
	err = s.handleFieldMd(FieldMdHandleTypeInsert, curTable, modifiedFields)
	if err != nil {
		return err
	}

	// call watchers
	if hasWatcher {
		ReportTableRecordUpdate(s.dba.ServiceId(), s.dba.BranchId(), s.mainKey, oriTable, curTable, modifiedFields)
	}

	return nil

}

func (s *SoIncIdWrap) Modify(f func(tInfo *SoIncId), errArgs ...interface{}) *SoIncIdWrap {
	err := s.modify(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoIncIdWrap.Modify failed: %s", err.Error()), errArgs...))
	}
	return s
}

func (s *SoIncIdWrap) SetCounter(p uint64, errArgs ...interface{}) *SoIncIdWrap {
	err := s.modify(func(r *SoIncId) {
		r.Counter = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoIncIdWrap.SetCounter( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoIncIdWrap) checkSortAndUniFieldValidity(curTable *SoIncId, fields map[string]bool) error {
	if curTable != nil && fields != nil && len(fields) > 0 {

	}
	return nil
}

//Get all the modified fields in the table
func (s *SoIncIdWrap) getModifiedFields(oriTable *SoIncId, curTable *SoIncId) (map[string]bool, bool, error) {
	if oriTable == nil {
		return nil, false, errors.New("table info is nil, can't get modified fields")
	}
	hasWatcher := false
	fields := make(map[string]bool)

	if !reflect.DeepEqual(oriTable.Counter, curTable.Counter) {
		fields["Counter"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasCounterWatcher
	}

	hasWatcher = hasWatcher || s.watcherFlag.WholeWatcher
	return fields, hasWatcher, nil
}

func (s *SoIncIdWrap) handleFieldMd(t FieldMdHandleType, so *SoIncId, fields map[string]bool) error {
	if so == nil {
		return errors.New("fail to modify empty table")
	}

	//there is no field need to modify
	if fields == nil || len(fields) < 1 {
		return nil
	}

	errStr := ""

	if fields["Counter"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldCounter(so.Counter, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "Counter")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldCounter(so.Counter, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "Counter")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldCounter(so.Counter, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "Counter")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	return nil
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoIncIdWrap) delAllSortKeys(br bool, val *SoIncId) bool {
	if s.dba == nil {
		return false
	}
	res := true

	return res
}

func (s *SoIncIdWrap) insertAllSortKeys(val *SoIncId) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoIncId fail ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoIncIdWrap) removeIncId() error {
	if s.dba == nil {
		return errors.New("database is nil")
	}

	s.initWatcherFlag()

	var oldVal *SoIncId
	if s.watcherFlag.AnyWatcher {
		oldVal = s.getIncId()
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

func (s *SoIncIdWrap) RemoveIncId(errMsgs ...interface{}) *SoIncIdWrap {
	err := s.removeIncId()
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoIncIdWrap.RemoveIncId failed: %s", err.Error()), errMsgs...))
	}
	return s
}

////////////// SECTION Members Get/Modify ///////////////

func (s *SoIncIdWrap) GetCounter() uint64 {
	res := true
	msg := &SoIncId{}
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
				return msg.Counter
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.Counter
}

func (s *SoIncIdWrap) mdFieldCounter(p uint64, isCheck bool, isDel bool, isInsert bool,
	so *SoIncId) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkCounterIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldCounter(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldCounter(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoIncIdWrap) delFieldCounter(so *SoIncId) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoIncIdWrap) insertFieldCounter(so *SoIncId) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoIncIdWrap) checkCounterIsMetMdCondition(p uint64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoIncIdWrap) GetId() int32 {
	res := true
	msg := &SoIncId{}
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
				return msg.Id
			}
		}
	}
	if !res {
		var tmpValue int32
		return tmpValue
	}
	return msg.Id
}

/////////////// SECTION Private function ////////////////

func (s *SoIncIdWrap) update(sa *SoIncId) bool {
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

func (s *SoIncIdWrap) getIncId() *SoIncId {
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

	res := &SoIncId{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoIncIdWrap) updateIncId(so *SoIncId) error {
	if s.dba == nil {
		return errors.New("update fail:the db is nil")
	}

	if so == nil {
		return errors.New("update fail: the SoIncId is nil")
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

func (s *SoIncIdWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := IncIdIdRow
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

func (s *SoIncIdWrap) delAllUniKeys(br bool, val *SoIncId) bool {
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

func (s *SoIncIdWrap) delUniKeysWithNames(names map[string]string, val *SoIncId) bool {
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

func (s *SoIncIdWrap) insertAllUniKeys(val *SoIncId) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoIncId fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyId(val) {
		return sucFields, errors.New("insert unique Field Id fail while insert table ")
	}
	sucFields["Id"] = "Id"

	return sucFields, nil
}

func (s *SoIncIdWrap) delUniKeyId(sa *SoIncId) bool {
	if s.dba == nil {
		return false
	}
	pre := IncIdIdUniTable
	kList := []interface{}{pre}
	if sa != nil {

		sub := sa.Id
		kList = append(kList, sub)
	} else {
		sub := s.GetId()

		kList = append(kList, sub)

	}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoIncIdWrap) insertUniKeyId(sa *SoIncId) bool {
	if s.dba == nil || sa == nil {
		return false
	}

	pre := IncIdIdUniTable
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
	val := SoUniqueIncIdById{}
	val.Id = sa.Id

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniIncIdIdWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniIncIdIdWrap(db iservices.IDatabaseRW) *UniIncIdIdWrap {
	if db == nil {
		return nil
	}
	wrap := UniIncIdIdWrap{Dba: db}
	return &wrap
}

func (s *UniIncIdIdWrap) UniQueryId(start *int32) *SoIncIdWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := IncIdIdUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueIncIdById{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoIncIdWrap(s.Dba, &res.Id)
			return wrap
		}
	}
	return nil
}

////////////// SECTION Watchers ///////////////

type IncIdWatcherFlag struct {
	HasCounterWatcher bool

	WholeWatcher bool
	AnyWatcher   bool
}

var (
	IncIdTable = &TableInfo{
		Name:    "IncId",
		Primary: "Id",
		Record:  reflect.TypeOf((*SoIncId)(nil)).Elem(),
	}
	IncIdWatcherFlags     = make(map[uint32]IncIdWatcherFlag)
	IncIdWatcherFlagsLock sync.RWMutex
)

func IncIdWatcherFlagOfDb(dbSvcId uint32) IncIdWatcherFlag {
	IncIdWatcherFlagsLock.RLock()
	defer IncIdWatcherFlagsLock.RUnlock()
	return IncIdWatcherFlags[dbSvcId]
}

func IncIdRecordWatcherChanged(dbSvcId uint32) {
	var flag IncIdWatcherFlag
	flag.WholeWatcher = HasTableRecordWatcher(dbSvcId, IncIdTable.Record, "")
	flag.AnyWatcher = flag.WholeWatcher

	flag.HasCounterWatcher = HasTableRecordWatcher(dbSvcId, IncIdTable.Record, "Counter")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasCounterWatcher

	IncIdWatcherFlagsLock.Lock()
	IncIdWatcherFlags[dbSvcId] = flag
	IncIdWatcherFlagsLock.Unlock()
}

////////////// SECTION Json query ///////////////

func IncIdQuery(db iservices.IDatabaseRW, keyJson string) (valueJson string, err error) {
	k := new(int32)
	d := json.NewDecoder(bytes.NewReader([]byte(keyJson)))
	d.UseNumber()
	if err = d.Decode(k); err != nil {
		return
	}
	if v := NewSoIncIdWrap(db, k).getIncId(); v == nil {
		err = errors.New("not found")
	} else {
		var jbytes []byte
		if jbytes, err = json.Marshal(v); err == nil {
			valueJson = string(jbytes)
		}
	}
	return
}

func init() {
	RegisterTableWatcherChangedCallback(IncIdTable.Record, IncIdRecordWatcherChanged)
	RegisterTableJsonQuery("IncId", IncIdQuery)
}
