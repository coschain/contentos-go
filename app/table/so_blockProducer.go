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
	BlockProducerOwnerTable    uint32 = 2440644301
	BlockProducerBpVestTable   uint32 = 3083635068
	BlockProducerOwnerUniTable uint32 = 404338461

	BlockProducerOwnerRow uint32 = 259692740
)

////////////// SECTION Wrap Define ///////////////
type SoBlockProducerWrap struct {
	dba       iservices.IDatabaseRW
	mainKey   *prototype.AccountName
	mKeyFlag  int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf   []byte //the buffer after the main key is encoded with prefix
	mBuf      []byte //the value after the main key is encoded
	mdFuncMap map[string]interface{}
}

func NewSoBlockProducerWrap(dba iservices.IDatabaseRW, key *prototype.AccountName) *SoBlockProducerWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoBlockProducerWrap{dba, key, -1, nil, nil, nil}
	return result
}

func (s *SoBlockProducerWrap) CheckExist() bool {
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

func (s *SoBlockProducerWrap) MustExist(errMsgs ...interface{}) *SoBlockProducerWrap {
	if !s.CheckExist() {
		panic(bindErrorInfo(fmt.Sprintf("SoBlockProducerWrap.MustExist: %v not found", s.mainKey), errMsgs...))
	}
	return s
}

func (s *SoBlockProducerWrap) MustNotExist(errMsgs ...interface{}) *SoBlockProducerWrap {
	if s.CheckExist() {
		panic(bindErrorInfo(fmt.Sprintf("SoBlockProducerWrap.MustNotExist: %v already exists", s.mainKey), errMsgs...))
	}
	return s
}

func (s *SoBlockProducerWrap) create(f func(tInfo *SoBlockProducer)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoBlockProducer{}
	f(val)
	if val.Owner == nil {
		val.Owner = s.mainKey
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
	if BlockProducerHasAnyWatcher {
		ReportTableRecordInsert(s.mainKey, val)
	}

	return nil
}

func (s *SoBlockProducerWrap) Create(f func(tInfo *SoBlockProducer), errArgs ...interface{}) *SoBlockProducerWrap {
	err := s.create(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Errorf("SoBlockProducerWrap.Create failed: %s", err.Error()), errArgs...))
	}
	return s
}

func (s *SoBlockProducerWrap) getMainKeyBuf() ([]byte, error) {
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

func (s *SoBlockProducerWrap) modify(f func(tInfo *SoBlockProducer)) error {
	if !s.CheckExist() {
		return errors.New("the SoBlockProducer table does not exist. Please create a table first")
	}
	oriTable := s.getBlockProducer()
	if oriTable == nil {
		return errors.New("fail to get origin table SoBlockProducer")
	}

	curTable := s.getBlockProducer()
	if curTable == nil {
		return errors.New("fail to create current table SoBlockProducer")
	}
	f(curTable)

	//the main key is not support modify
	if !reflect.DeepEqual(curTable.Owner, oriTable.Owner) {
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
	err = s.updateBlockProducer(curTable)
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

func (s *SoBlockProducerWrap) Modify(f func(tInfo *SoBlockProducer), errArgs ...interface{}) *SoBlockProducerWrap {
	err := s.modify(f)
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoBlockProducerWrap.Modify failed: %s", err.Error()), errArgs...))
	}
	return s
}

func (s *SoBlockProducerWrap) SetAccountCreateFee(p *prototype.Coin, errArgs ...interface{}) *SoBlockProducerWrap {
	err := s.modify(func(r *SoBlockProducer) {
		r.AccountCreateFee = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoBlockProducerWrap.SetAccountCreateFee( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoBlockProducerWrap) SetBpVest(p *prototype.BpVestId, errArgs ...interface{}) *SoBlockProducerWrap {
	err := s.modify(func(r *SoBlockProducer) {
		r.BpVest = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoBlockProducerWrap.SetBpVest( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoBlockProducerWrap) SetCreatedTime(p *prototype.TimePointSec, errArgs ...interface{}) *SoBlockProducerWrap {
	err := s.modify(func(r *SoBlockProducer) {
		r.CreatedTime = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoBlockProducerWrap.SetCreatedTime( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoBlockProducerWrap) SetEpochDuration(p uint64, errArgs ...interface{}) *SoBlockProducerWrap {
	err := s.modify(func(r *SoBlockProducer) {
		r.EpochDuration = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoBlockProducerWrap.SetEpochDuration( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoBlockProducerWrap) SetPerTicketPrice(p *prototype.Coin, errArgs ...interface{}) *SoBlockProducerWrap {
	err := s.modify(func(r *SoBlockProducer) {
		r.PerTicketPrice = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoBlockProducerWrap.SetPerTicketPrice( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoBlockProducerWrap) SetPerTicketWeight(p uint64, errArgs ...interface{}) *SoBlockProducerWrap {
	err := s.modify(func(r *SoBlockProducer) {
		r.PerTicketWeight = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoBlockProducerWrap.SetPerTicketWeight( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoBlockProducerWrap) SetProposedStaminaFree(p uint64, errArgs ...interface{}) *SoBlockProducerWrap {
	err := s.modify(func(r *SoBlockProducer) {
		r.ProposedStaminaFree = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoBlockProducerWrap.SetProposedStaminaFree( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoBlockProducerWrap) SetSigningKey(p *prototype.PublicKeyType, errArgs ...interface{}) *SoBlockProducerWrap {
	err := s.modify(func(r *SoBlockProducer) {
		r.SigningKey = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoBlockProducerWrap.SetSigningKey( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoBlockProducerWrap) SetTopNAcquireFreeToken(p uint32, errArgs ...interface{}) *SoBlockProducerWrap {
	err := s.modify(func(r *SoBlockProducer) {
		r.TopNAcquireFreeToken = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoBlockProducerWrap.SetTopNAcquireFreeToken( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoBlockProducerWrap) SetTpsExpected(p uint64, errArgs ...interface{}) *SoBlockProducerWrap {
	err := s.modify(func(r *SoBlockProducer) {
		r.TpsExpected = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoBlockProducerWrap.SetTpsExpected( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoBlockProducerWrap) SetUrl(p string, errArgs ...interface{}) *SoBlockProducerWrap {
	err := s.modify(func(r *SoBlockProducer) {
		r.Url = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoBlockProducerWrap.SetUrl( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoBlockProducerWrap) SetVoterCount(p uint64, errArgs ...interface{}) *SoBlockProducerWrap {
	err := s.modify(func(r *SoBlockProducer) {
		r.VoterCount = p
	})
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoBlockProducerWrap.SetVoterCount( %v ) failed: %s", p, err.Error()), errArgs...))
	}
	return s
}

func (s *SoBlockProducerWrap) checkSortAndUniFieldValidity(curTable *SoBlockProducer, fields map[string]bool) error {
	if curTable != nil && fields != nil && len(fields) > 0 {

		if fields["BpVest"] && curTable.BpVest == nil {
			return errors.New("sort field BpVest can't be modified to nil")
		}

	}
	return nil
}

//Get all the modified fields in the table
func (s *SoBlockProducerWrap) getModifiedFields(oriTable *SoBlockProducer, curTable *SoBlockProducer) (map[string]bool, bool, error) {
	if oriTable == nil {
		return nil, false, errors.New("table info is nil, can't get modified fields")
	}
	hasWatcher := false
	fields := make(map[string]bool)

	if !reflect.DeepEqual(oriTable.AccountCreateFee, curTable.AccountCreateFee) {
		fields["AccountCreateFee"] = true
		hasWatcher = hasWatcher || BlockProducerHasAccountCreateFeeWatcher
	}

	if !reflect.DeepEqual(oriTable.BpVest, curTable.BpVest) {
		fields["BpVest"] = true
		hasWatcher = hasWatcher || BlockProducerHasBpVestWatcher
	}

	if !reflect.DeepEqual(oriTable.CreatedTime, curTable.CreatedTime) {
		fields["CreatedTime"] = true
		hasWatcher = hasWatcher || BlockProducerHasCreatedTimeWatcher
	}

	if !reflect.DeepEqual(oriTable.EpochDuration, curTable.EpochDuration) {
		fields["EpochDuration"] = true
		hasWatcher = hasWatcher || BlockProducerHasEpochDurationWatcher
	}

	if !reflect.DeepEqual(oriTable.PerTicketPrice, curTable.PerTicketPrice) {
		fields["PerTicketPrice"] = true
		hasWatcher = hasWatcher || BlockProducerHasPerTicketPriceWatcher
	}

	if !reflect.DeepEqual(oriTable.PerTicketWeight, curTable.PerTicketWeight) {
		fields["PerTicketWeight"] = true
		hasWatcher = hasWatcher || BlockProducerHasPerTicketWeightWatcher
	}

	if !reflect.DeepEqual(oriTable.ProposedStaminaFree, curTable.ProposedStaminaFree) {
		fields["ProposedStaminaFree"] = true
		hasWatcher = hasWatcher || BlockProducerHasProposedStaminaFreeWatcher
	}

	if !reflect.DeepEqual(oriTable.SigningKey, curTable.SigningKey) {
		fields["SigningKey"] = true
		hasWatcher = hasWatcher || BlockProducerHasSigningKeyWatcher
	}

	if !reflect.DeepEqual(oriTable.TopNAcquireFreeToken, curTable.TopNAcquireFreeToken) {
		fields["TopNAcquireFreeToken"] = true
		hasWatcher = hasWatcher || BlockProducerHasTopNAcquireFreeTokenWatcher
	}

	if !reflect.DeepEqual(oriTable.TpsExpected, curTable.TpsExpected) {
		fields["TpsExpected"] = true
		hasWatcher = hasWatcher || BlockProducerHasTpsExpectedWatcher
	}

	if !reflect.DeepEqual(oriTable.Url, curTable.Url) {
		fields["Url"] = true
		hasWatcher = hasWatcher || BlockProducerHasUrlWatcher
	}

	if !reflect.DeepEqual(oriTable.VoterCount, curTable.VoterCount) {
		fields["VoterCount"] = true
		hasWatcher = hasWatcher || BlockProducerHasVoterCountWatcher
	}

	hasWatcher = hasWatcher || BlockProducerHasWholeWatcher
	return fields, hasWatcher, nil
}

func (s *SoBlockProducerWrap) handleFieldMd(t FieldMdHandleType, so *SoBlockProducer, fields map[string]bool) error {
	if so == nil {
		return errors.New("fail to modify empty table")
	}

	//there is no field need to modify
	if fields == nil || len(fields) < 1 {
		return nil
	}

	errStr := ""

	if fields["AccountCreateFee"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldAccountCreateFee(so.AccountCreateFee, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "AccountCreateFee")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldAccountCreateFee(so.AccountCreateFee, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "AccountCreateFee")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldAccountCreateFee(so.AccountCreateFee, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "AccountCreateFee")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["BpVest"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldBpVest(so.BpVest, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "BpVest")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldBpVest(so.BpVest, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "BpVest")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldBpVest(so.BpVest, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "BpVest")
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

	if fields["EpochDuration"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldEpochDuration(so.EpochDuration, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "EpochDuration")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldEpochDuration(so.EpochDuration, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "EpochDuration")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldEpochDuration(so.EpochDuration, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "EpochDuration")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["PerTicketPrice"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldPerTicketPrice(so.PerTicketPrice, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "PerTicketPrice")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldPerTicketPrice(so.PerTicketPrice, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "PerTicketPrice")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldPerTicketPrice(so.PerTicketPrice, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "PerTicketPrice")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["PerTicketWeight"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldPerTicketWeight(so.PerTicketWeight, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "PerTicketWeight")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldPerTicketWeight(so.PerTicketWeight, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "PerTicketWeight")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldPerTicketWeight(so.PerTicketWeight, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "PerTicketWeight")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["ProposedStaminaFree"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldProposedStaminaFree(so.ProposedStaminaFree, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "ProposedStaminaFree")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldProposedStaminaFree(so.ProposedStaminaFree, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "ProposedStaminaFree")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldProposedStaminaFree(so.ProposedStaminaFree, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "ProposedStaminaFree")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["SigningKey"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldSigningKey(so.SigningKey, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "SigningKey")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldSigningKey(so.SigningKey, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "SigningKey")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldSigningKey(so.SigningKey, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "SigningKey")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["TopNAcquireFreeToken"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldTopNAcquireFreeToken(so.TopNAcquireFreeToken, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "TopNAcquireFreeToken")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldTopNAcquireFreeToken(so.TopNAcquireFreeToken, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "TopNAcquireFreeToken")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldTopNAcquireFreeToken(so.TopNAcquireFreeToken, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "TopNAcquireFreeToken")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	if fields["TpsExpected"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldTpsExpected(so.TpsExpected, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "TpsExpected")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldTpsExpected(so.TpsExpected, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "TpsExpected")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldTpsExpected(so.TpsExpected, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "TpsExpected")
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

	if fields["VoterCount"] {
		res := true
		if t == FieldMdHandleTypeCheck {
			res = s.mdFieldVoterCount(so.VoterCount, true, false, false, so)
			errStr = fmt.Sprintf("fail to modify exist value of %v", "VoterCount")
		} else if t == FieldMdHandleTypeDel {
			res = s.mdFieldVoterCount(so.VoterCount, false, true, false, so)
			errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", "VoterCount")
		} else if t == FieldMdHandleTypeInsert {
			res = s.mdFieldVoterCount(so.VoterCount, false, false, true, so)
			errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", "VoterCount")
		}
		if !res {
			return errors.New(errStr)
		}
	}

	return nil
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoBlockProducerWrap) delSortKeyOwner(sa *SoBlockProducer) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListBlockProducerByOwner{}
	if sa == nil {
		val.Owner = s.GetOwner()
	} else {
		val.Owner = sa.Owner
	}
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoBlockProducerWrap) insertSortKeyOwner(sa *SoBlockProducer) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListBlockProducerByOwner{}
	val.Owner = sa.Owner
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

func (s *SoBlockProducerWrap) delSortKeyBpVest(sa *SoBlockProducer) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListBlockProducerByBpVest{}
	if sa == nil {
		val.BpVest = s.GetBpVest()
		val.Owner = s.mainKey

	} else {
		val.BpVest = sa.BpVest
		val.Owner = sa.Owner
	}
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoBlockProducerWrap) insertSortKeyBpVest(sa *SoBlockProducer) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListBlockProducerByBpVest{}
	val.Owner = sa.Owner
	val.BpVest = sa.BpVest
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

func (s *SoBlockProducerWrap) delAllSortKeys(br bool, val *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delSortKeyOwner(val) {
		if br {
			return false
		} else {
			res = false
		}
	}
	if !s.delSortKeyBpVest(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoBlockProducerWrap) insertAllSortKeys(val *SoBlockProducer) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoBlockProducer fail ")
	}
	if !s.insertSortKeyOwner(val) {
		return errors.New("insert sort Field Owner fail while insert table ")
	}
	if !s.insertSortKeyBpVest(val) {
		return errors.New("insert sort Field BpVest fail while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoBlockProducerWrap) removeBlockProducer() error {
	if s.dba == nil {
		return errors.New("database is nil")
	}

	var oldVal *SoBlockProducer
	if BlockProducerHasAnyWatcher {
		oldVal = s.getBlockProducer()
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
		if BlockProducerHasAnyWatcher && oldVal != nil {
			ReportTableRecordDelete(s.mainKey, oldVal)
		}
		return nil
	} else {
		return fmt.Errorf("database.Delete failed: %s", err.Error())
	}
}

func (s *SoBlockProducerWrap) RemoveBlockProducer(errMsgs ...interface{}) *SoBlockProducerWrap {
	err := s.removeBlockProducer()
	if err != nil {
		panic(bindErrorInfo(fmt.Sprintf("SoBlockProducerWrap.RemoveBlockProducer failed: %s", err.Error()), errMsgs...))
	}
	return s
}

////////////// SECTION Members Get/Modify ///////////////

func (s *SoBlockProducerWrap) GetAccountCreateFee() *prototype.Coin {
	res := true
	msg := &SoBlockProducer{}
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
				return msg.AccountCreateFee
			}
		}
	}
	if !res {
		return nil

	}
	return msg.AccountCreateFee
}

func (s *SoBlockProducerWrap) mdFieldAccountCreateFee(p *prototype.Coin, isCheck bool, isDel bool, isInsert bool,
	so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkAccountCreateFeeIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldAccountCreateFee(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldAccountCreateFee(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoBlockProducerWrap) delFieldAccountCreateFee(so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) insertFieldAccountCreateFee(so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) checkAccountCreateFeeIsMetMdCondition(p *prototype.Coin) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) GetBpVest() *prototype.BpVestId {
	res := true
	msg := &SoBlockProducer{}
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
				return msg.BpVest
			}
		}
	}
	if !res {
		return nil

	}
	return msg.BpVest
}

func (s *SoBlockProducerWrap) mdFieldBpVest(p *prototype.BpVestId, isCheck bool, isDel bool, isInsert bool,
	so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkBpVestIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldBpVest(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldBpVest(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoBlockProducerWrap) delFieldBpVest(so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyBpVest(so) {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) insertFieldBpVest(so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyBpVest(so) {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) checkBpVestIsMetMdCondition(p *prototype.BpVestId) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) GetCreatedTime() *prototype.TimePointSec {
	res := true
	msg := &SoBlockProducer{}
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

func (s *SoBlockProducerWrap) mdFieldCreatedTime(p *prototype.TimePointSec, isCheck bool, isDel bool, isInsert bool,
	so *SoBlockProducer) bool {
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

func (s *SoBlockProducerWrap) delFieldCreatedTime(so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) insertFieldCreatedTime(so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) checkCreatedTimeIsMetMdCondition(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) GetEpochDuration() uint64 {
	res := true
	msg := &SoBlockProducer{}
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
				return msg.EpochDuration
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.EpochDuration
}

func (s *SoBlockProducerWrap) mdFieldEpochDuration(p uint64, isCheck bool, isDel bool, isInsert bool,
	so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkEpochDurationIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldEpochDuration(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldEpochDuration(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoBlockProducerWrap) delFieldEpochDuration(so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) insertFieldEpochDuration(so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) checkEpochDurationIsMetMdCondition(p uint64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) GetOwner() *prototype.AccountName {
	res := true
	msg := &SoBlockProducer{}
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

func (s *SoBlockProducerWrap) GetPerTicketPrice() *prototype.Coin {
	res := true
	msg := &SoBlockProducer{}
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
				return msg.PerTicketPrice
			}
		}
	}
	if !res {
		return nil

	}
	return msg.PerTicketPrice
}

func (s *SoBlockProducerWrap) mdFieldPerTicketPrice(p *prototype.Coin, isCheck bool, isDel bool, isInsert bool,
	so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkPerTicketPriceIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldPerTicketPrice(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldPerTicketPrice(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoBlockProducerWrap) delFieldPerTicketPrice(so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) insertFieldPerTicketPrice(so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) checkPerTicketPriceIsMetMdCondition(p *prototype.Coin) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) GetPerTicketWeight() uint64 {
	res := true
	msg := &SoBlockProducer{}
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
				return msg.PerTicketWeight
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.PerTicketWeight
}

func (s *SoBlockProducerWrap) mdFieldPerTicketWeight(p uint64, isCheck bool, isDel bool, isInsert bool,
	so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkPerTicketWeightIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldPerTicketWeight(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldPerTicketWeight(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoBlockProducerWrap) delFieldPerTicketWeight(so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) insertFieldPerTicketWeight(so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) checkPerTicketWeightIsMetMdCondition(p uint64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) GetProposedStaminaFree() uint64 {
	res := true
	msg := &SoBlockProducer{}
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
				return msg.ProposedStaminaFree
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.ProposedStaminaFree
}

func (s *SoBlockProducerWrap) mdFieldProposedStaminaFree(p uint64, isCheck bool, isDel bool, isInsert bool,
	so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkProposedStaminaFreeIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldProposedStaminaFree(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldProposedStaminaFree(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoBlockProducerWrap) delFieldProposedStaminaFree(so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) insertFieldProposedStaminaFree(so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) checkProposedStaminaFreeIsMetMdCondition(p uint64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) GetSigningKey() *prototype.PublicKeyType {
	res := true
	msg := &SoBlockProducer{}
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
				return msg.SigningKey
			}
		}
	}
	if !res {
		return nil

	}
	return msg.SigningKey
}

func (s *SoBlockProducerWrap) mdFieldSigningKey(p *prototype.PublicKeyType, isCheck bool, isDel bool, isInsert bool,
	so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkSigningKeyIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldSigningKey(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldSigningKey(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoBlockProducerWrap) delFieldSigningKey(so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) insertFieldSigningKey(so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) checkSigningKeyIsMetMdCondition(p *prototype.PublicKeyType) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) GetTopNAcquireFreeToken() uint32 {
	res := true
	msg := &SoBlockProducer{}
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
				return msg.TopNAcquireFreeToken
			}
		}
	}
	if !res {
		var tmpValue uint32
		return tmpValue
	}
	return msg.TopNAcquireFreeToken
}

func (s *SoBlockProducerWrap) mdFieldTopNAcquireFreeToken(p uint32, isCheck bool, isDel bool, isInsert bool,
	so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkTopNAcquireFreeTokenIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldTopNAcquireFreeToken(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldTopNAcquireFreeToken(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoBlockProducerWrap) delFieldTopNAcquireFreeToken(so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) insertFieldTopNAcquireFreeToken(so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) checkTopNAcquireFreeTokenIsMetMdCondition(p uint32) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) GetTpsExpected() uint64 {
	res := true
	msg := &SoBlockProducer{}
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
				return msg.TpsExpected
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.TpsExpected
}

func (s *SoBlockProducerWrap) mdFieldTpsExpected(p uint64, isCheck bool, isDel bool, isInsert bool,
	so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkTpsExpectedIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldTpsExpected(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldTpsExpected(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoBlockProducerWrap) delFieldTpsExpected(so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) insertFieldTpsExpected(so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) checkTpsExpectedIsMetMdCondition(p uint64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) GetUrl() string {
	res := true
	msg := &SoBlockProducer{}
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

func (s *SoBlockProducerWrap) mdFieldUrl(p string, isCheck bool, isDel bool, isInsert bool,
	so *SoBlockProducer) bool {
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

func (s *SoBlockProducerWrap) delFieldUrl(so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) insertFieldUrl(so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) checkUrlIsMetMdCondition(p string) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) GetVoterCount() uint64 {
	res := true
	msg := &SoBlockProducer{}
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
				return msg.VoterCount
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.VoterCount
}

func (s *SoBlockProducerWrap) mdFieldVoterCount(p uint64, isCheck bool, isDel bool, isInsert bool,
	so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkVoterCountIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldVoterCount(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldVoterCount(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoBlockProducerWrap) delFieldVoterCount(so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) insertFieldVoterCount(so *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) checkVoterCountIsMetMdCondition(p uint64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

////////////// SECTION List Keys ///////////////
type SBlockProducerOwnerWrap struct {
	Dba iservices.IDatabaseRW
}

func NewBlockProducerOwnerWrap(db iservices.IDatabaseRW) *SBlockProducerOwnerWrap {
	if db == nil {
		return nil
	}
	wrap := SBlockProducerOwnerWrap{Dba: db}
	return &wrap
}

func (s *SBlockProducerOwnerWrap) GetMainVal(val []byte) *prototype.AccountName {
	res := &SoListBlockProducerByOwner{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Owner

}

func (s *SBlockProducerOwnerWrap) GetSubVal(val []byte) *prototype.AccountName {
	res := &SoListBlockProducerByOwner{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.Owner

}

func (m *SoListBlockProducerByOwner) OpeEncode() ([]byte, error) {
	pre := BlockProducerOwnerTable
	sub := m.Owner
	if sub == nil {
		return nil, errors.New("the pro Owner is nil")
	}
	sub1 := m.Owner
	if sub1 == nil {
		return nil, errors.New("the mainkey Owner is nil")
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
func (s *SBlockProducerOwnerWrap) ForEachByOrder(start *prototype.AccountName, end *prototype.AccountName, lastMainKey *prototype.AccountName,
	lastSubVal *prototype.AccountName, f func(mVal *prototype.AccountName, sVal *prototype.AccountName, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := BlockProducerOwnerTable
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
type SBlockProducerBpVestWrap struct {
	Dba iservices.IDatabaseRW
}

func NewBlockProducerBpVestWrap(db iservices.IDatabaseRW) *SBlockProducerBpVestWrap {
	if db == nil {
		return nil
	}
	wrap := SBlockProducerBpVestWrap{Dba: db}
	return &wrap
}

func (s *SBlockProducerBpVestWrap) GetMainVal(val []byte) *prototype.AccountName {
	res := &SoListBlockProducerByBpVest{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Owner

}

func (s *SBlockProducerBpVestWrap) GetSubVal(val []byte) *prototype.BpVestId {
	res := &SoListBlockProducerByBpVest{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.BpVest

}

func (m *SoListBlockProducerByBpVest) OpeEncode() ([]byte, error) {
	pre := BlockProducerBpVestTable
	sub := m.BpVest
	if sub == nil {
		return nil, errors.New("the pro BpVest is nil")
	}
	sub1 := m.Owner
	if sub1 == nil {
		return nil, errors.New("the mainkey Owner is nil")
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
func (s *SBlockProducerBpVestWrap) ForEachByRevOrder(start *prototype.BpVestId, end *prototype.BpVestId, lastMainKey *prototype.AccountName,
	lastSubVal *prototype.BpVestId, f func(mVal *prototype.AccountName, sVal *prototype.BpVestId, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := BlockProducerBpVestTable
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

func (s *SoBlockProducerWrap) update(sa *SoBlockProducer) bool {
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

func (s *SoBlockProducerWrap) getBlockProducer() *SoBlockProducer {
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

	res := &SoBlockProducer{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoBlockProducerWrap) updateBlockProducer(so *SoBlockProducer) error {
	if s.dba == nil {
		return errors.New("update fail:the db is nil")
	}

	if so == nil {
		return errors.New("update fail: the SoBlockProducer is nil")
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

func (s *SoBlockProducerWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := BlockProducerOwnerRow
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

func (s *SoBlockProducerWrap) delAllUniKeys(br bool, val *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delUniKeyOwner(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoBlockProducerWrap) delUniKeysWithNames(names map[string]string, val *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if len(names["Owner"]) > 0 {
		if !s.delUniKeyOwner(val) {
			res = false
		}
	}

	return res
}

func (s *SoBlockProducerWrap) insertAllUniKeys(val *SoBlockProducer) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoBlockProducer fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyOwner(val) {
		return sucFields, errors.New("insert unique Field Owner fail while insert table ")
	}
	sucFields["Owner"] = "Owner"

	return sucFields, nil
}

func (s *SoBlockProducerWrap) delUniKeyOwner(sa *SoBlockProducer) bool {
	if s.dba == nil {
		return false
	}
	pre := BlockProducerOwnerUniTable
	kList := []interface{}{pre}
	if sa != nil {
		if sa.Owner == nil {
			return false
		}

		sub := sa.Owner
		kList = append(kList, sub)
	} else {
		sub := s.GetOwner()
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

func (s *SoBlockProducerWrap) insertUniKeyOwner(sa *SoBlockProducer) bool {
	if s.dba == nil || sa == nil {
		return false
	}

	pre := BlockProducerOwnerUniTable
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
	val := SoUniqueBlockProducerByOwner{}
	val.Owner = sa.Owner

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniBlockProducerOwnerWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniBlockProducerOwnerWrap(db iservices.IDatabaseRW) *UniBlockProducerOwnerWrap {
	if db == nil {
		return nil
	}
	wrap := UniBlockProducerOwnerWrap{Dba: db}
	return &wrap
}

func (s *UniBlockProducerOwnerWrap) UniQueryOwner(start *prototype.AccountName) *SoBlockProducerWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := BlockProducerOwnerUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueBlockProducerByOwner{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoBlockProducerWrap(s.Dba, res.Owner)

			return wrap
		}
	}
	return nil
}

////////////// SECTION Watchers ///////////////
var (
	BlockProducerRecordType = reflect.TypeOf((*SoBlockProducer)(nil)).Elem() // table record type

	BlockProducerHasAccountCreateFeeWatcher bool // any watcher on member AccountCreateFee?

	BlockProducerHasBpVestWatcher bool // any watcher on member BpVest?

	BlockProducerHasCreatedTimeWatcher bool // any watcher on member CreatedTime?

	BlockProducerHasEpochDurationWatcher bool // any watcher on member EpochDuration?

	BlockProducerHasPerTicketPriceWatcher bool // any watcher on member PerTicketPrice?

	BlockProducerHasPerTicketWeightWatcher bool // any watcher on member PerTicketWeight?

	BlockProducerHasProposedStaminaFreeWatcher bool // any watcher on member ProposedStaminaFree?

	BlockProducerHasSigningKeyWatcher bool // any watcher on member SigningKey?

	BlockProducerHasTopNAcquireFreeTokenWatcher bool // any watcher on member TopNAcquireFreeToken?

	BlockProducerHasTpsExpectedWatcher bool // any watcher on member TpsExpected?

	BlockProducerHasUrlWatcher bool // any watcher on member Url?

	BlockProducerHasVoterCountWatcher bool // any watcher on member VoterCount?

	BlockProducerHasWholeWatcher bool // any watcher on the whole record?
	BlockProducerHasAnyWatcher   bool // any watcher?
)

func BlockProducerRecordWatcherChanged() {
	BlockProducerHasWholeWatcher = HasTableRecordWatcher(BlockProducerRecordType, "")
	BlockProducerHasAnyWatcher = BlockProducerHasWholeWatcher

	BlockProducerHasAccountCreateFeeWatcher = HasTableRecordWatcher(BlockProducerRecordType, "AccountCreateFee")
	BlockProducerHasAnyWatcher = BlockProducerHasAnyWatcher || BlockProducerHasAccountCreateFeeWatcher

	BlockProducerHasBpVestWatcher = HasTableRecordWatcher(BlockProducerRecordType, "BpVest")
	BlockProducerHasAnyWatcher = BlockProducerHasAnyWatcher || BlockProducerHasBpVestWatcher

	BlockProducerHasCreatedTimeWatcher = HasTableRecordWatcher(BlockProducerRecordType, "CreatedTime")
	BlockProducerHasAnyWatcher = BlockProducerHasAnyWatcher || BlockProducerHasCreatedTimeWatcher

	BlockProducerHasEpochDurationWatcher = HasTableRecordWatcher(BlockProducerRecordType, "EpochDuration")
	BlockProducerHasAnyWatcher = BlockProducerHasAnyWatcher || BlockProducerHasEpochDurationWatcher

	BlockProducerHasPerTicketPriceWatcher = HasTableRecordWatcher(BlockProducerRecordType, "PerTicketPrice")
	BlockProducerHasAnyWatcher = BlockProducerHasAnyWatcher || BlockProducerHasPerTicketPriceWatcher

	BlockProducerHasPerTicketWeightWatcher = HasTableRecordWatcher(BlockProducerRecordType, "PerTicketWeight")
	BlockProducerHasAnyWatcher = BlockProducerHasAnyWatcher || BlockProducerHasPerTicketWeightWatcher

	BlockProducerHasProposedStaminaFreeWatcher = HasTableRecordWatcher(BlockProducerRecordType, "ProposedStaminaFree")
	BlockProducerHasAnyWatcher = BlockProducerHasAnyWatcher || BlockProducerHasProposedStaminaFreeWatcher

	BlockProducerHasSigningKeyWatcher = HasTableRecordWatcher(BlockProducerRecordType, "SigningKey")
	BlockProducerHasAnyWatcher = BlockProducerHasAnyWatcher || BlockProducerHasSigningKeyWatcher

	BlockProducerHasTopNAcquireFreeTokenWatcher = HasTableRecordWatcher(BlockProducerRecordType, "TopNAcquireFreeToken")
	BlockProducerHasAnyWatcher = BlockProducerHasAnyWatcher || BlockProducerHasTopNAcquireFreeTokenWatcher

	BlockProducerHasTpsExpectedWatcher = HasTableRecordWatcher(BlockProducerRecordType, "TpsExpected")
	BlockProducerHasAnyWatcher = BlockProducerHasAnyWatcher || BlockProducerHasTpsExpectedWatcher

	BlockProducerHasUrlWatcher = HasTableRecordWatcher(BlockProducerRecordType, "Url")
	BlockProducerHasAnyWatcher = BlockProducerHasAnyWatcher || BlockProducerHasUrlWatcher

	BlockProducerHasVoterCountWatcher = HasTableRecordWatcher(BlockProducerRecordType, "VoterCount")
	BlockProducerHasAnyWatcher = BlockProducerHasAnyWatcher || BlockProducerHasVoterCountWatcher

}

func init() {
	RegisterTableWatcherChangedCallback(BlockProducerRecordType, BlockProducerRecordWatcherChanged)
}
