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
	WitnessOwnerTable     uint32 = 3588322158
	WitnessVoteCountTable uint32 = 2256540653
	WitnessOwnerUniTable  uint32 = 2680327584

	WitnessOwnerRow uint32 = 514612480
)

////////////// SECTION Wrap Define ///////////////
type SoWitnessWrap struct {
	dba       iservices.IDatabaseRW
	mainKey   *prototype.AccountName
	mKeyFlag  int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf   []byte //the buffer after the main key is encoded with prefix
	mBuf      []byte //the value after the main key is encoded
	mdFuncMap map[string]interface{}
}

func NewSoWitnessWrap(dba iservices.IDatabaseRW, key *prototype.AccountName) *SoWitnessWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoWitnessWrap{dba, key, -1, nil, nil, nil}
	return result
}

func (s *SoWitnessWrap) CheckExist() bool {
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

func (s *SoWitnessWrap) Create(f func(tInfo *SoWitness)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoWitness{}
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

	return nil
}

func (s *SoWitnessWrap) getMainKeyBuf() ([]byte, error) {
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

func (s *SoWitnessWrap) Md(f func(tInfo *SoWitness)) error {
	t := &SoWitness{}
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

	mKeyName := "Owner"
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

	sa := s.getWitness()
	if sa == nil {
		return errors.New("fail to get table SoWitness")
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
	err = s.updateWitness(sa)
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

func (s *SoWitnessWrap) handleFieldMd(t FieldMdHandleType, so *SoWitness, fMap map[string]interface{}) error {
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

func (s *SoWitnessWrap) delSortKeyOwner(sa *SoWitness) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListWitnessByOwner{}
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

func (s *SoWitnessWrap) insertSortKeyOwner(sa *SoWitness) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListWitnessByOwner{}
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

func (s *SoWitnessWrap) delSortKeyVoteCount(sa *SoWitness) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListWitnessByVoteCount{}
	if sa == nil {
		val.VoteCount = s.GetVoteCount()
		val.Owner = s.mainKey

	} else {
		val.VoteCount = sa.VoteCount
		val.Owner = sa.Owner
	}

	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoWitnessWrap) insertSortKeyVoteCount(sa *SoWitness) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListWitnessByVoteCount{}
	val.Owner = sa.Owner
	val.VoteCount = sa.VoteCount
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

func (s *SoWitnessWrap) delAllSortKeys(br bool, val *SoWitness) bool {
	if s.dba == nil {
		return false
	}
	res := true

	if !s.delSortKeyVoteCount(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoWitnessWrap) insertAllSortKeys(val *SoWitness) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoWitness fail ")
	}

	if !s.insertSortKeyVoteCount(val) {
		return errors.New("insert sort Field VoteCount fail while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoWitnessWrap) RemoveWitness() bool {
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

func (s *SoWitnessWrap) GetActive() bool {
	res := true
	msg := &SoWitness{}
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
				return msg.Active
			}
		}
	}
	if !res {
		var tmpValue bool
		return tmpValue
	}
	return msg.Active
}

func (s *SoWitnessWrap) mdFieldActive(p bool, isCheck bool, isDel bool, isInsert bool,
	so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkActiveIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldActive(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldActive(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoWitnessWrap) delFieldActive(so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) insertFieldActive(so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) checkActiveIsMetMdCondition(p bool) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) GetCreatedTime() *prototype.TimePointSec {
	res := true
	msg := &SoWitness{}
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

func (s *SoWitnessWrap) mdFieldCreatedTime(p *prototype.TimePointSec, isCheck bool, isDel bool, isInsert bool,
	so *SoWitness) bool {
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

func (s *SoWitnessWrap) delFieldCreatedTime(so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) insertFieldCreatedTime(so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) checkCreatedTimeIsMetMdCondition(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) GetLastAslot() uint32 {
	res := true
	msg := &SoWitness{}
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
				return msg.LastAslot
			}
		}
	}
	if !res {
		var tmpValue uint32
		return tmpValue
	}
	return msg.LastAslot
}

func (s *SoWitnessWrap) mdFieldLastAslot(p uint32, isCheck bool, isDel bool, isInsert bool,
	so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkLastAslotIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldLastAslot(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldLastAslot(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoWitnessWrap) delFieldLastAslot(so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) insertFieldLastAslot(so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) checkLastAslotIsMetMdCondition(p uint32) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) GetLastConfirmedBlockNum() uint32 {
	res := true
	msg := &SoWitness{}
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
				return msg.LastConfirmedBlockNum
			}
		}
	}
	if !res {
		var tmpValue uint32
		return tmpValue
	}
	return msg.LastConfirmedBlockNum
}

func (s *SoWitnessWrap) mdFieldLastConfirmedBlockNum(p uint32, isCheck bool, isDel bool, isInsert bool,
	so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkLastConfirmedBlockNumIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldLastConfirmedBlockNum(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldLastConfirmedBlockNum(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoWitnessWrap) delFieldLastConfirmedBlockNum(so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) insertFieldLastConfirmedBlockNum(so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) checkLastConfirmedBlockNumIsMetMdCondition(p uint32) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) GetLastWork() *prototype.Sha256 {
	res := true
	msg := &SoWitness{}
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
				return msg.LastWork
			}
		}
	}
	if !res {
		return nil

	}
	return msg.LastWork
}

func (s *SoWitnessWrap) mdFieldLastWork(p *prototype.Sha256, isCheck bool, isDel bool, isInsert bool,
	so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkLastWorkIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldLastWork(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldLastWork(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoWitnessWrap) delFieldLastWork(so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) insertFieldLastWork(so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) checkLastWorkIsMetMdCondition(p *prototype.Sha256) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) GetOwner() *prototype.AccountName {
	res := true
	msg := &SoWitness{}
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

func (s *SoWitnessWrap) GetPowWorker() uint32 {
	res := true
	msg := &SoWitness{}
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
				return msg.PowWorker
			}
		}
	}
	if !res {
		var tmpValue uint32
		return tmpValue
	}
	return msg.PowWorker
}

func (s *SoWitnessWrap) mdFieldPowWorker(p uint32, isCheck bool, isDel bool, isInsert bool,
	so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkPowWorkerIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldPowWorker(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldPowWorker(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoWitnessWrap) delFieldPowWorker(so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) insertFieldPowWorker(so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) checkPowWorkerIsMetMdCondition(p uint32) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) GetProposedStaminaFree() uint64 {
	res := true
	msg := &SoWitness{}
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

func (s *SoWitnessWrap) mdFieldProposedStaminaFree(p uint64, isCheck bool, isDel bool, isInsert bool,
	so *SoWitness) bool {
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

func (s *SoWitnessWrap) delFieldProposedStaminaFree(so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) insertFieldProposedStaminaFree(so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) checkProposedStaminaFreeIsMetMdCondition(p uint64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) GetRunningVersion() uint32 {
	res := true
	msg := &SoWitness{}
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
				return msg.RunningVersion
			}
		}
	}
	if !res {
		var tmpValue uint32
		return tmpValue
	}
	return msg.RunningVersion
}

func (s *SoWitnessWrap) mdFieldRunningVersion(p uint32, isCheck bool, isDel bool, isInsert bool,
	so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkRunningVersionIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldRunningVersion(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldRunningVersion(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoWitnessWrap) delFieldRunningVersion(so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) insertFieldRunningVersion(so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) checkRunningVersionIsMetMdCondition(p uint32) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) GetSigningKey() *prototype.PublicKeyType {
	res := true
	msg := &SoWitness{}
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

func (s *SoWitnessWrap) mdFieldSigningKey(p *prototype.PublicKeyType, isCheck bool, isDel bool, isInsert bool,
	so *SoWitness) bool {
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

func (s *SoWitnessWrap) delFieldSigningKey(so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) insertFieldSigningKey(so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) checkSigningKeyIsMetMdCondition(p *prototype.PublicKeyType) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) GetTotalMissed() uint32 {
	res := true
	msg := &SoWitness{}
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
				return msg.TotalMissed
			}
		}
	}
	if !res {
		var tmpValue uint32
		return tmpValue
	}
	return msg.TotalMissed
}

func (s *SoWitnessWrap) mdFieldTotalMissed(p uint32, isCheck bool, isDel bool, isInsert bool,
	so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkTotalMissedIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldTotalMissed(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldTotalMissed(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoWitnessWrap) delFieldTotalMissed(so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) insertFieldTotalMissed(so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) checkTotalMissedIsMetMdCondition(p uint32) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) GetTpsExpected() uint64 {
	res := true
	msg := &SoWitness{}
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

func (s *SoWitnessWrap) mdFieldTpsExpected(p uint64, isCheck bool, isDel bool, isInsert bool,
	so *SoWitness) bool {
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

func (s *SoWitnessWrap) delFieldTpsExpected(so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) insertFieldTpsExpected(so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) checkTpsExpectedIsMetMdCondition(p uint64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) GetUrl() string {
	res := true
	msg := &SoWitness{}
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

func (s *SoWitnessWrap) mdFieldUrl(p string, isCheck bool, isDel bool, isInsert bool,
	so *SoWitness) bool {
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

func (s *SoWitnessWrap) delFieldUrl(so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) insertFieldUrl(so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) checkUrlIsMetMdCondition(p string) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoWitnessWrap) GetVoteCount() uint64 {
	res := true
	msg := &SoWitness{}
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
				return msg.VoteCount
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.VoteCount
}

func (s *SoWitnessWrap) mdFieldVoteCount(p uint64, isCheck bool, isDel bool, isInsert bool,
	so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkVoteCountIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldVoteCount(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldVoteCount(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoWitnessWrap) delFieldVoteCount(so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyVoteCount(so) {
		return false
	}

	return true
}

func (s *SoWitnessWrap) insertFieldVoteCount(so *SoWitness) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyVoteCount(so) {
		return false
	}

	return true
}

func (s *SoWitnessWrap) checkVoteCountIsMetMdCondition(p uint64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

////////////// SECTION List Keys ///////////////
type SWitnessOwnerWrap struct {
	Dba iservices.IDatabaseRW
}

func NewWitnessOwnerWrap(db iservices.IDatabaseRW) *SWitnessOwnerWrap {
	if db == nil {
		return nil
	}
	wrap := SWitnessOwnerWrap{Dba: db}
	return &wrap
}

func (s *SWitnessOwnerWrap) GetMainVal(val []byte) *prototype.AccountName {
	res := &SoListWitnessByOwner{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Owner

}

func (s *SWitnessOwnerWrap) GetSubVal(val []byte) *prototype.AccountName {
	res := &SoListWitnessByOwner{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.Owner

}

func (m *SoListWitnessByOwner) OpeEncode() ([]byte, error) {
	pre := WitnessOwnerTable
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
func (s *SWitnessOwnerWrap) ForEachByOrder(start *prototype.AccountName, end *prototype.AccountName, lastMainKey *prototype.AccountName,
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
	pre := WitnessOwnerTable
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
type SWitnessVoteCountWrap struct {
	Dba iservices.IDatabaseRW
}

func NewWitnessVoteCountWrap(db iservices.IDatabaseRW) *SWitnessVoteCountWrap {
	if db == nil {
		return nil
	}
	wrap := SWitnessVoteCountWrap{Dba: db}
	return &wrap
}

func (s *SWitnessVoteCountWrap) GetMainVal(val []byte) *prototype.AccountName {
	res := &SoListWitnessByVoteCount{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Owner

}

func (s *SWitnessVoteCountWrap) GetSubVal(val []byte) *uint64 {
	res := &SoListWitnessByVoteCount{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return &res.VoteCount

}

func (m *SoListWitnessByVoteCount) OpeEncode() ([]byte, error) {
	pre := WitnessVoteCountTable
	sub := m.VoteCount

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
func (s *SWitnessVoteCountWrap) ForEachByRevOrder(start *uint64, end *uint64, lastMainKey *prototype.AccountName,
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
	pre := WitnessVoteCountTable
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

func (s *SoWitnessWrap) update(sa *SoWitness) bool {
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

func (s *SoWitnessWrap) getWitness() *SoWitness {
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

	res := &SoWitness{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoWitnessWrap) updateWitness(so *SoWitness) error {
	if s.dba == nil {
		return errors.New("update fail:the db is nil")
	}

	if so == nil {
		return errors.New("update fail: the SoWitness is nil")
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

func (s *SoWitnessWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := WitnessOwnerRow
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

func (s *SoWitnessWrap) delAllUniKeys(br bool, val *SoWitness) bool {
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

func (s *SoWitnessWrap) delUniKeysWithNames(names map[string]string, val *SoWitness) bool {
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

func (s *SoWitnessWrap) insertAllUniKeys(val *SoWitness) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoWitness fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyOwner(val) {
		return sucFields, errors.New("insert unique Field Owner fail while insert table ")
	}
	sucFields["Owner"] = "Owner"

	return sucFields, nil
}

func (s *SoWitnessWrap) delUniKeyOwner(sa *SoWitness) bool {
	if s.dba == nil {
		return false
	}
	pre := WitnessOwnerUniTable
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

func (s *SoWitnessWrap) insertUniKeyOwner(sa *SoWitness) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	pre := WitnessOwnerUniTable
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
	val := SoUniqueWitnessByOwner{}
	val.Owner = sa.Owner

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniWitnessOwnerWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniWitnessOwnerWrap(db iservices.IDatabaseRW) *UniWitnessOwnerWrap {
	if db == nil {
		return nil
	}
	wrap := UniWitnessOwnerWrap{Dba: db}
	return &wrap
}

func (s *UniWitnessOwnerWrap) UniQueryOwner(start *prototype.AccountName) *SoWitnessWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := WitnessOwnerUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueWitnessByOwner{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoWitnessWrap(s.Dba, res.Owner)

			return wrap
		}
	}
	return nil
}

func (s *SoWitnessWrap) getMdFuncMap() map[string]interface{} {
	if s.mdFuncMap != nil && len(s.mdFuncMap) > 0 {
		return s.mdFuncMap
	}
	m := map[string]interface{}{}

	m["Active"] = s.mdFieldActive

	m["CreatedTime"] = s.mdFieldCreatedTime

	m["LastAslot"] = s.mdFieldLastAslot

	m["LastConfirmedBlockNum"] = s.mdFieldLastConfirmedBlockNum

	m["LastWork"] = s.mdFieldLastWork

	m["PowWorker"] = s.mdFieldPowWorker

	m["ProposedStaminaFree"] = s.mdFieldProposedStaminaFree

	m["RunningVersion"] = s.mdFieldRunningVersion

	m["SigningKey"] = s.mdFieldSigningKey

	m["TotalMissed"] = s.mdFieldTotalMissed

	m["TpsExpected"] = s.mdFieldTpsExpected

	m["Url"] = s.mdFieldUrl

	m["VoteCount"] = s.mdFieldVoteCount

	if len(m) > 0 {
		s.mdFuncMap = m
	}
	return m
}
