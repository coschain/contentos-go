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
	ExtUserPostPostCreatedOrderTable uint32 = 555226009
	ExtUserPostPostIdUniTable        uint32 = 2411654352
	ExtUserPostPostCreatedOrderCell  uint32 = 4157191683
	ExtUserPostPostIdCell            uint32 = 3675031809
)

////////////// SECTION Wrap Define ///////////////
type SoExtUserPostWrap struct {
	dba      iservices.IDatabaseRW
	mainKey  *uint64
	mKeyFlag int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf  []byte //the buffer after the main key is encoded with prefix
	mBuf     []byte //the value after the main key is encoded
}

func NewSoExtUserPostWrap(dba iservices.IDatabaseRW, key *uint64) *SoExtUserPostWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoExtUserPostWrap{dba, key, -1, nil, nil}
	return result
}

func (s *SoExtUserPostWrap) CheckExist() bool {
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

func (s *SoExtUserPostWrap) Create(f func(tInfo *SoExtUserPost)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoExtUserPost{}
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

func (s *SoExtUserPostWrap) getMainKeyBuf() ([]byte, error) {
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

func (s *SoExtUserPostWrap) delSortKeyPostCreatedOrder(sa *SoExtUserPost) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListExtUserPostByPostCreatedOrder{}
	if sa == nil {
		key, err := s.encodeMemKey("PostCreatedOrder")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemExtUserPostByPostCreatedOrder{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		val.PostCreatedOrder = ori.PostCreatedOrder
		val.PostId = *s.mainKey
	} else {
		val.PostCreatedOrder = sa.PostCreatedOrder
		val.PostId = sa.PostId
	}

	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoExtUserPostWrap) insertSortKeyPostCreatedOrder(sa *SoExtUserPost) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListExtUserPostByPostCreatedOrder{}
	val.PostId = sa.PostId
	val.PostCreatedOrder = sa.PostCreatedOrder
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

func (s *SoExtUserPostWrap) delAllSortKeys(br bool, val *SoExtUserPost) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delSortKeyPostCreatedOrder(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoExtUserPostWrap) insertAllSortKeys(val *SoExtUserPost) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoExtUserPost fail ")
	}
	if !s.insertSortKeyPostCreatedOrder(val) {
		return errors.New("insert sort Field PostCreatedOrder fail while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoExtUserPostWrap) RemoveExtUserPost() bool {
	if s.dba == nil {
		return false
	}
	val := &SoExtUserPost{}
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
func (s *SoExtUserPostWrap) getMemKeyPrefix(fName string) uint32 {
	if fName == "PostCreatedOrder" {
		return ExtUserPostPostCreatedOrderCell
	}
	if fName == "PostId" {
		return ExtUserPostPostIdCell
	}

	return 0
}

func (s *SoExtUserPostWrap) encodeMemKey(fName string) ([]byte, error) {
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

func (s *SoExtUserPostWrap) saveAllMemKeys(tInfo *SoExtUserPost, br bool) error {
	if s.dba == nil {
		return errors.New("save member Field fail , the db is nil")
	}

	if tInfo == nil {
		return errors.New("save member Field fail , the data is nil ")
	}
	var err error = nil
	errDes := ""
	if err = s.saveMemKeyPostCreatedOrder(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "PostCreatedOrder", err)
		}
	}
	if err = s.saveMemKeyPostId(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "PostId", err)
		}
	}

	if len(errDes) > 0 {
		return errors.New(errDes)
	}
	return err
}

func (s *SoExtUserPostWrap) delAllMemKeys(br bool, tInfo *SoExtUserPost) error {
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

func (s *SoExtUserPostWrap) delMemKey(fName string) error {
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

func (s *SoExtUserPostWrap) saveMemKeyPostCreatedOrder(tInfo *SoExtUserPost) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemExtUserPostByPostCreatedOrder{}
	val.PostCreatedOrder = tInfo.PostCreatedOrder
	key, err := s.encodeMemKey("PostCreatedOrder")
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

func (s *SoExtUserPostWrap) GetPostCreatedOrder() *prototype.UserPostCreateOrder {
	res := true
	msg := &SoMemExtUserPostByPostCreatedOrder{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("PostCreatedOrder")
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
				return msg.PostCreatedOrder
			}
		}
	}
	if !res {
		return nil

	}
	return msg.PostCreatedOrder
}

func (s *SoExtUserPostWrap) MdPostCreatedOrder(p *prototype.UserPostCreateOrder) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("PostCreatedOrder")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemExtUserPostByPostCreatedOrder{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoExtUserPost{}
	sa.PostId = *s.mainKey
	sa.PostCreatedOrder = ori.PostCreatedOrder

	if !s.delSortKeyPostCreatedOrder(sa) {
		return false
	}
	ori.PostCreatedOrder = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.PostCreatedOrder = p

	if !s.insertSortKeyPostCreatedOrder(sa) {
		return false
	}

	return true
}

func (s *SoExtUserPostWrap) saveMemKeyPostId(tInfo *SoExtUserPost) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemExtUserPostByPostId{}
	val.PostId = tInfo.PostId
	key, err := s.encodeMemKey("PostId")
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

func (s *SoExtUserPostWrap) GetPostId() uint64 {
	res := true
	msg := &SoMemExtUserPostByPostId{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("PostId")
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
				return msg.PostId
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.PostId
}

////////////// SECTION List Keys ///////////////
type SExtUserPostPostCreatedOrderWrap struct {
	Dba iservices.IDatabaseRW
}

func NewExtUserPostPostCreatedOrderWrap(db iservices.IDatabaseRW) *SExtUserPostPostCreatedOrderWrap {
	if db == nil {
		return nil
	}
	wrap := SExtUserPostPostCreatedOrderWrap{Dba: db}
	return &wrap
}

func (s *SExtUserPostPostCreatedOrderWrap) GetMainVal(val []byte) *uint64 {
	res := &SoListExtUserPostByPostCreatedOrder{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return &res.PostId

}

func (s *SExtUserPostPostCreatedOrderWrap) GetSubVal(val []byte) *prototype.UserPostCreateOrder {
	res := &SoListExtUserPostByPostCreatedOrder{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.PostCreatedOrder

}

func (m *SoListExtUserPostByPostCreatedOrder) OpeEncode() ([]byte, error) {
	pre := ExtUserPostPostCreatedOrderTable
	sub := m.PostCreatedOrder
	if sub == nil {
		return nil, errors.New("the pro PostCreatedOrder is nil")
	}
	sub1 := m.PostId

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
func (s *SExtUserPostPostCreatedOrderWrap) ForEachByOrder(start *prototype.UserPostCreateOrder, end *prototype.UserPostCreateOrder, lastMainKey *uint64,
	lastSubVal *prototype.UserPostCreateOrder, f func(mVal *uint64, sVal *prototype.UserPostCreateOrder, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := ExtUserPostPostCreatedOrderTable
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
func (s *SExtUserPostPostCreatedOrderWrap) ForEachByRevOrder(start *prototype.UserPostCreateOrder, end *prototype.UserPostCreateOrder, lastMainKey *uint64,
	lastSubVal *prototype.UserPostCreateOrder, f func(mVal *uint64, sVal *prototype.UserPostCreateOrder, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := ExtUserPostPostCreatedOrderTable
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

func (s *SoExtUserPostWrap) update(sa *SoExtUserPost) bool {
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

func (s *SoExtUserPostWrap) getExtUserPost() *SoExtUserPost {
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

	res := &SoExtUserPost{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoExtUserPostWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := s.getMemKeyPrefix("PostId")
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

func (s *SoExtUserPostWrap) delAllUniKeys(br bool, val *SoExtUserPost) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delUniKeyPostId(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoExtUserPostWrap) delUniKeysWithNames(names map[string]string, val *SoExtUserPost) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if len(names["PostId"]) > 0 {
		if !s.delUniKeyPostId(val) {
			res = false
		}
	}

	return res
}

func (s *SoExtUserPostWrap) insertAllUniKeys(val *SoExtUserPost) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoExtUserPost fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyPostId(val) {
		return sucFields, errors.New("insert unique Field PostId fail while insert table ")
	}
	sucFields["PostId"] = "PostId"

	return sucFields, nil
}

func (s *SoExtUserPostWrap) delUniKeyPostId(sa *SoExtUserPost) bool {
	if s.dba == nil {
		return false
	}
	pre := ExtUserPostPostIdUniTable
	kList := []interface{}{pre}
	if sa != nil {

		sub := sa.PostId
		kList = append(kList, sub)
	} else {
		key, err := s.encodeMemKey("PostId")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemExtUserPostByPostId{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		sub := ori.PostId
		kList = append(kList, sub)

	}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoExtUserPostWrap) insertUniKeyPostId(sa *SoExtUserPost) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	pre := ExtUserPostPostIdUniTable
	sub := sa.PostId
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
	val := SoUniqueExtUserPostByPostId{}
	val.PostId = sa.PostId

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniExtUserPostPostIdWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniExtUserPostPostIdWrap(db iservices.IDatabaseRW) *UniExtUserPostPostIdWrap {
	if db == nil {
		return nil
	}
	wrap := UniExtUserPostPostIdWrap{Dba: db}
	return &wrap
}

func (s *UniExtUserPostPostIdWrap) UniQueryPostId(start *uint64) *SoExtUserPostWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := ExtUserPostPostIdUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueExtUserPostByPostId{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoExtUserPostWrap(s.Dba, &res.PostId)
			return wrap
		}
	}
	return nil
}
