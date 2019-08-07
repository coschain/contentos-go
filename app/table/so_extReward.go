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
	ExtRewardBlockHeightTable uint32 = 82025141
	ExtRewardIdUniTable       uint32 = 1999553764

	ExtRewardIdRow uint32 = 3421017858
)

////////////// SECTION Wrap Define ///////////////
type SoExtRewardWrap struct {
	dba       iservices.IDatabaseRW
	mainKey   *prototype.RewardCashoutId
	mKeyFlag  int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf   []byte //the buffer after the main key is encoded with prefix
	mBuf      []byte //the value after the main key is encoded
	mdFuncMap map[string]interface{}
}

func NewSoExtRewardWrap(dba iservices.IDatabaseRW, key *prototype.RewardCashoutId) *SoExtRewardWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoExtRewardWrap{dba, key, -1, nil, nil, nil}
	return result
}

func (s *SoExtRewardWrap) CheckExist() bool {
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

func (s *SoExtRewardWrap) Create(f func(tInfo *SoExtReward)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoExtReward{}
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

func (s *SoExtRewardWrap) getMainKeyBuf() ([]byte, error) {
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

func (s *SoExtRewardWrap) Modify(f func(tInfo *SoExtReward)) error {
	if !s.CheckExist() {
		return errors.New("the SoExtReward table does not exist. Please create a table first")
	}
	oriTable := s.getExtReward()
	if oriTable == nil {
		return errors.New("fail to get origin table SoExtReward")
	}
	curTable := *oriTable
	f(&curTable)

	//the main key is not support modify
	if !reflect.DeepEqual(curTable.Id, oriTable.Id) {
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
	err = s.updateExtReward(&curTable)
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

func (s *SoExtRewardWrap) SetBlockHeight(p uint64) bool {
	err := s.Modify(func(r *SoExtReward) {
		r.BlockHeight = p
	})
	return err == nil
}

func (s *SoExtRewardWrap) SetReward(p *prototype.Vest) bool {
	err := s.Modify(func(r *SoExtReward) {
		r.Reward = p
	})
	return err == nil
}

func (s *SoExtRewardWrap) checkSortAndUniFieldValidity(curTable *SoExtReward, fieldSli []string) error {
	if curTable != nil && fieldSli != nil && len(fieldSli) > 0 {
		for _, fName := range fieldSli {
			if len(fName) > 0 {

			}
		}
	}
	return nil
}

//Get all the modified fields in the table
func (s *SoExtRewardWrap) getModifiedFields(oriTable *SoExtReward, curTable *SoExtReward) ([]string, error) {
	if oriTable == nil {
		return nil, errors.New("table info is nil, can't get modified fields")
	}
	var list []string

	if !reflect.DeepEqual(oriTable.BlockHeight, curTable.BlockHeight) {
		list = append(list, "BlockHeight")
	}

	if !reflect.DeepEqual(oriTable.Reward, curTable.Reward) {
		list = append(list, "Reward")
	}

	return list, nil
}

func (s *SoExtRewardWrap) handleFieldMd(t FieldMdHandleType, so *SoExtReward, fSli []string) error {
	if so == nil {
		return errors.New("fail to modify empty table")
	}

	//there is no field need to modify
	if fSli == nil || len(fSli) < 1 {
		return nil
	}

	errStr := ""
	for _, fName := range fSli {

		if fName == "BlockHeight" {
			res := true
			if t == FieldMdHandleTypeCheck {
				res = s.mdFieldBlockHeight(so.BlockHeight, true, false, false, so)
				errStr = fmt.Sprintf("fail to modify exist value of %v", fName)
			} else if t == FieldMdHandleTypeDel {
				res = s.mdFieldBlockHeight(so.BlockHeight, false, true, false, so)
				errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", fName)
			} else if t == FieldMdHandleTypeInsert {
				res = s.mdFieldBlockHeight(so.BlockHeight, false, false, true, so)
				errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", fName)
			}
			if !res {
				return errors.New(errStr)
			}
		}

		if fName == "Reward" {
			res := true
			if t == FieldMdHandleTypeCheck {
				res = s.mdFieldReward(so.Reward, true, false, false, so)
				errStr = fmt.Sprintf("fail to modify exist value of %v", fName)
			} else if t == FieldMdHandleTypeDel {
				res = s.mdFieldReward(so.Reward, false, true, false, so)
				errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", fName)
			} else if t == FieldMdHandleTypeInsert {
				res = s.mdFieldReward(so.Reward, false, false, true, so)
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

func (s *SoExtRewardWrap) delSortKeyBlockHeight(sa *SoExtReward) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListExtRewardByBlockHeight{}
	if sa == nil {
		val.BlockHeight = s.GetBlockHeight()
		val.Id = s.mainKey

	} else {
		val.BlockHeight = sa.BlockHeight
		val.Id = sa.Id
	}
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoExtRewardWrap) insertSortKeyBlockHeight(sa *SoExtReward) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListExtRewardByBlockHeight{}
	val.Id = sa.Id
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

func (s *SoExtRewardWrap) delAllSortKeys(br bool, val *SoExtReward) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delSortKeyBlockHeight(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoExtRewardWrap) insertAllSortKeys(val *SoExtReward) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoExtReward fail ")
	}
	if !s.insertSortKeyBlockHeight(val) {
		return errors.New("insert sort Field BlockHeight fail while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoExtRewardWrap) RemoveExtReward() bool {
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

func (s *SoExtRewardWrap) GetBlockHeight() uint64 {
	res := true
	msg := &SoExtReward{}
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

func (s *SoExtRewardWrap) mdFieldBlockHeight(p uint64, isCheck bool, isDel bool, isInsert bool,
	so *SoExtReward) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkBlockHeightIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldBlockHeight(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldBlockHeight(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoExtRewardWrap) delFieldBlockHeight(so *SoExtReward) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyBlockHeight(so) {
		return false
	}

	return true
}

func (s *SoExtRewardWrap) insertFieldBlockHeight(so *SoExtReward) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyBlockHeight(so) {
		return false
	}

	return true
}

func (s *SoExtRewardWrap) checkBlockHeightIsMetMdCondition(p uint64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoExtRewardWrap) GetId() *prototype.RewardCashoutId {
	res := true
	msg := &SoExtReward{}
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
		return nil

	}
	return msg.Id
}

func (s *SoExtRewardWrap) GetReward() *prototype.Vest {
	res := true
	msg := &SoExtReward{}
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
				return msg.Reward
			}
		}
	}
	if !res {
		return nil

	}
	return msg.Reward
}

func (s *SoExtRewardWrap) mdFieldReward(p *prototype.Vest, isCheck bool, isDel bool, isInsert bool,
	so *SoExtReward) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkRewardIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldReward(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldReward(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoExtRewardWrap) delFieldReward(so *SoExtReward) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoExtRewardWrap) insertFieldReward(so *SoExtReward) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoExtRewardWrap) checkRewardIsMetMdCondition(p *prototype.Vest) bool {
	if s.dba == nil {
		return false
	}

	return true
}

////////////// SECTION List Keys ///////////////
type SExtRewardBlockHeightWrap struct {
	Dba iservices.IDatabaseRW
}

func NewExtRewardBlockHeightWrap(db iservices.IDatabaseRW) *SExtRewardBlockHeightWrap {
	if db == nil {
		return nil
	}
	wrap := SExtRewardBlockHeightWrap{Dba: db}
	return &wrap
}

func (s *SExtRewardBlockHeightWrap) GetMainVal(val []byte) *prototype.RewardCashoutId {
	res := &SoListExtRewardByBlockHeight{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Id

}

func (s *SExtRewardBlockHeightWrap) GetSubVal(val []byte) *uint64 {
	res := &SoListExtRewardByBlockHeight{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return &res.BlockHeight

}

func (m *SoListExtRewardByBlockHeight) OpeEncode() ([]byte, error) {
	pre := ExtRewardBlockHeightTable
	sub := m.BlockHeight

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
//lastMainKey: the main key of the last one of last page
//lastSubVal: the value  of the last one of last page
//
func (s *SExtRewardBlockHeightWrap) ForEachByOrder(start *uint64, end *uint64, lastMainKey *prototype.RewardCashoutId,
	lastSubVal *uint64, f func(mVal *prototype.RewardCashoutId, sVal *uint64, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := ExtRewardBlockHeightTable
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
func (s *SExtRewardBlockHeightWrap) ForEachByRevOrder(start *uint64, end *uint64, lastMainKey *prototype.RewardCashoutId,
	lastSubVal *uint64, f func(mVal *prototype.RewardCashoutId, sVal *uint64, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := ExtRewardBlockHeightTable
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
	var idx uint32 = 0
	s.Dba.Iterate(eBuf, sBuf, true, func(key, value []byte) bool {
		idx++
		return f(s.GetMainVal(value), s.GetSubVal(value), idx)
	})
	return nil
}

/////////////// SECTION Private function ////////////////

func (s *SoExtRewardWrap) update(sa *SoExtReward) bool {
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

func (s *SoExtRewardWrap) getExtReward() *SoExtReward {
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

	res := &SoExtReward{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoExtRewardWrap) updateExtReward(so *SoExtReward) error {
	if s.dba == nil {
		return errors.New("update fail:the db is nil")
	}

	if so == nil {
		return errors.New("update fail: the SoExtReward is nil")
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

func (s *SoExtRewardWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := ExtRewardIdRow
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

func (s *SoExtRewardWrap) delAllUniKeys(br bool, val *SoExtReward) bool {
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

func (s *SoExtRewardWrap) delUniKeysWithNames(names map[string]string, val *SoExtReward) bool {
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

func (s *SoExtRewardWrap) insertAllUniKeys(val *SoExtReward) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoExtReward fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyId(val) {
		return sucFields, errors.New("insert unique Field Id fail while insert table ")
	}
	sucFields["Id"] = "Id"

	return sucFields, nil
}

func (s *SoExtRewardWrap) delUniKeyId(sa *SoExtReward) bool {
	if s.dba == nil {
		return false
	}
	pre := ExtRewardIdUniTable
	kList := []interface{}{pre}
	if sa != nil {
		if sa.Id == nil {
			return false
		}

		sub := sa.Id
		kList = append(kList, sub)
	} else {
		sub := s.GetId()
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

func (s *SoExtRewardWrap) insertUniKeyId(sa *SoExtReward) bool {
	if s.dba == nil || sa == nil {
		return false
	}

	pre := ExtRewardIdUniTable
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
	val := SoUniqueExtRewardById{}
	val.Id = sa.Id

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniExtRewardIdWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniExtRewardIdWrap(db iservices.IDatabaseRW) *UniExtRewardIdWrap {
	if db == nil {
		return nil
	}
	wrap := UniExtRewardIdWrap{Dba: db}
	return &wrap
}

func (s *UniExtRewardIdWrap) UniQueryId(start *prototype.RewardCashoutId) *SoExtRewardWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := ExtRewardIdUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueExtRewardById{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoExtRewardWrap(s.Dba, res.Id)

			return wrap
		}
	}
	return nil
}
