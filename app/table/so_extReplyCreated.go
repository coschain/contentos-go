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
	ExtReplyCreatedTable             = []byte("ExtReplyCreatedTable")
	ExtReplyCreatedCreatedOrderTable = []byte("ExtReplyCreatedCreatedOrderTable")
	ExtReplyCreatedPostIdUniTable    = []byte("ExtReplyCreatedPostIdUniTable")
)

////////////// SECTION Wrap Define ///////////////
type SoExtReplyCreatedWrap struct {
	dba     iservices.IDatabaseService
	mainKey *uint64
}

func NewSoExtReplyCreatedWrap(dba iservices.IDatabaseService, key *uint64) *SoExtReplyCreatedWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoExtReplyCreatedWrap{dba, key}
	return result
}

func (s *SoExtReplyCreatedWrap) CheckExist() bool {
	if s.dba == nil {
		return false
	}
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return false
	}

	res, err := s.dba.Has(keyBuf)
	if err != nil {
		return false
	}

	return res
}

func (s *SoExtReplyCreatedWrap) Create(f func(tInfo *SoExtReplyCreated)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoExtReplyCreated{}
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
		return err
	}

	// update sort list keys
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

func (s *SoExtReplyCreatedWrap) encodeMemKey(fName string) ([]byte, error) {
	if len(fName) < 1 || s.mainKey == nil {
		return nil, errors.New("field name or main key is empty")
	}
	pre := "ExtReplyCreated" + fName + "cell"
	kList := []interface{}{pre, s.mainKey}
	key, err := kope.EncodeSlice(kList)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (so *SoExtReplyCreatedWrap) saveAllMemKeys(tInfo *SoExtReplyCreated, br bool) error {
	if so.dba == nil {
		return errors.New("save member Field fail , the db is nil")
	}

	if tInfo == nil {
		return errors.New("save member Field fail , the data is nil ")
	}
	var err error = nil
	errDes := ""
	if err = so.saveMemKeyCreatedOrder(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "CreatedOrder", err)
		}
	}
	if err = so.saveMemKeyPostId(tInfo); err != nil {
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

func (so *SoExtReplyCreatedWrap) delAllMemKeys(br bool, tInfo *SoExtReplyCreated) error {
	if so.dba == nil {
		return errors.New("the db is nil")
	}
	t := reflect.TypeOf(*tInfo)
	errDesc := ""
	for k := 0; k < t.NumField(); k++ {
		name := t.Field(k).Name
		if len(name) > 0 && !strings.HasPrefix(name, "XXX_") {
			err := so.delMemKey(name)
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

func (so *SoExtReplyCreatedWrap) delMemKey(fName string) error {
	if so.dba == nil {
		return errors.New("the db is nil")
	}
	if len(fName) <= 0 {
		return errors.New("the field name is empty ")
	}
	key, err := so.encodeMemKey(fName)
	if err != nil {
		return err
	}
	err = so.dba.Delete(key)
	return err
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoExtReplyCreatedWrap) delSortKeyCreatedOrder(sa *SoExtReplyCreated) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListExtReplyCreatedByCreatedOrder{}
	if sa == nil {
		key, err := s.encodeMemKey("CreatedOrder")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemExtReplyCreatedByCreatedOrder{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		val.CreatedOrder = ori.CreatedOrder
		val.PostId = *s.mainKey
	} else {
		val.CreatedOrder = sa.CreatedOrder
		val.PostId = sa.PostId
	}

	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoExtReplyCreatedWrap) insertSortKeyCreatedOrder(sa *SoExtReplyCreated) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListExtReplyCreatedByCreatedOrder{}
	val.PostId = sa.PostId
	val.CreatedOrder = sa.CreatedOrder
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

func (s *SoExtReplyCreatedWrap) delAllSortKeys(br bool, val *SoExtReplyCreated) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delSortKeyCreatedOrder(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoExtReplyCreatedWrap) insertAllSortKeys(val *SoExtReplyCreated) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoExtReplyCreated fail ")
	}
	if !s.insertSortKeyCreatedOrder(val) {
		return errors.New("insert sort Field CreatedOrder fail while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoExtReplyCreatedWrap) RemoveExtReplyCreated() bool {
	if s.dba == nil {
		return false
	}
	val := &SoExtReplyCreated{}
	//delete sort list key
	if res := s.delAllSortKeys(true, nil); !res {
		return false
	}

	//delete unique list
	if res := s.delAllUniKeys(true, nil); !res {
		return false
	}

	err := s.delAllMemKeys(true, val)
	return err == nil
}

////////////// SECTION Members Get/Modify ///////////////
func (s *SoExtReplyCreatedWrap) saveMemKeyCreatedOrder(tInfo *SoExtReplyCreated) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemExtReplyCreatedByCreatedOrder{}
	val.CreatedOrder = tInfo.CreatedOrder
	key, err := s.encodeMemKey("CreatedOrder")
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

func (s *SoExtReplyCreatedWrap) GetCreatedOrder() *prototype.ReplyCreatedOrder {
	res := true
	msg := &SoMemExtReplyCreatedByCreatedOrder{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("CreatedOrder")
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
				return msg.CreatedOrder
			}
		}
	}
	if !res {
		return nil

	}
	return msg.CreatedOrder
}

func (s *SoExtReplyCreatedWrap) MdCreatedOrder(p *prototype.ReplyCreatedOrder) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("CreatedOrder")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemExtReplyCreatedByCreatedOrder{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoExtReplyCreated{}
	sa.PostId = *s.mainKey
	sa.CreatedOrder = ori.CreatedOrder

	if !s.delSortKeyCreatedOrder(sa) {
		return false
	}
	ori.CreatedOrder = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.CreatedOrder = p

	if !s.insertSortKeyCreatedOrder(sa) {
		return false
	}

	return true
}

func (s *SoExtReplyCreatedWrap) saveMemKeyPostId(tInfo *SoExtReplyCreated) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemExtReplyCreatedByPostId{}
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

func (s *SoExtReplyCreatedWrap) GetPostId() uint64 {
	res := true
	msg := &SoMemExtReplyCreatedByPostId{}
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
type SExtReplyCreatedCreatedOrderWrap struct {
	Dba iservices.IDatabaseService
}

func NewExtReplyCreatedCreatedOrderWrap(db iservices.IDatabaseService) *SExtReplyCreatedCreatedOrderWrap {
	if db == nil {
		return nil
	}
	wrap := SExtReplyCreatedCreatedOrderWrap{Dba: db}
	return &wrap
}

func (s *SExtReplyCreatedCreatedOrderWrap) DelIterator(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SExtReplyCreatedCreatedOrderWrap) GetMainVal(iterator iservices.IDatabaseIterator) *uint64 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListExtReplyCreatedByCreatedOrder{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return &res.PostId

}

func (s *SExtReplyCreatedCreatedOrderWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.ReplyCreatedOrder {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListExtReplyCreatedByCreatedOrder{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.CreatedOrder

}

func (m *SoListExtReplyCreatedByCreatedOrder) OpeEncode() ([]byte, error) {
	pre := ExtReplyCreatedCreatedOrderTable
	sub := m.CreatedOrder
	if sub == nil {
		return nil, errors.New("the pro CreatedOrder is nil")
	}
	sub1 := m.PostId

	kList := []interface{}{pre, sub, sub1}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
}

//Query sort by reverse order
//
//f: callback for each traversal , primary 、sub key、idx(the number of times it has been iterated)
//as arguments to the callback function
//if the return value of f is true,continue iterating until the end iteration;
//otherwise stop iteration immediately
//
func (s *SExtReplyCreatedCreatedOrderWrap) ForEachByRevOrder(start *prototype.ReplyCreatedOrder, end *prototype.ReplyCreatedOrder,
	f func(mVal *uint64, sVal *prototype.ReplyCreatedOrder, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if f == nil {
		return nil
	}
	pre := ExtReplyCreatedCreatedOrderTable
	skeyList := []interface{}{pre}
	if start != nil {
		skeyList = append(skeyList, start)
	} else {
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

func (s *SoExtReplyCreatedWrap) update(sa *SoExtReplyCreated) bool {
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

func (s *SoExtReplyCreatedWrap) getExtReplyCreated() *SoExtReplyCreated {
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

	res := &SoExtReplyCreated{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoExtReplyCreatedWrap) encodeMainKey() ([]byte, error) {
	pre := "ExtReplyCreated" + "PostId" + "cell"
	sub := s.mainKey
	if sub == nil {
		return nil, errors.New("the mainKey is nil")
	}
	kList := []interface{}{pre, sub}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
}

////////////// Unique Query delete/insert/query ///////////////

func (s *SoExtReplyCreatedWrap) delAllUniKeys(br bool, val *SoExtReplyCreated) bool {
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

func (s *SoExtReplyCreatedWrap) delUniKeysWithNames(names map[string]string, val *SoExtReplyCreated) bool {
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

func (s *SoExtReplyCreatedWrap) insertAllUniKeys(val *SoExtReplyCreated) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoExtReplyCreated fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyPostId(val) {
		return sucFields, errors.New("insert unique Field PostId fail while insert table ")
	}
	sucFields["PostId"] = "PostId"

	return sucFields, nil
}

func (s *SoExtReplyCreatedWrap) delUniKeyPostId(sa *SoExtReplyCreated) bool {
	if s.dba == nil {
		return false
	}
	pre := ExtReplyCreatedPostIdUniTable
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
		ori := &SoMemExtReplyCreatedByPostId{}
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

func (s *SoExtReplyCreatedWrap) insertUniKeyPostId(sa *SoExtReplyCreated) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	uniWrap := UniExtReplyCreatedPostIdWrap{}
	uniWrap.Dba = s.dba
	res := uniWrap.UniQueryPostId(&sa.PostId)

	if res != nil {
		//the unique key is already exist
		return false
	}
	val := SoUniqueExtReplyCreatedByPostId{}
	val.PostId = sa.PostId

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	pre := ExtReplyCreatedPostIdUniTable
	sub := sa.PostId
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniExtReplyCreatedPostIdWrap struct {
	Dba iservices.IDatabaseService
}

func NewUniExtReplyCreatedPostIdWrap(db iservices.IDatabaseService) *UniExtReplyCreatedPostIdWrap {
	if db == nil {
		return nil
	}
	wrap := UniExtReplyCreatedPostIdWrap{Dba: db}
	return &wrap
}

func (s *UniExtReplyCreatedPostIdWrap) UniQueryPostId(start *uint64) *SoExtReplyCreatedWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := ExtReplyCreatedPostIdUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueExtReplyCreatedByPostId{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoExtReplyCreatedWrap(s.Dba, &res.PostId)
			return wrap
		}
	}
	return nil
}
