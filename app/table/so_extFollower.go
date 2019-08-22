package table

import (
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/coschain/contentos-go/common/encoding/kope"
	"github.com/coschain/contentos-go/iservices"
	prototype "github.com/coschain/contentos-go/prototype"
	proto "github.com/golang/protobuf/proto"
)

////////////// SECTION Prefix Mark ///////////////
var (
	ExtFollowerFollowerCreatedOrderTable uint32 = 1742944534
	ExtFollowerFollowerInfoUniTable      uint32 = 15777514

	ExtFollowerFollowerInfoRow uint32 = 3902153462
)

////////////// SECTION Wrap Define ///////////////
type SoExtFollowerWrap struct {
	dba         iservices.IDatabaseRW
	mainKey     *prototype.FollowerRelation
	watcherFlag *ExtFollowerWatcherFlag
	mKeyFlag    int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf     []byte //the buffer after the main key is encoded with prefix
	mBuf        []byte //the value after the main key is encoded
	mdFuncMap   map[string]interface{}
}

func NewSoExtFollowerWrap(dba iservices.IDatabaseRW, key *prototype.FollowerRelation) *SoExtFollowerWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoExtFollowerWrap{dba, key, nil, -1, nil, nil, nil}
	result.initWatcherFlag()
	return result
}

func (s *SoExtFollowerWrap) CheckExist() bool {
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

func (s *SoExtFollowerWrap) MustExist(errMsgs ...interface{}) *SoExtFollowerWrap {
	if !s.CheckExist() {
		panic(bindErrorInfo(fmt.Sprintf("SoExtFollowerWrap.MustExist: %v not found", s.mainKey), errMsgs...))
	}
	return s
}

func (s *SoExtFollowerWrap) MustNotExist(errMsgs ...interface{}) *SoExtFollowerWrap {
	if s.CheckExist() {
		panic(bindErrorInfo(fmt.Sprintf("SoExtFollowerWrap.MustNotExist: %v already exists", s.mainKey), errMsgs...))
	}
	return s
}

func (s *SoExtFollowerWrap) initWatcherFlag() {
	if s.watcherFlag == nil {
		s.watcherFlag = new(ExtFollowerWatcherFlag)
		*(s.watcherFlag) = ExtFollowerWatcherFlagOfDb(s.dba.ServiceId())
	}
}

func (s *SoExtFollowerWrap) create(f func(tInfo *SoExtFollower)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoExtFollower{}
	f(val)
	if val.FollowerInfo == nil {
		val.FollowerInfo = s.mainKey
	}
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

func (s *SoExtFollowerWrap) Create(f func(tInfo *SoExtFollower), errArgs ...interface{}) *SoExtFollowerWrap {
	err := s.create(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Errorf("SoExtFollowerWrap.Create failed: %s", err.Error()), errArgs...))
	}
	return s
}

func (s *SoExtFollowerWrap) getMainKeyBuf() ([]byte, error) {
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

func (s *SoExtFollowerWrap) modify(f func(tInfo *SoExtFollower)) error {
	if !s.CheckExist() {
		return errors.New("the SoExtFollower table does not exist. Please create a table first")
	}
	oriTable := s.getExtFollower()
	if oriTable == nil {
		return errors.New("fail to get origin table SoExtFollower")
	}

	curTable := s.getExtFollower()
	if curTable == nil {
		return errors.New("fail to create current table SoExtFollower")
	}
	f(curTable)

	//the main key is not support modify
	if !reflect.DeepEqual(curTable.FollowerInfo, oriTable.FollowerInfo) {
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
	err = s.updateExtFollower(curTable)
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

func (s *SoExtFollowerWrap) Modify(f func(tInfo *SoExtFollower), errArgs ...interface{}) *SoExtFollowerWrap {
	err := s.modify(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoExtFollowerWrap.Modify failed: %s", err.Error()), errArgs...))
	}
	return s
}

func (s *SoExtFollowerWrap) SetFollowerCreatedOrder(p *prototype.FollowerCreatedOrder, errArgs ...interface{}) *SoExtFollowerWrap {
	err := s.modify(func(r *SoExtFollower) {
		r.FollowerCreatedOrder = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoExtFollowerWrap.SetFollowerCreatedOrder( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoExtFollowerWrap) checkSortAndUniFieldValidity(curTable *SoExtFollower, fields map[string]bool) error {
	if curTable != nil && fields != nil && len(fields) > 0 {

		if fields["FollowerCreatedOrder"] && curTable.FollowerCreatedOrder == nil {
			return errors.New("sort field FollowerCreatedOrder can't be modified to nil")
		}

	}
	return nil
}

//Get all the modified fields in the table
func (s *SoExtFollowerWrap) getModifiedFields(oriTable *SoExtFollower, curTable *SoExtFollower) (map[string]bool, bool, error) {
	if oriTable == nil {
		return nil, false, errors.New("table info is nil, can't get modified fields")
	}
	hasWatcher := false
	fields := make(map[string]bool)

	if !reflect.DeepEqual(oriTable.FollowerCreatedOrder, curTable.FollowerCreatedOrder) {
		fields["FollowerCreatedOrder"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasFollowerCreatedOrderWatcher
	}

	hasWatcher = hasWatcher || s.watcherFlag.WholeWatcher
	return fields, hasWatcher, nil
}

func (s *SoExtFollowerWrap) handleFieldMd(t FieldMdHandleType, so *SoExtFollower, fields map[string]bool) error {
	if so == nil {
		return errors.New("fail to modify empty table")
	}

	//there is no field need to modify
	if fields == nil || len(fields) < 1 {
		return nil
	}

	errStr := ""

	if fields["FollowerCreatedOrder"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldFollowerCreatedOrder(so.FollowerCreatedOrder, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "FollowerCreatedOrder")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldFollowerCreatedOrder(so.FollowerCreatedOrder, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "FollowerCreatedOrder")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldFollowerCreatedOrder(so.FollowerCreatedOrder, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "FollowerCreatedOrder")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	return nil
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoExtFollowerWrap) delSortKeyFollowerCreatedOrder(sa *SoExtFollower) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListExtFollowerByFollowerCreatedOrder{}
	if sa == nil {
		val.FollowerCreatedOrder = s.GetFollowerCreatedOrder()
		val.FollowerInfo = s.mainKey

	} else {
		val.FollowerCreatedOrder = sa.FollowerCreatedOrder
		val.FollowerInfo = sa.FollowerInfo
	}
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoExtFollowerWrap) insertSortKeyFollowerCreatedOrder(sa *SoExtFollower) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListExtFollowerByFollowerCreatedOrder{}
	val.FollowerInfo = sa.FollowerInfo
	val.FollowerCreatedOrder = sa.FollowerCreatedOrder
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

func (s *SoExtFollowerWrap) delAllSortKeys(br bool, val *SoExtFollower) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delSortKeyFollowerCreatedOrder(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoExtFollowerWrap) insertAllSortKeys(val *SoExtFollower) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoExtFollower fail ")
	}
	if !s.insertSortKeyFollowerCreatedOrder(val) {
		return errors.New("insert sort Field FollowerCreatedOrder fail while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoExtFollowerWrap) removeExtFollower() error {
	if s.dba == nil {
		return errors.New("database is nil")
	}

	s.initWatcherFlag()

	var oldVal *SoExtFollower
	if s.watcherFlag.AnyWatcher {
		oldVal = s.getExtFollower()
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

func (s *SoExtFollowerWrap) RemoveExtFollower(errMsgs ...interface{}) *SoExtFollowerWrap {
	err := s.removeExtFollower()
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoExtFollowerWrap.RemoveExtFollower failed: %s", err.Error()), errMsgs...))
	}
	return s
}

////////////// SECTION Members Get/Modify ///////////////

func (s *SoExtFollowerWrap) GetFollowerCreatedOrder() *prototype.FollowerCreatedOrder {
	res := true
	msg := &SoExtFollower{}
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
				return msg.FollowerCreatedOrder
			}
		}
	}
	if !res {
		return nil

	}
	return msg.FollowerCreatedOrder
}

func (s *SoExtFollowerWrap) mdFieldFollowerCreatedOrder(p *prototype.FollowerCreatedOrder, isCheck bool, isDel bool, isInsert bool,
	so *SoExtFollower) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkFollowerCreatedOrderIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldFollowerCreatedOrder(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldFollowerCreatedOrder(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoExtFollowerWrap) delFieldFollowerCreatedOrder(so *SoExtFollower) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyFollowerCreatedOrder(so) {
		return false
	}

	return true
}

func (s *SoExtFollowerWrap) insertFieldFollowerCreatedOrder(so *SoExtFollower) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyFollowerCreatedOrder(so) {
		return false
	}

	return true
}

func (s *SoExtFollowerWrap) checkFollowerCreatedOrderIsMetMdCondition(p *prototype.FollowerCreatedOrder) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoExtFollowerWrap) GetFollowerInfo() *prototype.FollowerRelation {
	res := true
	msg := &SoExtFollower{}
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
				return msg.FollowerInfo
			}
		}
	}
	if !res {
		return nil

	}
	return msg.FollowerInfo
}

////////////// SECTION List Keys ///////////////
type SExtFollowerFollowerCreatedOrderWrap struct {
	Dba iservices.IDatabaseRW
}

func NewExtFollowerFollowerCreatedOrderWrap(db iservices.IDatabaseRW) *SExtFollowerFollowerCreatedOrderWrap {
	if db == nil {
		return nil
	}
	wrap := SExtFollowerFollowerCreatedOrderWrap{Dba: db}
	return &wrap
}

func (s *SExtFollowerFollowerCreatedOrderWrap) GetMainVal(val []byte) *prototype.FollowerRelation {
	res := &SoListExtFollowerByFollowerCreatedOrder{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.FollowerInfo

}

func (s *SExtFollowerFollowerCreatedOrderWrap) GetSubVal(val []byte) *prototype.FollowerCreatedOrder {
	res := &SoListExtFollowerByFollowerCreatedOrder{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.FollowerCreatedOrder

}

func (m *SoListExtFollowerByFollowerCreatedOrder) OpeEncode() ([]byte, error) {
	pre := ExtFollowerFollowerCreatedOrderTable
	sub := m.FollowerCreatedOrder
	if sub == nil {
		return nil, errors.New("the pro FollowerCreatedOrder is nil")
	}
	sub1 := m.FollowerInfo
	if sub1 == nil {
		return nil, errors.New("the mainkey FollowerInfo is nil")
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
func (s *SExtFollowerFollowerCreatedOrderWrap) ForEachByOrder(start *prototype.FollowerCreatedOrder, end *prototype.FollowerCreatedOrder, lastMainKey *prototype.FollowerRelation,
	lastSubVal *prototype.FollowerCreatedOrder, f func(mVal *prototype.FollowerRelation, sVal *prototype.FollowerCreatedOrder, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := ExtFollowerFollowerCreatedOrderTable
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

func (s *SoExtFollowerWrap) update(sa *SoExtFollower) bool {
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

func (s *SoExtFollowerWrap) getExtFollower() *SoExtFollower {
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

	res := &SoExtFollower{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoExtFollowerWrap) updateExtFollower(so *SoExtFollower) error {
	if s.dba == nil {
		return errors.New("update fail:the db is nil")
	}

	if so == nil {
		return errors.New("update fail: the SoExtFollower is nil")
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

func (s *SoExtFollowerWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := ExtFollowerFollowerInfoRow
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

func (s *SoExtFollowerWrap) delAllUniKeys(br bool, val *SoExtFollower) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delUniKeyFollowerInfo(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoExtFollowerWrap) delUniKeysWithNames(names map[string]string, val *SoExtFollower) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if len(names["FollowerInfo"]) > 0 {
		if !s.delUniKeyFollowerInfo(val) {
			res = false
		}
	}

	return res
}

func (s *SoExtFollowerWrap) insertAllUniKeys(val *SoExtFollower) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoExtFollower fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyFollowerInfo(val) {
		return sucFields, errors.New("insert unique Field FollowerInfo fail while insert table ")
	}
	sucFields["FollowerInfo"] = "FollowerInfo"

	return sucFields, nil
}

func (s *SoExtFollowerWrap) delUniKeyFollowerInfo(sa *SoExtFollower) bool {
	if s.dba == nil {
		return false
	}
	pre := ExtFollowerFollowerInfoUniTable
	kList := []interface{}{pre}
	if sa != nil {
		if sa.FollowerInfo == nil {
			return false
		}

		sub := sa.FollowerInfo
		kList = append(kList, sub)
	} else {
		sub := s.GetFollowerInfo()
		if sub == nil {
			return true
		}

		kList = append(kList, sub)

	}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoExtFollowerWrap) insertUniKeyFollowerInfo(sa *SoExtFollower) bool {
	if s.dba == nil || sa == nil {
		return false
	}

	pre := ExtFollowerFollowerInfoUniTable
	sub := sa.FollowerInfo
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
	val := SoUniqueExtFollowerByFollowerInfo{}
	val.FollowerInfo = sa.FollowerInfo

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniExtFollowerFollowerInfoWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniExtFollowerFollowerInfoWrap(db iservices.IDatabaseRW) *UniExtFollowerFollowerInfoWrap {
	if db == nil {
		return nil
	}
	wrap := UniExtFollowerFollowerInfoWrap{Dba: db}
	return &wrap
}

func (s *UniExtFollowerFollowerInfoWrap) UniQueryFollowerInfo(start *prototype.FollowerRelation) *SoExtFollowerWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := ExtFollowerFollowerInfoUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueExtFollowerByFollowerInfo{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoExtFollowerWrap(s.Dba, res.FollowerInfo)

			return wrap
		}
	}
	return nil
}

////////////// SECTION Watchers ///////////////

type ExtFollowerWatcherFlag struct {
	HasFollowerCreatedOrderWatcher bool

	WholeWatcher bool
	AnyWatcher   bool
}

var (
	ExtFollowerRecordType       = reflect.TypeOf((*SoExtFollower)(nil)).Elem()
	ExtFollowerWatcherFlags     = make(map[uint32]ExtFollowerWatcherFlag)
	ExtFollowerWatcherFlagsLock sync.RWMutex
)

func ExtFollowerWatcherFlagOfDb(dbSvcId uint32) ExtFollowerWatcherFlag {
	ExtFollowerWatcherFlagsLock.RLock()
	defer ExtFollowerWatcherFlagsLock.RUnlock()
	return ExtFollowerWatcherFlags[dbSvcId]
}

func ExtFollowerRecordWatcherChanged(dbSvcId uint32) {
	var flag ExtFollowerWatcherFlag
	flag.WholeWatcher = HasTableRecordWatcher(dbSvcId, ExtFollowerRecordType, "")
	flag.AnyWatcher = flag.WholeWatcher

	flag.HasFollowerCreatedOrderWatcher = HasTableRecordWatcher(dbSvcId, ExtFollowerRecordType, "FollowerCreatedOrder")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasFollowerCreatedOrderWatcher

	ExtFollowerWatcherFlagsLock.Lock()
	ExtFollowerWatcherFlags[dbSvcId] = flag
	ExtFollowerWatcherFlagsLock.Unlock()
}

func init() {
	RegisterTableWatcherChangedCallback(ExtFollowerRecordType, ExtFollowerRecordWatcherChanged)
}
