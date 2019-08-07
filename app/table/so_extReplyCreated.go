package table

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/coschain/contentos-go/common/encoding/kope"
	"github.com/coschain/contentos-go/iservices"
	prototype "github.com/coschain/contentos-go/prototype"
	proto "github.com/golang/protobuf/proto"
)

////////////// SECTION Prefix Mark ///////////////
var (
	ExtReplyCreatedCreatedOrderTable uint32 = 874612931
	ExtReplyCreatedPostIdUniTable    uint32 = 457271836

	ExtReplyCreatedPostIdRow uint32 = 3284075428
)

////////////// SECTION Wrap Define ///////////////
type SoExtReplyCreatedWrap struct {
	dba       iservices.IDatabaseRW
	mainKey   *uint64
	mKeyFlag  int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf   []byte //the buffer after the main key is encoded with prefix
	mBuf      []byte //the value after the main key is encoded
	mdFuncMap map[string]interface{}
}

func NewSoExtReplyCreatedWrap(dba iservices.IDatabaseRW, key *uint64) *SoExtReplyCreatedWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoExtReplyCreatedWrap{dba, key, -1, nil, nil, nil}
	return result
}

func (s *SoExtReplyCreatedWrap) CheckExist() bool {
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

func (s *SoExtReplyCreatedWrap) MustExist() *SoExtReplyCreatedWrap {
	if !s.CheckExist() {
		panic(fmt.Errorf("SoExtReplyCreatedWrap.MustExist: %v not found", s.mainKey))
	}
	return s
}

func (s *SoExtReplyCreatedWrap) MustNotExist() *SoExtReplyCreatedWrap {
	if s.CheckExist() {
		panic(fmt.Errorf("SoExtReplyCreatedWrap.MustNotExist: %v already exists", s.mainKey))
	}
	return s
}

func (s *SoExtReplyCreatedWrap) create(f func(tInfo *SoExtReplyCreated)) error {
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

func (s *SoExtReplyCreatedWrap) Create(f func(tInfo *SoExtReplyCreated), errArgs ...interface{}) *SoExtReplyCreatedWrap {
	err := s.create(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Errorf("SoExtReplyCreatedWrap.Create failed: %s", err.Error()), errArgs...))
	}
	return s
}

func (s *SoExtReplyCreatedWrap) getMainKeyBuf() ([]byte, error) {
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

func (s *SoExtReplyCreatedWrap) modify(f func(tInfo *SoExtReplyCreated)) error {
	if !s.CheckExist() {
		return errors.New("the SoExtReplyCreated table does not exist. Please create a table first")
	}
	oriTable := s.getExtReplyCreated()
	if oriTable == nil {
		return errors.New("fail to get origin table SoExtReplyCreated")
	}
	curTable := *oriTable
	f(&curTable)

	//the main key is not support modify
	if !reflect.DeepEqual(curTable.PostId, oriTable.PostId) {
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
	err = s.updateExtReplyCreated(&curTable)
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

func (s *SoExtReplyCreatedWrap) Modify(f func(tInfo *SoExtReplyCreated), errArgs ...interface{}) *SoExtReplyCreatedWrap {
	err := s.modify(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoExtReplyCreatedWrap.Modify failed: %s", err.Error()), errArgs...))
	}
	return s
}

func (s *SoExtReplyCreatedWrap) SetCreatedOrder(p *prototype.ReplyCreatedOrder, errArgs ...interface{}) *SoExtReplyCreatedWrap {
	err := s.modify(func(r *SoExtReplyCreated) {
		r.CreatedOrder = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoExtReplyCreatedWrap.SetCreatedOrder( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoExtReplyCreatedWrap) checkSortAndUniFieldValidity(curTable *SoExtReplyCreated, fieldSli []string) error {
	if curTable != nil && fieldSli != nil && len(fieldSli) > 0 {
		for _, fName := range fieldSli {
			if len(fName) > 0 {

				if fName == "CreatedOrder" && curTable.CreatedOrder == nil {
					return errors.New("sort field CreatedOrder can't be modified to nil")
				}

			}
		}
	}
	return nil
}

//Get all the modified fields in the table
func (s *SoExtReplyCreatedWrap) getModifiedFields(oriTable *SoExtReplyCreated, curTable *SoExtReplyCreated) ([]string, error) {
	if oriTable == nil {
		return nil, errors.New("table info is nil, can't get modified fields")
	}
	var list []string

	if !reflect.DeepEqual(oriTable.CreatedOrder, curTable.CreatedOrder) {
		list = append(list, "CreatedOrder")
	}

	return list, nil
}

func (s *SoExtReplyCreatedWrap) handleFieldMd(t FieldMdHandleType, so *SoExtReplyCreated, fSli []string) error {
	if so == nil {
		return errors.New("fail to modify empty table")
	}

	//there is no field need to modify
	if fSli == nil || len(fSli) < 1 {
		return nil
	}

	errStr := ""
	for _, fName := range fSli {

		if fName == "CreatedOrder" {
			res := true
			if t == FieldMdHandleTypeCheck {
				res = s.mdFieldCreatedOrder(so.CreatedOrder, true, false, false, so)
				errStr = fmt.Sprintf("fail to modify exist value of %v", fName)
			} else if t == FieldMdHandleTypeDel {
				res = s.mdFieldCreatedOrder(so.CreatedOrder, false, true, false, so)
				errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", fName)
			} else if t == FieldMdHandleTypeInsert {
				res = s.mdFieldCreatedOrder(so.CreatedOrder, false, false, true, so)
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

func (s *SoExtReplyCreatedWrap) delSortKeyCreatedOrder(sa *SoExtReplyCreated) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListExtReplyCreatedByCreatedOrder{}
	if sa == nil {
		val.CreatedOrder = s.GetCreatedOrder()
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

func (s *SoExtReplyCreatedWrap) removeExtReplyCreated() error {
	if s.dba == nil {
		return errors.New("database is nil")
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
		return nil
	} else {
		return fmt.Errorf("database.Delete failed: %s", err.Error())
	}
}

func (s *SoExtReplyCreatedWrap) RemoveExtReplyCreated(errMsgs ...interface{}) *SoExtReplyCreatedWrap {
	err := s.removeExtReplyCreated()
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoExtReplyCreatedWrap.RemoveExtReplyCreated failed: %s", err.Error()), errMsgs...))
	}
	return s
}

////////////// SECTION Members Get/Modify ///////////////

func (s *SoExtReplyCreatedWrap) GetCreatedOrder() *prototype.ReplyCreatedOrder {
	res := true
	msg := &SoExtReplyCreated{}
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
				return msg.CreatedOrder
			}
		}
	}
	if !res {
		return nil

	}
	return msg.CreatedOrder
}

func (s *SoExtReplyCreatedWrap) mdFieldCreatedOrder(p *prototype.ReplyCreatedOrder, isCheck bool, isDel bool, isInsert bool,
	so *SoExtReplyCreated) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkCreatedOrderIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldCreatedOrder(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldCreatedOrder(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoExtReplyCreatedWrap) delFieldCreatedOrder(so *SoExtReplyCreated) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyCreatedOrder(so) {
		return false
	}

	return true
}

func (s *SoExtReplyCreatedWrap) insertFieldCreatedOrder(so *SoExtReplyCreated) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyCreatedOrder(so) {
		return false
	}

	return true
}

func (s *SoExtReplyCreatedWrap) checkCreatedOrderIsMetMdCondition(p *prototype.ReplyCreatedOrder) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoExtReplyCreatedWrap) GetPostId() uint64 {
	res := true
	msg := &SoExtReplyCreated{}
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
	Dba iservices.IDatabaseRW
}

func NewExtReplyCreatedCreatedOrderWrap(db iservices.IDatabaseRW) *SExtReplyCreatedCreatedOrderWrap {
	if db == nil {
		return nil
	}
	wrap := SExtReplyCreatedCreatedOrderWrap{Dba: db}
	return &wrap
}

func (s *SExtReplyCreatedCreatedOrderWrap) GetMainVal(val []byte) *uint64 {
	res := &SoListExtReplyCreatedByCreatedOrder{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return &res.PostId

}

func (s *SExtReplyCreatedCreatedOrderWrap) GetSubVal(val []byte) *prototype.ReplyCreatedOrder {
	res := &SoListExtReplyCreatedByCreatedOrder{}
	err := proto.Unmarshal(val, res)
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
func (s *SExtReplyCreatedCreatedOrderWrap) ForEachByRevOrder(start *prototype.ReplyCreatedOrder, end *prototype.ReplyCreatedOrder, lastMainKey *uint64,
	lastSubVal *prototype.ReplyCreatedOrder, f func(mVal *uint64, sVal *prototype.ReplyCreatedOrder, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := ExtReplyCreatedCreatedOrderTable
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

func (s *SoExtReplyCreatedWrap) updateExtReplyCreated(so *SoExtReplyCreated) error {
	if s.dba == nil {
		return errors.New("update fail:the db is nil")
	}

	if so == nil {
		return errors.New("update fail: the SoExtReplyCreated is nil")
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

func (s *SoExtReplyCreatedWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := ExtReplyCreatedPostIdRow
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
		sub := s.GetPostId()

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

	pre := ExtReplyCreatedPostIdUniTable
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
	val := SoUniqueExtReplyCreatedByPostId{}
	val.PostId = sa.PostId

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniExtReplyCreatedPostIdWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniExtReplyCreatedPostIdWrap(db iservices.IDatabaseRW) *UniExtReplyCreatedPostIdWrap {
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
