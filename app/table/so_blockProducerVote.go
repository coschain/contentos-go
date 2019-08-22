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
	BlockProducerVoteBlockProducerIdTable    uint32 = 2788429790
	BlockProducerVoteBlockProducerIdUniTable uint32 = 800023394
	BlockProducerVoteVoterNameUniTable       uint32 = 3078178695

	BlockProducerVoteBlockProducerIdRow uint32 = 3268669708
)

////////////// SECTION Wrap Define ///////////////
type SoBlockProducerVoteWrap struct {
	dba         iservices.IDatabaseRW
	mainKey     *prototype.BpBlockProducerId
	watcherFlag *BlockProducerVoteWatcherFlag
	mKeyFlag    int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf     []byte //the buffer after the main key is encoded with prefix
	mBuf        []byte //the value after the main key is encoded
	mdFuncMap   map[string]interface{}
}

func NewSoBlockProducerVoteWrap(dba iservices.IDatabaseRW, key *prototype.BpBlockProducerId) *SoBlockProducerVoteWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoBlockProducerVoteWrap{dba, key, nil, -1, nil, nil, nil}
	result.initWatcherFlag()
	return result
}

func (s *SoBlockProducerVoteWrap) CheckExist() bool {
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

func (s *SoBlockProducerVoteWrap) MustExist(errMsgs ...interface{}) *SoBlockProducerVoteWrap {
	if !s.CheckExist() {
		panic(bindErrorInfo(fmt.Sprintf("SoBlockProducerVoteWrap.MustExist: %v not found", s.mainKey), errMsgs...))
	}
	return s
}

func (s *SoBlockProducerVoteWrap) MustNotExist(errMsgs ...interface{}) *SoBlockProducerVoteWrap {
	if s.CheckExist() {
		panic(bindErrorInfo(fmt.Sprintf("SoBlockProducerVoteWrap.MustNotExist: %v already exists", s.mainKey), errMsgs...))
	}
	return s
}

func (s *SoBlockProducerVoteWrap) initWatcherFlag() {
	if s.watcherFlag == nil {
		s.watcherFlag = new(BlockProducerVoteWatcherFlag)
		*(s.watcherFlag) = BlockProducerVoteWatcherFlagOfDb(s.dba.ServiceId())
	}
}

func (s *SoBlockProducerVoteWrap) create(f func(tInfo *SoBlockProducerVote)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoBlockProducerVote{}
	f(val)
	if val.BlockProducerId == nil {
		val.BlockProducerId = s.mainKey
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
		ReportTableRecordInsert(s.dba.ServiceId(), s.mainKey, val)
	}

	return nil
}

func (s *SoBlockProducerVoteWrap) Create(f func(tInfo *SoBlockProducerVote), errArgs ...interface{}) *SoBlockProducerVoteWrap {
	err := s.create(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Errorf("SoBlockProducerVoteWrap.Create failed: %s", err.Error()), errArgs...))
	}
	return s
}

func (s *SoBlockProducerVoteWrap) getMainKeyBuf() ([]byte, error) {
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

func (s *SoBlockProducerVoteWrap) modify(f func(tInfo *SoBlockProducerVote)) error {
	if !s.CheckExist() {
		return errors.New("the SoBlockProducerVote table does not exist. Please create a table first")
	}
	oriTable := s.getBlockProducerVote()
	if oriTable == nil {
		return errors.New("fail to get origin table SoBlockProducerVote")
	}

	curTable := s.getBlockProducerVote()
	if curTable == nil {
		return errors.New("fail to create current table SoBlockProducerVote")
	}
	f(curTable)

	//the main key is not support modify
	if !reflect.DeepEqual(curTable.BlockProducerId, oriTable.BlockProducerId) {
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
	err = s.updateBlockProducerVote(curTable)
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
		ReportTableRecordUpdate(s.dba.ServiceId(), s.mainKey, oriTable, curTable)
	}

	return nil

}

func (s *SoBlockProducerVoteWrap) Modify(f func(tInfo *SoBlockProducerVote), errArgs ...interface{}) *SoBlockProducerVoteWrap {
	err := s.modify(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoBlockProducerVoteWrap.Modify failed: %s", err.Error()), errArgs...))
	}
	return s
}

func (s *SoBlockProducerVoteWrap) SetVoteTime(p *prototype.TimePointSec, errArgs ...interface{}) *SoBlockProducerVoteWrap {
	err := s.modify(func(r *SoBlockProducerVote) {
		r.VoteTime = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoBlockProducerVoteWrap.SetVoteTime( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoBlockProducerVoteWrap) SetVoterName(p *prototype.AccountName, errArgs ...interface{}) *SoBlockProducerVoteWrap {
	err := s.modify(func(r *SoBlockProducerVote) {
		r.VoterName = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoBlockProducerVoteWrap.SetVoterName( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoBlockProducerVoteWrap) checkSortAndUniFieldValidity(curTable *SoBlockProducerVote, fields map[string]bool) error {
	if curTable != nil && fields != nil && len(fields) > 0 {

		if fields["VoterName"] && curTable.VoterName == nil {
			return errors.New("unique field VoterName can't be modified to nil")
		}

	}
	return nil
}

//Get all the modified fields in the table
func (s *SoBlockProducerVoteWrap) getModifiedFields(oriTable *SoBlockProducerVote, curTable *SoBlockProducerVote) (map[string]bool, bool, error) {
	if oriTable == nil {
		return nil, false, errors.New("table info is nil, can't get modified fields")
	}
	hasWatcher := false
	fields := make(map[string]bool)

	if !reflect.DeepEqual(oriTable.VoteTime, curTable.VoteTime) {
		fields["VoteTime"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasVoteTimeWatcher
	}

	if !reflect.DeepEqual(oriTable.VoterName, curTable.VoterName) {
		fields["VoterName"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasVoterNameWatcher
	}

	hasWatcher = hasWatcher || s.watcherFlag.WholeWatcher
	return fields, hasWatcher, nil
}

func (s *SoBlockProducerVoteWrap) handleFieldMd(t FieldMdHandleType, so *SoBlockProducerVote, fields map[string]bool) error {
	if so == nil {
		return errors.New("fail to modify empty table")
	}

	//there is no field need to modify
	if fields == nil || len(fields) < 1 {
		return nil
	}

	errStr := ""

	if fields["VoteTime"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldVoteTime(so.VoteTime, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "VoteTime")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldVoteTime(so.VoteTime, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "VoteTime")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldVoteTime(so.VoteTime, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "VoteTime")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["VoterName"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldVoterName(so.VoterName, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "VoterName")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldVoterName(so.VoterName, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "VoterName")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldVoterName(so.VoterName, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "VoterName")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	return nil
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoBlockProducerVoteWrap) delSortKeyBlockProducerId(sa *SoBlockProducerVote) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListBlockProducerVoteByBlockProducerId{}
	if sa == nil {
		val.BlockProducerId = s.GetBlockProducerId()
	} else {
		val.BlockProducerId = sa.BlockProducerId
	}
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoBlockProducerVoteWrap) insertSortKeyBlockProducerId(sa *SoBlockProducerVote) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListBlockProducerVoteByBlockProducerId{}
	val.BlockProducerId = sa.BlockProducerId
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

func (s *SoBlockProducerVoteWrap) delAllSortKeys(br bool, val *SoBlockProducerVote) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delSortKeyBlockProducerId(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoBlockProducerVoteWrap) insertAllSortKeys(val *SoBlockProducerVote) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoBlockProducerVote fail ")
	}
	if !s.insertSortKeyBlockProducerId(val) {
		return errors.New("insert sort Field BlockProducerId fail while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoBlockProducerVoteWrap) removeBlockProducerVote() error {
	if s.dba == nil {
		return errors.New("database is nil")
	}

	s.initWatcherFlag()

	var oldVal *SoBlockProducerVote
	if s.watcherFlag.AnyWatcher {
		oldVal = s.getBlockProducerVote()
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
			ReportTableRecordDelete(s.dba.ServiceId(), s.mainKey, oldVal)
		}
		return nil
	} else {
		return fmt.Errorf("database.Delete failed: %s", err.Error())
	}
}

func (s *SoBlockProducerVoteWrap) RemoveBlockProducerVote(errMsgs ...interface{}) *SoBlockProducerVoteWrap {
	err := s.removeBlockProducerVote()
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoBlockProducerVoteWrap.RemoveBlockProducerVote failed: %s", err.Error()), errMsgs...))
	}
	return s
}

////////////// SECTION Members Get/Modify ///////////////

func (s *SoBlockProducerVoteWrap) GetBlockProducerId() *prototype.BpBlockProducerId {
	res := true
	msg := &SoBlockProducerVote{}
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
				return msg.BlockProducerId
			}
		}
	}
	if !res {
		return nil

	}
	return msg.BlockProducerId
}

func (s *SoBlockProducerVoteWrap) GetVoteTime() *prototype.TimePointSec {
	res := true
	msg := &SoBlockProducerVote{}
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
				return msg.VoteTime
			}
		}
	}
	if !res {
		return nil

	}
	return msg.VoteTime
}

func (s *SoBlockProducerVoteWrap) mdFieldVoteTime(p *prototype.TimePointSec, isCheck bool, isDel bool, isInsert bool,
	so *SoBlockProducerVote) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkVoteTimeIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldVoteTime(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldVoteTime(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoBlockProducerVoteWrap) delFieldVoteTime(so *SoBlockProducerVote) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerVoteWrap) insertFieldVoteTime(so *SoBlockProducerVote) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerVoteWrap) checkVoteTimeIsMetMdCondition(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerVoteWrap) GetVoterName() *prototype.AccountName {
	res := true
	msg := &SoBlockProducerVote{}
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
				return msg.VoterName
			}
		}
	}
	if !res {
		return nil

	}
	return msg.VoterName
}

func (s *SoBlockProducerVoteWrap) mdFieldVoterName(p *prototype.AccountName, isCheck bool, isDel bool, isInsert bool,
	so *SoBlockProducerVote) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkVoterNameIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldVoterName(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldVoterName(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoBlockProducerVoteWrap) delFieldVoterName(so *SoBlockProducerVote) bool {
	if s.dba == nil {
		return false
	}

	if !s.delUniKeyVoterName(so) {
		return false
	}

	return true
}

func (s *SoBlockProducerVoteWrap) insertFieldVoterName(so *SoBlockProducerVote) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertUniKeyVoterName(so) {
		return false
	}
	return true
}

func (s *SoBlockProducerVoteWrap) checkVoterNameIsMetMdCondition(p *prototype.AccountName) bool {
	if s.dba == nil {
		return false
	}

	//judge the unique value if is exist
	uniWrap := UniBlockProducerVoteVoterNameWrap{}
	uniWrap.Dba = s.dba
	res := uniWrap.UniQueryVoterName(p)

	if res != nil {
		//the unique value to be modified is already exist
		return false
	}

	return true
}

////////////// SECTION List Keys ///////////////
type SBlockProducerVoteBlockProducerIdWrap struct {
	Dba iservices.IDatabaseRW
}

func NewBlockProducerVoteBlockProducerIdWrap(db iservices.IDatabaseRW) *SBlockProducerVoteBlockProducerIdWrap {
	if db == nil {
		return nil
	}
	wrap := SBlockProducerVoteBlockProducerIdWrap{Dba: db}
	return &wrap
}

func (s *SBlockProducerVoteBlockProducerIdWrap) GetMainVal(val []byte) *prototype.BpBlockProducerId {
	res := &SoListBlockProducerVoteByBlockProducerId{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.BlockProducerId

}

func (s *SBlockProducerVoteBlockProducerIdWrap) GetSubVal(val []byte) *prototype.BpBlockProducerId {
	res := &SoListBlockProducerVoteByBlockProducerId{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.BlockProducerId

}

func (m *SoListBlockProducerVoteByBlockProducerId) OpeEncode() ([]byte, error) {
	pre := BlockProducerVoteBlockProducerIdTable
	sub := m.BlockProducerId
	if sub == nil {
		return nil, errors.New("the pro BlockProducerId is nil")
	}
	sub1 := m.BlockProducerId
	if sub1 == nil {
		return nil, errors.New("the mainkey BlockProducerId is nil")
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
func (s *SBlockProducerVoteBlockProducerIdWrap) ForEachByOrder(start *prototype.BpBlockProducerId, end *prototype.BpBlockProducerId, lastMainKey *prototype.BpBlockProducerId,
	lastSubVal *prototype.BpBlockProducerId, f func(mVal *prototype.BpBlockProducerId, sVal *prototype.BpBlockProducerId, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := BlockProducerVoteBlockProducerIdTable
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

func (s *SoBlockProducerVoteWrap) update(sa *SoBlockProducerVote) bool {
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

func (s *SoBlockProducerVoteWrap) getBlockProducerVote() *SoBlockProducerVote {
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

	res := &SoBlockProducerVote{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoBlockProducerVoteWrap) updateBlockProducerVote(so *SoBlockProducerVote) error {
	if s.dba == nil {
		return errors.New("update fail:the db is nil")
	}

	if so == nil {
		return errors.New("update fail: the SoBlockProducerVote is nil")
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

func (s *SoBlockProducerVoteWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := BlockProducerVoteBlockProducerIdRow
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

func (s *SoBlockProducerVoteWrap) delAllUniKeys(br bool, val *SoBlockProducerVote) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delUniKeyBlockProducerId(val) {
		if br {
			return false
		} else {
			res = false
		}
	}
	if !s.delUniKeyVoterName(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoBlockProducerVoteWrap) delUniKeysWithNames(names map[string]string, val *SoBlockProducerVote) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if len(names["BlockProducerId"]) > 0 {
		if !s.delUniKeyBlockProducerId(val) {
			res = false
		}
	}
	if len(names["VoterName"]) > 0 {
		if !s.delUniKeyVoterName(val) {
			res = false
		}
	}

	return res
}

func (s *SoBlockProducerVoteWrap) insertAllUniKeys(val *SoBlockProducerVote) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoBlockProducerVote fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyBlockProducerId(val) {
		return sucFields, errors.New("insert unique Field BlockProducerId fail while insert table ")
	}
	sucFields["BlockProducerId"] = "BlockProducerId"
	if !s.insertUniKeyVoterName(val) {
		return sucFields, errors.New("insert unique Field VoterName fail while insert table ")
	}
	sucFields["VoterName"] = "VoterName"

	return sucFields, nil
}

func (s *SoBlockProducerVoteWrap) delUniKeyBlockProducerId(sa *SoBlockProducerVote) bool {
	if s.dba == nil {
		return false
	}
	pre := BlockProducerVoteBlockProducerIdUniTable
	kList := []interface{}{pre}
	if sa != nil {
		if sa.BlockProducerId == nil {
			return false
		}

		sub := sa.BlockProducerId
		kList = append(kList, sub)
	} else {
		sub := s.GetBlockProducerId()
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

func (s *SoBlockProducerVoteWrap) insertUniKeyBlockProducerId(sa *SoBlockProducerVote) bool {
	if s.dba == nil || sa == nil {
		return false
	}

	pre := BlockProducerVoteBlockProducerIdUniTable
	sub := sa.BlockProducerId
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
	val := SoUniqueBlockProducerVoteByBlockProducerId{}
	val.BlockProducerId = sa.BlockProducerId

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniBlockProducerVoteBlockProducerIdWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniBlockProducerVoteBlockProducerIdWrap(db iservices.IDatabaseRW) *UniBlockProducerVoteBlockProducerIdWrap {
	if db == nil {
		return nil
	}
	wrap := UniBlockProducerVoteBlockProducerIdWrap{Dba: db}
	return &wrap
}

func (s *UniBlockProducerVoteBlockProducerIdWrap) UniQueryBlockProducerId(start *prototype.BpBlockProducerId) *SoBlockProducerVoteWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := BlockProducerVoteBlockProducerIdUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueBlockProducerVoteByBlockProducerId{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoBlockProducerVoteWrap(s.Dba, res.BlockProducerId)

			return wrap
		}
	}
	return nil
}

func (s *SoBlockProducerVoteWrap) delUniKeyVoterName(sa *SoBlockProducerVote) bool {
	if s.dba == nil {
		return false
	}
	pre := BlockProducerVoteVoterNameUniTable
	kList := []interface{}{pre}
	if sa != nil {
		if sa.VoterName == nil {
			return false
		}

		sub := sa.VoterName
		kList = append(kList, sub)
	} else {
		sub := s.GetVoterName()
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

func (s *SoBlockProducerVoteWrap) insertUniKeyVoterName(sa *SoBlockProducerVote) bool {
	if s.dba == nil || sa == nil {
		return false
	}

	pre := BlockProducerVoteVoterNameUniTable
	sub := sa.VoterName
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
	val := SoUniqueBlockProducerVoteByVoterName{}
	val.BlockProducerId = sa.BlockProducerId
	val.VoterName = sa.VoterName

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniBlockProducerVoteVoterNameWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniBlockProducerVoteVoterNameWrap(db iservices.IDatabaseRW) *UniBlockProducerVoteVoterNameWrap {
	if db == nil {
		return nil
	}
	wrap := UniBlockProducerVoteVoterNameWrap{Dba: db}
	return &wrap
}

func (s *UniBlockProducerVoteVoterNameWrap) UniQueryVoterName(start *prototype.AccountName) *SoBlockProducerVoteWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := BlockProducerVoteVoterNameUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueBlockProducerVoteByVoterName{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoBlockProducerVoteWrap(s.Dba, res.BlockProducerId)

			return wrap
		}
	}
	return nil
}

////////////// SECTION Watchers ///////////////

type BlockProducerVoteWatcherFlag struct {
	HasVoteTimeWatcher bool

	HasVoterNameWatcher bool

	WholeWatcher bool
	AnyWatcher   bool
}

var (
	BlockProducerVoteRecordType       = reflect.TypeOf((*SoBlockProducerVote)(nil)).Elem()
	BlockProducerVoteWatcherFlags     = make(map[uint32]BlockProducerVoteWatcherFlag)
	BlockProducerVoteWatcherFlagsLock sync.RWMutex
)

func BlockProducerVoteWatcherFlagOfDb(dbSvcId uint32) BlockProducerVoteWatcherFlag {
	BlockProducerVoteWatcherFlagsLock.RLock()
	defer BlockProducerVoteWatcherFlagsLock.RUnlock()
	return BlockProducerVoteWatcherFlags[dbSvcId]
}

func BlockProducerVoteRecordWatcherChanged(dbSvcId uint32) {
	var flag BlockProducerVoteWatcherFlag
	flag.WholeWatcher = HasTableRecordWatcher(dbSvcId, BlockProducerVoteRecordType, "")
	flag.AnyWatcher = flag.WholeWatcher

	flag.HasVoteTimeWatcher = HasTableRecordWatcher(dbSvcId, BlockProducerVoteRecordType, "VoteTime")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasVoteTimeWatcher

	flag.HasVoterNameWatcher = HasTableRecordWatcher(dbSvcId, BlockProducerVoteRecordType, "VoterName")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasVoterNameWatcher

	BlockProducerVoteWatcherFlagsLock.Lock()
	BlockProducerVoteWatcherFlags[dbSvcId] = flag
	BlockProducerVoteWatcherFlagsLock.Unlock()
}

func init() {
	RegisterTableWatcherChangedCallback(BlockProducerVoteRecordType, BlockProducerVoteRecordWatcherChanged)
}
