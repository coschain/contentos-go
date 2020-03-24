package table

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/coschain/contentos-go/common/encoding/kope"
	"github.com/coschain/contentos-go/iservices"
	prototype "github.com/coschain/contentos-go/prototype"
	proto "github.com/golang/protobuf/proto"
)

////////////// SECTION Prefix Mark ///////////////
var (
	VestDelegationFromAccountTable   uint32 = 3132651932
	VestDelegationToAccountTable     uint32 = 3713114966
	VestDelegationMaturityBlockTable uint32 = 2354530309
	VestDelegationDeliveryBlockTable uint32 = 4292113245
	VestDelegationDeliveringTable    uint32 = 2725281102
	VestDelegationIdUniTable         uint32 = 4102010180

	VestDelegationIdRow uint32 = 2762572716
)

////////////// SECTION Wrap Define ///////////////
type SoVestDelegationWrap struct {
	dba         iservices.IDatabaseRW
	mainKey     *uint64
	watcherFlag *VestDelegationWatcherFlag
	mKeyFlag    int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf     []byte //the buffer after the main key is encoded with prefix
	mBuf        []byte //the value after the main key is encoded
	mdFuncMap   map[string]interface{}
}

func NewSoVestDelegationWrap(dba iservices.IDatabaseRW, key *uint64) *SoVestDelegationWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoVestDelegationWrap{dba, key, nil, -1, nil, nil, nil}
	result.initWatcherFlag()
	return result
}

func (s *SoVestDelegationWrap) CheckExist() bool {
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

func (s *SoVestDelegationWrap) MustExist(errMsgs ...interface{}) *SoVestDelegationWrap {
	if !s.CheckExist() {
		panic(bindErrorInfo(fmt.Sprintf("SoVestDelegationWrap.MustExist: %v not found", s.mainKey), errMsgs...))
	}
	return s
}

func (s *SoVestDelegationWrap) MustNotExist(errMsgs ...interface{}) *SoVestDelegationWrap {
	if s.CheckExist() {
		panic(bindErrorInfo(fmt.Sprintf("SoVestDelegationWrap.MustNotExist: %v already exists", s.mainKey), errMsgs...))
	}
	return s
}

func (s *SoVestDelegationWrap) initWatcherFlag() {
	if s.watcherFlag == nil {
		s.watcherFlag = new(VestDelegationWatcherFlag)
		*(s.watcherFlag) = VestDelegationWatcherFlagOfDb(s.dba.ServiceId())
	}
}

func (s *SoVestDelegationWrap) create(f func(tInfo *SoVestDelegation)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoVestDelegation{}
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

	// call watchers
	s.initWatcherFlag()
	if s.watcherFlag.AnyWatcher {
		ReportTableRecordInsert(s.dba.ServiceId(), s.dba.BranchId(), s.mainKey, val)
	}

	return nil
}

func (s *SoVestDelegationWrap) Create(f func(tInfo *SoVestDelegation), errArgs ...interface{}) *SoVestDelegationWrap {
	err := s.create(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Errorf("SoVestDelegationWrap.Create failed: %s", err.Error()), errArgs...))
	}
	return s
}

func (s *SoVestDelegationWrap) getMainKeyBuf() ([]byte, error) {
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

func (s *SoVestDelegationWrap) modify(f func(tInfo *SoVestDelegation)) error {
	if !s.CheckExist() {
		return errors.New("the SoVestDelegation table does not exist. Please create a table first")
	}
	oriTable := s.getVestDelegation()
	if oriTable == nil {
		return errors.New("fail to get origin table SoVestDelegation")
	}

	curTable := s.getVestDelegation()
	if curTable == nil {
		return errors.New("fail to create current table SoVestDelegation")
	}
	f(curTable)

	//the main key is not support modify
	if !reflect.DeepEqual(curTable.Id, oriTable.Id) {
		return errors.New("primary key does not support modification")
	}

	s.initWatcherFlag()
	modifiedFields, hasWatcher, err := s.getModifiedFields(oriTable, curTable)
	if err != nil {
		return err
	}

	if modifiedFields == nil || len(modifiedFields) < 1 {
		return nil
	}

	//check whether modify sort and unique field to nil
	err = s.checkSortAndUniFieldValidity(curTable, modifiedFields)
	if err != nil {
		return err
	}

	//check unique
	err = s.handleFieldMd(FieldMdHandleTypeCheck, curTable, modifiedFields)
	if err != nil {
		return err
	}

	//delete sort and unique key
	err = s.handleFieldMd(FieldMdHandleTypeDel, oriTable, modifiedFields)
	if err != nil {
		return err
	}

	//update table
	err = s.updateVestDelegation(curTable)
	if err != nil {
		return err
	}

	//insert sort and unique key
	err = s.handleFieldMd(FieldMdHandleTypeInsert, curTable, modifiedFields)
	if err != nil {
		return err
	}

	// call watchers
	if hasWatcher {
		ReportTableRecordUpdate(s.dba.ServiceId(), s.dba.BranchId(), s.mainKey, oriTable, curTable, modifiedFields)
	}

	return nil

}

func (s *SoVestDelegationWrap) Modify(f func(tInfo *SoVestDelegation), errArgs ...interface{}) *SoVestDelegationWrap {
	err := s.modify(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoVestDelegationWrap.Modify failed: %s", err.Error()), errArgs...))
	}
	return s
}

func (s *SoVestDelegationWrap) SetAmount(p *prototype.Vest, errArgs ...interface{}) *SoVestDelegationWrap {
	err := s.modify(func(r *SoVestDelegation) {
		r.Amount = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoVestDelegationWrap.SetAmount( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoVestDelegationWrap) SetCreatedBlock(p uint64, errArgs ...interface{}) *SoVestDelegationWrap {
	err := s.modify(func(r *SoVestDelegation) {
		r.CreatedBlock = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoVestDelegationWrap.SetCreatedBlock( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoVestDelegationWrap) SetDelivering(p bool, errArgs ...interface{}) *SoVestDelegationWrap {
	err := s.modify(func(r *SoVestDelegation) {
		r.Delivering = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoVestDelegationWrap.SetDelivering( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoVestDelegationWrap) SetDeliveryBlock(p uint64, errArgs ...interface{}) *SoVestDelegationWrap {
	err := s.modify(func(r *SoVestDelegation) {
		r.DeliveryBlock = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoVestDelegationWrap.SetDeliveryBlock( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoVestDelegationWrap) SetFromAccount(p *prototype.AccountName, errArgs ...interface{}) *SoVestDelegationWrap {
	err := s.modify(func(r *SoVestDelegation) {
		r.FromAccount = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoVestDelegationWrap.SetFromAccount( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoVestDelegationWrap) SetMaturityBlock(p uint64, errArgs ...interface{}) *SoVestDelegationWrap {
	err := s.modify(func(r *SoVestDelegation) {
		r.MaturityBlock = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoVestDelegationWrap.SetMaturityBlock( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoVestDelegationWrap) SetToAccount(p *prototype.AccountName, errArgs ...interface{}) *SoVestDelegationWrap {
	err := s.modify(func(r *SoVestDelegation) {
		r.ToAccount = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoVestDelegationWrap.SetToAccount( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoVestDelegationWrap) checkSortAndUniFieldValidity(curTable *SoVestDelegation, fields map[string]bool) error {
	if curTable != nil && fields != nil && len(fields) > 0 {

		if fields["FromAccount"] && curTable.FromAccount == nil {
			return errors.New("sort field FromAccount can't be modified to nil")
		}

		if fields["ToAccount"] && curTable.ToAccount == nil {
			return errors.New("sort field ToAccount can't be modified to nil")
		}

	}
	return nil
}

//Get all the modified fields in the table
func (s *SoVestDelegationWrap) getModifiedFields(oriTable *SoVestDelegation, curTable *SoVestDelegation) (map[string]bool, bool, error) {
	if oriTable == nil {
		return nil, false, errors.New("table info is nil, can't get modified fields")
	}
	hasWatcher := false
	fields := make(map[string]bool)

	if !reflect.DeepEqual(oriTable.Amount, curTable.Amount) {
		fields["Amount"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasAmountWatcher
	}

	if !reflect.DeepEqual(oriTable.CreatedBlock, curTable.CreatedBlock) {
		fields["CreatedBlock"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasCreatedBlockWatcher
	}

	if !reflect.DeepEqual(oriTable.Delivering, curTable.Delivering) {
		fields["Delivering"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasDeliveringWatcher
	}

	if !reflect.DeepEqual(oriTable.DeliveryBlock, curTable.DeliveryBlock) {
		fields["DeliveryBlock"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasDeliveryBlockWatcher
	}

	if !reflect.DeepEqual(oriTable.FromAccount, curTable.FromAccount) {
		fields["FromAccount"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasFromAccountWatcher
	}

	if !reflect.DeepEqual(oriTable.MaturityBlock, curTable.MaturityBlock) {
		fields["MaturityBlock"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasMaturityBlockWatcher
	}

	if !reflect.DeepEqual(oriTable.ToAccount, curTable.ToAccount) {
		fields["ToAccount"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasToAccountWatcher
	}

	hasWatcher = hasWatcher || s.watcherFlag.WholeWatcher
	return fields, hasWatcher, nil
}

func (s *SoVestDelegationWrap) handleFieldMd(t FieldMdHandleType, so *SoVestDelegation, fields map[string]bool) error {
	if so == nil {
		return errors.New("fail to modify empty table")
	}

	//there is no field need to modify
	if fields == nil || len(fields) < 1 {
		return nil
	}

	errStr := ""

	if fields["Amount"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldAmount(so.Amount, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "Amount")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldAmount(so.Amount, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "Amount")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldAmount(so.Amount, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "Amount")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["CreatedBlock"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldCreatedBlock(so.CreatedBlock, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "CreatedBlock")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldCreatedBlock(so.CreatedBlock, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "CreatedBlock")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldCreatedBlock(so.CreatedBlock, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "CreatedBlock")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["Delivering"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldDelivering(so.Delivering, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "Delivering")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldDelivering(so.Delivering, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "Delivering")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldDelivering(so.Delivering, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "Delivering")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["DeliveryBlock"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldDeliveryBlock(so.DeliveryBlock, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "DeliveryBlock")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldDeliveryBlock(so.DeliveryBlock, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "DeliveryBlock")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldDeliveryBlock(so.DeliveryBlock, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "DeliveryBlock")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["FromAccount"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldFromAccount(so.FromAccount, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "FromAccount")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldFromAccount(so.FromAccount, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "FromAccount")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldFromAccount(so.FromAccount, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "FromAccount")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["MaturityBlock"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldMaturityBlock(so.MaturityBlock, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "MaturityBlock")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldMaturityBlock(so.MaturityBlock, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "MaturityBlock")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldMaturityBlock(so.MaturityBlock, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "MaturityBlock")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["ToAccount"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldToAccount(so.ToAccount, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "ToAccount")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldToAccount(so.ToAccount, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "ToAccount")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldToAccount(so.ToAccount, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "ToAccount")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	return nil
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoVestDelegationWrap) delSortKeyFromAccount(sa *SoVestDelegation) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListVestDelegationByFromAccount{}
	if sa == nil {
		val.FromAccount = s.GetFromAccount()
		val.Id = *s.mainKey
	} else {
		val.FromAccount = sa.FromAccount
		val.Id = sa.Id
	}
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoVestDelegationWrap) insertSortKeyFromAccount(sa *SoVestDelegation) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListVestDelegationByFromAccount{}
	val.Id = sa.Id
	val.FromAccount = sa.FromAccount
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

func (s *SoVestDelegationWrap) delSortKeyToAccount(sa *SoVestDelegation) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListVestDelegationByToAccount{}
	if sa == nil {
		val.ToAccount = s.GetToAccount()
		val.Id = *s.mainKey
	} else {
		val.ToAccount = sa.ToAccount
		val.Id = sa.Id
	}
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoVestDelegationWrap) insertSortKeyToAccount(sa *SoVestDelegation) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListVestDelegationByToAccount{}
	val.Id = sa.Id
	val.ToAccount = sa.ToAccount
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

func (s *SoVestDelegationWrap) delSortKeyMaturityBlock(sa *SoVestDelegation) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListVestDelegationByMaturityBlock{}
	if sa == nil {
		val.MaturityBlock = s.GetMaturityBlock()
		val.Id = *s.mainKey
	} else {
		val.MaturityBlock = sa.MaturityBlock
		val.Id = sa.Id
	}
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoVestDelegationWrap) insertSortKeyMaturityBlock(sa *SoVestDelegation) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListVestDelegationByMaturityBlock{}
	val.Id = sa.Id
	val.MaturityBlock = sa.MaturityBlock
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

func (s *SoVestDelegationWrap) delSortKeyDeliveryBlock(sa *SoVestDelegation) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListVestDelegationByDeliveryBlock{}
	if sa == nil {
		val.DeliveryBlock = s.GetDeliveryBlock()
		val.Id = *s.mainKey
	} else {
		val.DeliveryBlock = sa.DeliveryBlock
		val.Id = sa.Id
	}
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoVestDelegationWrap) insertSortKeyDeliveryBlock(sa *SoVestDelegation) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListVestDelegationByDeliveryBlock{}
	val.Id = sa.Id
	val.DeliveryBlock = sa.DeliveryBlock
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

func (s *SoVestDelegationWrap) delSortKeyDelivering(sa *SoVestDelegation) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListVestDelegationByDelivering{}
	if sa == nil {
		val.Delivering = s.GetDelivering()
		val.Id = *s.mainKey
	} else {
		val.Delivering = sa.Delivering
		val.Id = sa.Id
	}
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoVestDelegationWrap) insertSortKeyDelivering(sa *SoVestDelegation) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListVestDelegationByDelivering{}
	val.Id = sa.Id
	val.Delivering = sa.Delivering
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

func (s *SoVestDelegationWrap) delAllSortKeys(br bool, val *SoVestDelegation) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delSortKeyFromAccount(val) {
		if br {
			return false
		} else {
			res = false
		}
	}
	if !s.delSortKeyToAccount(val) {
		if br {
			return false
		} else {
			res = false
		}
	}
	if !s.delSortKeyMaturityBlock(val) {
		if br {
			return false
		} else {
			res = false
		}
	}
	if !s.delSortKeyDeliveryBlock(val) {
		if br {
			return false
		} else {
			res = false
		}
	}
	if !s.delSortKeyDelivering(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoVestDelegationWrap) insertAllSortKeys(val *SoVestDelegation) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoVestDelegation fail ")
	}
	if !s.insertSortKeyFromAccount(val) {
		return errors.New("insert sort Field FromAccount fail while insert table ")
	}
	if !s.insertSortKeyToAccount(val) {
		return errors.New("insert sort Field ToAccount fail while insert table ")
	}
	if !s.insertSortKeyMaturityBlock(val) {
		return errors.New("insert sort Field MaturityBlock fail while insert table ")
	}
	if !s.insertSortKeyDeliveryBlock(val) {
		return errors.New("insert sort Field DeliveryBlock fail while insert table ")
	}
	if !s.insertSortKeyDelivering(val) {
		return errors.New("insert sort Field Delivering fail while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoVestDelegationWrap) removeVestDelegation() error {
	if s.dba == nil {
		return errors.New("database is nil")
	}

	s.initWatcherFlag()

	var oldVal *SoVestDelegation
	if s.watcherFlag.AnyWatcher {
		oldVal = s.getVestDelegation()
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
		if s.watcherFlag.AnyWatcher && oldVal != nil {
			ReportTableRecordDelete(s.dba.ServiceId(), s.dba.BranchId(), s.mainKey, oldVal)
		}
		return nil
	} else {
		return fmt.Errorf("database.Delete failed: %s", err.Error())
	}
}

func (s *SoVestDelegationWrap) RemoveVestDelegation(errMsgs ...interface{}) *SoVestDelegationWrap {
	err := s.removeVestDelegation()
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoVestDelegationWrap.RemoveVestDelegation failed: %s", err.Error()), errMsgs...))
	}
	return s
}

////////////// SECTION Members Get/Modify ///////////////

func (s *SoVestDelegationWrap) GetAmount() *prototype.Vest {
	res := true
	msg := &SoVestDelegation{}
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
				return msg.Amount
			}
		}
	}
	if !res {
		return nil

	}
	return msg.Amount
}

func (s *SoVestDelegationWrap) mdFieldAmount(p *prototype.Vest, isCheck bool, isDel bool, isInsert bool,
	so *SoVestDelegation) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkAmountIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldAmount(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldAmount(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoVestDelegationWrap) delFieldAmount(so *SoVestDelegation) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoVestDelegationWrap) insertFieldAmount(so *SoVestDelegation) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoVestDelegationWrap) checkAmountIsMetMdCondition(p *prototype.Vest) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoVestDelegationWrap) GetCreatedBlock() uint64 {
	res := true
	msg := &SoVestDelegation{}
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
				return msg.CreatedBlock
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.CreatedBlock
}

func (s *SoVestDelegationWrap) mdFieldCreatedBlock(p uint64, isCheck bool, isDel bool, isInsert bool,
	so *SoVestDelegation) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkCreatedBlockIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldCreatedBlock(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldCreatedBlock(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoVestDelegationWrap) delFieldCreatedBlock(so *SoVestDelegation) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoVestDelegationWrap) insertFieldCreatedBlock(so *SoVestDelegation) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoVestDelegationWrap) checkCreatedBlockIsMetMdCondition(p uint64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoVestDelegationWrap) GetDelivering() bool {
	res := true
	msg := &SoVestDelegation{}
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
				return msg.Delivering
			}
		}
	}
	if !res {
		var tmpValue bool
		return tmpValue
	}
	return msg.Delivering
}

func (s *SoVestDelegationWrap) mdFieldDelivering(p bool, isCheck bool, isDel bool, isInsert bool,
	so *SoVestDelegation) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkDeliveringIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldDelivering(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldDelivering(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoVestDelegationWrap) delFieldDelivering(so *SoVestDelegation) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyDelivering(so) {
		return false
	}

	return true
}

func (s *SoVestDelegationWrap) insertFieldDelivering(so *SoVestDelegation) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyDelivering(so) {
		return false
	}

	return true
}

func (s *SoVestDelegationWrap) checkDeliveringIsMetMdCondition(p bool) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoVestDelegationWrap) GetDeliveryBlock() uint64 {
	res := true
	msg := &SoVestDelegation{}
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
				return msg.DeliveryBlock
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.DeliveryBlock
}

func (s *SoVestDelegationWrap) mdFieldDeliveryBlock(p uint64, isCheck bool, isDel bool, isInsert bool,
	so *SoVestDelegation) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkDeliveryBlockIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldDeliveryBlock(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldDeliveryBlock(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoVestDelegationWrap) delFieldDeliveryBlock(so *SoVestDelegation) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyDeliveryBlock(so) {
		return false
	}

	return true
}

func (s *SoVestDelegationWrap) insertFieldDeliveryBlock(so *SoVestDelegation) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyDeliveryBlock(so) {
		return false
	}

	return true
}

func (s *SoVestDelegationWrap) checkDeliveryBlockIsMetMdCondition(p uint64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoVestDelegationWrap) GetFromAccount() *prototype.AccountName {
	res := true
	msg := &SoVestDelegation{}
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
				return msg.FromAccount
			}
		}
	}
	if !res {
		return nil

	}
	return msg.FromAccount
}

func (s *SoVestDelegationWrap) mdFieldFromAccount(p *prototype.AccountName, isCheck bool, isDel bool, isInsert bool,
	so *SoVestDelegation) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkFromAccountIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldFromAccount(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldFromAccount(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoVestDelegationWrap) delFieldFromAccount(so *SoVestDelegation) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyFromAccount(so) {
		return false
	}

	return true
}

func (s *SoVestDelegationWrap) insertFieldFromAccount(so *SoVestDelegation) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyFromAccount(so) {
		return false
	}

	return true
}

func (s *SoVestDelegationWrap) checkFromAccountIsMetMdCondition(p *prototype.AccountName) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoVestDelegationWrap) GetId() uint64 {
	res := true
	msg := &SoVestDelegation{}
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
		var tmpValue uint64
		return tmpValue
	}
	return msg.Id
}

func (s *SoVestDelegationWrap) GetMaturityBlock() uint64 {
	res := true
	msg := &SoVestDelegation{}
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
				return msg.MaturityBlock
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.MaturityBlock
}

func (s *SoVestDelegationWrap) mdFieldMaturityBlock(p uint64, isCheck bool, isDel bool, isInsert bool,
	so *SoVestDelegation) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkMaturityBlockIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldMaturityBlock(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldMaturityBlock(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoVestDelegationWrap) delFieldMaturityBlock(so *SoVestDelegation) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyMaturityBlock(so) {
		return false
	}

	return true
}

func (s *SoVestDelegationWrap) insertFieldMaturityBlock(so *SoVestDelegation) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyMaturityBlock(so) {
		return false
	}

	return true
}

func (s *SoVestDelegationWrap) checkMaturityBlockIsMetMdCondition(p uint64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoVestDelegationWrap) GetToAccount() *prototype.AccountName {
	res := true
	msg := &SoVestDelegation{}
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
				return msg.ToAccount
			}
		}
	}
	if !res {
		return nil

	}
	return msg.ToAccount
}

func (s *SoVestDelegationWrap) mdFieldToAccount(p *prototype.AccountName, isCheck bool, isDel bool, isInsert bool,
	so *SoVestDelegation) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkToAccountIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldToAccount(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldToAccount(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoVestDelegationWrap) delFieldToAccount(so *SoVestDelegation) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyToAccount(so) {
		return false
	}

	return true
}

func (s *SoVestDelegationWrap) insertFieldToAccount(so *SoVestDelegation) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyToAccount(so) {
		return false
	}

	return true
}

func (s *SoVestDelegationWrap) checkToAccountIsMetMdCondition(p *prototype.AccountName) bool {
	if s.dba == nil {
		return false
	}

	return true
}

////////////// SECTION List Keys ///////////////
type SVestDelegationFromAccountWrap struct {
	Dba iservices.IDatabaseRW
}

func NewVestDelegationFromAccountWrap(db iservices.IDatabaseRW) *SVestDelegationFromAccountWrap {
	if db == nil {
		return nil
	}
	wrap := SVestDelegationFromAccountWrap{Dba: db}
	return &wrap
}

func (s *SVestDelegationFromAccountWrap) GetMainVal(val []byte) *uint64 {
	res := &SoListVestDelegationByFromAccount{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return &res.Id

}

func (s *SVestDelegationFromAccountWrap) GetSubVal(val []byte) *prototype.AccountName {
	res := &SoListVestDelegationByFromAccount{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.FromAccount

}

func (m *SoListVestDelegationByFromAccount) OpeEncode() ([]byte, error) {
	pre := VestDelegationFromAccountTable
	sub := m.FromAccount
	if sub == nil {
		return nil, errors.New("the pro FromAccount is nil")
	}
	sub1 := m.Id

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
func (s *SVestDelegationFromAccountWrap) ForEachByOrder(start *prototype.AccountName, end *prototype.AccountName, lastMainKey *uint64,
	lastSubVal *prototype.AccountName, f func(mVal *uint64, sVal *prototype.AccountName, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := VestDelegationFromAccountTable
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
func (s *SVestDelegationFromAccountWrap) ForEachByRevOrder(start *prototype.AccountName, end *prototype.AccountName, lastMainKey *uint64,
	lastSubVal *prototype.AccountName, f func(mVal *uint64, sVal *prototype.AccountName, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := VestDelegationFromAccountTable
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
type SVestDelegationToAccountWrap struct {
	Dba iservices.IDatabaseRW
}

func NewVestDelegationToAccountWrap(db iservices.IDatabaseRW) *SVestDelegationToAccountWrap {
	if db == nil {
		return nil
	}
	wrap := SVestDelegationToAccountWrap{Dba: db}
	return &wrap
}

func (s *SVestDelegationToAccountWrap) GetMainVal(val []byte) *uint64 {
	res := &SoListVestDelegationByToAccount{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return &res.Id

}

func (s *SVestDelegationToAccountWrap) GetSubVal(val []byte) *prototype.AccountName {
	res := &SoListVestDelegationByToAccount{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.ToAccount

}

func (m *SoListVestDelegationByToAccount) OpeEncode() ([]byte, error) {
	pre := VestDelegationToAccountTable
	sub := m.ToAccount
	if sub == nil {
		return nil, errors.New("the pro ToAccount is nil")
	}
	sub1 := m.Id

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
func (s *SVestDelegationToAccountWrap) ForEachByOrder(start *prototype.AccountName, end *prototype.AccountName, lastMainKey *uint64,
	lastSubVal *prototype.AccountName, f func(mVal *uint64, sVal *prototype.AccountName, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := VestDelegationToAccountTable
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
func (s *SVestDelegationToAccountWrap) ForEachByRevOrder(start *prototype.AccountName, end *prototype.AccountName, lastMainKey *uint64,
	lastSubVal *prototype.AccountName, f func(mVal *uint64, sVal *prototype.AccountName, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := VestDelegationToAccountTable
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
type SVestDelegationMaturityBlockWrap struct {
	Dba iservices.IDatabaseRW
}

func NewVestDelegationMaturityBlockWrap(db iservices.IDatabaseRW) *SVestDelegationMaturityBlockWrap {
	if db == nil {
		return nil
	}
	wrap := SVestDelegationMaturityBlockWrap{Dba: db}
	return &wrap
}

func (s *SVestDelegationMaturityBlockWrap) GetMainVal(val []byte) *uint64 {
	res := &SoListVestDelegationByMaturityBlock{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return &res.Id

}

func (s *SVestDelegationMaturityBlockWrap) GetSubVal(val []byte) *uint64 {
	res := &SoListVestDelegationByMaturityBlock{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return &res.MaturityBlock

}

func (m *SoListVestDelegationByMaturityBlock) OpeEncode() ([]byte, error) {
	pre := VestDelegationMaturityBlockTable
	sub := m.MaturityBlock

	sub1 := m.Id

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
func (s *SVestDelegationMaturityBlockWrap) ForEachByOrder(start *uint64, end *uint64, lastMainKey *uint64,
	lastSubVal *uint64, f func(mVal *uint64, sVal *uint64, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := VestDelegationMaturityBlockTable
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
func (s *SVestDelegationMaturityBlockWrap) ForEachByRevOrder(start *uint64, end *uint64, lastMainKey *uint64,
	lastSubVal *uint64, f func(mVal *uint64, sVal *uint64, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := VestDelegationMaturityBlockTable
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
type SVestDelegationDeliveryBlockWrap struct {
	Dba iservices.IDatabaseRW
}

func NewVestDelegationDeliveryBlockWrap(db iservices.IDatabaseRW) *SVestDelegationDeliveryBlockWrap {
	if db == nil {
		return nil
	}
	wrap := SVestDelegationDeliveryBlockWrap{Dba: db}
	return &wrap
}

func (s *SVestDelegationDeliveryBlockWrap) GetMainVal(val []byte) *uint64 {
	res := &SoListVestDelegationByDeliveryBlock{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return &res.Id

}

func (s *SVestDelegationDeliveryBlockWrap) GetSubVal(val []byte) *uint64 {
	res := &SoListVestDelegationByDeliveryBlock{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return &res.DeliveryBlock

}

func (m *SoListVestDelegationByDeliveryBlock) OpeEncode() ([]byte, error) {
	pre := VestDelegationDeliveryBlockTable
	sub := m.DeliveryBlock

	sub1 := m.Id

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
func (s *SVestDelegationDeliveryBlockWrap) ForEachByOrder(start *uint64, end *uint64, lastMainKey *uint64,
	lastSubVal *uint64, f func(mVal *uint64, sVal *uint64, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := VestDelegationDeliveryBlockTable
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
func (s *SVestDelegationDeliveryBlockWrap) ForEachByRevOrder(start *uint64, end *uint64, lastMainKey *uint64,
	lastSubVal *uint64, f func(mVal *uint64, sVal *uint64, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := VestDelegationDeliveryBlockTable
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
type SVestDelegationDeliveringWrap struct {
	Dba iservices.IDatabaseRW
}

func NewVestDelegationDeliveringWrap(db iservices.IDatabaseRW) *SVestDelegationDeliveringWrap {
	if db == nil {
		return nil
	}
	wrap := SVestDelegationDeliveringWrap{Dba: db}
	return &wrap
}

func (s *SVestDelegationDeliveringWrap) GetMainVal(val []byte) *uint64 {
	res := &SoListVestDelegationByDelivering{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return &res.Id

}

func (s *SVestDelegationDeliveringWrap) GetSubVal(val []byte) *bool {
	res := &SoListVestDelegationByDelivering{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return &res.Delivering

}

func (m *SoListVestDelegationByDelivering) OpeEncode() ([]byte, error) {
	pre := VestDelegationDeliveringTable
	sub := m.Delivering

	sub1 := m.Id

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
func (s *SVestDelegationDeliveringWrap) ForEachByOrder(start *bool, end *bool, lastMainKey *uint64,
	lastSubVal *bool, f func(mVal *uint64, sVal *bool, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := VestDelegationDeliveringTable
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
func (s *SVestDelegationDeliveringWrap) ForEachByRevOrder(start *bool, end *bool, lastMainKey *uint64,
	lastSubVal *bool, f func(mVal *uint64, sVal *bool, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := VestDelegationDeliveringTable
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

func (s *SoVestDelegationWrap) update(sa *SoVestDelegation) bool {
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

func (s *SoVestDelegationWrap) getVestDelegation() *SoVestDelegation {
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

	res := &SoVestDelegation{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoVestDelegationWrap) updateVestDelegation(so *SoVestDelegation) error {
	if s.dba == nil {
		return errors.New("update fail:the db is nil")
	}

	if so == nil {
		return errors.New("update fail: the SoVestDelegation is nil")
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

func (s *SoVestDelegationWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := VestDelegationIdRow
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

func (s *SoVestDelegationWrap) delAllUniKeys(br bool, val *SoVestDelegation) bool {
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

func (s *SoVestDelegationWrap) delUniKeysWithNames(names map[string]string, val *SoVestDelegation) bool {
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

func (s *SoVestDelegationWrap) insertAllUniKeys(val *SoVestDelegation) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoVestDelegation fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyId(val) {
		return sucFields, errors.New("insert unique Field Id fail while insert table ")
	}
	sucFields["Id"] = "Id"

	return sucFields, nil
}

func (s *SoVestDelegationWrap) delUniKeyId(sa *SoVestDelegation) bool {
	if s.dba == nil {
		return false
	}
	pre := VestDelegationIdUniTable
	kList := []interface{}{pre}
	if sa != nil {

		sub := sa.Id
		kList = append(kList, sub)
	} else {
		sub := s.GetId()

		kList = append(kList, sub)

	}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoVestDelegationWrap) insertUniKeyId(sa *SoVestDelegation) bool {
	if s.dba == nil || sa == nil {
		return false
	}

	pre := VestDelegationIdUniTable
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
	val := SoUniqueVestDelegationById{}
	val.Id = sa.Id

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniVestDelegationIdWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniVestDelegationIdWrap(db iservices.IDatabaseRW) *UniVestDelegationIdWrap {
	if db == nil {
		return nil
	}
	wrap := UniVestDelegationIdWrap{Dba: db}
	return &wrap
}

func (s *UniVestDelegationIdWrap) UniQueryId(start *uint64) *SoVestDelegationWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := VestDelegationIdUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueVestDelegationById{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoVestDelegationWrap(s.Dba, &res.Id)
			return wrap
		}
	}
	return nil
}

////////////// SECTION Watchers ///////////////

type VestDelegationWatcherFlag struct {
	HasAmountWatcher bool

	HasCreatedBlockWatcher bool

	HasDeliveringWatcher bool

	HasDeliveryBlockWatcher bool

	HasFromAccountWatcher bool

	HasMaturityBlockWatcher bool

	HasToAccountWatcher bool

	WholeWatcher bool
	AnyWatcher   bool
}

var (
	VestDelegationTable = &TableInfo{
		Name:    "VestDelegation",
		Primary: "Id",
		Record:  reflect.TypeOf((*SoVestDelegation)(nil)).Elem(),
	}
	VestDelegationWatcherFlags     = make(map[uint32]VestDelegationWatcherFlag)
	VestDelegationWatcherFlagsLock sync.RWMutex
)

func VestDelegationWatcherFlagOfDb(dbSvcId uint32) VestDelegationWatcherFlag {
	VestDelegationWatcherFlagsLock.RLock()
	defer VestDelegationWatcherFlagsLock.RUnlock()
	return VestDelegationWatcherFlags[dbSvcId]
}

func VestDelegationRecordWatcherChanged(dbSvcId uint32) {
	var flag VestDelegationWatcherFlag
	flag.WholeWatcher = HasTableRecordWatcher(dbSvcId, VestDelegationTable.Record, "")
	flag.AnyWatcher = flag.WholeWatcher

	flag.HasAmountWatcher = HasTableRecordWatcher(dbSvcId, VestDelegationTable.Record, "Amount")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasAmountWatcher

	flag.HasCreatedBlockWatcher = HasTableRecordWatcher(dbSvcId, VestDelegationTable.Record, "CreatedBlock")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasCreatedBlockWatcher

	flag.HasDeliveringWatcher = HasTableRecordWatcher(dbSvcId, VestDelegationTable.Record, "Delivering")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasDeliveringWatcher

	flag.HasDeliveryBlockWatcher = HasTableRecordWatcher(dbSvcId, VestDelegationTable.Record, "DeliveryBlock")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasDeliveryBlockWatcher

	flag.HasFromAccountWatcher = HasTableRecordWatcher(dbSvcId, VestDelegationTable.Record, "FromAccount")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasFromAccountWatcher

	flag.HasMaturityBlockWatcher = HasTableRecordWatcher(dbSvcId, VestDelegationTable.Record, "MaturityBlock")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasMaturityBlockWatcher

	flag.HasToAccountWatcher = HasTableRecordWatcher(dbSvcId, VestDelegationTable.Record, "ToAccount")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasToAccountWatcher

	VestDelegationWatcherFlagsLock.Lock()
	VestDelegationWatcherFlags[dbSvcId] = flag
	VestDelegationWatcherFlagsLock.Unlock()
}

////////////// SECTION Json query ///////////////

func VestDelegationQuery(db iservices.IDatabaseRW, keyJson string) (valueJson string, err error) {
	k := new(uint64)
	d := json.NewDecoder(bytes.NewReader([]byte(keyJson)))
	d.UseNumber()
	if err = d.Decode(k); err != nil {
		return
	}
	if v := NewSoVestDelegationWrap(db, k).getVestDelegation(); v == nil {
		err = errors.New("not found")
	} else {
		var jbytes []byte
		if jbytes, err = json.Marshal(v); err == nil {
			valueJson = string(jbytes)
		}
	}
	return
}

func init() {
	RegisterTableWatcherChangedCallback(VestDelegationTable.Record, VestDelegationRecordWatcherChanged)
	RegisterTableJsonQuery("VestDelegation", VestDelegationQuery)
}
