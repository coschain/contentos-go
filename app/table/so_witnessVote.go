package table

import (
	"errors"
	fmt "fmt"
	"reflect"

	"github.com/coschain/contentos-go/common/encoding/kope"
	"github.com/coschain/contentos-go/iservices"
	prototype "github.com/coschain/contentos-go/prototype"
	proto "github.com/golang/protobuf/proto"
)

////////////// SECTION Prefix Mark ///////////////
var (
	WitnessVoteVoterIdTable    uint32 = 1793351735
	WitnessVoteVoterIdUniTable uint32 = 3978964285

	WitnessVoteVoterIdRow uint32 = 211978786
)

////////////// SECTION Wrap Define ///////////////
type SoWitnessVoteWrap struct {
	dba       iservices.IDatabaseRW
	mainKey   *prototype.BpVoterId
	mKeyFlag  int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf   []byte //the buffer after the main key is encoded with prefix
	mBuf      []byte //the value after the main key is encoded
	mdFuncMap map[string]interface{}
}

func NewSoWitnessVoteWrap(dba iservices.IDatabaseRW, key *prototype.BpVoterId) *SoWitnessVoteWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoWitnessVoteWrap{dba, key, -1, nil, nil, nil}
	return result
}

func (s *SoWitnessVoteWrap) CheckExist() bool {
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

func (s *SoWitnessVoteWrap) Create(f func(tInfo *SoWitnessVote)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoWitnessVote{}
	f(val)
	if val.VoterId == nil {
		val.VoterId = s.mainKey
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
	return nil
}

func (s *SoWitnessVoteWrap) getMainKeyBuf() ([]byte, error) {
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

func (s *SoWitnessVoteWrap) Md(f func(tInfo *SoWitnessVote)) error {
	if !s.CheckExist() {
		return errors.New("the SoWitnessVote table does not exist. Please create a table first")
	}
	oriTable := s.getWitnessVote()
	if oriTable == nil {
		return errors.New("fail to get origin table SoWitnessVote")
	}
	curTable := *oriTable
	f(&curTable)

	//the main key is not support modify
	if !reflect.DeepEqual(curTable.VoterId, oriTable.VoterId) {
		return errors.New("primary key does not support modification")
	}

	fieldSli, err := s.getModifiedFields(oriTable, &curTable)
	if err != nil {
		return err
	}

	if fieldSli == nil || len(fieldSli) < 1 {
		return nil
	}

	//check whether modify sort and unique field to nil
	err = s.checkSortAndUniFieldValidity(&curTable, fieldSli)
	if err != nil {
		return err
	}

	//check unique
	err = s.handleFieldMd(FieldMdHandleTypeCheck, &curTable, fieldSli)
	if err != nil {
		return err
	}

	//delete sort and unique key
	err = s.handleFieldMd(FieldMdHandleTypeDel, oriTable, fieldSli)
	if err != nil {
		return err
	}

	//update table
	err = s.updateWitnessVote(&curTable)
	if err != nil {
		return err
	}

	//insert sort and unique key
	err = s.handleFieldMd(FieldMdHandleTypeInsert, &curTable, fieldSli)
	if err != nil {
		return err
	}

	return nil

}

func (s *SoWitnessVoteWrap) checkSortAndUniFieldValidity(curTable *SoWitnessVote, fieldSli []string) error {
	if curTable != nil && fieldSli != nil && len(fieldSli) > 0 {
		for _, fName := range fieldSli {
			if len(fName) > 0 {

			}
		}
	}
	return nil
}

//Get all the modified fields in the table
func (s *SoWitnessVoteWrap) getModifiedFields(oriTable *SoWitnessVote, curTable *SoWitnessVote) ([]string, error) {
	if oriTable == nil {
		return nil, errors.New("table info is nil, can't get modified fields")
	}
	var list []string

	if !reflect.DeepEqual(oriTable.VoteTime, curTable.VoteTime) {
		list = append(list, "VoteTime")
	}

	if !reflect.DeepEqual(oriTable.WitnessId, curTable.WitnessId) {
		list = append(list, "WitnessId")
	}

	return list, nil
}

func (s *SoWitnessVoteWrap) handleFieldMd(t FieldMdHandleType, so *SoWitnessVote, fSli []string) error {
	if so == nil {
		return errors.New("fail to modify empty table")
	}

	//there is no field need to modify
	if fSli == nil || len(fSli) < 1 {
		return nil
	}

	errStr := ""
	for _, fName := range fSli {

		if fName == "VoteTime" {
			res := true
			if t == FieldMdHandleTypeCheck {
				res = s.mdFieldVoteTime(so.VoteTime, true, false, false, so)
				errStr = fmt.Sprintf("fail to modify exist value of %v", fName)
			} else if t == FieldMdHandleTypeDel {
				res = s.mdFieldVoteTime(so.VoteTime, false, true, false, so)
				errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", fName)
			} else if t == FieldMdHandleTypeInsert {
				res = s.mdFieldVoteTime(so.VoteTime, false, false, true, so)
				errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", fName)
			}
			if !res {
				return errors.New(errStr)
			}
		}

		if fName == "WitnessId" {
			res := true
			if t == FieldMdHandleTypeCheck {
				res = s.mdFieldWitnessId(so.WitnessId, true, false, false, so)
				errStr = fmt.Sprintf("fail to modify exist value of %v", fName)
			} else if t == FieldMdHandleTypeDel {
				res = s.mdFieldWitnessId(so.WitnessId, false, true, false, so)
				errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", fName)
			} else if t == FieldMdHandleTypeInsert {
				res = s.mdFieldWitnessId(so.WitnessId, false, false, true, so)
				errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", fName)
			}
			if !res {
				return errors.New(errStr)
			}
		}

	}

	return nil
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoWitnessVoteWrap) delSortKeyVoterId(sa *SoWitnessVote) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListWitnessVoteByVoterId{}
	if sa == nil {
		val.VoterId = s.GetVoterId()
	} else {
		val.VoterId = sa.VoterId
	}
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoWitnessVoteWrap) insertSortKeyVoterId(sa *SoWitnessVote) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListWitnessVoteByVoterId{}
	val.VoterId = sa.VoterId
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

func (s *SoWitnessVoteWrap) delAllSortKeys(br bool, val *SoWitnessVote) bool {
	if s.dba == nil {
		return false
	}
	res := true

	return res
}

func (s *SoWitnessVoteWrap) insertAllSortKeys(val *SoWitnessVote) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoWitnessVote fail ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoWitnessVoteWrap) RemoveWitnessVote() bool {
	if s.dba == nil {
		return false
	}
	//delete sort list key
	if res := s.delAllSortKeys(true, nil); !res {
		return false
	}

	//delete unique list
	if res := s.delAllUniKeys(true, nil); !res {
		return false
	}

	//delete table
	key, err := s.encodeMainKey()
	if err != nil {
		return false
	}
	err = s.dba.Delete(key)
	if err == nil {
		s.mKeyBuf = nil
		s.mKeyFlag = -1
		return true
	} else {
		return false
	}
}

////////////// SECTION Members Get/Modify ///////////////

func (s *SoWitnessVoteWrap) GetVoteTime() *prototype.TimePointSec {
	res := true
	msg := &SoWitnessVote{}
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

func (s *SoWitnessVoteWrap) mdFieldVoteTime(p *prototype.TimePointSec, isCheck bool, isDel bool, isInsert bool,
	so *SoWitnessVote) bool {
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

func (s *SoWitnessVoteWrap) delFieldVoteTime(so *SoWitnessVote) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessVoteWrap) insertFieldVoteTime(so *SoWitnessVote) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessVoteWrap) checkVoteTimeIsMetMdCondition(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessVoteWrap) GetVoterId() *prototype.BpVoterId {
	res := true
	msg := &SoWitnessVote{}
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
				return msg.VoterId
			}
		}
	}
	if !res {
		return nil

	}
	return msg.VoterId
}

func (s *SoWitnessVoteWrap) GetWitnessId() *prototype.BpWitnessId {
	res := true
	msg := &SoWitnessVote{}
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
				return msg.WitnessId
			}
		}
	}
	if !res {
		return nil

	}
	return msg.WitnessId
}

func (s *SoWitnessVoteWrap) mdFieldWitnessId(p *prototype.BpWitnessId, isCheck bool, isDel bool, isInsert bool,
	so *SoWitnessVote) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkWitnessIdIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldWitnessId(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldWitnessId(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoWitnessVoteWrap) delFieldWitnessId(so *SoWitnessVote) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessVoteWrap) insertFieldWitnessId(so *SoWitnessVote) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessVoteWrap) checkWitnessIdIsMetMdCondition(p *prototype.BpWitnessId) bool {
	if s.dba == nil {
		return false
	}

	return true
}

////////////// SECTION List Keys ///////////////
type SWitnessVoteVoterIdWrap struct {
	Dba iservices.IDatabaseRW
}

func NewWitnessVoteVoterIdWrap(db iservices.IDatabaseRW) *SWitnessVoteVoterIdWrap {
	if db == nil {
		return nil
	}
	wrap := SWitnessVoteVoterIdWrap{Dba: db}
	return &wrap
}

func (s *SWitnessVoteVoterIdWrap) GetMainVal(val []byte) *prototype.BpVoterId {
	res := &SoListWitnessVoteByVoterId{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.VoterId

}

func (s *SWitnessVoteVoterIdWrap) GetSubVal(val []byte) *prototype.BpVoterId {
	res := &SoListWitnessVoteByVoterId{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.VoterId

}

func (m *SoListWitnessVoteByVoterId) OpeEncode() ([]byte, error) {
	pre := WitnessVoteVoterIdTable
	sub := m.VoterId
	if sub == nil {
		return nil, errors.New("the pro VoterId is nil")
	}
	sub1 := m.VoterId
	if sub1 == nil {
		return nil, errors.New("the mainkey VoterId is nil")
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
func (s *SWitnessVoteVoterIdWrap) ForEachByOrder(start *prototype.BpVoterId, end *prototype.BpVoterId, lastMainKey *prototype.BpVoterId,
	lastSubVal *prototype.BpVoterId, f func(mVal *prototype.BpVoterId, sVal *prototype.BpVoterId, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := WitnessVoteVoterIdTable
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

func (s *SoWitnessVoteWrap) update(sa *SoWitnessVote) bool {
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

func (s *SoWitnessVoteWrap) getWitnessVote() *SoWitnessVote {
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

	res := &SoWitnessVote{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoWitnessVoteWrap) updateWitnessVote(so *SoWitnessVote) error {
	if s.dba == nil {
		return errors.New("update fail:the db is nil")
	}

	if so == nil {
		return errors.New("update fail: the SoWitnessVote is nil")
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

func (s *SoWitnessVoteWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := WitnessVoteVoterIdRow
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

func (s *SoWitnessVoteWrap) delAllUniKeys(br bool, val *SoWitnessVote) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delUniKeyVoterId(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoWitnessVoteWrap) delUniKeysWithNames(names map[string]string, val *SoWitnessVote) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if len(names["VoterId"]) > 0 {
		if !s.delUniKeyVoterId(val) {
			res = false
		}
	}

	return res
}

func (s *SoWitnessVoteWrap) insertAllUniKeys(val *SoWitnessVote) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoWitnessVote fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyVoterId(val) {
		return sucFields, errors.New("insert unique Field VoterId fail while insert table ")
	}
	sucFields["VoterId"] = "VoterId"

	return sucFields, nil
}

func (s *SoWitnessVoteWrap) delUniKeyVoterId(sa *SoWitnessVote) bool {
	if s.dba == nil {
		return false
	}
	pre := WitnessVoteVoterIdUniTable
	kList := []interface{}{pre}
	if sa != nil {
		if sa.VoterId == nil {
			return false
		}

		sub := sa.VoterId
		kList = append(kList, sub)
	} else {
		sub := s.GetVoterId()
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

func (s *SoWitnessVoteWrap) insertUniKeyVoterId(sa *SoWitnessVote) bool {
	if s.dba == nil || sa == nil {
		return false
	}

	pre := WitnessVoteVoterIdUniTable
	sub := sa.VoterId
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
	val := SoUniqueWitnessVoteByVoterId{}
	val.VoterId = sa.VoterId

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniWitnessVoteVoterIdWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniWitnessVoteVoterIdWrap(db iservices.IDatabaseRW) *UniWitnessVoteVoterIdWrap {
	if db == nil {
		return nil
	}
	wrap := UniWitnessVoteVoterIdWrap{Dba: db}
	return &wrap
}

func (s *UniWitnessVoteVoterIdWrap) UniQueryVoterId(start *prototype.BpVoterId) *SoWitnessVoteWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := WitnessVoteVoterIdUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueWitnessVoteByVoterId{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoWitnessVoteWrap(s.Dba, res.VoterId)

			return wrap
		}
	}
	return nil
}
