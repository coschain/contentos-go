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
	ExtRewardBlockHeightTable uint32 = 82025141
	ExtRewardIdUniTable       uint32 = 1999553764
	ExtRewardBlockHeightCell  uint32 = 1670527665
	ExtRewardIdCell           uint32 = 885470707
	ExtRewardRewardCell       uint32 = 1285045296
)

////////////// SECTION Wrap Define ///////////////
type SoExtRewardWrap struct {
	dba      iservices.IDatabaseRW
	mainKey  *prototype.RewardCashoutId
	mKeyFlag int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf  []byte //the buffer after the main key is encoded with prefix
	mBuf     []byte //the value after the main key is encoded
}

func NewSoExtRewardWrap(dba iservices.IDatabaseRW, key *prototype.RewardCashoutId) *SoExtRewardWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoExtRewardWrap{dba, key, -1, nil, nil}
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

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoExtRewardWrap) delSortKeyBlockHeight(sa *SoExtReward) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListExtRewardByBlockHeight{}
	if sa == nil {
		key, err := s.encodeMemKey("BlockHeight")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemExtRewardByBlockHeight{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		val.BlockHeight = ori.BlockHeight
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
	val := &SoExtReward{}
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
func (s *SoExtRewardWrap) getMemKeyPrefix(fName string) uint32 {
	if fName == "BlockHeight" {
		return ExtRewardBlockHeightCell
	}
	if fName == "Id" {
		return ExtRewardIdCell
	}
	if fName == "Reward" {
		return ExtRewardRewardCell
	}

	return 0
}

func (s *SoExtRewardWrap) encodeMemKey(fName string) ([]byte, error) {
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

func (s *SoExtRewardWrap) saveAllMemKeys(tInfo *SoExtReward, br bool) error {
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
	if err = s.saveMemKeyId(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Id", err)
		}
	}
	if err = s.saveMemKeyReward(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Reward", err)
		}
	}

	if len(errDes) > 0 {
		return errors.New(errDes)
	}
	return err
}

func (s *SoExtRewardWrap) delAllMemKeys(br bool, tInfo *SoExtReward) error {
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

func (s *SoExtRewardWrap) delMemKey(fName string) error {
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

func (s *SoExtRewardWrap) saveMemKeyBlockHeight(tInfo *SoExtReward) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemExtRewardByBlockHeight{}
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

func (s *SoExtRewardWrap) GetBlockHeight() uint64 {
	res := true
	msg := &SoMemExtRewardByBlockHeight{}
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

func (s *SoExtRewardWrap) MdBlockHeight(p uint64) bool {
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
	ori := &SoMemExtRewardByBlockHeight{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoExtReward{}
	sa.Id = s.mainKey

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

func (s *SoExtRewardWrap) saveMemKeyId(tInfo *SoExtReward) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemExtRewardById{}
	val.Id = tInfo.Id
	key, err := s.encodeMemKey("Id")
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

func (s *SoExtRewardWrap) GetId() *prototype.RewardCashoutId {
	res := true
	msg := &SoMemExtRewardById{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Id")
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

func (s *SoExtRewardWrap) saveMemKeyReward(tInfo *SoExtReward) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemExtRewardByReward{}
	val.Reward = tInfo.Reward
	key, err := s.encodeMemKey("Reward")
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

func (s *SoExtRewardWrap) GetReward() *prototype.Vest {
	res := true
	msg := &SoMemExtRewardByReward{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Reward")
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

func (s *SoExtRewardWrap) MdReward(p *prototype.Vest) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Reward")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemExtRewardByReward{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoExtReward{}
	sa.Id = s.mainKey

	sa.Reward = ori.Reward

	ori.Reward = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Reward = p

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

func (s *SExtRewardBlockHeightWrap) DelIterator(iterator iservices.IDatabaseIterator) {
	if iterator == nil {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SExtRewardBlockHeightWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.RewardCashoutId {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListExtRewardByBlockHeight{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Id

}

func (s *SExtRewardBlockHeightWrap) GetSubVal(iterator iservices.IDatabaseIterator) *uint64 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListExtRewardByBlockHeight{}
	err = proto.Unmarshal(val, res)
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

func (s *SoExtRewardWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := s.getMemKeyPrefix("Id")
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
		key, err := s.encodeMemKey("Id")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemExtRewardById{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		sub := ori.Id
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
