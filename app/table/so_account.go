package table

import (
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
	AccountCreatedTimeTable           uint32 = 2128286283
	AccountBalanceTable               uint32 = 4012029019
	AccountVestTable                  uint32 = 2512254821
	AccountBpVoteCountTable           uint32 = 2264397557
	AccountPostCountTable             uint32 = 1518203339
	AccountCreatedTrxCountTable       uint32 = 2604810499
	AccountNextPowerdownBlockNumTable uint32 = 1928824877
	AccountNameUniTable               uint32 = 2528390520
	AccountPubKeyUniTable             uint32 = 598545409

	AccountNameRow uint32 = 3130128817
)

////////////// SECTION Wrap Define ///////////////
type SoAccountWrap struct {
	dba         iservices.IDatabaseRW
	mainKey     *prototype.AccountName
	watcherFlag *AccountWatcherFlag
	mKeyFlag    int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf     []byte //the buffer after the main key is encoded with prefix
	mBuf        []byte //the value after the main key is encoded
	mdFuncMap   map[string]interface{}
}

func NewSoAccountWrap(dba iservices.IDatabaseRW, key *prototype.AccountName) *SoAccountWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoAccountWrap{dba, key, nil, -1, nil, nil, nil}
	result.initWatcherFlag()
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

func (s *SoAccountWrap) MustExist(errMsgs ...interface{}) *SoAccountWrap {
	if !s.CheckExist() {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.MustExist: %v not found", s.mainKey), errMsgs...))
	}
	return s
}

func (s *SoAccountWrap) MustNotExist(errMsgs ...interface{}) *SoAccountWrap {
	if s.CheckExist() {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.MustNotExist: %v already exists", s.mainKey), errMsgs...))
	}
	return s
}

func (s *SoAccountWrap) initWatcherFlag() {
	if s.watcherFlag == nil {
		s.watcherFlag = new(AccountWatcherFlag)
		*(s.watcherFlag) = AccountWatcherFlagOfDb(s.dba.ServiceId())
	}
}

func (s *SoAccountWrap) create(f func(tInfo *SoAccount)) error {
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

	s.mKeyFlag = 1

	// call watchers
	s.initWatcherFlag()
	if s.watcherFlag.AnyWatcher {
		ReportTableRecordInsert(s.dba.ServiceId(), s.dba.BranchId(), s.mainKey, val)
	}

	return nil
}

func (s *SoAccountWrap) Create(f func(tInfo *SoAccount), errArgs ...interface{}) *SoAccountWrap {
	err := s.create(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Errorf("SoAccountWrap.Create failed: %s", err.Error()), errArgs...))
	}
	return s
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

func (s *SoAccountWrap) modify(f func(tInfo *SoAccount)) error {
	if !s.CheckExist() {
		return errors.New("the SoAccount table does not exist. Please create a table first")
	}
	oriTable := s.getAccount()
	if oriTable == nil {
		return errors.New("fail to get origin table SoAccount")
	}

	curTable := s.getAccount()
	if curTable == nil {
		return errors.New("fail to create current table SoAccount")
	}
	f(curTable)

	//the main key is not support modify
	if !reflect.DeepEqual(curTable.Name, oriTable.Name) {
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
	err = s.updateAccount(curTable)
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

func (s *SoAccountWrap) Modify(f func(tInfo *SoAccount), errArgs ...interface{}) *SoAccountWrap {
	err := s.modify(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.Modify failed: %s", err.Error()), errArgs...))
	}
	return s
}

func (s *SoAccountWrap) SetBalance(p *prototype.Coin, errArgs ...interface{}) *SoAccountWrap {
	err := s.modify(func(r *SoAccount) {
		r.Balance = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.SetBalance( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoAccountWrap) SetBpVoteCount(p uint32, errArgs ...interface{}) *SoAccountWrap {
	err := s.modify(func(r *SoAccount) {
		r.BpVoteCount = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.SetBpVoteCount( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoAccountWrap) SetChargedTicket(p uint32, errArgs ...interface{}) *SoAccountWrap {
	err := s.modify(func(r *SoAccount) {
		r.ChargedTicket = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.SetChargedTicket( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoAccountWrap) SetCreatedTime(p *prototype.TimePointSec, errArgs ...interface{}) *SoAccountWrap {
	err := s.modify(func(r *SoAccount) {
		r.CreatedTime = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.SetCreatedTime( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoAccountWrap) SetCreatedTrxCount(p uint32, errArgs ...interface{}) *SoAccountWrap {
	err := s.modify(func(r *SoAccount) {
		r.CreatedTrxCount = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.SetCreatedTrxCount( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoAccountWrap) SetCreator(p *prototype.AccountName, errArgs ...interface{}) *SoAccountWrap {
	err := s.modify(func(r *SoAccount) {
		r.Creator = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.SetCreator( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoAccountWrap) SetEachPowerdownRate(p *prototype.Vest, errArgs ...interface{}) *SoAccountWrap {
	err := s.modify(func(r *SoAccount) {
		r.EachPowerdownRate = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.SetEachPowerdownRate( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoAccountWrap) SetFreeze(p uint32, errArgs ...interface{}) *SoAccountWrap {
	err := s.modify(func(r *SoAccount) {
		r.Freeze = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.SetFreeze( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoAccountWrap) SetFreezeMemo(p string, errArgs ...interface{}) *SoAccountWrap {
	err := s.modify(func(r *SoAccount) {
		r.FreezeMemo = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.SetFreezeMemo( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoAccountWrap) SetHasPowerdown(p *prototype.Vest, errArgs ...interface{}) *SoAccountWrap {
	err := s.modify(func(r *SoAccount) {
		r.HasPowerdown = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.SetHasPowerdown( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoAccountWrap) SetLastPostTime(p *prototype.TimePointSec, errArgs ...interface{}) *SoAccountWrap {
	err := s.modify(func(r *SoAccount) {
		r.LastPostTime = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.SetLastPostTime( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoAccountWrap) SetLastStakeTime(p *prototype.TimePointSec, errArgs ...interface{}) *SoAccountWrap {
	err := s.modify(func(r *SoAccount) {
		r.LastStakeTime = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.SetLastStakeTime( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoAccountWrap) SetLastVoteTime(p *prototype.TimePointSec, errArgs ...interface{}) *SoAccountWrap {
	err := s.modify(func(r *SoAccount) {
		r.LastVoteTime = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.SetLastVoteTime( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoAccountWrap) SetNextPowerdownBlockNum(p uint64, errArgs ...interface{}) *SoAccountWrap {
	err := s.modify(func(r *SoAccount) {
		r.NextPowerdownBlockNum = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.SetNextPowerdownBlockNum( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoAccountWrap) SetPostCount(p uint32, errArgs ...interface{}) *SoAccountWrap {
	err := s.modify(func(r *SoAccount) {
		r.PostCount = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.SetPostCount( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoAccountWrap) SetPubKey(p *prototype.PublicKeyType, errArgs ...interface{}) *SoAccountWrap {
	err := s.modify(func(r *SoAccount) {
		r.PubKey = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.SetPubKey( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoAccountWrap) SetReputation(p uint32, errArgs ...interface{}) *SoAccountWrap {
	err := s.modify(func(r *SoAccount) {
		r.Reputation = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.SetReputation( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoAccountWrap) SetReputationMemo(p string, errArgs ...interface{}) *SoAccountWrap {
	err := s.modify(func(r *SoAccount) {
		r.ReputationMemo = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.SetReputationMemo( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoAccountWrap) SetStakeVestForMe(p *prototype.Vest, errArgs ...interface{}) *SoAccountWrap {
	err := s.modify(func(r *SoAccount) {
		r.StakeVestForMe = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.SetStakeVestForMe( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoAccountWrap) SetStakeVestFromMe(p *prototype.Vest, errArgs ...interface{}) *SoAccountWrap {
	err := s.modify(func(r *SoAccount) {
		r.StakeVestFromMe = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.SetStakeVestFromMe( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoAccountWrap) SetStamina(p uint64, errArgs ...interface{}) *SoAccountWrap {
	err := s.modify(func(r *SoAccount) {
		r.Stamina = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.SetStamina( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoAccountWrap) SetStaminaFree(p uint64, errArgs ...interface{}) *SoAccountWrap {
	err := s.modify(func(r *SoAccount) {
		r.StaminaFree = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.SetStaminaFree( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoAccountWrap) SetStaminaFreeUseBlock(p uint64, errArgs ...interface{}) *SoAccountWrap {
	err := s.modify(func(r *SoAccount) {
		r.StaminaFreeUseBlock = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.SetStaminaFreeUseBlock( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoAccountWrap) SetStaminaUseBlock(p uint64, errArgs ...interface{}) *SoAccountWrap {
	err := s.modify(func(r *SoAccount) {
		r.StaminaUseBlock = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.SetStaminaUseBlock( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoAccountWrap) SetToPowerdown(p *prototype.Vest, errArgs ...interface{}) *SoAccountWrap {
	err := s.modify(func(r *SoAccount) {
		r.ToPowerdown = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.SetToPowerdown( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoAccountWrap) SetVest(p *prototype.Vest, errArgs ...interface{}) *SoAccountWrap {
	err := s.modify(func(r *SoAccount) {
		r.Vest = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.SetVest( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoAccountWrap) SetVotePower(p uint32, errArgs ...interface{}) *SoAccountWrap {
	err := s.modify(func(r *SoAccount) {
		r.VotePower = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.SetVotePower( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoAccountWrap) checkSortAndUniFieldValidity(curTable *SoAccount, fields map[string]bool) error {
	if curTable != nil && fields != nil && len(fields) > 0 {

		if fields["CreatedTime"] && curTable.CreatedTime == nil {
			return errors.New("sort field CreatedTime can't be modified to nil")
		}

		if fields["Balance"] && curTable.Balance == nil {
			return errors.New("sort field Balance can't be modified to nil")
		}

		if fields["Vest"] && curTable.Vest == nil {
			return errors.New("sort field Vest can't be modified to nil")
		}

		if fields["PubKey"] && curTable.PubKey == nil {
			return errors.New("unique field PubKey can't be modified to nil")
		}

	}
	return nil
}

//Get all the modified fields in the table
func (s *SoAccountWrap) getModifiedFields(oriTable *SoAccount, curTable *SoAccount) (map[string]bool, bool, error) {
	if oriTable == nil {
		return nil, false, errors.New("table info is nil, can't get modified fields")
	}
	hasWatcher := false
	fields := make(map[string]bool)

	if !reflect.DeepEqual(oriTable.Balance, curTable.Balance) {
		fields["Balance"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasBalanceWatcher
	}

	if !reflect.DeepEqual(oriTable.BpVoteCount, curTable.BpVoteCount) {
		fields["BpVoteCount"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasBpVoteCountWatcher
	}

	if !reflect.DeepEqual(oriTable.ChargedTicket, curTable.ChargedTicket) {
		fields["ChargedTicket"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasChargedTicketWatcher
	}

	if !reflect.DeepEqual(oriTable.CreatedTime, curTable.CreatedTime) {
		fields["CreatedTime"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasCreatedTimeWatcher
	}

	if !reflect.DeepEqual(oriTable.CreatedTrxCount, curTable.CreatedTrxCount) {
		fields["CreatedTrxCount"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasCreatedTrxCountWatcher
	}

	if !reflect.DeepEqual(oriTable.Creator, curTable.Creator) {
		fields["Creator"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasCreatorWatcher
	}

	if !reflect.DeepEqual(oriTable.EachPowerdownRate, curTable.EachPowerdownRate) {
		fields["EachPowerdownRate"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasEachPowerdownRateWatcher
	}

	if !reflect.DeepEqual(oriTable.Freeze, curTable.Freeze) {
		fields["Freeze"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasFreezeWatcher
	}

	if !reflect.DeepEqual(oriTable.FreezeMemo, curTable.FreezeMemo) {
		fields["FreezeMemo"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasFreezeMemoWatcher
	}

	if !reflect.DeepEqual(oriTable.HasPowerdown, curTable.HasPowerdown) {
		fields["HasPowerdown"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasHasPowerdownWatcher
	}

	if !reflect.DeepEqual(oriTable.LastPostTime, curTable.LastPostTime) {
		fields["LastPostTime"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasLastPostTimeWatcher
	}

	if !reflect.DeepEqual(oriTable.LastStakeTime, curTable.LastStakeTime) {
		fields["LastStakeTime"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasLastStakeTimeWatcher
	}

	if !reflect.DeepEqual(oriTable.LastVoteTime, curTable.LastVoteTime) {
		fields["LastVoteTime"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasLastVoteTimeWatcher
	}

	if !reflect.DeepEqual(oriTable.NextPowerdownBlockNum, curTable.NextPowerdownBlockNum) {
		fields["NextPowerdownBlockNum"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasNextPowerdownBlockNumWatcher
	}

	if !reflect.DeepEqual(oriTable.PostCount, curTable.PostCount) {
		fields["PostCount"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasPostCountWatcher
	}

	if !reflect.DeepEqual(oriTable.PubKey, curTable.PubKey) {
		fields["PubKey"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasPubKeyWatcher
	}

	if !reflect.DeepEqual(oriTable.Reputation, curTable.Reputation) {
		fields["Reputation"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasReputationWatcher
	}

	if !reflect.DeepEqual(oriTable.ReputationMemo, curTable.ReputationMemo) {
		fields["ReputationMemo"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasReputationMemoWatcher
	}

	if !reflect.DeepEqual(oriTable.StakeVestForMe, curTable.StakeVestForMe) {
		fields["StakeVestForMe"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasStakeVestForMeWatcher
	}

	if !reflect.DeepEqual(oriTable.StakeVestFromMe, curTable.StakeVestFromMe) {
		fields["StakeVestFromMe"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasStakeVestFromMeWatcher
	}

	if !reflect.DeepEqual(oriTable.Stamina, curTable.Stamina) {
		fields["Stamina"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasStaminaWatcher
	}

	if !reflect.DeepEqual(oriTable.StaminaFree, curTable.StaminaFree) {
		fields["StaminaFree"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasStaminaFreeWatcher
	}

	if !reflect.DeepEqual(oriTable.StaminaFreeUseBlock, curTable.StaminaFreeUseBlock) {
		fields["StaminaFreeUseBlock"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasStaminaFreeUseBlockWatcher
	}

	if !reflect.DeepEqual(oriTable.StaminaUseBlock, curTable.StaminaUseBlock) {
		fields["StaminaUseBlock"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasStaminaUseBlockWatcher
	}

	if !reflect.DeepEqual(oriTable.ToPowerdown, curTable.ToPowerdown) {
		fields["ToPowerdown"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasToPowerdownWatcher
	}

	if !reflect.DeepEqual(oriTable.Vest, curTable.Vest) {
		fields["Vest"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasVestWatcher
	}

	if !reflect.DeepEqual(oriTable.VotePower, curTable.VotePower) {
		fields["VotePower"] = true
		hasWatcher = hasWatcher || s.watcherFlag.HasVotePowerWatcher
	}

	hasWatcher = hasWatcher || s.watcherFlag.WholeWatcher
	return fields, hasWatcher, nil
}

func (s *SoAccountWrap) handleFieldMd(t FieldMdHandleType, so *SoAccount, fields map[string]bool) error {
	if so == nil {
		return errors.New("fail to modify empty table")
	}

	//there is no field need to modify
	if fields == nil || len(fields) < 1 {
		return nil
	}

	errStr := ""

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

	if fields["BpVoteCount"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldBpVoteCount(so.BpVoteCount, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "BpVoteCount")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldBpVoteCount(so.BpVoteCount, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "BpVoteCount")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldBpVoteCount(so.BpVoteCount, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "BpVoteCount")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["ChargedTicket"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldChargedTicket(so.ChargedTicket, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "ChargedTicket")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldChargedTicket(so.ChargedTicket, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "ChargedTicket")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldChargedTicket(so.ChargedTicket, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "ChargedTicket")
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

	if fields["CreatedTrxCount"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldCreatedTrxCount(so.CreatedTrxCount, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "CreatedTrxCount")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldCreatedTrxCount(so.CreatedTrxCount, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "CreatedTrxCount")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldCreatedTrxCount(so.CreatedTrxCount, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "CreatedTrxCount")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["Creator"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldCreator(so.Creator, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "Creator")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldCreator(so.Creator, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "Creator")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldCreator(so.Creator, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "Creator")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["EachPowerdownRate"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldEachPowerdownRate(so.EachPowerdownRate, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "EachPowerdownRate")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldEachPowerdownRate(so.EachPowerdownRate, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "EachPowerdownRate")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldEachPowerdownRate(so.EachPowerdownRate, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "EachPowerdownRate")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["Freeze"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldFreeze(so.Freeze, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "Freeze")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldFreeze(so.Freeze, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "Freeze")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldFreeze(so.Freeze, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "Freeze")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["FreezeMemo"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldFreezeMemo(so.FreezeMemo, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "FreezeMemo")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldFreezeMemo(so.FreezeMemo, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "FreezeMemo")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldFreezeMemo(so.FreezeMemo, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "FreezeMemo")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["HasPowerdown"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldHasPowerdown(so.HasPowerdown, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "HasPowerdown")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldHasPowerdown(so.HasPowerdown, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "HasPowerdown")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldHasPowerdown(so.HasPowerdown, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "HasPowerdown")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["LastPostTime"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldLastPostTime(so.LastPostTime, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "LastPostTime")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldLastPostTime(so.LastPostTime, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "LastPostTime")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldLastPostTime(so.LastPostTime, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "LastPostTime")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["LastStakeTime"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldLastStakeTime(so.LastStakeTime, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "LastStakeTime")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldLastStakeTime(so.LastStakeTime, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "LastStakeTime")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldLastStakeTime(so.LastStakeTime, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "LastStakeTime")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["LastVoteTime"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldLastVoteTime(so.LastVoteTime, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "LastVoteTime")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldLastVoteTime(so.LastVoteTime, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "LastVoteTime")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldLastVoteTime(so.LastVoteTime, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "LastVoteTime")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["NextPowerdownBlockNum"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldNextPowerdownBlockNum(so.NextPowerdownBlockNum, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "NextPowerdownBlockNum")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldNextPowerdownBlockNum(so.NextPowerdownBlockNum, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "NextPowerdownBlockNum")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldNextPowerdownBlockNum(so.NextPowerdownBlockNum, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "NextPowerdownBlockNum")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["PostCount"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldPostCount(so.PostCount, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "PostCount")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldPostCount(so.PostCount, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "PostCount")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldPostCount(so.PostCount, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "PostCount")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["PubKey"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldPubKey(so.PubKey, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "PubKey")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldPubKey(so.PubKey, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "PubKey")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldPubKey(so.PubKey, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "PubKey")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["Reputation"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldReputation(so.Reputation, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "Reputation")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldReputation(so.Reputation, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "Reputation")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldReputation(so.Reputation, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "Reputation")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["ReputationMemo"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldReputationMemo(so.ReputationMemo, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "ReputationMemo")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldReputationMemo(so.ReputationMemo, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "ReputationMemo")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldReputationMemo(so.ReputationMemo, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "ReputationMemo")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["StakeVestForMe"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldStakeVestForMe(so.StakeVestForMe, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "StakeVestForMe")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldStakeVestForMe(so.StakeVestForMe, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "StakeVestForMe")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldStakeVestForMe(so.StakeVestForMe, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "StakeVestForMe")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["StakeVestFromMe"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldStakeVestFromMe(so.StakeVestFromMe, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "StakeVestFromMe")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldStakeVestFromMe(so.StakeVestFromMe, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "StakeVestFromMe")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldStakeVestFromMe(so.StakeVestFromMe, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "StakeVestFromMe")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["Stamina"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldStamina(so.Stamina, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "Stamina")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldStamina(so.Stamina, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "Stamina")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldStamina(so.Stamina, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "Stamina")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["StaminaFree"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldStaminaFree(so.StaminaFree, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "StaminaFree")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldStaminaFree(so.StaminaFree, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "StaminaFree")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldStaminaFree(so.StaminaFree, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "StaminaFree")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["StaminaFreeUseBlock"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldStaminaFreeUseBlock(so.StaminaFreeUseBlock, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "StaminaFreeUseBlock")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldStaminaFreeUseBlock(so.StaminaFreeUseBlock, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "StaminaFreeUseBlock")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldStaminaFreeUseBlock(so.StaminaFreeUseBlock, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "StaminaFreeUseBlock")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["StaminaUseBlock"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldStaminaUseBlock(so.StaminaUseBlock, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "StaminaUseBlock")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldStaminaUseBlock(so.StaminaUseBlock, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "StaminaUseBlock")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldStaminaUseBlock(so.StaminaUseBlock, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "StaminaUseBlock")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["ToPowerdown"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldToPowerdown(so.ToPowerdown, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "ToPowerdown")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldToPowerdown(so.ToPowerdown, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "ToPowerdown")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldToPowerdown(so.ToPowerdown, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "ToPowerdown")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["Vest"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldVest(so.Vest, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "Vest")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldVest(so.Vest, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "Vest")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldVest(so.Vest, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "Vest")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["VotePower"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldVotePower(so.VotePower, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "VotePower")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldVotePower(so.VotePower, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "VotePower")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldVotePower(so.VotePower, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "VotePower")
		}
		if !res {
			return errors.New(errStr)
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

func (s *SoAccountWrap) delSortKeyVest(sa *SoAccount) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListAccountByVest{}
	if sa == nil {
		val.Vest = s.GetVest()
		val.Name = s.mainKey

	} else {
		val.Vest = sa.Vest
		val.Name = sa.Name
	}
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoAccountWrap) insertSortKeyVest(sa *SoAccount) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListAccountByVest{}
	val.Name = sa.Name
	val.Vest = sa.Vest
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
	if !s.delSortKeyVest(val) {
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
	if !s.insertSortKeyVest(val) {
		return errors.New("insert sort Field Vest fail while insert table ")
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

func (s *SoAccountWrap) removeAccount() error {
	if s.dba == nil {
		return errors.New("database is nil")
	}

	s.initWatcherFlag()

	var oldVal *SoAccount
	if s.watcherFlag.AnyWatcher {
		oldVal = s.getAccount()
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

func (s *SoAccountWrap) RemoveAccount(errMsgs ...interface{}) *SoAccountWrap {
	err := s.removeAccount()
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoAccountWrap.RemoveAccount failed: %s", err.Error()), errMsgs...))
	}
	return s
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

func (s *SoAccountWrap) GetChargedTicket() uint32 {
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
				return msg.ChargedTicket
			}
		}
	}
	if !res {
		var tmpValue uint32
		return tmpValue
	}
	return msg.ChargedTicket
}

func (s *SoAccountWrap) mdFieldChargedTicket(p uint32, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkChargedTicketIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldChargedTicket(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldChargedTicket(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldChargedTicket(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldChargedTicket(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkChargedTicketIsMetMdCondition(p uint32) bool {
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

func (s *SoAccountWrap) GetFreeze() uint32 {
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
				return msg.Freeze
			}
		}
	}
	if !res {
		var tmpValue uint32
		return tmpValue
	}
	return msg.Freeze
}

func (s *SoAccountWrap) mdFieldFreeze(p uint32, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkFreezeIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldFreeze(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldFreeze(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldFreeze(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldFreeze(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkFreezeIsMetMdCondition(p uint32) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) GetFreezeMemo() string {
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
				return msg.FreezeMemo
			}
		}
	}
	if !res {
		var tmpValue string
		return tmpValue
	}
	return msg.FreezeMemo
}

func (s *SoAccountWrap) mdFieldFreezeMemo(p string, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkFreezeMemoIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldFreezeMemo(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldFreezeMemo(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldFreezeMemo(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldFreezeMemo(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkFreezeMemoIsMetMdCondition(p string) bool {
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

func (s *SoAccountWrap) GetPubKey() *prototype.PublicKeyType {
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
				return msg.PubKey
			}
		}
	}
	if !res {
		return nil

	}
	return msg.PubKey
}

func (s *SoAccountWrap) mdFieldPubKey(p *prototype.PublicKeyType, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkPubKeyIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldPubKey(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldPubKey(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldPubKey(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if !s.delUniKeyPubKey(so) {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldPubKey(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertUniKeyPubKey(so) {
		return false
	}
	return true
}

func (s *SoAccountWrap) checkPubKeyIsMetMdCondition(p *prototype.PublicKeyType) bool {
	if s.dba == nil {
		return false
	}

	//judge the unique value if is exist
	uniWrap := UniAccountPubKeyWrap{}
	uniWrap.Dba = s.dba
	res := uniWrap.UniQueryPubKey(p)

	if res != nil {
		//the unique value to be modified is already exist
		return false
	}

	return true
}

func (s *SoAccountWrap) GetReputation() uint32 {
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
				return msg.Reputation
			}
		}
	}
	if !res {
		var tmpValue uint32
		return tmpValue
	}
	return msg.Reputation
}

func (s *SoAccountWrap) mdFieldReputation(p uint32, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkReputationIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldReputation(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldReputation(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldReputation(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldReputation(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkReputationIsMetMdCondition(p uint32) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) GetReputationMemo() string {
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
				return msg.ReputationMemo
			}
		}
	}
	if !res {
		var tmpValue string
		return tmpValue
	}
	return msg.ReputationMemo
}

func (s *SoAccountWrap) mdFieldReputationMemo(p string, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkReputationMemoIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldReputationMemo(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldReputationMemo(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldReputationMemo(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldReputationMemo(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkReputationMemoIsMetMdCondition(p string) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) GetStakeVestForMe() *prototype.Vest {
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
				return msg.StakeVestForMe
			}
		}
	}
	if !res {
		return nil

	}
	return msg.StakeVestForMe
}

func (s *SoAccountWrap) mdFieldStakeVestForMe(p *prototype.Vest, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkStakeVestForMeIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldStakeVestForMe(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldStakeVestForMe(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldStakeVestForMe(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldStakeVestForMe(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkStakeVestForMeIsMetMdCondition(p *prototype.Vest) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) GetStakeVestFromMe() *prototype.Vest {
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
				return msg.StakeVestFromMe
			}
		}
	}
	if !res {
		return nil

	}
	return msg.StakeVestFromMe
}

func (s *SoAccountWrap) mdFieldStakeVestFromMe(p *prototype.Vest, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkStakeVestFromMeIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldStakeVestFromMe(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldStakeVestFromMe(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldStakeVestFromMe(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldStakeVestFromMe(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkStakeVestFromMeIsMetMdCondition(p *prototype.Vest) bool {
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

func (s *SoAccountWrap) GetVest() *prototype.Vest {
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
				return msg.Vest
			}
		}
	}
	if !res {
		return nil

	}
	return msg.Vest
}

func (s *SoAccountWrap) mdFieldVest(p *prototype.Vest, isCheck bool, isDel bool, isInsert bool,
	so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkVestIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldVest(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldVest(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoAccountWrap) delFieldVest(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyVest(so) {
		return false
	}

	return true
}

func (s *SoAccountWrap) insertFieldVest(so *SoAccount) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyVest(so) {
		return false
	}

	return true
}

func (s *SoAccountWrap) checkVestIsMetMdCondition(p *prototype.Vest) bool {
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
type SAccountVestWrap struct {
	Dba iservices.IDatabaseRW
}

func NewAccountVestWrap(db iservices.IDatabaseRW) *SAccountVestWrap {
	if db == nil {
		return nil
	}
	wrap := SAccountVestWrap{Dba: db}
	return &wrap
}

func (s *SAccountVestWrap) GetMainVal(val []byte) *prototype.AccountName {
	res := &SoListAccountByVest{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Name

}

func (s *SAccountVestWrap) GetSubVal(val []byte) *prototype.Vest {
	res := &SoListAccountByVest{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.Vest

}

func (m *SoListAccountByVest) OpeEncode() ([]byte, error) {
	pre := AccountVestTable
	sub := m.Vest
	if sub == nil {
		return nil, errors.New("the pro Vest is nil")
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
func (s *SAccountVestWrap) ForEachByRevOrder(start *prototype.Vest, end *prototype.Vest, lastMainKey *prototype.AccountName,
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
	pre := AccountVestTable
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
	if !s.delUniKeyPubKey(val) {
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
	if len(names["PubKey"]) > 0 {
		if !s.delUniKeyPubKey(val) {
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
	if !s.insertUniKeyPubKey(val) {
		return sucFields, errors.New("insert unique Field PubKey fail while insert table ")
	}
	sucFields["PubKey"] = "PubKey"

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

func (s *SoAccountWrap) delUniKeyPubKey(sa *SoAccount) bool {
	if s.dba == nil {
		return false
	}
	pre := AccountPubKeyUniTable
	kList := []interface{}{pre}
	if sa != nil {
		if sa.PubKey == nil {
			return false
		}

		sub := sa.PubKey
		kList = append(kList, sub)
	} else {
		sub := s.GetPubKey()
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

func (s *SoAccountWrap) insertUniKeyPubKey(sa *SoAccount) bool {
	if s.dba == nil || sa == nil {
		return false
	}

	pre := AccountPubKeyUniTable
	sub := sa.PubKey
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
	val := SoUniqueAccountByPubKey{}
	val.Name = sa.Name
	val.PubKey = sa.PubKey

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniAccountPubKeyWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniAccountPubKeyWrap(db iservices.IDatabaseRW) *UniAccountPubKeyWrap {
	if db == nil {
		return nil
	}
	wrap := UniAccountPubKeyWrap{Dba: db}
	return &wrap
}

func (s *UniAccountPubKeyWrap) UniQueryPubKey(start *prototype.PublicKeyType) *SoAccountWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := AccountPubKeyUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueAccountByPubKey{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoAccountWrap(s.Dba, res.Name)

			return wrap
		}
	}
	return nil
}

////////////// SECTION Watchers ///////////////

type AccountWatcherFlag struct {
	HasBalanceWatcher bool

	HasBpVoteCountWatcher bool

	HasChargedTicketWatcher bool

	HasCreatedTimeWatcher bool

	HasCreatedTrxCountWatcher bool

	HasCreatorWatcher bool

	HasEachPowerdownRateWatcher bool

	HasFreezeWatcher bool

	HasFreezeMemoWatcher bool

	HasHasPowerdownWatcher bool

	HasLastPostTimeWatcher bool

	HasLastStakeTimeWatcher bool

	HasLastVoteTimeWatcher bool

	HasNextPowerdownBlockNumWatcher bool

	HasPostCountWatcher bool

	HasPubKeyWatcher bool

	HasReputationWatcher bool

	HasReputationMemoWatcher bool

	HasStakeVestForMeWatcher bool

	HasStakeVestFromMeWatcher bool

	HasStaminaWatcher bool

	HasStaminaFreeWatcher bool

	HasStaminaFreeUseBlockWatcher bool

	HasStaminaUseBlockWatcher bool

	HasToPowerdownWatcher bool

	HasVestWatcher bool

	HasVotePowerWatcher bool

	WholeWatcher bool
	AnyWatcher   bool
}

var (
	AccountTable = &TableInfo{
		Name:    "Account",
		Primary: "Name",
		Record:  reflect.TypeOf((*SoAccount)(nil)).Elem(),
	}
	AccountWatcherFlags     = make(map[uint32]AccountWatcherFlag)
	AccountWatcherFlagsLock sync.RWMutex
)

func AccountWatcherFlagOfDb(dbSvcId uint32) AccountWatcherFlag {
	AccountWatcherFlagsLock.RLock()
	defer AccountWatcherFlagsLock.RUnlock()
	return AccountWatcherFlags[dbSvcId]
}

func AccountRecordWatcherChanged(dbSvcId uint32) {
	var flag AccountWatcherFlag
	flag.WholeWatcher = HasTableRecordWatcher(dbSvcId, AccountTable.Record, "")
	flag.AnyWatcher = flag.WholeWatcher

	flag.HasBalanceWatcher = HasTableRecordWatcher(dbSvcId, AccountTable.Record, "Balance")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasBalanceWatcher

	flag.HasBpVoteCountWatcher = HasTableRecordWatcher(dbSvcId, AccountTable.Record, "BpVoteCount")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasBpVoteCountWatcher

	flag.HasChargedTicketWatcher = HasTableRecordWatcher(dbSvcId, AccountTable.Record, "ChargedTicket")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasChargedTicketWatcher

	flag.HasCreatedTimeWatcher = HasTableRecordWatcher(dbSvcId, AccountTable.Record, "CreatedTime")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasCreatedTimeWatcher

	flag.HasCreatedTrxCountWatcher = HasTableRecordWatcher(dbSvcId, AccountTable.Record, "CreatedTrxCount")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasCreatedTrxCountWatcher

	flag.HasCreatorWatcher = HasTableRecordWatcher(dbSvcId, AccountTable.Record, "Creator")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasCreatorWatcher

	flag.HasEachPowerdownRateWatcher = HasTableRecordWatcher(dbSvcId, AccountTable.Record, "EachPowerdownRate")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasEachPowerdownRateWatcher

	flag.HasFreezeWatcher = HasTableRecordWatcher(dbSvcId, AccountTable.Record, "Freeze")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasFreezeWatcher

	flag.HasFreezeMemoWatcher = HasTableRecordWatcher(dbSvcId, AccountTable.Record, "FreezeMemo")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasFreezeMemoWatcher

	flag.HasHasPowerdownWatcher = HasTableRecordWatcher(dbSvcId, AccountTable.Record, "HasPowerdown")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasHasPowerdownWatcher

	flag.HasLastPostTimeWatcher = HasTableRecordWatcher(dbSvcId, AccountTable.Record, "LastPostTime")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasLastPostTimeWatcher

	flag.HasLastStakeTimeWatcher = HasTableRecordWatcher(dbSvcId, AccountTable.Record, "LastStakeTime")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasLastStakeTimeWatcher

	flag.HasLastVoteTimeWatcher = HasTableRecordWatcher(dbSvcId, AccountTable.Record, "LastVoteTime")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasLastVoteTimeWatcher

	flag.HasNextPowerdownBlockNumWatcher = HasTableRecordWatcher(dbSvcId, AccountTable.Record, "NextPowerdownBlockNum")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasNextPowerdownBlockNumWatcher

	flag.HasPostCountWatcher = HasTableRecordWatcher(dbSvcId, AccountTable.Record, "PostCount")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasPostCountWatcher

	flag.HasPubKeyWatcher = HasTableRecordWatcher(dbSvcId, AccountTable.Record, "PubKey")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasPubKeyWatcher

	flag.HasReputationWatcher = HasTableRecordWatcher(dbSvcId, AccountTable.Record, "Reputation")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasReputationWatcher

	flag.HasReputationMemoWatcher = HasTableRecordWatcher(dbSvcId, AccountTable.Record, "ReputationMemo")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasReputationMemoWatcher

	flag.HasStakeVestForMeWatcher = HasTableRecordWatcher(dbSvcId, AccountTable.Record, "StakeVestForMe")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasStakeVestForMeWatcher

	flag.HasStakeVestFromMeWatcher = HasTableRecordWatcher(dbSvcId, AccountTable.Record, "StakeVestFromMe")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasStakeVestFromMeWatcher

	flag.HasStaminaWatcher = HasTableRecordWatcher(dbSvcId, AccountTable.Record, "Stamina")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasStaminaWatcher

	flag.HasStaminaFreeWatcher = HasTableRecordWatcher(dbSvcId, AccountTable.Record, "StaminaFree")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasStaminaFreeWatcher

	flag.HasStaminaFreeUseBlockWatcher = HasTableRecordWatcher(dbSvcId, AccountTable.Record, "StaminaFreeUseBlock")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasStaminaFreeUseBlockWatcher

	flag.HasStaminaUseBlockWatcher = HasTableRecordWatcher(dbSvcId, AccountTable.Record, "StaminaUseBlock")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasStaminaUseBlockWatcher

	flag.HasToPowerdownWatcher = HasTableRecordWatcher(dbSvcId, AccountTable.Record, "ToPowerdown")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasToPowerdownWatcher

	flag.HasVestWatcher = HasTableRecordWatcher(dbSvcId, AccountTable.Record, "Vest")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasVestWatcher

	flag.HasVotePowerWatcher = HasTableRecordWatcher(dbSvcId, AccountTable.Record, "VotePower")
	flag.AnyWatcher = flag.AnyWatcher || flag.HasVotePowerWatcher

	AccountWatcherFlagsLock.Lock()
	AccountWatcherFlags[dbSvcId] = flag
	AccountWatcherFlagsLock.Unlock()
}

func init() {
	RegisterTableWatcherChangedCallback(AccountTable.Record, AccountRecordWatcherChanged)
}
