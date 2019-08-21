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
	ContractCreatedTimeTable uint32 = 1292005739
	ContractApplyCountTable  uint32 = 2694332342
	ContractIdUniTable       uint32 = 4175408872

	ContractIdRow uint32 = 1374288427
)

////////////// SECTION Wrap Define ///////////////
type SoContractWrap struct {
	dba       iservices.IDatabaseRW
	mainKey   *prototype.ContractId
	mKeyFlag  int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf   []byte //the buffer after the main key is encoded with prefix
	mBuf      []byte //the value after the main key is encoded
	mdFuncMap map[string]interface{}
}

func NewSoContractWrap(dba iservices.IDatabaseRW, key *prototype.ContractId) *SoContractWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoContractWrap{dba, key, -1, nil, nil, nil}
	return result
}

func (s *SoContractWrap) CheckExist() bool {
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

func (s *SoContractWrap) MustExist(errMsgs ...interface{}) *SoContractWrap {
	if !s.CheckExist() {
		panic(bindErrorInfo(fmt.Sprintf("SoContractWrap.MustExist: %v not found", s.mainKey), errMsgs...))
	}
	return s
}

func (s *SoContractWrap) MustNotExist(errMsgs ...interface{}) *SoContractWrap {
	if s.CheckExist() {
		panic(bindErrorInfo(fmt.Sprintf("SoContractWrap.MustNotExist: %v already exists", s.mainKey), errMsgs...))
	}
	return s
}

func (s *SoContractWrap) create(f func(tInfo *SoContract)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoContract{}
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

	// call watchers
	if ContractHasAnyWatcher {
		ReportTableRecordInsert(s.mainKey, val)
	}

	return nil
}

func (s *SoContractWrap) Create(f func(tInfo *SoContract), errArgs ...interface{}) *SoContractWrap {
	err := s.create(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Errorf("SoContractWrap.Create failed: %s", err.Error()), errArgs...))
	}
	return s
}

func (s *SoContractWrap) getMainKeyBuf() ([]byte, error) {
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

func (s *SoContractWrap) modify(f func(tInfo *SoContract)) error {
	if !s.CheckExist() {
		return errors.New("the SoContract table does not exist. Please create a table first")
	}
	oriTable := s.getContract()
	if oriTable == nil {
		return errors.New("fail to get origin table SoContract")
	}

	curTable := s.getContract()
	if curTable == nil {
		return errors.New("fail to create current table SoContract")
	}
	f(curTable)

	//the main key is not support modify
	if !reflect.DeepEqual(curTable.Id, oriTable.Id) {
		return errors.New("primary key does not support modification")
	}

	fieldSli, hasWatcher, err := s.getModifiedFields(oriTable, curTable)
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
	err = s.updateContract(curTable)
	if err != nil {
		return err
	}

	//insert sort and unique key
	err = s.handleFieldMd(FieldMdHandleTypeInsert, curTable, fieldSli)
	if err != nil {
		return err
	}

	// call watchers
	if hasWatcher {
		ReportTableRecordUpdate(s.mainKey, oriTable, curTable)
	}

	return nil

}

func (s *SoContractWrap) Modify(f func(tInfo *SoContract), errArgs ...interface{}) *SoContractWrap {
	err := s.modify(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoContractWrap.Modify failed: %s", err.Error()), errArgs...))
	}
	return s
}

func (s *SoContractWrap) SetAbi(p string, errArgs ...interface{}) *SoContractWrap {
	err := s.modify(func(r *SoContract) {
		r.Abi = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoContractWrap.SetAbi( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoContractWrap) SetApplyCount(p uint32, errArgs ...interface{}) *SoContractWrap {
	err := s.modify(func(r *SoContract) {
		r.ApplyCount = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoContractWrap.SetApplyCount( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoContractWrap) SetBalance(p *prototype.Coin, errArgs ...interface{}) *SoContractWrap {
	err := s.modify(func(r *SoContract) {
		r.Balance = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoContractWrap.SetBalance( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoContractWrap) SetCode(p []byte, errArgs ...interface{}) *SoContractWrap {
	err := s.modify(func(r *SoContract) {
		r.Code = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoContractWrap.SetCode( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoContractWrap) SetCreatedTime(p *prototype.TimePointSec, errArgs ...interface{}) *SoContractWrap {
	err := s.modify(func(r *SoContract) {
		r.CreatedTime = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoContractWrap.SetCreatedTime( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoContractWrap) SetDescribe(p string, errArgs ...interface{}) *SoContractWrap {
	err := s.modify(func(r *SoContract) {
		r.Describe = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoContractWrap.SetDescribe( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoContractWrap) SetHash(p *prototype.Sha256, errArgs ...interface{}) *SoContractWrap {
	err := s.modify(func(r *SoContract) {
		r.Hash = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoContractWrap.SetHash( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoContractWrap) SetUpgradeable(p bool, errArgs ...interface{}) *SoContractWrap {
	err := s.modify(func(r *SoContract) {
		r.Upgradeable = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoContractWrap.SetUpgradeable( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoContractWrap) SetUrl(p string, errArgs ...interface{}) *SoContractWrap {
	err := s.modify(func(r *SoContract) {
		r.Url = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoContractWrap.SetUrl( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoContractWrap) checkSortAndUniFieldValidity(curTable *SoContract, fields map[string]bool) error {
	if curTable != nil && fields != nil && len(fields) > 0 {

		if fields["CreatedTime"] && curTable.CreatedTime == nil {
			return errors.New("sort field CreatedTime can't be modified to nil")
		}

	}
	return nil
}

//Get all the modified fields in the table
func (s *SoContractWrap) getModifiedFields(oriTable *SoContract, curTable *SoContract) (map[string]bool, bool, error) {
	if oriTable == nil {
		return nil, false, errors.New("table info is nil, can't get modified fields")
	}
	hasWatcher := false
	fields := make(map[string]bool)

	if !reflect.DeepEqual(oriTable.Abi, curTable.Abi) {
		fields["Abi"] = true
		hasWatcher = hasWatcher || ContractHasAbiWatcher
	}

	if !reflect.DeepEqual(oriTable.ApplyCount, curTable.ApplyCount) {
		fields["ApplyCount"] = true
		hasWatcher = hasWatcher || ContractHasApplyCountWatcher
	}

	if !reflect.DeepEqual(oriTable.Balance, curTable.Balance) {
		fields["Balance"] = true
		hasWatcher = hasWatcher || ContractHasBalanceWatcher
	}

	if !reflect.DeepEqual(oriTable.Code, curTable.Code) {
		fields["Code"] = true
		hasWatcher = hasWatcher || ContractHasCodeWatcher
	}

	if !reflect.DeepEqual(oriTable.CreatedTime, curTable.CreatedTime) {
		fields["CreatedTime"] = true
		hasWatcher = hasWatcher || ContractHasCreatedTimeWatcher
	}

	if !reflect.DeepEqual(oriTable.Describe, curTable.Describe) {
		fields["Describe"] = true
		hasWatcher = hasWatcher || ContractHasDescribeWatcher
	}

	if !reflect.DeepEqual(oriTable.Hash, curTable.Hash) {
		fields["Hash"] = true
		hasWatcher = hasWatcher || ContractHasHashWatcher
	}

	if !reflect.DeepEqual(oriTable.Upgradeable, curTable.Upgradeable) {
		fields["Upgradeable"] = true
		hasWatcher = hasWatcher || ContractHasUpgradeableWatcher
	}

	if !reflect.DeepEqual(oriTable.Url, curTable.Url) {
		fields["Url"] = true
		hasWatcher = hasWatcher || ContractHasUrlWatcher
	}

	hasWatcher = hasWatcher || ContractHasWholeWatcher
	return fields, hasWatcher, nil
}

func (s *SoContractWrap) handleFieldMd(t FieldMdHandleType, so *SoContract, fields map[string]bool) error {
	if so == nil {
		return errors.New("fail to modify empty table")
	}

	//there is no field need to modify
	if fields == nil || len(fields) < 1 {
		return nil
	}

	errStr := ""

	if fields["Abi"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldAbi(so.Abi, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "Abi")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldAbi(so.Abi, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "Abi")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldAbi(so.Abi, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "Abi")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["ApplyCount"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldApplyCount(so.ApplyCount, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "ApplyCount")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldApplyCount(so.ApplyCount, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "ApplyCount")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldApplyCount(so.ApplyCount, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "ApplyCount")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["Balance"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldBalance(so.Balance, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "Balance")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldBalance(so.Balance, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "Balance")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldBalance(so.Balance, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "Balance")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["Code"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldCode(so.Code, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "Code")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldCode(so.Code, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "Code")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldCode(so.Code, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "Code")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["CreatedTime"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldCreatedTime(so.CreatedTime, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "CreatedTime")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldCreatedTime(so.CreatedTime, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "CreatedTime")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldCreatedTime(so.CreatedTime, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "CreatedTime")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["Describe"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldDescribe(so.Describe, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "Describe")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldDescribe(so.Describe, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "Describe")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldDescribe(so.Describe, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "Describe")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["Hash"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldHash(so.Hash, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "Hash")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldHash(so.Hash, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "Hash")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldHash(so.Hash, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "Hash")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["Upgradeable"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldUpgradeable(so.Upgradeable, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "Upgradeable")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldUpgradeable(so.Upgradeable, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "Upgradeable")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldUpgradeable(so.Upgradeable, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "Upgradeable")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["Url"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldUrl(so.Url, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "Url")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldUrl(so.Url, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "Url")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldUrl(so.Url, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "Url")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	return nil
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoContractWrap) delSortKeyCreatedTime(sa *SoContract) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListContractByCreatedTime{}
	if sa == nil {
		val.CreatedTime = s.GetCreatedTime()
		val.Id = s.mainKey

	} else {
		val.CreatedTime = sa.CreatedTime
		val.Id = sa.Id
	}
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoContractWrap) insertSortKeyCreatedTime(sa *SoContract) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListContractByCreatedTime{}
	val.Id = sa.Id
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

func (s *SoContractWrap) delSortKeyApplyCount(sa *SoContract) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListContractByApplyCount{}
	if sa == nil {
		val.ApplyCount = s.GetApplyCount()
		val.Id = s.mainKey

	} else {
		val.ApplyCount = sa.ApplyCount
		val.Id = sa.Id
	}
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoContractWrap) insertSortKeyApplyCount(sa *SoContract) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListContractByApplyCount{}
	val.Id = sa.Id
	val.ApplyCount = sa.ApplyCount
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

func (s *SoContractWrap) delAllSortKeys(br bool, val *SoContract) bool {
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
	if !s.delSortKeyApplyCount(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoContractWrap) insertAllSortKeys(val *SoContract) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoContract fail ")
	}
	if !s.insertSortKeyCreatedTime(val) {
		return errors.New("insert sort Field CreatedTime fail while insert table ")
	}
	if !s.insertSortKeyApplyCount(val) {
		return errors.New("insert sort Field ApplyCount fail while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoContractWrap) removeContract() error {
	if s.dba == nil {
		return errors.New("database is nil")
	}

	var oldVal *SoContract
	if ContractHasAnyWatcher {
		oldVal = s.getContract()
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

		// call watchers
		if ContractHasAnyWatcher && oldVal != nil {
			ReportTableRecordDelete(s.mainKey, oldVal)
		}
		return nil
	} else {
		return fmt.Errorf("database.Delete failed: %s", err.Error())
	}
}

func (s *SoContractWrap) RemoveContract(errMsgs ...interface{}) *SoContractWrap {
	err := s.removeContract()
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoContractWrap.RemoveContract failed: %s", err.Error()), errMsgs...))
	}
	return s
}

////////////// SECTION Members Get/Modify ///////////////

func (s *SoContractWrap) GetAbi() string {
	res := true
	msg := &SoContract{}
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
				return msg.Abi
			}
		}
	}
	if !res {
		var tmpValue string
		return tmpValue
	}
	return msg.Abi
}

func (s *SoContractWrap) mdFieldAbi(p string, isCheck bool, isDel bool, isInsert bool,
	so *SoContract) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkAbiIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldAbi(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldAbi(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoContractWrap) delFieldAbi(so *SoContract) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoContractWrap) insertFieldAbi(so *SoContract) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoContractWrap) checkAbiIsMetMdCondition(p string) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoContractWrap) GetApplyCount() uint32 {
	res := true
	msg := &SoContract{}
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
				return msg.ApplyCount
			}
		}
	}
	if !res {
		var tmpValue uint32
		return tmpValue
	}
	return msg.ApplyCount
}

func (s *SoContractWrap) mdFieldApplyCount(p uint32, isCheck bool, isDel bool, isInsert bool,
	so *SoContract) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkApplyCountIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldApplyCount(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldApplyCount(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoContractWrap) delFieldApplyCount(so *SoContract) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyApplyCount(so) {
		return false
	}

	return true
}

func (s *SoContractWrap) insertFieldApplyCount(so *SoContract) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyApplyCount(so) {
		return false
	}

	return true
}

func (s *SoContractWrap) checkApplyCountIsMetMdCondition(p uint32) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoContractWrap) GetBalance() *prototype.Coin {
	res := true
	msg := &SoContract{}
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

func (s *SoContractWrap) mdFieldBalance(p *prototype.Coin, isCheck bool, isDel bool, isInsert bool,
	so *SoContract) bool {
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

func (s *SoContractWrap) delFieldBalance(so *SoContract) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoContractWrap) insertFieldBalance(so *SoContract) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoContractWrap) checkBalanceIsMetMdCondition(p *prototype.Coin) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoContractWrap) GetCode() []byte {
	res := true
	msg := &SoContract{}
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
				return msg.Code
			}
		}
	}
	if !res {
		var tmpValue []byte
		return tmpValue
	}
	return msg.Code
}

func (s *SoContractWrap) mdFieldCode(p []byte, isCheck bool, isDel bool, isInsert bool,
	so *SoContract) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkCodeIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldCode(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldCode(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoContractWrap) delFieldCode(so *SoContract) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoContractWrap) insertFieldCode(so *SoContract) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoContractWrap) checkCodeIsMetMdCondition(p []byte) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoContractWrap) GetCreatedTime() *prototype.TimePointSec {
	res := true
	msg := &SoContract{}
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

func (s *SoContractWrap) mdFieldCreatedTime(p *prototype.TimePointSec, isCheck bool, isDel bool, isInsert bool,
	so *SoContract) bool {
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

func (s *SoContractWrap) delFieldCreatedTime(so *SoContract) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyCreatedTime(so) {
		return false
	}

	return true
}

func (s *SoContractWrap) insertFieldCreatedTime(so *SoContract) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyCreatedTime(so) {
		return false
	}

	return true
}

func (s *SoContractWrap) checkCreatedTimeIsMetMdCondition(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoContractWrap) GetDescribe() string {
	res := true
	msg := &SoContract{}
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
				return msg.Describe
			}
		}
	}
	if !res {
		var tmpValue string
		return tmpValue
	}
	return msg.Describe
}

func (s *SoContractWrap) mdFieldDescribe(p string, isCheck bool, isDel bool, isInsert bool,
	so *SoContract) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkDescribeIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldDescribe(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldDescribe(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoContractWrap) delFieldDescribe(so *SoContract) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoContractWrap) insertFieldDescribe(so *SoContract) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoContractWrap) checkDescribeIsMetMdCondition(p string) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoContractWrap) GetHash() *prototype.Sha256 {
	res := true
	msg := &SoContract{}
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
				return msg.Hash
			}
		}
	}
	if !res {
		return nil

	}
	return msg.Hash
}

func (s *SoContractWrap) mdFieldHash(p *prototype.Sha256, isCheck bool, isDel bool, isInsert bool,
	so *SoContract) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkHashIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldHash(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldHash(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoContractWrap) delFieldHash(so *SoContract) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoContractWrap) insertFieldHash(so *SoContract) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoContractWrap) checkHashIsMetMdCondition(p *prototype.Sha256) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoContractWrap) GetId() *prototype.ContractId {
	res := true
	msg := &SoContract{}
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

func (s *SoContractWrap) GetUpgradeable() bool {
	res := true
	msg := &SoContract{}
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
				return msg.Upgradeable
			}
		}
	}
	if !res {
		var tmpValue bool
		return tmpValue
	}
	return msg.Upgradeable
}

func (s *SoContractWrap) mdFieldUpgradeable(p bool, isCheck bool, isDel bool, isInsert bool,
	so *SoContract) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkUpgradeableIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldUpgradeable(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldUpgradeable(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoContractWrap) delFieldUpgradeable(so *SoContract) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoContractWrap) insertFieldUpgradeable(so *SoContract) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoContractWrap) checkUpgradeableIsMetMdCondition(p bool) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoContractWrap) GetUrl() string {
	res := true
	msg := &SoContract{}
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
				return msg.Url
			}
		}
	}
	if !res {
		var tmpValue string
		return tmpValue
	}
	return msg.Url
}

func (s *SoContractWrap) mdFieldUrl(p string, isCheck bool, isDel bool, isInsert bool,
	so *SoContract) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkUrlIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldUrl(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldUrl(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoContractWrap) delFieldUrl(so *SoContract) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoContractWrap) insertFieldUrl(so *SoContract) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoContractWrap) checkUrlIsMetMdCondition(p string) bool {
	if s.dba == nil {
		return false
	}

	return true
}

////////////// SECTION List Keys ///////////////
type SContractCreatedTimeWrap struct {
	Dba iservices.IDatabaseRW
}

func NewContractCreatedTimeWrap(db iservices.IDatabaseRW) *SContractCreatedTimeWrap {
	if db == nil {
		return nil
	}
	wrap := SContractCreatedTimeWrap{Dba: db}
	return &wrap
}

func (s *SContractCreatedTimeWrap) GetMainVal(val []byte) *prototype.ContractId {
	res := &SoListContractByCreatedTime{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Id

}

func (s *SContractCreatedTimeWrap) GetSubVal(val []byte) *prototype.TimePointSec {
	res := &SoListContractByCreatedTime{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.CreatedTime

}

func (m *SoListContractByCreatedTime) OpeEncode() ([]byte, error) {
	pre := ContractCreatedTimeTable
	sub := m.CreatedTime
	if sub == nil {
		return nil, errors.New("the pro CreatedTime is nil")
	}
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
func (s *SContractCreatedTimeWrap) ForEachByOrder(start *prototype.TimePointSec, end *prototype.TimePointSec, lastMainKey *prototype.ContractId,
	lastSubVal *prototype.TimePointSec, f func(mVal *prototype.ContractId, sVal *prototype.TimePointSec, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := ContractCreatedTimeTable
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
func (s *SContractCreatedTimeWrap) ForEachByRevOrder(start *prototype.TimePointSec, end *prototype.TimePointSec, lastMainKey *prototype.ContractId,
	lastSubVal *prototype.TimePointSec, f func(mVal *prototype.ContractId, sVal *prototype.TimePointSec, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := ContractCreatedTimeTable
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
type SContractApplyCountWrap struct {
	Dba iservices.IDatabaseRW
}

func NewContractApplyCountWrap(db iservices.IDatabaseRW) *SContractApplyCountWrap {
	if db == nil {
		return nil
	}
	wrap := SContractApplyCountWrap{Dba: db}
	return &wrap
}

func (s *SContractApplyCountWrap) GetMainVal(val []byte) *prototype.ContractId {
	res := &SoListContractByApplyCount{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Id

}

func (s *SContractApplyCountWrap) GetSubVal(val []byte) *uint32 {
	res := &SoListContractByApplyCount{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return &res.ApplyCount

}

func (m *SoListContractByApplyCount) OpeEncode() ([]byte, error) {
	pre := ContractApplyCountTable
	sub := m.ApplyCount

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
func (s *SContractApplyCountWrap) ForEachByOrder(start *uint32, end *uint32, lastMainKey *prototype.ContractId,
	lastSubVal *uint32, f func(mVal *prototype.ContractId, sVal *uint32, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := ContractApplyCountTable
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

func (s *SoContractWrap) update(sa *SoContract) bool {
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

func (s *SoContractWrap) getContract() *SoContract {
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

	res := &SoContract{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoContractWrap) updateContract(so *SoContract) error {
	if s.dba == nil {
		return errors.New("update fail:the db is nil")
	}

	if so == nil {
		return errors.New("update fail: the SoContract is nil")
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

func (s *SoContractWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := ContractIdRow
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

func (s *SoContractWrap) delAllUniKeys(br bool, val *SoContract) bool {
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

func (s *SoContractWrap) delUniKeysWithNames(names map[string]string, val *SoContract) bool {
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

func (s *SoContractWrap) insertAllUniKeys(val *SoContract) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoContract fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyId(val) {
		return sucFields, errors.New("insert unique Field Id fail while insert table ")
	}
	sucFields["Id"] = "Id"

	return sucFields, nil
}

func (s *SoContractWrap) delUniKeyId(sa *SoContract) bool {
	if s.dba == nil {
		return false
	}
	pre := ContractIdUniTable
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

func (s *SoContractWrap) insertUniKeyId(sa *SoContract) bool {
	if s.dba == nil || sa == nil {
		return false
	}

	pre := ContractIdUniTable
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
	val := SoUniqueContractById{}
	val.Id = sa.Id

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniContractIdWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniContractIdWrap(db iservices.IDatabaseRW) *UniContractIdWrap {
	if db == nil {
		return nil
	}
	wrap := UniContractIdWrap{Dba: db}
	return &wrap
}

func (s *UniContractIdWrap) UniQueryId(start *prototype.ContractId) *SoContractWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := ContractIdUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueContractById{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoContractWrap(s.Dba, res.Id)

			return wrap
		}
	}
	return nil
}

////////////// SECTION Watchers ///////////////
var (
	ContractRecordType = reflect.TypeOf((*SoContract)(nil)).Elem() // table record type

	ContractHasAbiWatcher bool // any watcher on member Abi?

	ContractHasApplyCountWatcher bool // any watcher on member ApplyCount?

	ContractHasBalanceWatcher bool // any watcher on member Balance?

	ContractHasCodeWatcher bool // any watcher on member Code?

	ContractHasCreatedTimeWatcher bool // any watcher on member CreatedTime?

	ContractHasDescribeWatcher bool // any watcher on member Describe?

	ContractHasHashWatcher bool // any watcher on member Hash?

	ContractHasUpgradeableWatcher bool // any watcher on member Upgradeable?

	ContractHasUrlWatcher bool // any watcher on member Url?

	ContractHasWholeWatcher bool // any watcher on the whole record?
	ContractHasAnyWatcher   bool // any watcher?
)

func ContractRecordWatcherChanged() {
	ContractHasWholeWatcher = HasTableRecordWatcher(ContractRecordType, "")
	ContractHasAnyWatcher = ContractHasWholeWatcher

	ContractHasAbiWatcher = HasTableRecordWatcher(ContractRecordType, "Abi")
	ContractHasAnyWatcher = ContractHasAnyWatcher || ContractHasAbiWatcher

	ContractHasApplyCountWatcher = HasTableRecordWatcher(ContractRecordType, "ApplyCount")
	ContractHasAnyWatcher = ContractHasAnyWatcher || ContractHasApplyCountWatcher

	ContractHasBalanceWatcher = HasTableRecordWatcher(ContractRecordType, "Balance")
	ContractHasAnyWatcher = ContractHasAnyWatcher || ContractHasBalanceWatcher

	ContractHasCodeWatcher = HasTableRecordWatcher(ContractRecordType, "Code")
	ContractHasAnyWatcher = ContractHasAnyWatcher || ContractHasCodeWatcher

	ContractHasCreatedTimeWatcher = HasTableRecordWatcher(ContractRecordType, "CreatedTime")
	ContractHasAnyWatcher = ContractHasAnyWatcher || ContractHasCreatedTimeWatcher

	ContractHasDescribeWatcher = HasTableRecordWatcher(ContractRecordType, "Describe")
	ContractHasAnyWatcher = ContractHasAnyWatcher || ContractHasDescribeWatcher

	ContractHasHashWatcher = HasTableRecordWatcher(ContractRecordType, "Hash")
	ContractHasAnyWatcher = ContractHasAnyWatcher || ContractHasHashWatcher

	ContractHasUpgradeableWatcher = HasTableRecordWatcher(ContractRecordType, "Upgradeable")
	ContractHasAnyWatcher = ContractHasAnyWatcher || ContractHasUpgradeableWatcher

	ContractHasUrlWatcher = HasTableRecordWatcher(ContractRecordType, "Url")
	ContractHasAnyWatcher = ContractHasAnyWatcher || ContractHasUrlWatcher

}

func init() {
	RegisterTableWatcherChangedCallback(ContractRecordType, ContractRecordWatcherChanged)
}
