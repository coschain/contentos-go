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
	ExtPostCreatedTable             = []byte("ExtPostCreatedTable")
	ExtPostCreatedCreatedOrderTable = []byte("ExtPostCreatedCreatedOrderTable")
	ExtPostCreatedPostIdUniTable    = []byte("ExtPostCreatedPostIdUniTable")
)

////////////// SECTION Wrap Define ///////////////
type SoExtPostCreatedWrap struct {
	dba     iservices.IDatabaseService
	mainKey *uint64
}

func NewSoExtPostCreatedWrap(dba iservices.IDatabaseService, key *uint64) *SoExtPostCreatedWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoExtPostCreatedWrap{dba, key}
	return result
}

func (s *SoExtPostCreatedWrap) CheckExist() bool {
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

func (s *SoExtPostCreatedWrap) Create(f func(tInfo *SoExtPostCreated)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoExtPostCreated{}
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
	if err = s.insertAllUniKeys(val); err != nil {
		s.delAllSortKeys(false, val)
		s.delAllUniKeys(false, val)
		s.dba.Delete(keyBuf)
		s.delAllMemKeys(false, val)
		return err
	}

	return nil
}

func (s *SoExtPostCreatedWrap) encodeMemKey(fName string) ([]byte, error) {
	if len(fName) < 1 || s.mainKey == nil {
		return nil, errors.New("field name or main key is empty")
	}
	pre := "ExtPostCreated" + fName + "cell"
	kList := []interface{}{pre, s.mainKey}
	key, err := kope.EncodeSlice(kList)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (so *SoExtPostCreatedWrap) saveAllMemKeys(tInfo *SoExtPostCreated, br bool) error {
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

func (so *SoExtPostCreatedWrap) delAllMemKeys(br bool, tInfo *SoExtPostCreated) error {
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

func (so *SoExtPostCreatedWrap) delMemKey(fName string) error {
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

func (s *SoExtPostCreatedWrap) delSortKeyCreatedOrder(sa *SoExtPostCreated) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListExtPostCreatedByCreatedOrder{}
	if sa == nil {
		key, err := s.encodeMemKey("CreatedOrder")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemExtPostCreatedByCreatedOrder{}
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

func (s *SoExtPostCreatedWrap) insertSortKeyCreatedOrder(sa *SoExtPostCreated) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListExtPostCreatedByCreatedOrder{}
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

func (s *SoExtPostCreatedWrap) delAllSortKeys(br bool, val *SoExtPostCreated) bool {
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

func (s *SoExtPostCreatedWrap) insertAllSortKeys(val *SoExtPostCreated) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoExtPostCreated fail ")
	}
	if !s.insertSortKeyCreatedOrder(val) {
		return errors.New("insert sort Field CreatedOrder fail while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoExtPostCreatedWrap) RemoveExtPostCreated() bool {
	if s.dba == nil {
		return false
	}
	val := &SoExtPostCreated{}
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
func (s *SoExtPostCreatedWrap) saveMemKeyCreatedOrder(tInfo *SoExtPostCreated) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemExtPostCreatedByCreatedOrder{}
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

func (s *SoExtPostCreatedWrap) GetCreatedOrder() *prototype.PostCreatedOrder {
	res := true
	msg := &SoMemExtPostCreatedByCreatedOrder{}
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

func (s *SoExtPostCreatedWrap) MdCreatedOrder(p *prototype.PostCreatedOrder) bool {
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
	ori := &SoMemExtPostCreatedByCreatedOrder{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoExtPostCreated{}
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

func (s *SoExtPostCreatedWrap) saveMemKeyPostId(tInfo *SoExtPostCreated) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemExtPostCreatedByPostId{}
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

func (s *SoExtPostCreatedWrap) GetPostId() uint64 {
	res := true
	msg := &SoMemExtPostCreatedByPostId{}
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
type SExtPostCreatedCreatedOrderWrap struct {
	Dba iservices.IDatabaseService
}

func NewExtPostCreatedCreatedOrderWrap(db iservices.IDatabaseService) *SExtPostCreatedCreatedOrderWrap {
	if db == nil {
		return nil
	}
	wrap := SExtPostCreatedCreatedOrderWrap{Dba: db}
	return &wrap
}

func (s *SExtPostCreatedCreatedOrderWrap) DelIterater(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SExtPostCreatedCreatedOrderWrap) GetMainVal(iterator iservices.IDatabaseIterator) *uint64 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListExtPostCreatedByCreatedOrder{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return &res.PostId

}

func (s *SExtPostCreatedCreatedOrderWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.PostCreatedOrder {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListExtPostCreatedByCreatedOrder{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.CreatedOrder

}

func (m *SoListExtPostCreatedByCreatedOrder) OpeEncode() ([]byte, error) {
	pre := ExtPostCreatedCreatedOrderTable
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
func (s *SExtPostCreatedCreatedOrderWrap) QueryListByRevOrder(start *prototype.PostCreatedOrder, end *prototype.PostCreatedOrder) iservices.IDatabaseIterator {
	if s.Dba == nil {
		return nil
	}
	pre := ExtPostCreatedCreatedOrderTable
	skeyList := []interface{}{pre}
	if start != nil {
		skeyList = append(skeyList, start)
	} else {
		skeyList = append(skeyList, kope.MaximumKey)
	}
	sBuf, cErr := kope.EncodeSlice(skeyList)
	if cErr != nil {
		return nil
	}
	eKeyList := []interface{}{pre}
	if end != nil {
		eKeyList = append(eKeyList, end)
	}
	eBuf, cErr := kope.EncodeSlice(eKeyList)
	if cErr != nil {
		return nil
	}
	//reverse the start and end when create ReversedIterator to query by reverse order
	iter := s.Dba.NewReversedIterator(eBuf, sBuf)
	return iter
}

/////////////// SECTION Private function ////////////////

func (s *SoExtPostCreatedWrap) update(sa *SoExtPostCreated) bool {
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

func (s *SoExtPostCreatedWrap) getExtPostCreated() *SoExtPostCreated {
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

	res := &SoExtPostCreated{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoExtPostCreatedWrap) encodeMainKey() ([]byte, error) {
	pre := "ExtPostCreated" + "PostId" + "cell"
	sub := s.mainKey
	if sub == nil {
		return nil, errors.New("the mainKey is nil")
	}
	kList := []interface{}{pre, sub}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
}

////////////// Unique Query delete/insert/query ///////////////

func (s *SoExtPostCreatedWrap) delAllUniKeys(br bool, val *SoExtPostCreated) bool {
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

func (s *SoExtPostCreatedWrap) insertAllUniKeys(val *SoExtPostCreated) error {
	if s.dba == nil {
		return errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert uniuqe Field fail,get the SoExtPostCreated fail ")
	}
	if !s.insertUniKeyPostId(val) {
		return errors.New("insert unique Field PostId fail while insert table ")
	}

	return nil
}

func (s *SoExtPostCreatedWrap) delUniKeyPostId(sa *SoExtPostCreated) bool {
	if s.dba == nil {
		return false
	}
	pre := ExtPostCreatedPostIdUniTable
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
		ori := &SoMemExtPostCreatedByPostId{}
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

func (s *SoExtPostCreatedWrap) insertUniKeyPostId(sa *SoExtPostCreated) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	uniWrap := UniExtPostCreatedPostIdWrap{}
	uniWrap.Dba = s.dba
	res := uniWrap.UniQueryPostId(&sa.PostId)

	if res != nil {
		//the unique key is already exist
		return false
	}
	val := SoUniqueExtPostCreatedByPostId{}
	val.PostId = sa.PostId

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	pre := ExtPostCreatedPostIdUniTable
	sub := sa.PostId
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniExtPostCreatedPostIdWrap struct {
	Dba iservices.IDatabaseService
}

func NewUniExtPostCreatedPostIdWrap(db iservices.IDatabaseService) *UniExtPostCreatedPostIdWrap {
	if db == nil {
		return nil
	}
	wrap := UniExtPostCreatedPostIdWrap{Dba: db}
	return &wrap
}

func (s *UniExtPostCreatedPostIdWrap) UniQueryPostId(start *uint64) *SoExtPostCreatedWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := ExtPostCreatedPostIdUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueExtPostCreatedByPostId{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoExtPostCreatedWrap(s.Dba, &res.PostId)
			return wrap
		}
	}
	return nil
}
