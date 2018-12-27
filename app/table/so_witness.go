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
	WitnessOwnerTable                = uint32(3588322158)
	WitnessVoteCountTable            = uint32(2256540653)
	WitnessOwnerUniTable             = uint32(2680327584)
	WitnessCreatedTimeCell           = uint32(732260124)
	WitnessLastAslotCell             = uint32(2989050122)
	WitnessLastConfirmedBlockNumCell = uint32(4183878646)
	WitnessLastWorkCell              = uint32(3441432781)
	WitnessOwnerCell                 = uint32(3659272213)
	WitnessPowWorkerCell             = uint32(217317251)
	WitnessRunningVersionCell        = uint32(3359126320)
	WitnessSigningKeyCell            = uint32(2433568317)
	WitnessTotalMissedCell           = uint32(348210894)
	WitnessUrlCell                   = uint32(261756480)
	WitnessVoteCountCell             = uint32(149922791)
	WitnessWitnessScheduleTypeCell   = uint32(1680633675)
)

////////////// SECTION Wrap Define ///////////////
type SoWitnessWrap struct {
	dba      iservices.IDatabaseService
	mainKey  *prototype.AccountName
	mKeyFlag int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf  []byte //the buffer after the main key is encoded with prefix
	mBuf     []byte //the value after the main key is encoded
}

func NewSoWitnessWrap(dba iservices.IDatabaseService, key *prototype.AccountName) *SoWitnessWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoWitnessWrap{dba, key, -1, nil, nil}
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
	err = s.saveAllMemKeys(val, true)
	if err != nil {
		s.delAllMemKeys(false, val)
		return err
	}

	// update srt list keys
	if err = s.insertAllSortKeys(val); err != nil {
		s.delAllSortKeys(false, val)
		s.dba.Delete(keyBuf)
		s.delAllMemKeys(false, val)
		return err
	}

	//update unique list
	if sucNames, err := s.insertAllUniKeys(val); err != nil {
		s.delAllSortKeys(false, val)
		s.delUniKeysWithNames(sucNames, val)
		s.dba.Delete(keyBuf)
		s.delAllMemKeys(false, val)
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

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoWitnessWrap) delSortKeyOwner(sa *SoWitness) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListWitnessByOwner{}
	if sa == nil {
		key, err := s.encodeMemKey("Owner")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemWitnessByOwner{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		val.Owner = ori.Owner
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
		key, err := s.encodeMemKey("VoteCount")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemWitnessByVoteCount{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		val.VoteCount = ori.VoteCount
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
	if !s.delSortKeyOwner(val) {
		if br {
			return false
		} else {
			res = false
		}
	}
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
	if !s.insertSortKeyOwner(val) {
		return errors.New("insert sort Field Owner fail while insert table ")
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
	val := &SoWitness{}
	//delete sort list key
	if res := s.delAllSortKeys(true, nil); !res {
		return false
	}

	//delete unique list
	if res := s.delAllUniKeys(true, nil); !res {
		return false
	}

	err := s.delAllMemKeys(true, val)
	if err == nil {
		s.mKeyBuf = nil
		s.mKeyFlag = -1
		return true
	} else {
		return false
	}
}

////////////// SECTION Members Get/Modify ///////////////
func (s *SoWitnessWrap) getMemKeyPrefix(fName string) uint32 {
	if fName == "CreatedTime" {
		return WitnessCreatedTimeCell
	}
	if fName == "LastAslot" {
		return WitnessLastAslotCell
	}
	if fName == "LastConfirmedBlockNum" {
		return WitnessLastConfirmedBlockNumCell
	}
	if fName == "LastWork" {
		return WitnessLastWorkCell
	}
	if fName == "Owner" {
		return WitnessOwnerCell
	}
	if fName == "PowWorker" {
		return WitnessPowWorkerCell
	}
	if fName == "RunningVersion" {
		return WitnessRunningVersionCell
	}
	if fName == "SigningKey" {
		return WitnessSigningKeyCell
	}
	if fName == "TotalMissed" {
		return WitnessTotalMissedCell
	}
	if fName == "Url" {
		return WitnessUrlCell
	}
	if fName == "VoteCount" {
		return WitnessVoteCountCell
	}
	if fName == "WitnessScheduleType" {
		return WitnessWitnessScheduleTypeCell
	}

	return 0
}

func (s *SoWitnessWrap) encodeMemKey(fName string) ([]byte, error) {
	if len(fName) < 1 || s.mainKey == nil {
		return nil, errors.New("field name or main key is empty")
	}
	pre := s.getMemKeyPrefix(fName)
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
	return kope.PackList(list), nil
}

func (s *SoWitnessWrap) saveAllMemKeys(tInfo *SoWitness, br bool) error {
	if s.dba == nil {
		return errors.New("save member Field fail , the db is nil")
	}

	if tInfo == nil {
		return errors.New("save member Field fail , the data is nil ")
	}
	var err error = nil
	errDes := ""
	if err = s.saveMemKeyCreatedTime(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "CreatedTime", err)
		}
	}
	if err = s.saveMemKeyLastAslot(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "LastAslot", err)
		}
	}
	if err = s.saveMemKeyLastConfirmedBlockNum(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "LastConfirmedBlockNum", err)
		}
	}
	if err = s.saveMemKeyLastWork(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "LastWork", err)
		}
	}
	if err = s.saveMemKeyOwner(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Owner", err)
		}
	}
	if err = s.saveMemKeyPowWorker(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "PowWorker", err)
		}
	}
	if err = s.saveMemKeyRunningVersion(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "RunningVersion", err)
		}
	}
	if err = s.saveMemKeySigningKey(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "SigningKey", err)
		}
	}
	if err = s.saveMemKeyTotalMissed(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "TotalMissed", err)
		}
	}
	if err = s.saveMemKeyUrl(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Url", err)
		}
	}
	if err = s.saveMemKeyVoteCount(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "VoteCount", err)
		}
	}
	if err = s.saveMemKeyWitnessScheduleType(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "WitnessScheduleType", err)
		}
	}

	if len(errDes) > 0 {
		return errors.New(errDes)
	}
	return err
}

func (s *SoWitnessWrap) delAllMemKeys(br bool, tInfo *SoWitness) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	t := reflect.TypeOf(*tInfo)
	errDesc := ""
	for k := 0; k < t.NumField(); k++ {
		name := t.Field(k).Name
		if len(name) > 0 && !strings.HasPrefix(name, "XXX_") {
			err := s.delMemKey(name)
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

func (s *SoWitnessWrap) delMemKey(fName string) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if len(fName) <= 0 {
		return errors.New("the field name is empty ")
	}
	key, err := s.encodeMemKey(fName)
	if err != nil {
		return err
	}
	err = s.dba.Delete(key)
	return err
}

func (s *SoWitnessWrap) saveMemKeyCreatedTime(tInfo *SoWitness) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemWitnessByCreatedTime{}
	val.CreatedTime = tInfo.CreatedTime
	key, err := s.encodeMemKey("CreatedTime")
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

func (s *SoWitnessWrap) GetCreatedTime() *prototype.TimePointSec {
	res := true
	msg := &SoMemWitnessByCreatedTime{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("CreatedTime")
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

func (s *SoWitnessWrap) MdCreatedTime(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("CreatedTime")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemWitnessByCreatedTime{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoWitness{}
	sa.Owner = s.mainKey

	sa.CreatedTime = ori.CreatedTime

	ori.CreatedTime = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.CreatedTime = p

	return true
}

func (s *SoWitnessWrap) saveMemKeyLastAslot(tInfo *SoWitness) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemWitnessByLastAslot{}
	val.LastAslot = tInfo.LastAslot
	key, err := s.encodeMemKey("LastAslot")
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

func (s *SoWitnessWrap) GetLastAslot() uint32 {
	res := true
	msg := &SoMemWitnessByLastAslot{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("LastAslot")
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

func (s *SoWitnessWrap) MdLastAslot(p uint32) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("LastAslot")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemWitnessByLastAslot{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoWitness{}
	sa.Owner = s.mainKey

	sa.LastAslot = ori.LastAslot

	ori.LastAslot = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.LastAslot = p

	return true
}

func (s *SoWitnessWrap) saveMemKeyLastConfirmedBlockNum(tInfo *SoWitness) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemWitnessByLastConfirmedBlockNum{}
	val.LastConfirmedBlockNum = tInfo.LastConfirmedBlockNum
	key, err := s.encodeMemKey("LastConfirmedBlockNum")
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

func (s *SoWitnessWrap) GetLastConfirmedBlockNum() uint32 {
	res := true
	msg := &SoMemWitnessByLastConfirmedBlockNum{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("LastConfirmedBlockNum")
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

func (s *SoWitnessWrap) MdLastConfirmedBlockNum(p uint32) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("LastConfirmedBlockNum")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemWitnessByLastConfirmedBlockNum{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoWitness{}
	sa.Owner = s.mainKey

	sa.LastConfirmedBlockNum = ori.LastConfirmedBlockNum

	ori.LastConfirmedBlockNum = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.LastConfirmedBlockNum = p

	return true
}

func (s *SoWitnessWrap) saveMemKeyLastWork(tInfo *SoWitness) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemWitnessByLastWork{}
	val.LastWork = tInfo.LastWork
	key, err := s.encodeMemKey("LastWork")
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

func (s *SoWitnessWrap) GetLastWork() *prototype.Sha256 {
	res := true
	msg := &SoMemWitnessByLastWork{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("LastWork")
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

func (s *SoWitnessWrap) MdLastWork(p *prototype.Sha256) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("LastWork")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemWitnessByLastWork{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoWitness{}
	sa.Owner = s.mainKey

	sa.LastWork = ori.LastWork

	ori.LastWork = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.LastWork = p

	return true
}

func (s *SoWitnessWrap) saveMemKeyOwner(tInfo *SoWitness) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemWitnessByOwner{}
	val.Owner = tInfo.Owner
	key, err := s.encodeMemKey("Owner")
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

func (s *SoWitnessWrap) GetOwner() *prototype.AccountName {
	res := true
	msg := &SoMemWitnessByOwner{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Owner")
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

func (s *SoWitnessWrap) saveMemKeyPowWorker(tInfo *SoWitness) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemWitnessByPowWorker{}
	val.PowWorker = tInfo.PowWorker
	key, err := s.encodeMemKey("PowWorker")
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

func (s *SoWitnessWrap) GetPowWorker() uint32 {
	res := true
	msg := &SoMemWitnessByPowWorker{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("PowWorker")
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

func (s *SoWitnessWrap) MdPowWorker(p uint32) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("PowWorker")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemWitnessByPowWorker{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoWitness{}
	sa.Owner = s.mainKey

	sa.PowWorker = ori.PowWorker

	ori.PowWorker = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.PowWorker = p

	return true
}

func (s *SoWitnessWrap) saveMemKeyRunningVersion(tInfo *SoWitness) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemWitnessByRunningVersion{}
	val.RunningVersion = tInfo.RunningVersion
	key, err := s.encodeMemKey("RunningVersion")
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

func (s *SoWitnessWrap) GetRunningVersion() uint32 {
	res := true
	msg := &SoMemWitnessByRunningVersion{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("RunningVersion")
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

func (s *SoWitnessWrap) MdRunningVersion(p uint32) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("RunningVersion")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemWitnessByRunningVersion{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoWitness{}
	sa.Owner = s.mainKey

	sa.RunningVersion = ori.RunningVersion

	ori.RunningVersion = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.RunningVersion = p

	return true
}

func (s *SoWitnessWrap) saveMemKeySigningKey(tInfo *SoWitness) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemWitnessBySigningKey{}
	val.SigningKey = tInfo.SigningKey
	key, err := s.encodeMemKey("SigningKey")
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

func (s *SoWitnessWrap) GetSigningKey() *prototype.PublicKeyType {
	res := true
	msg := &SoMemWitnessBySigningKey{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("SigningKey")
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

func (s *SoWitnessWrap) MdSigningKey(p *prototype.PublicKeyType) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("SigningKey")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemWitnessBySigningKey{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoWitness{}
	sa.Owner = s.mainKey

	sa.SigningKey = ori.SigningKey

	ori.SigningKey = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.SigningKey = p

	return true
}

func (s *SoWitnessWrap) saveMemKeyTotalMissed(tInfo *SoWitness) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemWitnessByTotalMissed{}
	val.TotalMissed = tInfo.TotalMissed
	key, err := s.encodeMemKey("TotalMissed")
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

func (s *SoWitnessWrap) GetTotalMissed() uint32 {
	res := true
	msg := &SoMemWitnessByTotalMissed{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("TotalMissed")
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

func (s *SoWitnessWrap) MdTotalMissed(p uint32) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("TotalMissed")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemWitnessByTotalMissed{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoWitness{}
	sa.Owner = s.mainKey

	sa.TotalMissed = ori.TotalMissed

	ori.TotalMissed = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.TotalMissed = p

	return true
}

func (s *SoWitnessWrap) saveMemKeyUrl(tInfo *SoWitness) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemWitnessByUrl{}
	val.Url = tInfo.Url
	key, err := s.encodeMemKey("Url")
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

func (s *SoWitnessWrap) GetUrl() string {
	res := true
	msg := &SoMemWitnessByUrl{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Url")
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

func (s *SoWitnessWrap) MdUrl(p string) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Url")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemWitnessByUrl{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoWitness{}
	sa.Owner = s.mainKey

	sa.Url = ori.Url

	ori.Url = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Url = p

	return true
}

func (s *SoWitnessWrap) saveMemKeyVoteCount(tInfo *SoWitness) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemWitnessByVoteCount{}
	val.VoteCount = tInfo.VoteCount
	key, err := s.encodeMemKey("VoteCount")
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

func (s *SoWitnessWrap) GetVoteCount() uint64 {
	res := true
	msg := &SoMemWitnessByVoteCount{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("VoteCount")
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

func (s *SoWitnessWrap) MdVoteCount(p uint64) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("VoteCount")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemWitnessByVoteCount{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoWitness{}
	sa.Owner = s.mainKey

	sa.VoteCount = ori.VoteCount

	if !s.delSortKeyVoteCount(sa) {
		return false
	}
	ori.VoteCount = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.VoteCount = p

	if !s.insertSortKeyVoteCount(sa) {
		return false
	}

	return true
}

func (s *SoWitnessWrap) saveMemKeyWitnessScheduleType(tInfo *SoWitness) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemWitnessByWitnessScheduleType{}
	val.WitnessScheduleType = tInfo.WitnessScheduleType
	key, err := s.encodeMemKey("WitnessScheduleType")
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

func (s *SoWitnessWrap) GetWitnessScheduleType() *prototype.WitnessScheduleType {
	res := true
	msg := &SoMemWitnessByWitnessScheduleType{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("WitnessScheduleType")
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
				return msg.WitnessScheduleType
			}
		}
	}
	if !res {
		return nil

	}
	return msg.WitnessScheduleType
}

func (s *SoWitnessWrap) MdWitnessScheduleType(p *prototype.WitnessScheduleType) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("WitnessScheduleType")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemWitnessByWitnessScheduleType{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoWitness{}
	sa.Owner = s.mainKey

	sa.WitnessScheduleType = ori.WitnessScheduleType

	ori.WitnessScheduleType = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.WitnessScheduleType = p

	return true
}

////////////// SECTION List Keys ///////////////
type SWitnessOwnerWrap struct {
	Dba iservices.IDatabaseService
}

func NewWitnessOwnerWrap(db iservices.IDatabaseService) *SWitnessOwnerWrap {
	if db == nil {
		return nil
	}
	wrap := SWitnessOwnerWrap{Dba: db}
	return &wrap
}

func (s *SWitnessOwnerWrap) DelIterator(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SWitnessOwnerWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListWitnessByOwner{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Owner

}

func (s *SWitnessOwnerWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListWitnessByOwner{}
	err = proto.Unmarshal(val, res)
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
func (s *SWitnessOwnerWrap) ForEachByOrder(start *prototype.AccountName, end *prototype.AccountName,
	f func(mVal *prototype.AccountName, sVal *prototype.AccountName, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if f == nil {
		return nil
	}
	pre := WitnessOwnerTable
	skeyList := []interface{}{pre}
	if start != nil {
		skeyList = append(skeyList, start)
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
	iterator := s.Dba.NewIterator(sBuf, eBuf)
	if iterator == nil {
		return errors.New("there is no data in range")
	}
	var idx uint32 = 0
	for iterator.Next() {
		idx++
		if isContinue := f(s.GetMainVal(iterator), s.GetSubVal(iterator), idx); !isContinue {
			break
		}
	}
	s.DelIterator(iterator)
	return nil
}

////////////// SECTION List Keys ///////////////
type SWitnessVoteCountWrap struct {
	Dba iservices.IDatabaseService
}

func NewWitnessVoteCountWrap(db iservices.IDatabaseService) *SWitnessVoteCountWrap {
	if db == nil {
		return nil
	}
	wrap := SWitnessVoteCountWrap{Dba: db}
	return &wrap
}

func (s *SWitnessVoteCountWrap) DelIterator(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SWitnessVoteCountWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListWitnessByVoteCount{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Owner

}

func (s *SWitnessVoteCountWrap) GetSubVal(iterator iservices.IDatabaseIterator) *uint64 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListWitnessByVoteCount{}
	err = proto.Unmarshal(val, res)
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
func (s *SWitnessVoteCountWrap) ForEachByRevOrder(start *uint64, end *uint64,
	f func(mVal *prototype.AccountName, sVal *uint64, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if f == nil {
		return nil
	}
	pre := WitnessVoteCountTable
	skeyList := []interface{}{pre}
	if start != nil {
		skeyList = append(skeyList, start)
	} else {
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
	//reverse the start and end when create ReversedIterator to query by reverse order
	iterator := s.Dba.NewReversedIterator(eBuf, sBuf)
	if iterator == nil {
		return errors.New("there is no data in range")
	}
	var idx uint32 = 0
	for iterator.Next() {
		idx++
		if isContinue := f(s.GetMainVal(iterator), s.GetSubVal(iterator), idx); !isContinue {
			break
		}
	}
	s.DelIterator(iterator)
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

func (s *SoWitnessWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := s.getMemKeyPrefix("Owner")
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
		key, err := s.encodeMemKey("Owner")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemWitnessByOwner{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		sub := ori.Owner
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
	Dba iservices.IDatabaseService
}

func NewUniWitnessOwnerWrap(db iservices.IDatabaseService) *UniWitnessOwnerWrap {
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
