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
	ExtPostCreatedCreatedOrderTable uint32 = 2563559990
	ExtPostCreatedPostIdUniTable    uint32 = 2848909971

	ExtPostCreatedPostIdRow uint32 = 1997264083
)

////////////// SECTION Wrap Define ///////////////
type SoExtPostCreatedWrap struct {
	dba       iservices.IDatabaseRW
	mainKey   *uint64
	mKeyFlag  int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf   []byte //the buffer after the main key is encoded with prefix
	mBuf      []byte //the value after the main key is encoded
	mdFuncMap map[string]interface{}
}

func NewSoExtPostCreatedWrap(dba iservices.IDatabaseRW, key *uint64) *SoExtPostCreatedWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoExtPostCreatedWrap{dba, key, -1, nil, nil, nil}
	return result
}

func (s *SoExtPostCreatedWrap) CheckExist() bool {
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

func (s *SoExtPostCreatedWrap) MustExist(errMsgs ...interface{}) *SoExtPostCreatedWrap {
	if !s.CheckExist() {
		panic(bindErrorInfo(fmt.Sprintf("SoExtPostCreatedWrap.MustExist: %v not found", s.mainKey), errMsgs...))
	}
	return s
}

func (s *SoExtPostCreatedWrap) MustNotExist(errMsgs ...interface{}) *SoExtPostCreatedWrap {
	if s.CheckExist() {
		panic(bindErrorInfo(fmt.Sprintf("SoExtPostCreatedWrap.MustNotExist: %v already exists", s.mainKey), errMsgs...))
	}
	return s
}

func (s *SoExtPostCreatedWrap) create(f func(tInfo *SoExtPostCreated)) error {
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

func (s *SoExtPostCreatedWrap) Create(f func(tInfo *SoExtPostCreated), errArgs ...interface{}) *SoExtPostCreatedWrap {
	err := s.create(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Errorf("SoExtPostCreatedWrap.Create failed: %s", err.Error()), errArgs...))
	}
	return s
}

func (s *SoExtPostCreatedWrap) getMainKeyBuf() ([]byte, error) {
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

func (s *SoExtPostCreatedWrap) modify(f func(tInfo *SoExtPostCreated)) error {
	if !s.CheckExist() {
		return errors.New("the SoExtPostCreated table does not exist. Please create a table first")
	}
	oriTable := s.getExtPostCreated()
	if oriTable == nil {
		return errors.New("fail to get origin table SoExtPostCreated")
	}

	curTable := s.getExtPostCreated()
	if curTable == nil {
		return errors.New("fail to create current table SoExtPostCreated")
	}
	f(curTable)

	//the main key is not support modify
	if !reflect.DeepEqual(curTable.PostId, oriTable.PostId) {
		return errors.New("primary key does not support modification")
	}

	fieldSli, err := s.getModifiedFields(oriTable, curTable)
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
	err = s.updateExtPostCreated(curTable)
	if err != nil {
		return err
	}

	//insert sort and unique key
	err = s.handleFieldMd(FieldMdHandleTypeInsert, curTable, fieldSli)
	if err != nil {
		return err
	}

	return nil

}

func (s *SoExtPostCreatedWrap) Modify(f func(tInfo *SoExtPostCreated), errArgs ...interface{}) *SoExtPostCreatedWrap {
	err := s.modify(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoExtPostCreatedWrap.Modify failed: %s", err.Error()), errArgs...))
	}
	return s
}

func (s *SoExtPostCreatedWrap) SetCreatedOrder(p *prototype.PostCreatedOrder, errArgs ...interface{}) *SoExtPostCreatedWrap {
	err := s.modify(func(r *SoExtPostCreated) {
		r.CreatedOrder = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoExtPostCreatedWrap.SetCreatedOrder( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoExtPostCreatedWrap) checkSortAndUniFieldValidity(curTable *SoExtPostCreated, fieldSli []string) error {
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
func (s *SoExtPostCreatedWrap) getModifiedFields(oriTable *SoExtPostCreated, curTable *SoExtPostCreated) ([]string, error) {
	if oriTable == nil {
		return nil, errors.New("table info is nil, can't get modified fields")
	}
	var list []string

	if !reflect.DeepEqual(oriTable.CreatedOrder, curTable.CreatedOrder) {
		list = append(list, "CreatedOrder")
	}

	return list, nil
}

func (s *SoExtPostCreatedWrap) handleFieldMd(t FieldMdHandleType, so *SoExtPostCreated, fSli []string) error {
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

func (s *SoExtPostCreatedWrap) delSortKeyCreatedOrder(sa *SoExtPostCreated) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListExtPostCreatedByCreatedOrder{}
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

func (s *SoExtPostCreatedWrap) removeExtPostCreated() error {
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

func (s *SoExtPostCreatedWrap) RemoveExtPostCreated(errMsgs ...interface{}) *SoExtPostCreatedWrap {
	err := s.removeExtPostCreated()
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoExtPostCreatedWrap.RemoveExtPostCreated failed: %s", err.Error()), errMsgs...))
	}
	return s
}

////////////// SECTION Members Get/Modify ///////////////

func (s *SoExtPostCreatedWrap) GetCreatedOrder() *prototype.PostCreatedOrder {
	res := true
	msg := &SoExtPostCreated{}
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

func (s *SoExtPostCreatedWrap) mdFieldCreatedOrder(p *prototype.PostCreatedOrder, isCheck bool, isDel bool, isInsert bool,
	so *SoExtPostCreated) bool {
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

func (s *SoExtPostCreatedWrap) delFieldCreatedOrder(so *SoExtPostCreated) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyCreatedOrder(so) {
		return false
	}

	return true
}

func (s *SoExtPostCreatedWrap) insertFieldCreatedOrder(so *SoExtPostCreated) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyCreatedOrder(so) {
		return false
	}

	return true
}

func (s *SoExtPostCreatedWrap) checkCreatedOrderIsMetMdCondition(p *prototype.PostCreatedOrder) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoExtPostCreatedWrap) GetPostId() uint64 {
	res := true
	msg := &SoExtPostCreated{}
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
type SExtPostCreatedCreatedOrderWrap struct {
	Dba iservices.IDatabaseRW
}

func NewExtPostCreatedCreatedOrderWrap(db iservices.IDatabaseRW) *SExtPostCreatedCreatedOrderWrap {
	if db == nil {
		return nil
	}
	wrap := SExtPostCreatedCreatedOrderWrap{Dba: db}
	return &wrap
}

func (s *SExtPostCreatedCreatedOrderWrap) GetMainVal(val []byte) *uint64 {
	res := &SoListExtPostCreatedByCreatedOrder{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return &res.PostId

}

func (s *SExtPostCreatedCreatedOrderWrap) GetSubVal(val []byte) *prototype.PostCreatedOrder {
	res := &SoListExtPostCreatedByCreatedOrder{}
	err := proto.Unmarshal(val, res)
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
func (s *SExtPostCreatedCreatedOrderWrap) ForEachByRevOrder(start *prototype.PostCreatedOrder, end *prototype.PostCreatedOrder, lastMainKey *uint64,
	lastSubVal *prototype.PostCreatedOrder, f func(mVal *uint64, sVal *prototype.PostCreatedOrder, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := ExtPostCreatedCreatedOrderTable
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

func (s *SoExtPostCreatedWrap) updateExtPostCreated(so *SoExtPostCreated) error {
	if s.dba == nil {
		return errors.New("update fail:the db is nil")
	}

	if so == nil {
		return errors.New("update fail: the SoExtPostCreated is nil")
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

func (s *SoExtPostCreatedWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := ExtPostCreatedPostIdRow
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

func (s *SoExtPostCreatedWrap) delUniKeysWithNames(names map[string]string, val *SoExtPostCreated) bool {
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

func (s *SoExtPostCreatedWrap) insertAllUniKeys(val *SoExtPostCreated) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoExtPostCreated fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyPostId(val) {
		return sucFields, errors.New("insert unique Field PostId fail while insert table ")
	}
	sucFields["PostId"] = "PostId"

	return sucFields, nil
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
		sub := s.GetPostId()

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

	pre := ExtPostCreatedPostIdUniTable
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
	val := SoUniqueExtPostCreatedByPostId{}
	val.PostId = sa.PostId

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniExtPostCreatedPostIdWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniExtPostCreatedPostIdWrap(db iservices.IDatabaseRW) *UniExtPostCreatedPostIdWrap {
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
