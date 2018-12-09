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
	AccountTable              = []byte("AccountTable")
	AccountCreatedTimeTable   = []byte("AccountCreatedTimeTable")
	AccountBalanceTable       = []byte("AccountBalanceTable")
	AccountVestingSharesTable = []byte("AccountVestingSharesTable")
	AccountBpVoteCountTable   = []byte("AccountBpVoteCountTable")
	AccountNameUniTable       = []byte("AccountNameUniTable")
)

////////////// SECTION Wrap Define ///////////////
type SoAccountWrap struct {
	dba     iservices.IDatabaseService
	mainKey *prototype.AccountName
}

func NewSoAccountWrap(dba iservices.IDatabaseService, key *prototype.AccountName) *SoAccountWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoAccountWrap{dba, key}
	return result
}

func (s *SoAccountWrap) CheckExist() bool {
	if s.dba == nil {
		return false
	}
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return false
	}

	res, err := s.dba.Has(keyBuf)
	if err != nil {
		return false
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
	err = s.saveAllMemKeys(val, true)
	if err != nil {
		return err
	}

	// update sort list keys
	if err = s.insertAllSortKeys(val); err != nil {
		s.delAllSortKeys(false, val)
		s.dba.Delete(keyBuf)
		s.delAllMemKeys(false, val)
		return err
	}

	//update unique list
	if err = s.insertAllUniKeys(val); err != nil {
		s.delAllSortKeys(false, val)
		s.delAllUniKeys(false, val)
		s.dba.Delete(keyBuf)
		s.delAllMemKeys(false, val)
		return err
	}

	return nil
}

func (s *SoAccountWrap) encodeMemKey(fName string) ([]byte, error) {
	if len(fName) < 1 || s.mainKey == nil {
		return nil, errors.New("field name or main key is empty")
	}
	pre := "Account" + fName + "cell"
	kList := []interface{}{pre, s.mainKey}
	key, err := kope.EncodeSlice(kList)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (so *SoAccountWrap) saveAllMemKeys(tInfo *SoAccount, br bool) error {
	if so.dba == nil {
		return errors.New("save member Field fail , the db is nil")
	}

	if tInfo == nil {
		return errors.New("save member Field fail , the data is nil ")
	}
	var err error = nil
	errDes := ""
	if err = so.saveMemKeyBalance(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Balance", err)
		}
	}
	if err = so.saveMemKeyBpVoteCount(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "BpVoteCount", err)
		}
	}
	if err = so.saveMemKeyCreatedTime(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "CreatedTime", err)
		}
	}
	if err = so.saveMemKeyCreator(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Creator", err)
		}
	}
	if err = so.saveMemKeyLastPostTime(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "LastPostTime", err)
		}
	}
	if err = so.saveMemKeyLastVoteTime(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "LastVoteTime", err)
		}
	}
	if err = so.saveMemKeyName(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Name", err)
		}
	}
	if err = so.saveMemKeyVestingShares(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "VestingShares", err)
		}
	}
	if err = so.saveMemKeyVotePower(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "VotePower", err)
		}
	}

	if len(errDes) > 0 {
		return errors.New(errDes)
	}
	return err
}

func (so *SoAccountWrap) delAllMemKeys(br bool, tInfo *SoAccount) error {
	if so.dba == nil {
		return errors.New("the db is nil")
	}
	t := reflect.TypeOf(*tInfo)
	errDesc := ""
	for k := 0; k < t.NumField(); k++ {
		name := t.Field(k).Name
		if len(name) > 0 && !strings.HasPrefix(name, "XXX_") {
			err := so.delMemKey(name)
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

func (so *SoAccountWrap) delMemKey(fName string) error {
	if so.dba == nil {
		return errors.New("the db is nil")
	}
	if len(fName) <= 0 {
		return errors.New("the field name is empty ")
	}
	key, err := so.encodeMemKey(fName)
	if err != nil {
		return err
	}
	err = so.dba.Delete(key)
	return err
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoAccountWrap) delSortKeyCreatedTime(sa *SoAccount) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListAccountByCreatedTime{}
	if sa == nil {
		key, err := s.encodeMemKey("CreatedTime")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemAccountByCreatedTime{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		val.CreatedTime = ori.CreatedTime
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
		key, err := s.encodeMemKey("Balance")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemAccountByBalance{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		val.Balance = ori.Balance
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
		key, err := s.encodeMemKey("VestingShares")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemAccountByVestingShares{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		val.VestingShares = ori.VestingShares
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
		key, err := s.encodeMemKey("BpVoteCount")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemAccountByBpVoteCount{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		val.BpVoteCount = ori.BpVoteCount
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

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoAccountWrap) RemoveAccount() bool {
	if s.dba == nil {
		return false
	}
	val := &SoAccount{}
	//delete sort list key
	if res := s.delAllSortKeys(true, nil); !res {
		return false
	}

	//delete unique list
	if res := s.delAllUniKeys(true, nil); !res {
		return false
	}

	err := s.delAllMemKeys(true, val)
	return err == nil
}

////////////// SECTION Members Get/Modify ///////////////
func (s *SoAccountWrap) saveMemKeyBalance(tInfo *SoAccount) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemAccountByBalance{}
	val.Balance = tInfo.Balance
	key, err := s.encodeMemKey("Balance")
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

func (s *SoAccountWrap) GetBalance() *prototype.Coin {
	res := true
	msg := &SoMemAccountByBalance{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Balance")
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

func (s *SoAccountWrap) MdBalance(p *prototype.Coin) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Balance")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemAccountByBalance{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoAccount{}
	sa.Name = s.mainKey

	sa.Balance = ori.Balance

	if !s.delSortKeyBalance(sa) {
		return false
	}
	ori.Balance = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Balance = p

	if !s.insertSortKeyBalance(sa) {
		return false
	}

	return true
}

func (s *SoAccountWrap) saveMemKeyBpVoteCount(tInfo *SoAccount) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemAccountByBpVoteCount{}
	val.BpVoteCount = tInfo.BpVoteCount
	key, err := s.encodeMemKey("BpVoteCount")
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

func (s *SoAccountWrap) GetBpVoteCount() uint32 {
	res := true
	msg := &SoMemAccountByBpVoteCount{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("BpVoteCount")
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

func (s *SoAccountWrap) MdBpVoteCount(p uint32) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("BpVoteCount")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemAccountByBpVoteCount{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoAccount{}
	sa.Name = s.mainKey

	sa.BpVoteCount = ori.BpVoteCount

	if !s.delSortKeyBpVoteCount(sa) {
		return false
	}
	ori.BpVoteCount = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.BpVoteCount = p

	if !s.insertSortKeyBpVoteCount(sa) {
		return false
	}

	return true
}

func (s *SoAccountWrap) saveMemKeyCreatedTime(tInfo *SoAccount) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemAccountByCreatedTime{}
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

func (s *SoAccountWrap) GetCreatedTime() *prototype.TimePointSec {
	res := true
	msg := &SoMemAccountByCreatedTime{}
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

func (s *SoAccountWrap) MdCreatedTime(p *prototype.TimePointSec) bool {
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
	ori := &SoMemAccountByCreatedTime{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoAccount{}
	sa.Name = s.mainKey

	sa.CreatedTime = ori.CreatedTime

	if !s.delSortKeyCreatedTime(sa) {
		return false
	}
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

	if !s.insertSortKeyCreatedTime(sa) {
		return false
	}

	return true
}

func (s *SoAccountWrap) saveMemKeyCreator(tInfo *SoAccount) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemAccountByCreator{}
	val.Creator = tInfo.Creator
	key, err := s.encodeMemKey("Creator")
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

func (s *SoAccountWrap) GetCreator() *prototype.AccountName {
	res := true
	msg := &SoMemAccountByCreator{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Creator")
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

func (s *SoAccountWrap) MdCreator(p *prototype.AccountName) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Creator")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemAccountByCreator{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoAccount{}
	sa.Name = s.mainKey

	sa.Creator = ori.Creator

	ori.Creator = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Creator = p

	return true
}

func (s *SoAccountWrap) saveMemKeyLastPostTime(tInfo *SoAccount) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemAccountByLastPostTime{}
	val.LastPostTime = tInfo.LastPostTime
	key, err := s.encodeMemKey("LastPostTime")
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

func (s *SoAccountWrap) GetLastPostTime() *prototype.TimePointSec {
	res := true
	msg := &SoMemAccountByLastPostTime{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("LastPostTime")
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

func (s *SoAccountWrap) MdLastPostTime(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("LastPostTime")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemAccountByLastPostTime{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoAccount{}
	sa.Name = s.mainKey

	sa.LastPostTime = ori.LastPostTime

	ori.LastPostTime = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.LastPostTime = p

	return true
}

func (s *SoAccountWrap) saveMemKeyLastVoteTime(tInfo *SoAccount) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemAccountByLastVoteTime{}
	val.LastVoteTime = tInfo.LastVoteTime
	key, err := s.encodeMemKey("LastVoteTime")
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

func (s *SoAccountWrap) GetLastVoteTime() *prototype.TimePointSec {
	res := true
	msg := &SoMemAccountByLastVoteTime{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("LastVoteTime")
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

func (s *SoAccountWrap) MdLastVoteTime(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("LastVoteTime")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemAccountByLastVoteTime{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoAccount{}
	sa.Name = s.mainKey

	sa.LastVoteTime = ori.LastVoteTime

	ori.LastVoteTime = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.LastVoteTime = p

	return true
}

func (s *SoAccountWrap) saveMemKeyName(tInfo *SoAccount) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemAccountByName{}
	val.Name = tInfo.Name
	key, err := s.encodeMemKey("Name")
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

func (s *SoAccountWrap) GetName() *prototype.AccountName {
	res := true
	msg := &SoMemAccountByName{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Name")
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

func (s *SoAccountWrap) saveMemKeyVestingShares(tInfo *SoAccount) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemAccountByVestingShares{}
	val.VestingShares = tInfo.VestingShares
	key, err := s.encodeMemKey("VestingShares")
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

func (s *SoAccountWrap) GetVestingShares() *prototype.Vest {
	res := true
	msg := &SoMemAccountByVestingShares{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("VestingShares")
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

func (s *SoAccountWrap) MdVestingShares(p *prototype.Vest) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("VestingShares")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemAccountByVestingShares{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoAccount{}
	sa.Name = s.mainKey

	sa.VestingShares = ori.VestingShares

	if !s.delSortKeyVestingShares(sa) {
		return false
	}
	ori.VestingShares = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.VestingShares = p

	if !s.insertSortKeyVestingShares(sa) {
		return false
	}

	return true
}

func (s *SoAccountWrap) saveMemKeyVotePower(tInfo *SoAccount) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemAccountByVotePower{}
	val.VotePower = tInfo.VotePower
	key, err := s.encodeMemKey("VotePower")
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

func (s *SoAccountWrap) GetVotePower() uint32 {
	res := true
	msg := &SoMemAccountByVotePower{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("VotePower")
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

func (s *SoAccountWrap) MdVotePower(p uint32) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("VotePower")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemAccountByVotePower{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoAccount{}
	sa.Name = s.mainKey

	sa.VotePower = ori.VotePower

	ori.VotePower = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.VotePower = p

	return true
}

////////////// SECTION List Keys ///////////////
type SAccountCreatedTimeWrap struct {
	Dba iservices.IDatabaseService
}

func NewAccountCreatedTimeWrap(db iservices.IDatabaseService) *SAccountCreatedTimeWrap {
	if db == nil {
		return nil
	}
	wrap := SAccountCreatedTimeWrap{Dba: db}
	return &wrap
}

func (s *SAccountCreatedTimeWrap) DelIterater(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SAccountCreatedTimeWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListAccountByCreatedTime{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Name

}

func (s *SAccountCreatedTimeWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.TimePointSec {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListAccountByCreatedTime{}
	err = proto.Unmarshal(val, res)
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

//Query sort by order
//start = nil  end = nil (query the db from start to end)
//start = nil (query from start the db)
//end = nil (query to the end of db)
func (s *SAccountCreatedTimeWrap) QueryListByOrder(start *prototype.TimePointSec, end *prototype.TimePointSec) iservices.IDatabaseIterator {
	if s.Dba == nil {
		return nil
	}
	pre := AccountCreatedTimeTable
	skeyList := []interface{}{pre}
	if start != nil {
		skeyList = append(skeyList, start)
	}
	sBuf, cErr := kope.EncodeSlice(skeyList)
	if cErr != nil {
		return nil
	}
	eKeyList := []interface{}{pre}
	if end != nil {
		eKeyList = append(eKeyList, end)
	} else {
		eKeyList = append(eKeyList, kope.MaximumKey)
	}
	eBuf, cErr := kope.EncodeSlice(eKeyList)
	if cErr != nil {
		return nil
	}
	return s.Dba.NewIterator(sBuf, eBuf)
}

////////////// SECTION List Keys ///////////////
type SAccountBalanceWrap struct {
	Dba iservices.IDatabaseService
}

func NewAccountBalanceWrap(db iservices.IDatabaseService) *SAccountBalanceWrap {
	if db == nil {
		return nil
	}
	wrap := SAccountBalanceWrap{Dba: db}
	return &wrap
}

func (s *SAccountBalanceWrap) DelIterater(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SAccountBalanceWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListAccountByBalance{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Name

}

func (s *SAccountBalanceWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.Coin {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListAccountByBalance{}
	err = proto.Unmarshal(val, res)
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

//Query sort by order
//start = nil  end = nil (query the db from start to end)
//start = nil (query from start the db)
//end = nil (query to the end of db)
func (s *SAccountBalanceWrap) QueryListByOrder(start *prototype.Coin, end *prototype.Coin) iservices.IDatabaseIterator {
	if s.Dba == nil {
		return nil
	}
	pre := AccountBalanceTable
	skeyList := []interface{}{pre}
	if start != nil {
		skeyList = append(skeyList, start)
	}
	sBuf, cErr := kope.EncodeSlice(skeyList)
	if cErr != nil {
		return nil
	}
	eKeyList := []interface{}{pre}
	if end != nil {
		eKeyList = append(eKeyList, end)
	} else {
		eKeyList = append(eKeyList, kope.MaximumKey)
	}
	eBuf, cErr := kope.EncodeSlice(eKeyList)
	if cErr != nil {
		return nil
	}
	return s.Dba.NewIterator(sBuf, eBuf)
}

////////////// SECTION List Keys ///////////////
type SAccountVestingSharesWrap struct {
	Dba iservices.IDatabaseService
}

func NewAccountVestingSharesWrap(db iservices.IDatabaseService) *SAccountVestingSharesWrap {
	if db == nil {
		return nil
	}
	wrap := SAccountVestingSharesWrap{Dba: db}
	return &wrap
}

func (s *SAccountVestingSharesWrap) DelIterater(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SAccountVestingSharesWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListAccountByVestingShares{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Name

}

func (s *SAccountVestingSharesWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.Vest {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListAccountByVestingShares{}
	err = proto.Unmarshal(val, res)
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

//Query sort by order
//start = nil  end = nil (query the db from start to end)
//start = nil (query from start the db)
//end = nil (query to the end of db)
func (s *SAccountVestingSharesWrap) QueryListByOrder(start *prototype.Vest, end *prototype.Vest) iservices.IDatabaseIterator {
	if s.Dba == nil {
		return nil
	}
	pre := AccountVestingSharesTable
	skeyList := []interface{}{pre}
	if start != nil {
		skeyList = append(skeyList, start)
	}
	sBuf, cErr := kope.EncodeSlice(skeyList)
	if cErr != nil {
		return nil
	}
	eKeyList := []interface{}{pre}
	if end != nil {
		eKeyList = append(eKeyList, end)
	} else {
		eKeyList = append(eKeyList, kope.MaximumKey)
	}
	eBuf, cErr := kope.EncodeSlice(eKeyList)
	if cErr != nil {
		return nil
	}
	return s.Dba.NewIterator(sBuf, eBuf)
}

////////////// SECTION List Keys ///////////////
type SAccountBpVoteCountWrap struct {
	Dba iservices.IDatabaseService
}

func NewAccountBpVoteCountWrap(db iservices.IDatabaseService) *SAccountBpVoteCountWrap {
	if db == nil {
		return nil
	}
	wrap := SAccountBpVoteCountWrap{Dba: db}
	return &wrap
}

func (s *SAccountBpVoteCountWrap) DelIterater(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SAccountBpVoteCountWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListAccountByBpVoteCount{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Name

}

func (s *SAccountBpVoteCountWrap) GetSubVal(iterator iservices.IDatabaseIterator) *uint32 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListAccountByBpVoteCount{}
	err = proto.Unmarshal(val, res)
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

//Query sort by order
//start = nil  end = nil (query the db from start to end)
//start = nil (query from start the db)
//end = nil (query to the end of db)
func (s *SAccountBpVoteCountWrap) QueryListByOrder(start *uint32, end *uint32) iservices.IDatabaseIterator {
	if s.Dba == nil {
		return nil
	}
	pre := AccountBpVoteCountTable
	skeyList := []interface{}{pre}
	if start != nil {
		skeyList = append(skeyList, start)
	}
	sBuf, cErr := kope.EncodeSlice(skeyList)
	if cErr != nil {
		return nil
	}
	eKeyList := []interface{}{pre}
	if end != nil {
		eKeyList = append(eKeyList, end)
	} else {
		eKeyList = append(eKeyList, kope.MaximumKey)
	}
	eBuf, cErr := kope.EncodeSlice(eKeyList)
	if cErr != nil {
		return nil
	}
	return s.Dba.NewIterator(sBuf, eBuf)
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

func (s *SoAccountWrap) encodeMainKey() ([]byte, error) {
	pre := "Account" + "Name" + "cell"
	sub := s.mainKey
	if sub == nil {
		return nil, errors.New("the mainKey is nil")
	}
	kList := []interface{}{pre, sub}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
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

	return res
}

func (s *SoAccountWrap) insertAllUniKeys(val *SoAccount) error {
	if s.dba == nil {
		return errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert uniuqe Field fail,get the SoAccount fail ")
	}
	if !s.insertUniKeyName(val) {
		return errors.New("insert unique Field Name fail while insert table ")
	}

	return nil
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
		key, err := s.encodeMemKey("Name")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemAccountByName{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		sub := ori.Name
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
	uniWrap := UniAccountNameWrap{}
	uniWrap.Dba = s.dba

	res := uniWrap.UniQueryName(sa.Name)
	if res != nil {
		//the unique key is already exist
		return false
	}
	val := SoUniqueAccountByName{}
	val.Name = sa.Name

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	pre := AccountNameUniTable
	sub := sa.Name
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniAccountNameWrap struct {
	Dba iservices.IDatabaseService
}

func NewUniAccountNameWrap(db iservices.IDatabaseService) *UniAccountNameWrap {
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
