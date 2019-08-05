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
	BlockProducerOwnerTable               uint32 = 2440644301
	BlockProducerBpVestTable              uint32 = 3083635068
	BlockProducerOwnerUniTable            uint32 = 404338461
	BlockProducerAccountCreateFeeCell     uint32 = 1268209114
	BlockProducerBpVestCell               uint32 = 721149257
	BlockProducerCreatedTimeCell          uint32 = 3166890368
	BlockProducerEpochDurationCell        uint32 = 3885317215
	BlockProducerOwnerCell                uint32 = 1174716537
	BlockProducerPerTicketPriceCell       uint32 = 3920773511
	BlockProducerPerTicketWeightCell      uint32 = 504810187
	BlockProducerProposedStaminaFreeCell  uint32 = 2410170773
	BlockProducerSigningKeyCell           uint32 = 609901110
	BlockProducerTopNAcquireFreeTokenCell uint32 = 2170463993
	BlockProducerTpsExpectedCell          uint32 = 164553831
	BlockProducerUrlCell                  uint32 = 3629420187
	BlockProducerVoterCountCell           uint32 = 1302707693
)

////////////// SECTION Wrap Define ///////////////
type SoBlockProducerWrap struct {
	dba      iservices.IDatabaseRW
	mainKey  *prototype.AccountName
	mKeyFlag int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf  []byte //the buffer after the main key is encoded with prefix
	mBuf     []byte //the value after the main key is encoded
}

func NewSoBlockProducerWrap(dba iservices.IDatabaseRW, key *prototype.AccountName) *SoBlockProducerWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoBlockProducerWrap{dba, key, -1, nil, nil}
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

func (s *SoBlockProducerWrap) Create(f func(tInfo *SoBlockProducer)) error {
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

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoBlockProducerWrap) delSortKeyOwner(sa *SoBlockProducer) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListBlockProducerByOwner{}
	if sa == nil {
		key, err := s.encodeMemKey("Owner")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemBlockProducerByOwner{}
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
		key, err := s.encodeMemKey("BpVest")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemBlockProducerByBpVest{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		val.BpVest = ori.BpVest
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

func (s *SoBlockProducerWrap) RemoveBlockProducer() bool {
	if s.dba == nil {
		return false
	}
	val := &SoBlockProducer{}
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
func (s *SoBlockProducerWrap) getMemKeyPrefix(fName string) uint32 {
	if fName == "AccountCreateFee" {
		return BlockProducerAccountCreateFeeCell
	}
	if fName == "BpVest" {
		return BlockProducerBpVestCell
	}
	if fName == "CreatedTime" {
		return BlockProducerCreatedTimeCell
	}
	if fName == "EpochDuration" {
		return BlockProducerEpochDurationCell
	}
	if fName == "Owner" {
		return BlockProducerOwnerCell
	}
	if fName == "PerTicketPrice" {
		return BlockProducerPerTicketPriceCell
	}
	if fName == "PerTicketWeight" {
		return BlockProducerPerTicketWeightCell
	}
	if fName == "ProposedStaminaFree" {
		return BlockProducerProposedStaminaFreeCell
	}
	if fName == "SigningKey" {
		return BlockProducerSigningKeyCell
	}
	if fName == "TopNAcquireFreeToken" {
		return BlockProducerTopNAcquireFreeTokenCell
	}
	if fName == "TpsExpected" {
		return BlockProducerTpsExpectedCell
	}
	if fName == "Url" {
		return BlockProducerUrlCell
	}
	if fName == "VoterCount" {
		return BlockProducerVoterCountCell
	}

	return 0
}

func (s *SoBlockProducerWrap) encodeMemKey(fName string) ([]byte, error) {
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

func (s *SoBlockProducerWrap) saveAllMemKeys(tInfo *SoBlockProducer, br bool) error {
	if s.dba == nil {
		return errors.New("save member Field fail , the db is nil")
	}

	if tInfo == nil {
		return errors.New("save member Field fail , the data is nil ")
	}
	var err error = nil
	errDes := ""
	if err = s.saveMemKeyAccountCreateFee(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "AccountCreateFee", err)
		}
	}
	if err = s.saveMemKeyBpVest(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "BpVest", err)
		}
	}
	if err = s.saveMemKeyCreatedTime(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "CreatedTime", err)
		}
	}
	if err = s.saveMemKeyEpochDuration(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "EpochDuration", err)
		}
	}
	if err = s.saveMemKeyOwner(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Owner", err)
		}
	}
	if err = s.saveMemKeyPerTicketPrice(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "PerTicketPrice", err)
		}
	}
	if err = s.saveMemKeyPerTicketWeight(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "PerTicketWeight", err)
		}
	}
	if err = s.saveMemKeyProposedStaminaFree(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "ProposedStaminaFree", err)
		}
	}
	if err = s.saveMemKeySigningKey(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "SigningKey", err)
		}
	}
	if err = s.saveMemKeyTopNAcquireFreeToken(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "TopNAcquireFreeToken", err)
		}
	}
	if err = s.saveMemKeyTpsExpected(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "TpsExpected", err)
		}
	}
	if err = s.saveMemKeyUrl(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Url", err)
		}
	}
	if err = s.saveMemKeyVoterCount(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "VoterCount", err)
		}
	}

	if len(errDes) > 0 {
		return errors.New(errDes)
	}
	return err
}

func (s *SoBlockProducerWrap) delAllMemKeys(br bool, tInfo *SoBlockProducer) error {
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

func (s *SoBlockProducerWrap) delMemKey(fName string) error {
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

func (s *SoBlockProducerWrap) saveMemKeyAccountCreateFee(tInfo *SoBlockProducer) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemBlockProducerByAccountCreateFee{}
	val.AccountCreateFee = tInfo.AccountCreateFee
	key, err := s.encodeMemKey("AccountCreateFee")
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

func (s *SoBlockProducerWrap) GetAccountCreateFee() *prototype.Coin {
	res := true
	msg := &SoMemBlockProducerByAccountCreateFee{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("AccountCreateFee")
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

func (s *SoBlockProducerWrap) MdAccountCreateFee(p *prototype.Coin) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("AccountCreateFee")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemBlockProducerByAccountCreateFee{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoBlockProducer{}
	sa.Owner = s.mainKey

	sa.AccountCreateFee = ori.AccountCreateFee

	ori.AccountCreateFee = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.AccountCreateFee = p

	return true
}

func (s *SoBlockProducerWrap) saveMemKeyBpVest(tInfo *SoBlockProducer) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemBlockProducerByBpVest{}
	val.BpVest = tInfo.BpVest
	key, err := s.encodeMemKey("BpVest")
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

func (s *SoBlockProducerWrap) GetBpVest() *prototype.BpVestId {
	res := true
	msg := &SoMemBlockProducerByBpVest{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("BpVest")
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

func (s *SoBlockProducerWrap) MdBpVest(p *prototype.BpVestId) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("BpVest")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemBlockProducerByBpVest{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoBlockProducer{}
	sa.Owner = s.mainKey

	sa.BpVest = ori.BpVest

	if !s.delSortKeyBpVest(sa) {
		return false
	}
	ori.BpVest = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.BpVest = p

	if !s.insertSortKeyBpVest(sa) {
		return false
	}

	return true
}

func (s *SoBlockProducerWrap) saveMemKeyCreatedTime(tInfo *SoBlockProducer) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemBlockProducerByCreatedTime{}
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

func (s *SoBlockProducerWrap) GetCreatedTime() *prototype.TimePointSec {
	res := true
	msg := &SoMemBlockProducerByCreatedTime{}
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

func (s *SoBlockProducerWrap) MdCreatedTime(p *prototype.TimePointSec) bool {
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
	ori := &SoMemBlockProducerByCreatedTime{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoBlockProducer{}
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

func (s *SoBlockProducerWrap) saveMemKeyEpochDuration(tInfo *SoBlockProducer) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemBlockProducerByEpochDuration{}
	val.EpochDuration = tInfo.EpochDuration
	key, err := s.encodeMemKey("EpochDuration")
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

func (s *SoBlockProducerWrap) GetEpochDuration() uint64 {
	res := true
	msg := &SoMemBlockProducerByEpochDuration{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("EpochDuration")
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

func (s *SoBlockProducerWrap) MdEpochDuration(p uint64) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("EpochDuration")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemBlockProducerByEpochDuration{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoBlockProducer{}
	sa.Owner = s.mainKey

	sa.EpochDuration = ori.EpochDuration

	ori.EpochDuration = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.EpochDuration = p

	return true
}

func (s *SoBlockProducerWrap) saveMemKeyOwner(tInfo *SoBlockProducer) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemBlockProducerByOwner{}
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

func (s *SoBlockProducerWrap) GetOwner() *prototype.AccountName {
	res := true
	msg := &SoMemBlockProducerByOwner{}
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

func (s *SoBlockProducerWrap) saveMemKeyPerTicketPrice(tInfo *SoBlockProducer) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemBlockProducerByPerTicketPrice{}
	val.PerTicketPrice = tInfo.PerTicketPrice
	key, err := s.encodeMemKey("PerTicketPrice")
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

func (s *SoBlockProducerWrap) GetPerTicketPrice() *prototype.Coin {
	res := true
	msg := &SoMemBlockProducerByPerTicketPrice{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("PerTicketPrice")
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

func (s *SoBlockProducerWrap) MdPerTicketPrice(p *prototype.Coin) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("PerTicketPrice")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemBlockProducerByPerTicketPrice{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoBlockProducer{}
	sa.Owner = s.mainKey

	sa.PerTicketPrice = ori.PerTicketPrice

	ori.PerTicketPrice = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.PerTicketPrice = p

	return true
}

func (s *SoBlockProducerWrap) saveMemKeyPerTicketWeight(tInfo *SoBlockProducer) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemBlockProducerByPerTicketWeight{}
	val.PerTicketWeight = tInfo.PerTicketWeight
	key, err := s.encodeMemKey("PerTicketWeight")
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

func (s *SoBlockProducerWrap) GetPerTicketWeight() uint64 {
	res := true
	msg := &SoMemBlockProducerByPerTicketWeight{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("PerTicketWeight")
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

func (s *SoBlockProducerWrap) MdPerTicketWeight(p uint64) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("PerTicketWeight")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemBlockProducerByPerTicketWeight{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoBlockProducer{}
	sa.Owner = s.mainKey

	sa.PerTicketWeight = ori.PerTicketWeight

	ori.PerTicketWeight = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.PerTicketWeight = p

	return true
}

func (s *SoBlockProducerWrap) saveMemKeyProposedStaminaFree(tInfo *SoBlockProducer) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemBlockProducerByProposedStaminaFree{}
	val.ProposedStaminaFree = tInfo.ProposedStaminaFree
	key, err := s.encodeMemKey("ProposedStaminaFree")
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

func (s *SoBlockProducerWrap) GetProposedStaminaFree() uint64 {
	res := true
	msg := &SoMemBlockProducerByProposedStaminaFree{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("ProposedStaminaFree")
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

func (s *SoBlockProducerWrap) MdProposedStaminaFree(p uint64) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("ProposedStaminaFree")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemBlockProducerByProposedStaminaFree{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoBlockProducer{}
	sa.Owner = s.mainKey

	sa.ProposedStaminaFree = ori.ProposedStaminaFree

	ori.ProposedStaminaFree = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.ProposedStaminaFree = p

	return true
}

func (s *SoBlockProducerWrap) saveMemKeySigningKey(tInfo *SoBlockProducer) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemBlockProducerBySigningKey{}
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

func (s *SoBlockProducerWrap) GetSigningKey() *prototype.PublicKeyType {
	res := true
	msg := &SoMemBlockProducerBySigningKey{}
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

func (s *SoBlockProducerWrap) MdSigningKey(p *prototype.PublicKeyType) bool {
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
	ori := &SoMemBlockProducerBySigningKey{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoBlockProducer{}
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

func (s *SoBlockProducerWrap) saveMemKeyTopNAcquireFreeToken(tInfo *SoBlockProducer) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemBlockProducerByTopNAcquireFreeToken{}
	val.TopNAcquireFreeToken = tInfo.TopNAcquireFreeToken
	key, err := s.encodeMemKey("TopNAcquireFreeToken")
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

func (s *SoBlockProducerWrap) GetTopNAcquireFreeToken() uint32 {
	res := true
	msg := &SoMemBlockProducerByTopNAcquireFreeToken{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("TopNAcquireFreeToken")
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

func (s *SoBlockProducerWrap) MdTopNAcquireFreeToken(p uint32) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("TopNAcquireFreeToken")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemBlockProducerByTopNAcquireFreeToken{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoBlockProducer{}
	sa.Owner = s.mainKey

	sa.TopNAcquireFreeToken = ori.TopNAcquireFreeToken

	ori.TopNAcquireFreeToken = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.TopNAcquireFreeToken = p

	return true
}

func (s *SoBlockProducerWrap) saveMemKeyTpsExpected(tInfo *SoBlockProducer) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemBlockProducerByTpsExpected{}
	val.TpsExpected = tInfo.TpsExpected
	key, err := s.encodeMemKey("TpsExpected")
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

func (s *SoBlockProducerWrap) GetTpsExpected() uint64 {
	res := true
	msg := &SoMemBlockProducerByTpsExpected{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("TpsExpected")
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

func (s *SoBlockProducerWrap) MdTpsExpected(p uint64) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("TpsExpected")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemBlockProducerByTpsExpected{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoBlockProducer{}
	sa.Owner = s.mainKey

	sa.TpsExpected = ori.TpsExpected

	ori.TpsExpected = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.TpsExpected = p

	return true
}

func (s *SoBlockProducerWrap) saveMemKeyUrl(tInfo *SoBlockProducer) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemBlockProducerByUrl{}
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

func (s *SoBlockProducerWrap) GetUrl() string {
	res := true
	msg := &SoMemBlockProducerByUrl{}
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

func (s *SoBlockProducerWrap) MdUrl(p string) bool {
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
	ori := &SoMemBlockProducerByUrl{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoBlockProducer{}
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

func (s *SoBlockProducerWrap) saveMemKeyVoterCount(tInfo *SoBlockProducer) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemBlockProducerByVoterCount{}
	val.VoterCount = tInfo.VoterCount
	key, err := s.encodeMemKey("VoterCount")
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

func (s *SoBlockProducerWrap) GetVoterCount() uint64 {
	res := true
	msg := &SoMemBlockProducerByVoterCount{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("VoterCount")
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

func (s *SoBlockProducerWrap) MdVoterCount(p uint64) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("VoterCount")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemBlockProducerByVoterCount{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoBlockProducer{}
	sa.Owner = s.mainKey

	sa.VoterCount = ori.VoterCount

	ori.VoterCount = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.VoterCount = p

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

func (s *SoBlockProducerWrap) encodeMainKey() ([]byte, error) {
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
		key, err := s.encodeMemKey("Owner")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemBlockProducerByOwner{}
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
