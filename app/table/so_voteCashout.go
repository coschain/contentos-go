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
	VoteCashoutCashoutBlockUniTable uint32 = 3832433085

	VoteCashoutCashoutBlockRow uint32 = 2140678679
)

////////////// SECTION Wrap Define ///////////////
type SoVoteCashoutWrap struct {
	dba         iservices.IDatabaseRW
	mainKey     *uint64
	watcherFlag *VoteCashoutWatcherFlag
	mKeyFlag    int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf     []byte //the buffer after the main key is encoded with prefix
	mBuf        []byte //the value after the main key is encoded
	mdFuncMap   map[string]interface{}
}

func NewSoVoteCashoutWrap(dba iservices.IDatabaseRW, key *uint64) *SoVoteCashoutWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoVoteCashoutWrap{dba, key, nil, -1, nil, nil, nil}
	result.initWatcherFlag()
	return result
}

func (s *SoVoteCashoutWrap) CheckExist() bool {
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

func (s *SoVoteCashoutWrap) MustExist(errMsgs ...interface{}) *SoVoteCashoutWrap {
	if !s.CheckExist() {
		panic(bindErrorInfo(fmt.Sprintf("SoVoteCashoutWrap.MustExist: %v not found", s.mainKey), errMsgs...))
	}
	return s
}

func (s *SoVoteCashoutWrap) MustNotExist(errMsgs ...interface{}) *SoVoteCashoutWrap {
	if s.CheckExist() {
		panic(bindErrorInfo(fmt.Sprintf("SoVoteCashoutWrap.MustNotExist: %v already exists", s.mainKey), errMsgs...))
	}
	return s
}

func (s *SoVoteCashoutWrap) initWatcherFlag() {
	if s.watcherFlag == nil {
		s.watcherFlag = new(VoteCashoutWatcherFlag)
		*(s.watcherFlag) = VoteCashoutWatcherFlagOfDb(s.dba.ServiceId())
	}
}

func (s *SoVoteCashoutWrap) create(f func(tInfo *SoVoteCashout)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoVoteCashout{}
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

func (s *SoVoteCashoutWrap) Create(f func(tInfo *SoVoteCashout), errArgs ...interface{}) *SoVoteCashoutWrap {
	err := s.create(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Errorf("SoVoteCashoutWrap.Create failed: %s", err.Error()), errArgs...))
	}
	return s
}

func (s *SoVoteCashoutWrap) getMainKeyBuf() ([]byte, error) {
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

func (s *SoVoteCashoutWrap) modify(f func(tInfo *SoVoteCashout)) error {
	if !s.CheckExist() {
		return errors.New("the SoVoteCashout table does not exist. Please create a table first")
	}
	oriTable := s.getVoteCashout()
	if oriTable == nil {
		return errors.New("fail to get origin table SoVoteCashout")
	}

	curTable := s.getVoteCashout()
	if curTable == nil {
		return errors.New("fail to create current table SoVoteCashout")
	}
	f(curTable)

	//the main key is not support modify
	if !reflect.DeepEqual(curTable.CashoutBlock, oriTable.CashoutBlock) {
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
	err = s.updateVoteCashout(curTable)
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

func (s *SoVoteCashoutWrap) Modify(f func(tInfo *SoVoteCashout), errArgs ...interface{}) *SoVoteCashoutWrap {
	err := s.modify(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoVoteCashoutWrap.Modify failed: %s", err.Error()), errArgs...))
	}
	return s
}

func (s *SoVoteCashoutWrap) SetVoterIds(p []*prototype.VoterId, errArgs ...interface{}) *SoVoteCashoutWrap {
	err := s.modify(func(r *SoVoteCashout) {
		r.VoterIds = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoVoteCashoutWrap.SetVoterIds( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoVoteCashoutWrap) checkSortAndUniFieldValidity(curTable *SoVoteCashout, fields map[string]bool) error {
	if curTable != nil && fields != nil && len(fields) > 0 {

	}
	return nil
}

//Get all the modified fields in the table
func (s *SoVoteCashoutWrap) getModifiedFields(oriTable *SoVoteCashout, curTable *SoVoteCashout) (map[string]bool, bool, error) {
	if oriTable == nil {
		return nil, false, errors.New("table info is nil, can't get modified fields")
	}
	hasWatcher := false
	fields := make(map[string]bool)

	if !reflect.DeepEqual(oriTable.VoterIds, curTable.VoterIds) {
		fields["VoterIds"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasVoterIdsWatcher
	}

	hasWatcher = hasWatcher || s.watcherFlag.WholeWatcher
	return fields, hasWatcher, nil
}

func (s *SoVoteCashoutWrap) handleFieldMd(t FieldMdHandleType, so *SoVoteCashout, fields map[string]bool) error {
	if so == nil {
		return errors.New("fail to modify empty table")
	}

	//there is no field need to modify
	if fields == nil || len(fields) < 1 {
		return nil
	}

	errStr := ""

	if fields["VoterIds"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldVoterIds(so.VoterIds, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "VoterIds")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldVoterIds(so.VoterIds, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "VoterIds")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldVoterIds(so.VoterIds, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "VoterIds")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	return nil
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoVoteCashoutWrap) delAllSortKeys(br bool, val *SoVoteCashout) bool {
	if s.dba == nil {
		return false
	}
	res := true

	return res
}

func (s *SoVoteCashoutWrap) insertAllSortKeys(val *SoVoteCashout) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoVoteCashout fail ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoVoteCashoutWrap) removeVoteCashout() error {
	if s.dba == nil {
		return errors.New("database is nil")
	}

	s.initWatcherFlag()

	var oldVal *SoVoteCashout
	if s.watcherFlag.AnyWatcher {
		oldVal = s.getVoteCashout()
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

func (s *SoVoteCashoutWrap) RemoveVoteCashout(errMsgs ...interface{}) *SoVoteCashoutWrap {
	err := s.removeVoteCashout()
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoVoteCashoutWrap.RemoveVoteCashout failed: %s", err.Error()), errMsgs...))
	}
	return s
}

////////////// SECTION Members Get/Modify ///////////////

func (s *SoVoteCashoutWrap) GetCashoutBlock() uint64 {
	res := true
	msg := &SoVoteCashout{}
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
				return msg.CashoutBlock
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.CashoutBlock
}

func (s *SoVoteCashoutWrap) GetVoterIds() []*prototype.VoterId {
	res := true
	msg := &SoVoteCashout{}
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
				return msg.VoterIds
			}
		}
	}
	if !res {
		var tmpValue []*prototype.VoterId
		return tmpValue
	}
	return msg.VoterIds
}

func (s *SoVoteCashoutWrap) mdFieldVoterIds(p []*prototype.VoterId, isCheck bool, isDel bool, isInsert bool,
	so *SoVoteCashout) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkVoterIdsIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldVoterIds(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldVoterIds(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoVoteCashoutWrap) delFieldVoterIds(so *SoVoteCashout) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoVoteCashoutWrap) insertFieldVoterIds(so *SoVoteCashout) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoVoteCashoutWrap) checkVoterIdsIsMetMdCondition(p []*prototype.VoterId) bool {
	if s.dba == nil {
		return false
	}

	return true
}

/////////////// SECTION Private function ////////////////

func (s *SoVoteCashoutWrap) update(sa *SoVoteCashout) bool {
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

func (s *SoVoteCashoutWrap) getVoteCashout() *SoVoteCashout {
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

	res := &SoVoteCashout{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoVoteCashoutWrap) updateVoteCashout(so *SoVoteCashout) error {
	if s.dba == nil {
		return errors.New("update fail:the db is nil")
	}

	if so == nil {
		return errors.New("update fail: the SoVoteCashout is nil")
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

func (s *SoVoteCashoutWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := VoteCashoutCashoutBlockRow
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

func (s *SoVoteCashoutWrap) delAllUniKeys(br bool, val *SoVoteCashout) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delUniKeyCashoutBlock(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoVoteCashoutWrap) delUniKeysWithNames(names map[string]string, val *SoVoteCashout) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if len(names["CashoutBlock"]) > 0 {
		if !s.delUniKeyCashoutBlock(val) {
			res = false
		}
	}

	return res
}

func (s *SoVoteCashoutWrap) insertAllUniKeys(val *SoVoteCashout) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoVoteCashout fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyCashoutBlock(val) {
		return sucFields, errors.New("insert unique Field CashoutBlock fail while insert table ")
	}
	sucFields["CashoutBlock"] = "CashoutBlock"

	return sucFields, nil
}

func (s *SoVoteCashoutWrap) delUniKeyCashoutBlock(sa *SoVoteCashout) bool {
	if s.dba == nil {
		return false
	}
	pre := VoteCashoutCashoutBlockUniTable
	kList := []interface{}{pre}
	if sa != nil {

		sub := sa.CashoutBlock
		kList = append(kList, sub)
	} else {
		sub := s.GetCashoutBlock()

		kList = append(kList, sub)

	}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoVoteCashoutWrap) insertUniKeyCashoutBlock(sa *SoVoteCashout) bool {
	if s.dba == nil || sa == nil {
		return false
	}

	pre := VoteCashoutCashoutBlockUniTable
	sub := sa.CashoutBlock
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
	val := SoUniqueVoteCashoutByCashoutBlock{}
	val.CashoutBlock = sa.CashoutBlock

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniVoteCashoutCashoutBlockWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniVoteCashoutCashoutBlockWrap(db iservices.IDatabaseRW) *UniVoteCashoutCashoutBlockWrap {
	if db == nil {
		return nil
	}
	wrap := UniVoteCashoutCashoutBlockWrap{Dba: db}
	return &wrap
}

func (s *UniVoteCashoutCashoutBlockWrap) UniQueryCashoutBlock(start *uint64) *SoVoteCashoutWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := VoteCashoutCashoutBlockUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueVoteCashoutByCashoutBlock{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoVoteCashoutWrap(s.Dba, &res.CashoutBlock)
			return wrap
		}
	}
	return nil
}

////////////// SECTION Watchers ///////////////

type VoteCashoutWatcherFlag struct {
	HasVoterIdsWatcher bool

	WholeWatcher bool
	AnyWatcher   bool
}

var (
	VoteCashoutRecordType       = reflect.TypeOf((*SoVoteCashout)(nil)).Elem()
	VoteCashoutWatcherFlags     = make(map[uint32]VoteCashoutWatcherFlag)
	VoteCashoutWatcherFlagsLock sync.RWMutex
)

func VoteCashoutWatcherFlagOfDb(dbSvcId uint32) VoteCashoutWatcherFlag {
	VoteCashoutWatcherFlagsLock.RLock()
	defer VoteCashoutWatcherFlagsLock.RUnlock()
	return VoteCashoutWatcherFlags[dbSvcId]
}

func VoteCashoutRecordWatcherChanged(dbSvcId uint32) {
	var flag VoteCashoutWatcherFlag
	flag.WholeWatcher = HasTableRecordWatcher(dbSvcId, VoteCashoutRecordType, "")
	flag.AnyWatcher = flag.WholeWatcher

	flag.HasVoterIdsWatcher = HasTableRecordWatcher(dbSvcId, VoteCashoutRecordType, "VoterIds")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasVoterIdsWatcher

	VoteCashoutWatcherFlagsLock.Lock()
	VoteCashoutWatcherFlags[dbSvcId] = flag
	VoteCashoutWatcherFlagsLock.Unlock()
}

func init() {
	RegisterTableWatcherChangedCallback(VoteCashoutRecordType, VoteCashoutRecordWatcherChanged)
}
