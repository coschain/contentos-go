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
	PostCreatedTable         uint32 = 3346451556
	PostCashoutBlockNumTable uint32 = 1826021466
	PostPostIdUniTable       uint32 = 157486700

	PostPostIdRow uint32 = 3809844522
)

////////////// SECTION Wrap Define ///////////////
type SoPostWrap struct {
	dba       iservices.IDatabaseRW
	mainKey   *uint64
	mKeyFlag  int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf   []byte //the buffer after the main key is encoded with prefix
	mBuf      []byte //the value after the main key is encoded
	mdFuncMap map[string]interface{}
}

func NewSoPostWrap(dba iservices.IDatabaseRW, key *uint64) *SoPostWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoPostWrap{dba, key, -1, nil, nil, nil}
	return result
}

func (s *SoPostWrap) CheckExist() bool {
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

func (s *SoPostWrap) Create(f func(tInfo *SoPost)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoPost{}
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

	return nil
}

func (s *SoPostWrap) getMainKeyBuf() ([]byte, error) {
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

func (s *SoPostWrap) Md(f func(tInfo *SoPost)) error {
	t := &SoPost{}
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

	mKeyName := "PostId"
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

	sa := s.getPost()
	if sa == nil {
		return errors.New("fail to get table SoPost")
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
	err = s.updatePost(sa)
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

func (s *SoPostWrap) handleFieldMd(t FieldMdHandleType, so *SoPost, fMap map[string]interface{}) error {
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

func (s *SoPostWrap) delSortKeyCreated(sa *SoPost) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListPostByCreated{}
	if sa == nil {
		val.Created = s.GetCreated()
		val.PostId = *s.mainKey
	} else {
		val.Created = sa.Created
		val.PostId = sa.PostId
	}

	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoPostWrap) insertSortKeyCreated(sa *SoPost) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListPostByCreated{}
	val.PostId = sa.PostId
	val.Created = sa.Created
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

func (s *SoPostWrap) delSortKeyCashoutBlockNum(sa *SoPost) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListPostByCashoutBlockNum{}
	if sa == nil {
		val.CashoutBlockNum = s.GetCashoutBlockNum()
		val.PostId = *s.mainKey
	} else {
		val.CashoutBlockNum = sa.CashoutBlockNum
		val.PostId = sa.PostId
	}

	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoPostWrap) insertSortKeyCashoutBlockNum(sa *SoPost) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListPostByCashoutBlockNum{}
	val.PostId = sa.PostId
	val.CashoutBlockNum = sa.CashoutBlockNum
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

func (s *SoPostWrap) delAllSortKeys(br bool, val *SoPost) bool {
	if s.dba == nil {
		return false
	}
	res := true

	if !s.delSortKeyCreated(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	if !s.delSortKeyCashoutBlockNum(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoPostWrap) insertAllSortKeys(val *SoPost) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoPost fail ")
	}

	if !s.insertSortKeyCreated(val) {
		return errors.New("insert sort Field Created fail while insert table ")
	}

	if !s.insertSortKeyCashoutBlockNum(val) {
		return errors.New("insert sort Field CashoutBlockNum fail while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoPostWrap) RemovePost() bool {
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

func (s *SoPostWrap) GetAuthor() *prototype.AccountName {
	res := true
	msg := &SoPost{}
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
				return msg.Author
			}
		}
	}
	if !res {
		return nil

	}
	return msg.Author
}

func (s *SoPostWrap) mdFieldAuthor(p *prototype.AccountName, isCheck bool, isDel bool, isInsert bool,
	so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkAuthorIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldAuthor(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldAuthor(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoPostWrap) delFieldAuthor(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) insertFieldAuthor(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) checkAuthorIsMetMdCondition(p *prototype.AccountName) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) GetBeneficiaries() []*prototype.BeneficiaryRouteType {
	res := true
	msg := &SoPost{}
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
				return msg.Beneficiaries
			}
		}
	}
	if !res {
		var tmpValue []*prototype.BeneficiaryRouteType
		return tmpValue
	}
	return msg.Beneficiaries
}

func (s *SoPostWrap) mdFieldBeneficiaries(p []*prototype.BeneficiaryRouteType, isCheck bool, isDel bool, isInsert bool,
	so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkBeneficiariesIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldBeneficiaries(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldBeneficiaries(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoPostWrap) delFieldBeneficiaries(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) insertFieldBeneficiaries(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) checkBeneficiariesIsMetMdCondition(p []*prototype.BeneficiaryRouteType) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) GetBody() string {
	res := true
	msg := &SoPost{}
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
				return msg.Body
			}
		}
	}
	if !res {
		var tmpValue string
		return tmpValue
	}
	return msg.Body
}

func (s *SoPostWrap) mdFieldBody(p string, isCheck bool, isDel bool, isInsert bool,
	so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkBodyIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldBody(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldBody(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoPostWrap) delFieldBody(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) insertFieldBody(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) checkBodyIsMetMdCondition(p string) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) GetCashoutBlockNum() uint64 {
	res := true
	msg := &SoPost{}
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
				return msg.CashoutBlockNum
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.CashoutBlockNum
}

func (s *SoPostWrap) mdFieldCashoutBlockNum(p uint64, isCheck bool, isDel bool, isInsert bool,
	so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkCashoutBlockNumIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldCashoutBlockNum(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldCashoutBlockNum(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoPostWrap) delFieldCashoutBlockNum(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyCashoutBlockNum(so) {
		return false
	}

	return true
}

func (s *SoPostWrap) insertFieldCashoutBlockNum(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyCashoutBlockNum(so) {
		return false
	}

	return true
}

func (s *SoPostWrap) checkCashoutBlockNumIsMetMdCondition(p uint64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) GetCategory() string {
	res := true
	msg := &SoPost{}
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
				return msg.Category
			}
		}
	}
	if !res {
		var tmpValue string
		return tmpValue
	}
	return msg.Category
}

func (s *SoPostWrap) mdFieldCategory(p string, isCheck bool, isDel bool, isInsert bool,
	so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkCategoryIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldCategory(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldCategory(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoPostWrap) delFieldCategory(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) insertFieldCategory(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) checkCategoryIsMetMdCondition(p string) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) GetChildren() uint32 {
	res := true
	msg := &SoPost{}
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
				return msg.Children
			}
		}
	}
	if !res {
		var tmpValue uint32
		return tmpValue
	}
	return msg.Children
}

func (s *SoPostWrap) mdFieldChildren(p uint32, isCheck bool, isDel bool, isInsert bool,
	so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkChildrenIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldChildren(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldChildren(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoPostWrap) delFieldChildren(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) insertFieldChildren(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) checkChildrenIsMetMdCondition(p uint32) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) GetCreated() *prototype.TimePointSec {
	res := true
	msg := &SoPost{}
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
				return msg.Created
			}
		}
	}
	if !res {
		return nil

	}
	return msg.Created
}

func (s *SoPostWrap) mdFieldCreated(p *prototype.TimePointSec, isCheck bool, isDel bool, isInsert bool,
	so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkCreatedIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldCreated(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldCreated(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoPostWrap) delFieldCreated(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyCreated(so) {
		return false
	}

	return true
}

func (s *SoPostWrap) insertFieldCreated(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyCreated(so) {
		return false
	}

	return true
}

func (s *SoPostWrap) checkCreatedIsMetMdCondition(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) GetDappRewards() *prototype.Vest {
	res := true
	msg := &SoPost{}
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
				return msg.DappRewards
			}
		}
	}
	if !res {
		return nil

	}
	return msg.DappRewards
}

func (s *SoPostWrap) mdFieldDappRewards(p *prototype.Vest, isCheck bool, isDel bool, isInsert bool,
	so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkDappRewardsIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldDappRewards(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldDappRewards(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoPostWrap) delFieldDappRewards(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) insertFieldDappRewards(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) checkDappRewardsIsMetMdCondition(p *prototype.Vest) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) GetDepth() uint32 {
	res := true
	msg := &SoPost{}
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
				return msg.Depth
			}
		}
	}
	if !res {
		var tmpValue uint32
		return tmpValue
	}
	return msg.Depth
}

func (s *SoPostWrap) mdFieldDepth(p uint32, isCheck bool, isDel bool, isInsert bool,
	so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkDepthIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldDepth(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldDepth(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoPostWrap) delFieldDepth(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) insertFieldDepth(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) checkDepthIsMetMdCondition(p uint32) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) GetLastPayout() *prototype.TimePointSec {
	res := true
	msg := &SoPost{}
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
				return msg.LastPayout
			}
		}
	}
	if !res {
		return nil

	}
	return msg.LastPayout
}

func (s *SoPostWrap) mdFieldLastPayout(p *prototype.TimePointSec, isCheck bool, isDel bool, isInsert bool,
	so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkLastPayoutIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldLastPayout(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldLastPayout(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoPostWrap) delFieldLastPayout(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) insertFieldLastPayout(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) checkLastPayoutIsMetMdCondition(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) GetParentId() uint64 {
	res := true
	msg := &SoPost{}
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
				return msg.ParentId
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.ParentId
}

func (s *SoPostWrap) mdFieldParentId(p uint64, isCheck bool, isDel bool, isInsert bool,
	so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkParentIdIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldParentId(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldParentId(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoPostWrap) delFieldParentId(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) insertFieldParentId(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) checkParentIdIsMetMdCondition(p uint64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) GetPostId() uint64 {
	res := true
	msg := &SoPost{}
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
				return msg.PostId
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.PostId
}

func (s *SoPostWrap) GetRewards() *prototype.Vest {
	res := true
	msg := &SoPost{}
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
				return msg.Rewards
			}
		}
	}
	if !res {
		return nil

	}
	return msg.Rewards
}

func (s *SoPostWrap) mdFieldRewards(p *prototype.Vest, isCheck bool, isDel bool, isInsert bool,
	so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkRewardsIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldRewards(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldRewards(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoPostWrap) delFieldRewards(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) insertFieldRewards(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) checkRewardsIsMetMdCondition(p *prototype.Vest) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) GetRootId() uint64 {
	res := true
	msg := &SoPost{}
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
				return msg.RootId
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.RootId
}

func (s *SoPostWrap) mdFieldRootId(p uint64, isCheck bool, isDel bool, isInsert bool,
	so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkRootIdIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldRootId(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldRootId(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoPostWrap) delFieldRootId(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) insertFieldRootId(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) checkRootIdIsMetMdCondition(p uint64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) GetTags() []string {
	res := true
	msg := &SoPost{}
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
				return msg.Tags
			}
		}
	}
	if !res {
		var tmpValue []string
		return tmpValue
	}
	return msg.Tags
}

func (s *SoPostWrap) mdFieldTags(p []string, isCheck bool, isDel bool, isInsert bool,
	so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkTagsIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldTags(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldTags(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoPostWrap) delFieldTags(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) insertFieldTags(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) checkTagsIsMetMdCondition(p []string) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) GetTitle() string {
	res := true
	msg := &SoPost{}
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
				return msg.Title
			}
		}
	}
	if !res {
		var tmpValue string
		return tmpValue
	}
	return msg.Title
}

func (s *SoPostWrap) mdFieldTitle(p string, isCheck bool, isDel bool, isInsert bool,
	so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkTitleIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldTitle(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldTitle(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoPostWrap) delFieldTitle(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) insertFieldTitle(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) checkTitleIsMetMdCondition(p string) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) GetVoteCnt() uint64 {
	res := true
	msg := &SoPost{}
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
				return msg.VoteCnt
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.VoteCnt
}

func (s *SoPostWrap) mdFieldVoteCnt(p uint64, isCheck bool, isDel bool, isInsert bool,
	so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkVoteCntIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldVoteCnt(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldVoteCnt(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoPostWrap) delFieldVoteCnt(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) insertFieldVoteCnt(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) checkVoteCntIsMetMdCondition(p uint64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) GetWeightedVp() uint64 {
	res := true
	msg := &SoPost{}
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
				return msg.WeightedVp
			}
		}
	}
	if !res {
		var tmpValue uint64
		return tmpValue
	}
	return msg.WeightedVp
}

func (s *SoPostWrap) mdFieldWeightedVp(p uint64, isCheck bool, isDel bool, isInsert bool,
	so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkWeightedVpIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldWeightedVp(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldWeightedVp(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoPostWrap) delFieldWeightedVp(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) insertFieldWeightedVp(so *SoPost) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoPostWrap) checkWeightedVpIsMetMdCondition(p uint64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

////////////// SECTION List Keys ///////////////
type SPostCreatedWrap struct {
	Dba iservices.IDatabaseRW
}

func NewPostCreatedWrap(db iservices.IDatabaseRW) *SPostCreatedWrap {
	if db == nil {
		return nil
	}
	wrap := SPostCreatedWrap{Dba: db}
	return &wrap
}

func (s *SPostCreatedWrap) GetMainVal(val []byte) *uint64 {
	res := &SoListPostByCreated{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return &res.PostId

}

func (s *SPostCreatedWrap) GetSubVal(val []byte) *prototype.TimePointSec {
	res := &SoListPostByCreated{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.Created

}

func (m *SoListPostByCreated) OpeEncode() ([]byte, error) {
	pre := PostCreatedTable
	sub := m.Created
	if sub == nil {
		return nil, errors.New("the pro Created is nil")
	}
	sub1 := m.PostId

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
func (s *SPostCreatedWrap) ForEachByOrder(start *prototype.TimePointSec, end *prototype.TimePointSec, lastMainKey *uint64,
	lastSubVal *prototype.TimePointSec, f func(mVal *uint64, sVal *prototype.TimePointSec, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := PostCreatedTable
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
func (s *SPostCreatedWrap) ForEachByRevOrder(start *prototype.TimePointSec, end *prototype.TimePointSec, lastMainKey *uint64,
	lastSubVal *prototype.TimePointSec, f func(mVal *uint64, sVal *prototype.TimePointSec, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := PostCreatedTable
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
type SPostCashoutBlockNumWrap struct {
	Dba iservices.IDatabaseRW
}

func NewPostCashoutBlockNumWrap(db iservices.IDatabaseRW) *SPostCashoutBlockNumWrap {
	if db == nil {
		return nil
	}
	wrap := SPostCashoutBlockNumWrap{Dba: db}
	return &wrap
}

func (s *SPostCashoutBlockNumWrap) GetMainVal(val []byte) *uint64 {
	res := &SoListPostByCashoutBlockNum{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return &res.PostId

}

func (s *SPostCashoutBlockNumWrap) GetSubVal(val []byte) *uint64 {
	res := &SoListPostByCashoutBlockNum{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return &res.CashoutBlockNum

}

func (m *SoListPostByCashoutBlockNum) OpeEncode() ([]byte, error) {
	pre := PostCashoutBlockNumTable
	sub := m.CashoutBlockNum

	sub1 := m.PostId

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
func (s *SPostCashoutBlockNumWrap) ForEachByOrder(start *uint64, end *uint64, lastMainKey *uint64,
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
	pre := PostCashoutBlockNumTable
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

func (s *SoPostWrap) update(sa *SoPost) bool {
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

func (s *SoPostWrap) getPost() *SoPost {
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

	res := &SoPost{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoPostWrap) updatePost(so *SoPost) error {
	if s.dba == nil {
		return errors.New("update fail:the db is nil")
	}

	if so == nil {
		return errors.New("update fail: the SoPost is nil")
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

func (s *SoPostWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := PostPostIdRow
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

func (s *SoPostWrap) delAllUniKeys(br bool, val *SoPost) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delUniKeyPostId(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoPostWrap) delUniKeysWithNames(names map[string]string, val *SoPost) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if len(names["PostId"]) > 0 {
		if !s.delUniKeyPostId(val) {
			res = false
		}
	}

	return res
}

func (s *SoPostWrap) insertAllUniKeys(val *SoPost) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoPost fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyPostId(val) {
		return sucFields, errors.New("insert unique Field PostId fail while insert table ")
	}
	sucFields["PostId"] = "PostId"

	return sucFields, nil
}

func (s *SoPostWrap) delUniKeyPostId(sa *SoPost) bool {
	if s.dba == nil {
		return false
	}
	pre := PostPostIdUniTable
	kList := []interface{}{pre}
	if sa != nil {

		sub := sa.PostId
		kList = append(kList, sub)
	} else {
		sub := s.GetPostId()
		kList = append(kList, sub)

	}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoPostWrap) insertUniKeyPostId(sa *SoPost) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	pre := PostPostIdUniTable
	sub := sa.PostId
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
	val := SoUniquePostByPostId{}
	val.PostId = sa.PostId

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniPostPostIdWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniPostPostIdWrap(db iservices.IDatabaseRW) *UniPostPostIdWrap {
	if db == nil {
		return nil
	}
	wrap := UniPostPostIdWrap{Dba: db}
	return &wrap
}

func (s *UniPostPostIdWrap) UniQueryPostId(start *uint64) *SoPostWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := PostPostIdUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniquePostByPostId{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoPostWrap(s.Dba, &res.PostId)
			return wrap
		}
	}
	return nil
}

func (s *SoPostWrap) getMdFuncMap() map[string]interface{} {
	if s.mdFuncMap != nil && len(s.mdFuncMap) > 0 {
		return s.mdFuncMap
	}
	m := map[string]interface{}{}

	m["Author"] = s.mdFieldAuthor

	m["Beneficiaries"] = s.mdFieldBeneficiaries

	m["Body"] = s.mdFieldBody

	m["CashoutBlockNum"] = s.mdFieldCashoutBlockNum

	m["Category"] = s.mdFieldCategory

	m["Children"] = s.mdFieldChildren

	m["Created"] = s.mdFieldCreated

	m["DappRewards"] = s.mdFieldDappRewards

	m["Depth"] = s.mdFieldDepth

	m["LastPayout"] = s.mdFieldLastPayout

	m["ParentId"] = s.mdFieldParentId

	m["Rewards"] = s.mdFieldRewards

	m["RootId"] = s.mdFieldRootId

	m["Tags"] = s.mdFieldTags

	m["Title"] = s.mdFieldTitle

	m["VoteCnt"] = s.mdFieldVoteCnt

	m["WeightedVp"] = s.mdFieldWeightedVp

	if len(m) > 0 {
		s.mdFuncMap = m
	}
	return m
}
