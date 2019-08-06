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
	GiftTicketTicketTable    uint32 = 1694240687
	GiftTicketCountTable     uint32 = 3991811728
	GiftTicketTicketUniTable uint32 = 4012059461

	GiftTicketTicketRow uint32 = 3884327903
)

////////////// SECTION Wrap Define ///////////////
type SoGiftTicketWrap struct {
	dba       iservices.IDatabaseRW
	mainKey   *prototype.GiftTicketKeyType
	mKeyFlag  int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf   []byte //the buffer after the main key is encoded with prefix
	mBuf      []byte //the value after the main key is encoded
	mdFuncMap map[string]interface{}
}

func NewSoGiftTicketWrap(dba iservices.IDatabaseRW, key *prototype.GiftTicketKeyType) *SoGiftTicketWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoGiftTicketWrap{dba, key, -1, nil, nil, nil}
	return result
}

func (s *SoGiftTicketWrap) CheckExist() bool {
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

func (s *SoGiftTicketWrap) Create(f func(tInfo *SoGiftTicket)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoGiftTicket{}
	f(val)
	if val.Ticket == nil {
		val.Ticket = s.mainKey
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

func (s *SoGiftTicketWrap) getMainKeyBuf() ([]byte, error) {
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

func (s *SoGiftTicketWrap) Modify(f func(tInfo *SoGiftTicket)) error {
	if !s.CheckExist() {
		return errors.New("the SoGiftTicket table does not exist. Please create a table first")
	}
	oriTable := s.getGiftTicket()
	if oriTable == nil {
		return errors.New("fail to get origin table SoGiftTicket")
	}
	curTable := *oriTable
	f(&curTable)

	//the main key is not support modify
	if !reflect.DeepEqual(curTable.Ticket, oriTable.Ticket) {
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
	err = s.updateGiftTicket(&curTable)
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

func (s *SoGiftTicketWrap) MdCount(p uint64) bool {
	err := s.Modify(func(r *SoGiftTicket) {
		r.Count = p
	})
	return err == nil
}

func (s *SoGiftTicketWrap) MdDenom(p uint64) bool {
	err := s.Modify(func(r *SoGiftTicket) {
		r.Denom = p
	})
	return err == nil
}

func (s *SoGiftTicketWrap) MdExpireBlock(p uint64) bool {
	err := s.Modify(func(r *SoGiftTicket) {
		r.ExpireBlock = p
	})
	return err == nil
}

func (s *SoGiftTicketWrap) checkSortAndUniFieldValidity(curTable *SoGiftTicket, fieldSli []string) error {
	if curTable != nil && fieldSli != nil && len(fieldSli) > 0 {
		for _, fName := range fieldSli {
			if len(fName) > 0 {

			}
		}
	}
	return nil
}

//Get all the modified fields in the table
func (s *SoGiftTicketWrap) getModifiedFields(oriTable *SoGiftTicket, curTable *SoGiftTicket) ([]string, error) {
	if oriTable == nil {
		return nil, errors.New("table info is nil, can't get modified fields")
	}
	var list []string

	if !reflect.DeepEqual(oriTable.Count, curTable.Count) {
		list = append(list, "Count")
	}

	if !reflect.DeepEqual(oriTable.Denom, curTable.Denom) {
		list = append(list, "Denom")
	}

	if !reflect.DeepEqual(oriTable.ExpireBlock, curTable.ExpireBlock) {
		list = append(list, "ExpireBlock")
	}

	return list, nil
}

func (s *SoGiftTicketWrap) handleFieldMd(t FieldMdHandleType, so *SoGiftTicket, fSli []string) error {
	if so == nil {
		return errors.New("fail to modify empty table")
	}

	//there is no field need to modify
	if fSli == nil || len(fSli) < 1 {
		return nil
	}

	errStr := ""
	for _, fName := range fSli {

		if fName == "Count" {
			res := true
			if t == FieldMdHandleTypeCheck {
				res = s.mdFieldCount(so.Count, true, false, false, so)
				errStr = fmt.Sprintf("fail to modify exist value of %v", fName)
			} else if t == FieldMdHandleTypeDel {
				res = s.mdFieldCount(so.Count, false, true, false, so)
				errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", fName)
			} else if t == FieldMdHandleTypeInsert {
				res = s.mdFieldCount(so.Count, false, false, true, so)
				errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", fName)
			}
			if !res {
				return errors.New(errStr)
			}
		}

		if fName == "Denom" {
			res := true
			if t == FieldMdHandleTypeCheck {
				res = s.mdFieldDenom(so.Denom, true, false, false, so)
				errStr = fmt.Sprintf("fail to modify exist value of %v", fName)
			} else if t == FieldMdHandleTypeDel {
				res = s.mdFieldDenom(so.Denom, false, true, false, so)
				errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", fName)
			} else if t == FieldMdHandleTypeInsert {
				res = s.mdFieldDenom(so.Denom, false, false, true, so)
				errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", fName)
			}
			if !res {
				return errors.New(errStr)
			}
		}

		if fName == "ExpireBlock" {
			res := true
			if t == FieldMdHandleTypeCheck {
				res = s.mdFieldExpireBlock(so.ExpireBlock, true, false, false, so)
				errStr = fmt.Sprintf("fail to modify exist value of %v", fName)
			} else if t == FieldMdHandleTypeDel {
				res = s.mdFieldExpireBlock(so.ExpireBlock, false, true, false, so)
				errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", fName)
			} else if t == FieldMdHandleTypeInsert {
				res = s.mdFieldExpireBlock(so.ExpireBlock, false, false, true, so)
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

func (s *SoGiftTicketWrap) delSortKeyTicket(sa *SoGiftTicket) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListGiftTicketByTicket{}
	if sa == nil {
		val.Ticket = s.GetTicket()
	} else {
		val.Ticket = sa.Ticket
	}
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoGiftTicketWrap) insertSortKeyTicket(sa *SoGiftTicket) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListGiftTicketByTicket{}
	val.Ticket = sa.Ticket
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

func (s *SoGiftTicketWrap) delSortKeyCount(sa *SoGiftTicket) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListGiftTicketByCount{}
	if sa == nil {
		val.Count = s.GetCount()
		val.Ticket = s.mainKey

	} else {
		val.Count = sa.Count
		val.Ticket = sa.Ticket
	}
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoGiftTicketWrap) insertSortKeyCount(sa *SoGiftTicket) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListGiftTicketByCount{}
	val.Ticket = sa.Ticket
	val.Count = sa.Count
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

func (s *SoGiftTicketWrap) delAllSortKeys(br bool, val *SoGiftTicket) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delSortKeyTicket(val) {
		if br {
			return false
		} else {
			res = false
		}
	}
	if !s.delSortKeyCount(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoGiftTicketWrap) insertAllSortKeys(val *SoGiftTicket) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoGiftTicket fail ")
	}
	if !s.insertSortKeyTicket(val) {
		return errors.New("insert sort Field Ticket fail while insert table ")
	}
	if !s.insertSortKeyCount(val) {
		return errors.New("insert sort Field Count fail while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoGiftTicketWrap) RemoveGiftTicket() bool {
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

func (s *SoGiftTicketWrap) GetCount() uint64 {
	res := true
	msg := &SoGiftTicket{}
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
				return msg.Count
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.Count
}

func (s *SoGiftTicketWrap) mdFieldCount(p uint64, isCheck bool, isDel bool, isInsert bool,
	so *SoGiftTicket) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkCountIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldCount(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldCount(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoGiftTicketWrap) delFieldCount(so *SoGiftTicket) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyCount(so) {
		return false
	}

	return true
}

func (s *SoGiftTicketWrap) insertFieldCount(so *SoGiftTicket) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyCount(so) {
		return false
	}

	return true
}

func (s *SoGiftTicketWrap) checkCountIsMetMdCondition(p uint64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoGiftTicketWrap) GetDenom() uint64 {
	res := true
	msg := &SoGiftTicket{}
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
				return msg.Denom
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.Denom
}

func (s *SoGiftTicketWrap) mdFieldDenom(p uint64, isCheck bool, isDel bool, isInsert bool,
	so *SoGiftTicket) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkDenomIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldDenom(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldDenom(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoGiftTicketWrap) delFieldDenom(so *SoGiftTicket) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoGiftTicketWrap) insertFieldDenom(so *SoGiftTicket) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoGiftTicketWrap) checkDenomIsMetMdCondition(p uint64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoGiftTicketWrap) GetExpireBlock() uint64 {
	res := true
	msg := &SoGiftTicket{}
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
				return msg.ExpireBlock
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.ExpireBlock
}

func (s *SoGiftTicketWrap) mdFieldExpireBlock(p uint64, isCheck bool, isDel bool, isInsert bool,
	so *SoGiftTicket) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkExpireBlockIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldExpireBlock(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldExpireBlock(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoGiftTicketWrap) delFieldExpireBlock(so *SoGiftTicket) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoGiftTicketWrap) insertFieldExpireBlock(so *SoGiftTicket) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoGiftTicketWrap) checkExpireBlockIsMetMdCondition(p uint64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoGiftTicketWrap) GetTicket() *prototype.GiftTicketKeyType {
	res := true
	msg := &SoGiftTicket{}
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
				return msg.Ticket
			}
		}
	}
	if !res {
		return nil

	}
	return msg.Ticket
}

////////////// SECTION List Keys ///////////////
type SGiftTicketTicketWrap struct {
	Dba iservices.IDatabaseRW
}

func NewGiftTicketTicketWrap(db iservices.IDatabaseRW) *SGiftTicketTicketWrap {
	if db == nil {
		return nil
	}
	wrap := SGiftTicketTicketWrap{Dba: db}
	return &wrap
}

func (s *SGiftTicketTicketWrap) GetMainVal(val []byte) *prototype.GiftTicketKeyType {
	res := &SoListGiftTicketByTicket{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Ticket

}

func (s *SGiftTicketTicketWrap) GetSubVal(val []byte) *prototype.GiftTicketKeyType {
	res := &SoListGiftTicketByTicket{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.Ticket

}

func (m *SoListGiftTicketByTicket) OpeEncode() ([]byte, error) {
	pre := GiftTicketTicketTable
	sub := m.Ticket
	if sub == nil {
		return nil, errors.New("the pro Ticket is nil")
	}
	sub1 := m.Ticket
	if sub1 == nil {
		return nil, errors.New("the mainkey Ticket is nil")
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
func (s *SGiftTicketTicketWrap) ForEachByOrder(start *prototype.GiftTicketKeyType, end *prototype.GiftTicketKeyType, lastMainKey *prototype.GiftTicketKeyType,
	lastSubVal *prototype.GiftTicketKeyType, f func(mVal *prototype.GiftTicketKeyType, sVal *prototype.GiftTicketKeyType, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := GiftTicketTicketTable
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
func (s *SGiftTicketTicketWrap) ForEachByRevOrder(start *prototype.GiftTicketKeyType, end *prototype.GiftTicketKeyType, lastMainKey *prototype.GiftTicketKeyType,
	lastSubVal *prototype.GiftTicketKeyType, f func(mVal *prototype.GiftTicketKeyType, sVal *prototype.GiftTicketKeyType, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := GiftTicketTicketTable
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

////////////// SECTION List Keys ///////////////
type SGiftTicketCountWrap struct {
	Dba iservices.IDatabaseRW
}

func NewGiftTicketCountWrap(db iservices.IDatabaseRW) *SGiftTicketCountWrap {
	if db == nil {
		return nil
	}
	wrap := SGiftTicketCountWrap{Dba: db}
	return &wrap
}

func (s *SGiftTicketCountWrap) GetMainVal(val []byte) *prototype.GiftTicketKeyType {
	res := &SoListGiftTicketByCount{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Ticket

}

func (s *SGiftTicketCountWrap) GetSubVal(val []byte) *uint64 {
	res := &SoListGiftTicketByCount{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return &res.Count

}

func (m *SoListGiftTicketByCount) OpeEncode() ([]byte, error) {
	pre := GiftTicketCountTable
	sub := m.Count

	sub1 := m.Ticket
	if sub1 == nil {
		return nil, errors.New("the mainkey Ticket is nil")
	}
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
func (s *SGiftTicketCountWrap) ForEachByRevOrder(start *uint64, end *uint64, lastMainKey *prototype.GiftTicketKeyType,
	lastSubVal *uint64, f func(mVal *prototype.GiftTicketKeyType, sVal *uint64, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := GiftTicketCountTable
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

func (s *SoGiftTicketWrap) update(sa *SoGiftTicket) bool {
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

func (s *SoGiftTicketWrap) getGiftTicket() *SoGiftTicket {
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

	res := &SoGiftTicket{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoGiftTicketWrap) updateGiftTicket(so *SoGiftTicket) error {
	if s.dba == nil {
		return errors.New("update fail:the db is nil")
	}

	if so == nil {
		return errors.New("update fail: the SoGiftTicket is nil")
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

func (s *SoGiftTicketWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := GiftTicketTicketRow
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

func (s *SoGiftTicketWrap) delAllUniKeys(br bool, val *SoGiftTicket) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delUniKeyTicket(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoGiftTicketWrap) delUniKeysWithNames(names map[string]string, val *SoGiftTicket) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if len(names["Ticket"]) > 0 {
		if !s.delUniKeyTicket(val) {
			res = false
		}
	}

	return res
}

func (s *SoGiftTicketWrap) insertAllUniKeys(val *SoGiftTicket) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoGiftTicket fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyTicket(val) {
		return sucFields, errors.New("insert unique Field Ticket fail while insert table ")
	}
	sucFields["Ticket"] = "Ticket"

	return sucFields, nil
}

func (s *SoGiftTicketWrap) delUniKeyTicket(sa *SoGiftTicket) bool {
	if s.dba == nil {
		return false
	}
	pre := GiftTicketTicketUniTable
	kList := []interface{}{pre}
	if sa != nil {
		if sa.Ticket == nil {
			return false
		}

		sub := sa.Ticket
		kList = append(kList, sub)
	} else {
		sub := s.GetTicket()
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

func (s *SoGiftTicketWrap) insertUniKeyTicket(sa *SoGiftTicket) bool {
	if s.dba == nil || sa == nil {
		return false
	}

	pre := GiftTicketTicketUniTable
	sub := sa.Ticket
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
	val := SoUniqueGiftTicketByTicket{}
	val.Ticket = sa.Ticket

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniGiftTicketTicketWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniGiftTicketTicketWrap(db iservices.IDatabaseRW) *UniGiftTicketTicketWrap {
	if db == nil {
		return nil
	}
	wrap := UniGiftTicketTicketWrap{Dba: db}
	return &wrap
}

func (s *UniGiftTicketTicketWrap) UniQueryTicket(start *prototype.GiftTicketKeyType) *SoGiftTicketWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := GiftTicketTicketUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueGiftTicketByTicket{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoGiftTicketWrap(s.Dba, res.Ticket)

			return wrap
		}
	}
	return nil
}
