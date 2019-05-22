package table

import (
	"encoding/json"
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
	AccountCreatedTimeTable           uint32 = 2128286283
	AccountBalanceTable               uint32 = 4012029019
	AccountVestingSharesTable         uint32 = 3830877790
	AccountBpVoteCountTable           uint32 = 2264397557
	AccountPostCountTable             uint32 = 1518203339
	AccountCreatedTrxCountTable       uint32 = 2604810499
	AccountNextPowerdownBlockNumTable uint32 = 1928824877
	AccountNameUniTable               uint32 = 2528390520
	AccountOwnerUniTable              uint32 = 4120855558

	AccountNameRow uint32 = 3130128817
)

////////////// SECTION Wrap Define ///////////////
type SoAccountWrap struct {
	dba       iservices.IDatabaseRW
	mainKey   *prototype.AccountName
	mKeyFlag  int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf   []byte //the buffer after the main key is encoded with prefix
	mBuf      []byte //the value after the main key is encoded
	mdFuncMap map[string]interface{}
}

func NewSoAccountWrap(dba iservices.IDatabaseRW, key *prototype.AccountName) *SoAccountWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoAccountWrap{dba, key, -1, nil, nil, nil}
	return result
}

func (s *SoAccountWrap) CheckExist() bool {
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

func (s *SoAccountWrap) Create(f func(tInfo *SoAccount)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoAccount{}
	f(val)
	if val.Name == nil {
		val.Name = s.mainKey
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

	return nil
}

func (s *SoAccountWrap) getMainKeyBuf() ([]byte, error) {
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

func (s *SoAccountWrap) Md(f func(tInfo *SoAccount)) error {
	t := &SoAccount{}
	f(t)
	js, err := json.Marshal(t)
	if err != nil {
		return err
	}
	fMap := make(map[string]interface{})
	err = json.Unmarshal(js, &fMap)
	if err != nil {
		return err
	}

	mKeyName := "Name"
	mKeyField := ""
	for name, _ := range fMap {
		if ConvTableFieldToPbFormat(name) == mKeyName {
			mKeyField = name
			break
		}
	}
	if len(mKeyField) > 0 {
		delete(fMap, mKeyField)
	}

	if len(fMap) < 1 {
		return errors.New("can't' modify empty struct")
	}

	sa := s.getAccount()
	if sa == nil {
		return errors.New("fail to get table SoAccount")
	}

	refVal := reflect.ValueOf(*t)
	el := reflect.ValueOf(sa).Elem()

	//check unique
	err = s.handleFieldMd(FieldMdHandleTypeCheck, t, fMap)
	if err != nil {
		return err
	}

	//delete sort and unique key
	err = s.handleFieldMd(FieldMdHandleTypeDel, sa, fMap)
	if err != nil {
		return err
	}

	//update table
	for f, _ := range fMap {
		fName := ConvTableFieldToPbFormat(f)
		val := refVal.FieldByName(fName)
		if _, ok := s.mdFuncMap[fName]; ok {
			el.FieldByName(fName).Set(val)
		}
	}
	err = s.updateAccount(sa)
	if err != nil {
		return err
	}

	//insert sort and unique key
	err = s.handleFieldMd(FieldMdHandleTypeInsert, sa, fMap)
	if err != nil {
		return err
	}

	return err

}

func (s *SoAccountWrap) handleFieldMd(t FieldMdHandleType, so *SoAccount, fMap map[string]interface{}) error {
	if so == nil || fMap == nil {
		return errors.New("fail to modify empty table")
	}

	mdFuncMap := s.getMdFuncMap()
	if len(mdFuncMap) < 1 {
		return errors.New("there is not exsit md function to md field")
	}
	errStr := ""
	refVal := reflect.ValueOf(*so)
	for f, _ := range fMap {
		fName := ConvTableFieldToPbFormat(f)
		val := refVal.FieldByName(fName)
		if _, ok := mdFuncMap[fName]; ok {
			f := reflect.ValueOf(s.mdFuncMap[fName])
			p := []reflect.Value{val, reflect.ValueOf(true), reflect.ValueOf(false), reflect.ValueOf(false), reflect.ValueOf(so)}
			errStr = fmt.Sprintf("fail to modify exist value of %v", fName)
			if t == FieldMdHandleTypeDel {
				p = []reflect.Value{val, reflect.ValueOf(false), reflect.ValueOf(true), reflect.ValueOf(false), reflect.ValueOf(so)}
				errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", fName)
			} else if t == FieldMdHandleTypeInsert {
				p = []reflect.Value{val, reflect.ValueOf(false), reflect.ValueOf(false), reflect.ValueOf(true), reflect.ValueOf(so)}
				errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", fName)
			}
			res := f.Call(p)
			if !(res[0].Bool()) {
				return errors.New(errStr)
			}
		}
	}

	return nil
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoAccountWrap) delSortKeyCreatedTime(sa *SoAccount) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListAccountByCreatedTime{}
	if sa == nil {
		val.CreatedTime = s.GetCreatedTime()
		val.Name = s.mainKey

	} else {
		val.CreatedTime = sa.CreatedTime
		val.Name = sa.Name
	}

	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoAccountWrap) insertSortKeyCreatedTime(sa *SoAccount) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListAccountByCreatedTime{}
	val.Name = sa.Name
	val.CreatedTime = sa.CreatedTime
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

func (s *SoAccountWrap) delSortKeyBalance(sa *SoAccount) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListAccountByBalance{}
	if sa == nil {
		val.Balance = s.GetBalance()
		val.Name = s.mainKey

	} else {
		val.Balance = sa.Balance
		val.Name = sa.Name
	}

	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoAccountWrap) insertSortKeyBalance(sa *SoAccount) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListAccountByBalance{}
	val.Name = sa.Name
	val.Balance = sa.Balance
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

func (s *SoAccountWrap) delSortKeyVestingShares(sa *SoAccount) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListAccountByVestingShares{}
	if sa == nil {
		val.VestingShares = s.GetVestingShares()
		val.Name = s.mainKey

	} else {
		val.VestingShares = sa.VestingShares
		val.Name = sa.Name
	}

	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoAccountWrap) insertSortKeyVestingShares(sa *SoAccount) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListAccountByVestingShares{}
	val.Name = sa.Name
	val.VestingShares = sa.VestingShares
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

func (s *SoAccountWrap) delSortKeyBpVoteCount(sa *SoAccount) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListAccountByBpVoteCount{}
	if sa == nil {
		val.BpVoteCount = s.GetBpVoteCount()
		val.Name = s.mainKey

	} else {
		val.BpVoteCount = sa.BpVoteCount
		val.Name = sa.Name
	}

	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoAccountWrap) insertSortKeyBpVoteCount(sa *SoAccount) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListAccountByBpVoteCount{}
	val.Name = sa.Name
	val.BpVoteCount = sa.BpVoteCount
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

func (s *SoAccountWrap) delSortKeyPostCount(sa *SoAccount) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListAccountByPostCount{}
	if sa == nil {
		val.PostCount = s.GetPostCount()
		val.Name = s.mainKey

	} else {
		val.PostCount = sa.PostCount
		val.Name = sa.Name
	}

	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoAccountWrap) insertSortKeyPostCount(sa *SoAccount) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListAccountByPostCount{}
	val.Name = sa.Name
	val.PostCount = sa.PostCount
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

func (s *SoAccountWrap) delSortKeyCreatedTrxCount(sa *SoAccount) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListAccountByCreatedTrxCount{}
	if sa == nil {
		val.CreatedTrxCount = s.GetCreatedTrxCount()
		val.Name = s.mainKey

	} else {
		val.CreatedTrxCount = sa.CreatedTrxCount
		val.Name = sa.Name
	}

	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoAccountWrap) insertSortKeyCreatedTrxCount(sa *SoAccount) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListAccountByCreatedTrxCount{}
	val.Name = sa.Name
	val.CreatedTrxCount = sa.CreatedTrxCount
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

func (s *SoAccountWrap) delSortKeyNextPowerdownBlockNum(sa *SoAccount) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListAccountByNextPowerdownBlockNum{}
	if sa == nil {
		val.NextPowerdownBlockNum = s.GetNextPowerdownBlockNum()
		val.Name = s.mainKey

	} else {
		val.NextPowerdownBlockNum = sa.NextPowerdownBlockNum
		val.Name = sa.Name
	}

	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoAccountWrap) insertSortKeyNextPowerdownBlockNum(sa *SoAccount) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListAccountByNextPowerdownBlockNum{}
	val.Name = sa.Name
	val.NextPowerdownBlockNum = sa.NextPowerdownBlockNum
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

func (s *SoAccountWrap) delAllSortKeys(br bool, val *SoAccount) bool {
	if s.dba == nil {
		return false
	}
	res := true

	if !s.delSortKeyCreatedTime(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	if !s.delSortKeyBalance(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	if !s.delSortKeyVestingShares(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	if !s.delSortKeyBpVoteCount(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	if !s.delSortKeyPostCount(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	if !s.delSortKeyCreatedTrxCount(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	if !s.delSortKeyNextPowerdownBlockNum(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoAccountWrap) insertAllSortKeys(val *SoAccount) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoAccount fail ")
	}

	if !s.insertSortKeyCreatedTime(val) {
		return errors.New("insert sort Field CreatedTime fail while insert table ")
	}

	if !s.insertSortKeyBalance(val) {
		return errors.New("insert sort Field Balance fail while insert table ")
	}

	if !s.insertSortKeyVestingShares(val) {
		return errors.New("insert sort Field VestingShares fail while insert table ")
	}

	if !s.insertSortKeyBpVoteCount(val) {
		return errors.New("insert sort Field BpVoteCount fail while insert table ")
	}

	if !s.insertSortKeyPostCount(val) {
		return errors.New("insert sort Field PostCount fail while insert table ")
	}

	if !s.insertSortKeyCreatedTrxCount(val) {
		return errors.New("insert sort Field CreatedTrxCount fail while insert table ")
	}

	if !s.insertSortKeyNextPowerdownBlockNum(val) {
		return errors.New("insert sort Field NextPowerdownBlockNum fail while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoAccountWrap) RemoveAccount() bool {
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

func (s *SoAccountWrap) GetBalance() *prototype.Coin {
	res := true
	msg := &SoAccount{}
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
				return msg.Balance
			}
		}
	}
	if !res {
		return nil

	}
	return msg.Balance
}

func (s *SoAccountWrap) mdFieldBalance(p *prototype.Coin, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkBalanceIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldBalance(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldBalance(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldBalance(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyBalance(so) {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldBalance(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyBalance(so) {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkBalanceIsMetMdCondition(p *prototype.Coin) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) GetBpVoteCount() uint32 {
	res := true
	msg := &SoAccount{}
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
				return msg.BpVoteCount
			}
		}
	}
	if !res {
		var tmpValue uint32
		return tmpValue
	}
	return msg.BpVoteCount
}

func (s *SoAccountWrap) mdFieldBpVoteCount(p uint32, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkBpVoteCountIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldBpVoteCount(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldBpVoteCount(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldBpVoteCount(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyBpVoteCount(so) {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldBpVoteCount(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyBpVoteCount(so) {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkBpVoteCountIsMetMdCondition(p uint32) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) GetCreatedTime() *prototype.TimePointSec {
	res := true
	msg := &SoAccount{}
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
				return msg.CreatedTime
			}
		}
	}
	if !res {
		return nil

	}
	return msg.CreatedTime
}

func (s *SoAccountWrap) mdFieldCreatedTime(p *prototype.TimePointSec, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkCreatedTimeIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldCreatedTime(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldCreatedTime(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldCreatedTime(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyCreatedTime(so) {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldCreatedTime(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyCreatedTime(so) {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkCreatedTimeIsMetMdCondition(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) GetCreatedTrxCount() uint32 {
	res := true
	msg := &SoAccount{}
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
				return msg.CreatedTrxCount
			}
		}
	}
	if !res {
		var tmpValue uint32
		return tmpValue
	}
	return msg.CreatedTrxCount
}

func (s *SoAccountWrap) mdFieldCreatedTrxCount(p uint32, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkCreatedTrxCountIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldCreatedTrxCount(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldCreatedTrxCount(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldCreatedTrxCount(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyCreatedTrxCount(so) {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldCreatedTrxCount(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyCreatedTrxCount(so) {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkCreatedTrxCountIsMetMdCondition(p uint32) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) GetCreator() *prototype.AccountName {
	res := true
	msg := &SoAccount{}
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
				return msg.Creator
			}
		}
	}
	if !res {
		return nil

	}
	return msg.Creator
}

func (s *SoAccountWrap) mdFieldCreator(p *prototype.AccountName, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkCreatorIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldCreator(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldCreator(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldCreator(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldCreator(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkCreatorIsMetMdCondition(p *prototype.AccountName) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) GetEachPowerdownRate() *prototype.Vest {
	res := true
	msg := &SoAccount{}
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
				return msg.EachPowerdownRate
			}
		}
	}
	if !res {
		return nil

	}
	return msg.EachPowerdownRate
}

func (s *SoAccountWrap) mdFieldEachPowerdownRate(p *prototype.Vest, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkEachPowerdownRateIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldEachPowerdownRate(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldEachPowerdownRate(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldEachPowerdownRate(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldEachPowerdownRate(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkEachPowerdownRateIsMetMdCondition(p *prototype.Vest) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) GetHasPowerdown() *prototype.Vest {
	res := true
	msg := &SoAccount{}
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
				return msg.HasPowerdown
			}
		}
	}
	if !res {
		return nil

	}
	return msg.HasPowerdown
}

func (s *SoAccountWrap) mdFieldHasPowerdown(p *prototype.Vest, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkHasPowerdownIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldHasPowerdown(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldHasPowerdown(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldHasPowerdown(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldHasPowerdown(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkHasPowerdownIsMetMdCondition(p *prototype.Vest) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) GetLastOwnerUpdate() *prototype.TimePointSec {
	res := true
	msg := &SoAccount{}
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
				return msg.LastOwnerUpdate
			}
		}
	}
	if !res {
		return nil

	}
	return msg.LastOwnerUpdate
}

func (s *SoAccountWrap) mdFieldLastOwnerUpdate(p *prototype.TimePointSec, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkLastOwnerUpdateIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldLastOwnerUpdate(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldLastOwnerUpdate(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldLastOwnerUpdate(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldLastOwnerUpdate(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkLastOwnerUpdateIsMetMdCondition(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) GetLastPostTime() *prototype.TimePointSec {
	res := true
	msg := &SoAccount{}
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
				return msg.LastPostTime
			}
		}
	}
	if !res {
		return nil

	}
	return msg.LastPostTime
}

func (s *SoAccountWrap) mdFieldLastPostTime(p *prototype.TimePointSec, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkLastPostTimeIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldLastPostTime(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldLastPostTime(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldLastPostTime(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldLastPostTime(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkLastPostTimeIsMetMdCondition(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) GetLastStakeTime() *prototype.TimePointSec {
	res := true
	msg := &SoAccount{}
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
				return msg.LastStakeTime
			}
		}
	}
	if !res {
		return nil

	}
	return msg.LastStakeTime
}

func (s *SoAccountWrap) mdFieldLastStakeTime(p *prototype.TimePointSec, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkLastStakeTimeIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldLastStakeTime(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldLastStakeTime(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldLastStakeTime(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldLastStakeTime(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkLastStakeTimeIsMetMdCondition(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) GetLastVoteTime() *prototype.TimePointSec {
	res := true
	msg := &SoAccount{}
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
				return msg.LastVoteTime
			}
		}
	}
	if !res {
		return nil

	}
	return msg.LastVoteTime
}

func (s *SoAccountWrap) mdFieldLastVoteTime(p *prototype.TimePointSec, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkLastVoteTimeIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldLastVoteTime(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldLastVoteTime(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldLastVoteTime(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldLastVoteTime(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkLastVoteTimeIsMetMdCondition(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) GetName() *prototype.AccountName {
	res := true
	msg := &SoAccount{}
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
				return msg.Name
			}
		}
	}
	if !res {
		return nil

	}
	return msg.Name
}

func (s *SoAccountWrap) GetNextPowerdownBlockNum() uint64 {
	res := true
	msg := &SoAccount{}
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
				return msg.NextPowerdownBlockNum
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.NextPowerdownBlockNum
}

func (s *SoAccountWrap) mdFieldNextPowerdownBlockNum(p uint64, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkNextPowerdownBlockNumIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldNextPowerdownBlockNum(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldNextPowerdownBlockNum(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldNextPowerdownBlockNum(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyNextPowerdownBlockNum(so) {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldNextPowerdownBlockNum(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyNextPowerdownBlockNum(so) {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkNextPowerdownBlockNumIsMetMdCondition(p uint64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) GetOwner() *prototype.PublicKeyType {
	res := true
	msg := &SoAccount{}
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
				return msg.Owner
			}
		}
	}
	if !res {
		return nil

	}
	return msg.Owner
}

func (s *SoAccountWrap) mdFieldOwner(p *prototype.PublicKeyType, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkOwnerIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldOwner(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldOwner(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldOwner(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if !s.delUniKeyOwner(so) {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldOwner(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertUniKeyOwner(so) {
		return false
	}
	return true
}

func (s *SoAccountWrap) checkOwnerIsMetMdCondition(p *prototype.PublicKeyType) bool {
	if s.dba == nil {
		return false
	}

	//judge the unique value if is exist
	uniWrap := UniAccountOwnerWrap{}
	uniWrap.Dba = s.dba
	res := uniWrap.UniQueryOwner(p)

	if res != nil {
		//the unique value to be modified is already exist
		return false
	}

	return true
}

func (s *SoAccountWrap) GetPostCount() uint32 {
	res := true
	msg := &SoAccount{}
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
				return msg.PostCount
			}
		}
	}
	if !res {
		var tmpValue uint32
		return tmpValue
	}
	return msg.PostCount
}

func (s *SoAccountWrap) mdFieldPostCount(p uint32, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkPostCountIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldPostCount(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldPostCount(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldPostCount(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyPostCount(so) {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldPostCount(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyPostCount(so) {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkPostCountIsMetMdCondition(p uint32) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) GetStakeVesting() *prototype.Vest {
	res := true
	msg := &SoAccount{}
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
				return msg.StakeVesting
			}
		}
	}
	if !res {
		return nil

	}
	return msg.StakeVesting
}

func (s *SoAccountWrap) mdFieldStakeVesting(p *prototype.Vest, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkStakeVestingIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldStakeVesting(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldStakeVesting(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldStakeVesting(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldStakeVesting(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkStakeVestingIsMetMdCondition(p *prototype.Vest) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) GetStamina() uint64 {
	res := true
	msg := &SoAccount{}
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
				return msg.Stamina
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.Stamina
}

func (s *SoAccountWrap) mdFieldStamina(p uint64, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkStaminaIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldStamina(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldStamina(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldStamina(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldStamina(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkStaminaIsMetMdCondition(p uint64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) GetStaminaFree() uint64 {
	res := true
	msg := &SoAccount{}
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
				return msg.StaminaFree
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.StaminaFree
}

func (s *SoAccountWrap) mdFieldStaminaFree(p uint64, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkStaminaFreeIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldStaminaFree(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldStaminaFree(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldStaminaFree(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldStaminaFree(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkStaminaFreeIsMetMdCondition(p uint64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) GetStaminaFreeUseBlock() uint64 {
	res := true
	msg := &SoAccount{}
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
				return msg.StaminaFreeUseBlock
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.StaminaFreeUseBlock
}

func (s *SoAccountWrap) mdFieldStaminaFreeUseBlock(p uint64, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkStaminaFreeUseBlockIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldStaminaFreeUseBlock(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldStaminaFreeUseBlock(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldStaminaFreeUseBlock(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldStaminaFreeUseBlock(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkStaminaFreeUseBlockIsMetMdCondition(p uint64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) GetStaminaUseBlock() uint64 {
	res := true
	msg := &SoAccount{}
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
				return msg.StaminaUseBlock
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.StaminaUseBlock
}

func (s *SoAccountWrap) mdFieldStaminaUseBlock(p uint64, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkStaminaUseBlockIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldStaminaUseBlock(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldStaminaUseBlock(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldStaminaUseBlock(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldStaminaUseBlock(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkStaminaUseBlockIsMetMdCondition(p uint64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) GetToPowerdown() *prototype.Vest {
	res := true
	msg := &SoAccount{}
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
				return msg.ToPowerdown
			}
		}
	}
	if !res {
		return nil

	}
	return msg.ToPowerdown
}

func (s *SoAccountWrap) mdFieldToPowerdown(p *prototype.Vest, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkToPowerdownIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldToPowerdown(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldToPowerdown(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldToPowerdown(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldToPowerdown(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkToPowerdownIsMetMdCondition(p *prototype.Vest) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) GetVestingShares() *prototype.Vest {
	res := true
	msg := &SoAccount{}
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
				return msg.VestingShares
			}
		}
	}
	if !res {
		return nil

	}
	return msg.VestingShares
}

func (s *SoAccountWrap) mdFieldVestingShares(p *prototype.Vest, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkVestingSharesIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldVestingShares(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldVestingShares(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldVestingShares(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyVestingShares(so) {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldVestingShares(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyVestingShares(so) {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkVestingSharesIsMetMdCondition(p *prototype.Vest) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) GetVotePower() uint32 {
	res := true
	msg := &SoAccount{}
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
				return msg.VotePower
			}
		}
	}
	if !res {
		var tmpValue uint32
		return tmpValue
	}
	return msg.VotePower
}

func (s *SoAccountWrap) mdFieldVotePower(p uint32, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkVotePowerIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldVotePower(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldVotePower(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldVotePower(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldVotePower(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkVotePowerIsMetMdCondition(p uint32) bool {
	if s.dba == nil {
		return false
	}

	return true
}

////////////// SECTION List Keys ///////////////
type SAccountCreatedTimeWrap struct {
	Dba iservices.IDatabaseRW
}

func NewAccountCreatedTimeWrap(db iservices.IDatabaseRW) *SAccountCreatedTimeWrap {
	if db == nil {
		return nil
	}
	wrap := SAccountCreatedTimeWrap{Dba: db}
	return &wrap
}

func (s *SAccountCreatedTimeWrap) GetMainVal(val []byte) *prototype.AccountName {
	res := &SoListAccountByCreatedTime{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Name

}

func (s *SAccountCreatedTimeWrap) GetSubVal(val []byte) *prototype.TimePointSec {
	res := &SoListAccountByCreatedTime{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.CreatedTime

}

func (m *SoListAccountByCreatedTime) OpeEncode() ([]byte, error) {
	pre := AccountCreatedTimeTable
	sub := m.CreatedTime
	if sub == nil {
		return nil, errors.New("the pro CreatedTime is nil")
	}
	sub1 := m.Name
	if sub1 == nil {
		return nil, errors.New("the mainkey Name is nil")
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
func (s *SAccountCreatedTimeWrap) ForEachByOrder(start *prototype.TimePointSec, end *prototype.TimePointSec, lastMainKey *prototype.AccountName,
	lastSubVal *prototype.TimePointSec, f func(mVal *prototype.AccountName, sVal *prototype.TimePointSec, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := AccountCreatedTimeTable
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
func (s *SAccountCreatedTimeWrap) ForEachByRevOrder(start *prototype.TimePointSec, end *prototype.TimePointSec, lastMainKey *prototype.AccountName,
	lastSubVal *prototype.TimePointSec, f func(mVal *prototype.AccountName, sVal *prototype.TimePointSec, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := AccountCreatedTimeTable
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
type SAccountBalanceWrap struct {
	Dba iservices.IDatabaseRW
}

func NewAccountBalanceWrap(db iservices.IDatabaseRW) *SAccountBalanceWrap {
	if db == nil {
		return nil
	}
	wrap := SAccountBalanceWrap{Dba: db}
	return &wrap
}

func (s *SAccountBalanceWrap) GetMainVal(val []byte) *prototype.AccountName {
	res := &SoListAccountByBalance{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Name

}

func (s *SAccountBalanceWrap) GetSubVal(val []byte) *prototype.Coin {
	res := &SoListAccountByBalance{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.Balance

}

func (m *SoListAccountByBalance) OpeEncode() ([]byte, error) {
	pre := AccountBalanceTable
	sub := m.Balance
	if sub == nil {
		return nil, errors.New("the pro Balance is nil")
	}
	sub1 := m.Name
	if sub1 == nil {
		return nil, errors.New("the mainkey Name is nil")
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
func (s *SAccountBalanceWrap) ForEachByOrder(start *prototype.Coin, end *prototype.Coin, lastMainKey *prototype.AccountName,
	lastSubVal *prototype.Coin, f func(mVal *prototype.AccountName, sVal *prototype.Coin, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := AccountBalanceTable
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
func (s *SAccountBalanceWrap) ForEachByRevOrder(start *prototype.Coin, end *prototype.Coin, lastMainKey *prototype.AccountName,
	lastSubVal *prototype.Coin, f func(mVal *prototype.AccountName, sVal *prototype.Coin, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := AccountBalanceTable
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
type SAccountVestingSharesWrap struct {
	Dba iservices.IDatabaseRW
}

func NewAccountVestingSharesWrap(db iservices.IDatabaseRW) *SAccountVestingSharesWrap {
	if db == nil {
		return nil
	}
	wrap := SAccountVestingSharesWrap{Dba: db}
	return &wrap
}

func (s *SAccountVestingSharesWrap) GetMainVal(val []byte) *prototype.AccountName {
	res := &SoListAccountByVestingShares{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Name

}

func (s *SAccountVestingSharesWrap) GetSubVal(val []byte) *prototype.Vest {
	res := &SoListAccountByVestingShares{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.VestingShares

}

func (m *SoListAccountByVestingShares) OpeEncode() ([]byte, error) {
	pre := AccountVestingSharesTable
	sub := m.VestingShares
	if sub == nil {
		return nil, errors.New("the pro VestingShares is nil")
	}
	sub1 := m.Name
	if sub1 == nil {
		return nil, errors.New("the mainkey Name is nil")
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
func (s *SAccountVestingSharesWrap) ForEachByRevOrder(start *prototype.Vest, end *prototype.Vest, lastMainKey *prototype.AccountName,
	lastSubVal *prototype.Vest, f func(mVal *prototype.AccountName, sVal *prototype.Vest, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := AccountVestingSharesTable
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
type SAccountBpVoteCountWrap struct {
	Dba iservices.IDatabaseRW
}

func NewAccountBpVoteCountWrap(db iservices.IDatabaseRW) *SAccountBpVoteCountWrap {
	if db == nil {
		return nil
	}
	wrap := SAccountBpVoteCountWrap{Dba: db}
	return &wrap
}

func (s *SAccountBpVoteCountWrap) GetMainVal(val []byte) *prototype.AccountName {
	res := &SoListAccountByBpVoteCount{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Name

}

func (s *SAccountBpVoteCountWrap) GetSubVal(val []byte) *uint32 {
	res := &SoListAccountByBpVoteCount{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return &res.BpVoteCount

}

func (m *SoListAccountByBpVoteCount) OpeEncode() ([]byte, error) {
	pre := AccountBpVoteCountTable
	sub := m.BpVoteCount

	sub1 := m.Name
	if sub1 == nil {
		return nil, errors.New("the mainkey Name is nil")
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
func (s *SAccountBpVoteCountWrap) ForEachByOrder(start *uint32, end *uint32, lastMainKey *prototype.AccountName,
	lastSubVal *uint32, f func(mVal *prototype.AccountName, sVal *uint32, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := AccountBpVoteCountTable
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

////////////// SECTION List Keys ///////////////
type SAccountPostCountWrap struct {
	Dba iservices.IDatabaseRW
}

func NewAccountPostCountWrap(db iservices.IDatabaseRW) *SAccountPostCountWrap {
	if db == nil {
		return nil
	}
	wrap := SAccountPostCountWrap{Dba: db}
	return &wrap
}

func (s *SAccountPostCountWrap) GetMainVal(val []byte) *prototype.AccountName {
	res := &SoListAccountByPostCount{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Name

}

func (s *SAccountPostCountWrap) GetSubVal(val []byte) *uint32 {
	res := &SoListAccountByPostCount{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return &res.PostCount

}

func (m *SoListAccountByPostCount) OpeEncode() ([]byte, error) {
	pre := AccountPostCountTable
	sub := m.PostCount

	sub1 := m.Name
	if sub1 == nil {
		return nil, errors.New("the mainkey Name is nil")
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
func (s *SAccountPostCountWrap) ForEachByOrder(start *uint32, end *uint32, lastMainKey *prototype.AccountName,
	lastSubVal *uint32, f func(mVal *prototype.AccountName, sVal *uint32, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := AccountPostCountTable
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

////////////// SECTION List Keys ///////////////
type SAccountCreatedTrxCountWrap struct {
	Dba iservices.IDatabaseRW
}

func NewAccountCreatedTrxCountWrap(db iservices.IDatabaseRW) *SAccountCreatedTrxCountWrap {
	if db == nil {
		return nil
	}
	wrap := SAccountCreatedTrxCountWrap{Dba: db}
	return &wrap
}

func (s *SAccountCreatedTrxCountWrap) GetMainVal(val []byte) *prototype.AccountName {
	res := &SoListAccountByCreatedTrxCount{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Name

}

func (s *SAccountCreatedTrxCountWrap) GetSubVal(val []byte) *uint32 {
	res := &SoListAccountByCreatedTrxCount{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return &res.CreatedTrxCount

}

func (m *SoListAccountByCreatedTrxCount) OpeEncode() ([]byte, error) {
	pre := AccountCreatedTrxCountTable
	sub := m.CreatedTrxCount

	sub1 := m.Name
	if sub1 == nil {
		return nil, errors.New("the mainkey Name is nil")
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
func (s *SAccountCreatedTrxCountWrap) ForEachByOrder(start *uint32, end *uint32, lastMainKey *prototype.AccountName,
	lastSubVal *uint32, f func(mVal *prototype.AccountName, sVal *uint32, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := AccountCreatedTrxCountTable
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

////////////// SECTION List Keys ///////////////
type SAccountNextPowerdownBlockNumWrap struct {
	Dba iservices.IDatabaseRW
}

func NewAccountNextPowerdownBlockNumWrap(db iservices.IDatabaseRW) *SAccountNextPowerdownBlockNumWrap {
	if db == nil {
		return nil
	}
	wrap := SAccountNextPowerdownBlockNumWrap{Dba: db}
	return &wrap
}

func (s *SAccountNextPowerdownBlockNumWrap) GetMainVal(val []byte) *prototype.AccountName {
	res := &SoListAccountByNextPowerdownBlockNum{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Name

}

func (s *SAccountNextPowerdownBlockNumWrap) GetSubVal(val []byte) *uint64 {
	res := &SoListAccountByNextPowerdownBlockNum{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return &res.NextPowerdownBlockNum

}

func (m *SoListAccountByNextPowerdownBlockNum) OpeEncode() ([]byte, error) {
	pre := AccountNextPowerdownBlockNumTable
	sub := m.NextPowerdownBlockNum

	sub1 := m.Name
	if sub1 == nil {
		return nil, errors.New("the mainkey Name is nil")
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
func (s *SAccountNextPowerdownBlockNumWrap) ForEachByOrder(start *uint64, end *uint64, lastMainKey *prototype.AccountName,
	lastSubVal *uint64, f func(mVal *prototype.AccountName, sVal *uint64, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := AccountNextPowerdownBlockNumTable
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

func (s *SoAccountWrap) update(sa *SoAccount) bool {
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

func (s *SoAccountWrap) getAccount() *SoAccount {
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

	res := &SoAccount{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoAccountWrap) updateAccount(so *SoAccount) error {
	if s.dba == nil {
		return errors.New("update fail:the db is nil")
	}

	if so == nil {
		return errors.New("update fail: the SoAccount is nil")
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

func (s *SoAccountWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := AccountNameRow
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

func (s *SoAccountWrap) delAllUniKeys(br bool, val *SoAccount) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delUniKeyName(val) {
		if br {
			return false
		} else {
			res = false
		}
	}
	if !s.delUniKeyOwner(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoAccountWrap) delUniKeysWithNames(names map[string]string, val *SoAccount) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if len(names["Name"]) > 0 {
		if !s.delUniKeyName(val) {
			res = false
		}
	}
	if len(names["Owner"]) > 0 {
		if !s.delUniKeyOwner(val) {
			res = false
		}
	}

	return res
}

func (s *SoAccountWrap) insertAllUniKeys(val *SoAccount) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoAccount fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyName(val) {
		return sucFields, errors.New("insert unique Field Name fail while insert table ")
	}
	sucFields["Name"] = "Name"
	if !s.insertUniKeyOwner(val) {
		return sucFields, errors.New("insert unique Field Owner fail while insert table ")
	}
	sucFields["Owner"] = "Owner"

	return sucFields, nil
}

func (s *SoAccountWrap) delUniKeyName(sa *SoAccount) bool {
	if s.dba == nil {
		return false
	}
	pre := AccountNameUniTable
	kList := []interface{}{pre}
	if sa != nil {

		if sa.Name == nil {
			return false
		}

		sub := sa.Name
		kList = append(kList, sub)
	} else {
		sub := s.GetName()
		kList = append(kList, sub)

	}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoAccountWrap) insertUniKeyName(sa *SoAccount) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	pre := AccountNameUniTable
	sub := sa.Name
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
	val := SoUniqueAccountByName{}
	val.Name = sa.Name

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniAccountNameWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniAccountNameWrap(db iservices.IDatabaseRW) *UniAccountNameWrap {
	if db == nil {
		return nil
	}
	wrap := UniAccountNameWrap{Dba: db}
	return &wrap
}

func (s *UniAccountNameWrap) UniQueryName(start *prototype.AccountName) *SoAccountWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := AccountNameUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueAccountByName{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoAccountWrap(s.Dba, res.Name)

			return wrap
		}
	}
	return nil
}

func (s *SoAccountWrap) delUniKeyOwner(sa *SoAccount) bool {
	if s.dba == nil {
		return false
	}
	pre := AccountOwnerUniTable
	kList := []interface{}{pre}
	if sa != nil {

		if sa.Owner == nil {
			return false
		}

		sub := sa.Owner
		kList = append(kList, sub)
	} else {
		sub := s.GetOwner()
		kList = append(kList, sub)

	}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoAccountWrap) insertUniKeyOwner(sa *SoAccount) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	pre := AccountOwnerUniTable
	sub := sa.Owner
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
	val := SoUniqueAccountByOwner{}
	val.Name = sa.Name
	val.Owner = sa.Owner

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniAccountOwnerWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniAccountOwnerWrap(db iservices.IDatabaseRW) *UniAccountOwnerWrap {
	if db == nil {
		return nil
	}
	wrap := UniAccountOwnerWrap{Dba: db}
	return &wrap
}

func (s *UniAccountOwnerWrap) UniQueryOwner(start *prototype.PublicKeyType) *SoAccountWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := AccountOwnerUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueAccountByOwner{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoAccountWrap(s.Dba, res.Name)

			return wrap
		}
	}
	return nil
}

func (s *SoAccountWrap) getMdFuncMap() map[string]interface{} {
	if s.mdFuncMap != nil && len(s.mdFuncMap) > 0 {
		return s.mdFuncMap
	}
	m := map[string]interface{}{}

	m["Balance"] = s.mdFieldBalance

	m["BpVoteCount"] = s.mdFieldBpVoteCount

	m["CreatedTime"] = s.mdFieldCreatedTime

	m["CreatedTrxCount"] = s.mdFieldCreatedTrxCount

	m["Creator"] = s.mdFieldCreator

	m["EachPowerdownRate"] = s.mdFieldEachPowerdownRate

	m["HasPowerdown"] = s.mdFieldHasPowerdown

	m["LastOwnerUpdate"] = s.mdFieldLastOwnerUpdate

	m["LastPostTime"] = s.mdFieldLastPostTime

	m["LastStakeTime"] = s.mdFieldLastStakeTime

	m["LastVoteTime"] = s.mdFieldLastVoteTime

	m["NextPowerdownBlockNum"] = s.mdFieldNextPowerdownBlockNum

	m["Owner"] = s.mdFieldOwner

	m["PostCount"] = s.mdFieldPostCount

	m["StakeVesting"] = s.mdFieldStakeVesting

	m["Stamina"] = s.mdFieldStamina

	m["StaminaFree"] = s.mdFieldStaminaFree

	m["StaminaFreeUseBlock"] = s.mdFieldStaminaFreeUseBlock

	m["StaminaUseBlock"] = s.mdFieldStaminaUseBlock

	m["ToPowerdown"] = s.mdFieldToPowerdown

	m["VestingShares"] = s.mdFieldVestingShares

	m["VotePower"] = s.mdFieldVotePower

	if len(m) > 0 {
		s.mdFuncMap = m
	}
	return m
}
