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
	DemoOwnerTable        uint32 = 1920714703
	DemoPostTimeTable     uint32 = 2261075900
	DemoLikeCountTable    uint32 = 418391101
	DemoIdxTable          uint32 = 2303787796
	DemoReplayCountTable  uint32 = 1154759843
	DemoTaglistTable      uint32 = 918597048
	DemoIdxUniTable       uint32 = 586852864
	DemoLikeCountUniTable uint32 = 1853028069
	DemoOwnerUniTable     uint32 = 3607866294

	DemoOwnerRow uint32 = 4002792218
)

////////////// SECTION Wrap Define ///////////////
type SoDemoWrap struct {
	dba       iservices.IDatabaseRW
	mainKey   *prototype.AccountName
	mKeyFlag  int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf   []byte //the buffer after the main key is encoded with prefix
	mBuf      []byte //the value after the main key is encoded
	mdFuncMap map[string]interface{}
}

func NewSoDemoWrap(dba iservices.IDatabaseRW, key *prototype.AccountName) *SoDemoWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoDemoWrap{dba, key, -1, nil, nil, nil}
	return result
}

func (s *SoDemoWrap) CheckExist() bool {
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

func (s *SoDemoWrap) Create(f func(tInfo *SoDemo)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoDemo{}
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

func (s *SoDemoWrap) getMainKeyBuf() ([]byte, error) {
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

func (s *SoDemoWrap) Md(f func(tInfo *SoDemo)) error {
	t := &SoDemo{}
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

	sa := s.getDemo()
	if sa == nil {
		return errors.New("fail to get table SoDemo")
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
	err = s.updateDemo(sa)
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

func (s *SoDemoWrap) handleFieldMd(t FieldMdHandleType, so *SoDemo, fMap map[string]interface{}) error {
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

func (s *SoDemoWrap) delSortKeyOwner(sa *SoDemo) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListDemoByOwner{}
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

func (s *SoDemoWrap) insertSortKeyOwner(sa *SoDemo) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListDemoByOwner{}
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

func (s *SoDemoWrap) delSortKeyPostTime(sa *SoDemo) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListDemoByPostTime{}
	if sa == nil {
		val.PostTime = s.GetPostTime()
		val.Owner = s.mainKey

	} else {
		val.PostTime = sa.PostTime
		val.Owner = sa.Owner
	}

	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoDemoWrap) insertSortKeyPostTime(sa *SoDemo) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListDemoByPostTime{}
	val.Owner = sa.Owner
	val.PostTime = sa.PostTime
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

func (s *SoDemoWrap) delSortKeyLikeCount(sa *SoDemo) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListDemoByLikeCount{}
	if sa == nil {
		val.LikeCount = s.GetLikeCount()
		val.Owner = s.mainKey

	} else {
		val.LikeCount = sa.LikeCount
		val.Owner = sa.Owner
	}

	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoDemoWrap) insertSortKeyLikeCount(sa *SoDemo) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListDemoByLikeCount{}
	val.Owner = sa.Owner
	val.LikeCount = sa.LikeCount
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

func (s *SoDemoWrap) delSortKeyIdx(sa *SoDemo) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListDemoByIdx{}
	if sa == nil {
		val.Idx = s.GetIdx()
		val.Owner = s.mainKey

	} else {
		val.Idx = sa.Idx
		val.Owner = sa.Owner
	}

	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoDemoWrap) insertSortKeyIdx(sa *SoDemo) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListDemoByIdx{}
	val.Owner = sa.Owner
	val.Idx = sa.Idx
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

func (s *SoDemoWrap) delSortKeyReplayCount(sa *SoDemo) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListDemoByReplayCount{}
	if sa == nil {
		val.ReplayCount = s.GetReplayCount()
		val.Owner = s.mainKey

	} else {
		val.ReplayCount = sa.ReplayCount
		val.Owner = sa.Owner
	}

	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoDemoWrap) insertSortKeyReplayCount(sa *SoDemo) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListDemoByReplayCount{}
	val.Owner = sa.Owner
	val.ReplayCount = sa.ReplayCount
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

func (s *SoDemoWrap) delSortKeyTaglist(sa *SoDemo) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListDemoByTaglist{}
	if sa == nil {
		val.Taglist = s.GetTaglist()
		val.Owner = s.mainKey

	} else {
		val.Taglist = sa.Taglist
		val.Owner = sa.Owner
	}

	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoDemoWrap) insertSortKeyTaglist(sa *SoDemo) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListDemoByTaglist{}
	val.Owner = sa.Owner
	val.Taglist = sa.Taglist
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

func (s *SoDemoWrap) delAllSortKeys(br bool, val *SoDemo) bool {
	if s.dba == nil {
		return false
	}
	res := true

	if !s.delSortKeyPostTime(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	if !s.delSortKeyLikeCount(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	if !s.delSortKeyIdx(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	if !s.delSortKeyReplayCount(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	if !s.delSortKeyTaglist(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoDemoWrap) insertAllSortKeys(val *SoDemo) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoDemo fail ")
	}

	if !s.insertSortKeyPostTime(val) {
		return errors.New("insert sort Field PostTime fail while insert table ")
	}

	if !s.insertSortKeyLikeCount(val) {
		return errors.New("insert sort Field LikeCount fail while insert table ")
	}

	if !s.insertSortKeyIdx(val) {
		return errors.New("insert sort Field Idx fail while insert table ")
	}

	if !s.insertSortKeyReplayCount(val) {
		return errors.New("insert sort Field ReplayCount fail while insert table ")
	}

	if !s.insertSortKeyTaglist(val) {
		return errors.New("insert sort Field Taglist fail while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoDemoWrap) RemoveDemo() bool {
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

func (s *SoDemoWrap) GetContent() string {
	res := true
	msg := &SoDemo{}
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
				return msg.Content
			}
		}
	}
	if !res {
		var tmpValue string
		return tmpValue
	}
	return msg.Content
}

func (s *SoDemoWrap) mdFieldContent(p string, isCheck bool, isDel bool, isInsert bool,
	so *SoDemo) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkContentIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldContent(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldContent(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoDemoWrap) delFieldContent(so *SoDemo) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoDemoWrap) insertFieldContent(so *SoDemo) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoDemoWrap) checkContentIsMetMdCondition(p string) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoDemoWrap) GetIdx() int64 {
	res := true
	msg := &SoDemo{}
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
				return msg.Idx
			}
		}
	}
	if !res {
		var tmpValue int64
		return tmpValue
	}
	return msg.Idx
}

func (s *SoDemoWrap) mdFieldIdx(p int64, isCheck bool, isDel bool, isInsert bool,
	so *SoDemo) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkIdxIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldIdx(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldIdx(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoDemoWrap) delFieldIdx(so *SoDemo) bool {
	if s.dba == nil {
		return false
	}
	if !s.delUniKeyIdx(so) {
		return false
	}

	if !s.delSortKeyIdx(so) {
		return false
	}

	return true
}

func (s *SoDemoWrap) insertFieldIdx(so *SoDemo) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyIdx(so) {
		return false
	}

	if !s.insertUniKeyIdx(so) {
		return false
	}

	return true
}

func (s *SoDemoWrap) checkIdxIsMetMdCondition(p int64) bool {
	if s.dba == nil {
		return false
	}
	//judge the unique value if is exist
	uniWrap := UniDemoIdxWrap{}
	uniWrap.Dba = s.dba
	res := uniWrap.UniQueryIdx(&p)
	if res != nil {
		//the unique value to be modified is already exist
		return false
	}

	return true
}

func (s *SoDemoWrap) GetLikeCount() int64 {
	res := true
	msg := &SoDemo{}
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
				return msg.LikeCount
			}
		}
	}
	if !res {
		var tmpValue int64
		return tmpValue
	}
	return msg.LikeCount
}

func (s *SoDemoWrap) mdFieldLikeCount(p int64, isCheck bool, isDel bool, isInsert bool,
	so *SoDemo) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkLikeCountIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldLikeCount(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldLikeCount(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoDemoWrap) delFieldLikeCount(so *SoDemo) bool {
	if s.dba == nil {
		return false
	}

	if !s.delUniKeyLikeCount(so) {
		return false
	}

	if !s.delSortKeyLikeCount(so) {
		return false
	}

	return true
}

func (s *SoDemoWrap) insertFieldLikeCount(so *SoDemo) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyLikeCount(so) {
		return false
	}

	if !s.insertUniKeyLikeCount(so) {
		return false
	}

	return true
}

func (s *SoDemoWrap) checkLikeCountIsMetMdCondition(p int64) bool {
	if s.dba == nil {
		return false
	}

	//judge the unique value if is exist
	uniWrap := UniDemoLikeCountWrap{}
	uniWrap.Dba = s.dba
	res := uniWrap.UniQueryLikeCount(&p)
	if res != nil {
		//the unique value to be modified is already exist
		return false
	}

	return true
}

func (s *SoDemoWrap) GetOwner() *prototype.AccountName {
	res := true
	msg := &SoDemo{}
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

func (s *SoDemoWrap) GetPostTime() *prototype.TimePointSec {
	res := true
	msg := &SoDemo{}
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
				return msg.PostTime
			}
		}
	}
	if !res {
		return nil

	}
	return msg.PostTime
}

func (s *SoDemoWrap) mdFieldPostTime(p *prototype.TimePointSec, isCheck bool, isDel bool, isInsert bool,
	so *SoDemo) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkPostTimeIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldPostTime(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldPostTime(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoDemoWrap) delFieldPostTime(so *SoDemo) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyPostTime(so) {
		return false
	}

	return true
}

func (s *SoDemoWrap) insertFieldPostTime(so *SoDemo) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyPostTime(so) {
		return false
	}

	return true
}

func (s *SoDemoWrap) checkPostTimeIsMetMdCondition(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoDemoWrap) GetReplayCount() int64 {
	res := true
	msg := &SoDemo{}
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
				return msg.ReplayCount
			}
		}
	}
	if !res {
		var tmpValue int64
		return tmpValue
	}
	return msg.ReplayCount
}

func (s *SoDemoWrap) mdFieldReplayCount(p int64, isCheck bool, isDel bool, isInsert bool,
	so *SoDemo) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkReplayCountIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldReplayCount(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldReplayCount(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoDemoWrap) delFieldReplayCount(so *SoDemo) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyReplayCount(so) {
		return false
	}

	return true
}

func (s *SoDemoWrap) insertFieldReplayCount(so *SoDemo) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyReplayCount(so) {
		return false
	}

	return true
}

func (s *SoDemoWrap) checkReplayCountIsMetMdCondition(p int64) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoDemoWrap) GetTaglist() []string {
	res := true
	msg := &SoDemo{}
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
				return msg.Taglist
			}
		}
	}
	if !res {
		var tmpValue []string
		return tmpValue
	}
	return msg.Taglist
}

func (s *SoDemoWrap) mdFieldTaglist(p []string, isCheck bool, isDel bool, isInsert bool,
	so *SoDemo) bool {
	if s.dba == nil {
		return false
	}

	if isCheck {
		res := s.checkTaglistIsMetMdCondition(p)
		if !res {
			return false
		}
	}

	if isDel {
		res := s.delFieldTaglist(so)
		if !res {
			return false
		}
	}

	if isInsert {
		res := s.insertFieldTaglist(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *SoDemoWrap) delFieldTaglist(so *SoDemo) bool {
	if s.dba == nil {
		return false
	}

	if !s.delSortKeyTaglist(so) {
		return false
	}

	return true
}

func (s *SoDemoWrap) insertFieldTaglist(so *SoDemo) bool {
	if s.dba == nil {
		return false
	}

	if !s.insertSortKeyTaglist(so) {
		return false
	}

	return true
}

func (s *SoDemoWrap) checkTaglistIsMetMdCondition(p []string) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoDemoWrap) GetTitle() string {
	res := true
	msg := &SoDemo{}
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

func (s *SoDemoWrap) mdFieldTitle(p string, isCheck bool, isDel bool, isInsert bool,
	so *SoDemo) bool {
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

func (s *SoDemoWrap) delFieldTitle(so *SoDemo) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoDemoWrap) insertFieldTitle(so *SoDemo) bool {
	if s.dba == nil {
		return false
	}

	return true
}

func (s *SoDemoWrap) checkTitleIsMetMdCondition(p string) bool {
	if s.dba == nil {
		return false
	}

	return true
}

////////////// SECTION List Keys ///////////////
type SDemoOwnerWrap struct {
	Dba iservices.IDatabaseRW
}

func NewDemoOwnerWrap(db iservices.IDatabaseRW) *SDemoOwnerWrap {
	if db == nil {
		return nil
	}
	wrap := SDemoOwnerWrap{Dba: db}
	return &wrap
}

func (s *SDemoOwnerWrap) GetMainVal(val []byte) *prototype.AccountName {
	res := &SoListDemoByOwner{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Owner

}

func (s *SDemoOwnerWrap) GetSubVal(val []byte) *prototype.AccountName {
	res := &SoListDemoByOwner{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.Owner

}

func (m *SoListDemoByOwner) OpeEncode() ([]byte, error) {
	pre := DemoOwnerTable
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
func (s *SDemoOwnerWrap) ForEachByRevOrder(start *prototype.AccountName, end *prototype.AccountName, lastMainKey *prototype.AccountName,
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
	pre := DemoOwnerTable
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
type SDemoPostTimeWrap struct {
	Dba iservices.IDatabaseRW
}

func NewDemoPostTimeWrap(db iservices.IDatabaseRW) *SDemoPostTimeWrap {
	if db == nil {
		return nil
	}
	wrap := SDemoPostTimeWrap{Dba: db}
	return &wrap
}

func (s *SDemoPostTimeWrap) GetMainVal(val []byte) *prototype.AccountName {
	res := &SoListDemoByPostTime{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Owner

}

func (s *SDemoPostTimeWrap) GetSubVal(val []byte) *prototype.TimePointSec {
	res := &SoListDemoByPostTime{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.PostTime

}

func (m *SoListDemoByPostTime) OpeEncode() ([]byte, error) {
	pre := DemoPostTimeTable
	sub := m.PostTime
	if sub == nil {
		return nil, errors.New("the pro PostTime is nil")
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
func (s *SDemoPostTimeWrap) ForEachByOrder(start *prototype.TimePointSec, end *prototype.TimePointSec, lastMainKey *prototype.AccountName,
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
	pre := DemoPostTimeTable
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
func (s *SDemoPostTimeWrap) ForEachByRevOrder(start *prototype.TimePointSec, end *prototype.TimePointSec, lastMainKey *prototype.AccountName,
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
	pre := DemoPostTimeTable
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
type SDemoLikeCountWrap struct {
	Dba iservices.IDatabaseRW
}

func NewDemoLikeCountWrap(db iservices.IDatabaseRW) *SDemoLikeCountWrap {
	if db == nil {
		return nil
	}
	wrap := SDemoLikeCountWrap{Dba: db}
	return &wrap
}

func (s *SDemoLikeCountWrap) GetMainVal(val []byte) *prototype.AccountName {
	res := &SoListDemoByLikeCount{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Owner

}

func (s *SDemoLikeCountWrap) GetSubVal(val []byte) *int64 {
	res := &SoListDemoByLikeCount{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return &res.LikeCount

}

func (m *SoListDemoByLikeCount) OpeEncode() ([]byte, error) {
	pre := DemoLikeCountTable
	sub := m.LikeCount

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
func (s *SDemoLikeCountWrap) ForEachByRevOrder(start *int64, end *int64, lastMainKey *prototype.AccountName,
	lastSubVal *int64, f func(mVal *prototype.AccountName, sVal *int64, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := DemoLikeCountTable
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
type SDemoIdxWrap struct {
	Dba iservices.IDatabaseRW
}

func NewDemoIdxWrap(db iservices.IDatabaseRW) *SDemoIdxWrap {
	if db == nil {
		return nil
	}
	wrap := SDemoIdxWrap{Dba: db}
	return &wrap
}

func (s *SDemoIdxWrap) GetMainVal(val []byte) *prototype.AccountName {
	res := &SoListDemoByIdx{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Owner

}

func (s *SDemoIdxWrap) GetSubVal(val []byte) *int64 {
	res := &SoListDemoByIdx{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return &res.Idx

}

func (m *SoListDemoByIdx) OpeEncode() ([]byte, error) {
	pre := DemoIdxTable
	sub := m.Idx

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
func (s *SDemoIdxWrap) ForEachByRevOrder(start *int64, end *int64, lastMainKey *prototype.AccountName,
	lastSubVal *int64, f func(mVal *prototype.AccountName, sVal *int64, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := DemoIdxTable
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
type SDemoReplayCountWrap struct {
	Dba iservices.IDatabaseRW
}

func NewDemoReplayCountWrap(db iservices.IDatabaseRW) *SDemoReplayCountWrap {
	if db == nil {
		return nil
	}
	wrap := SDemoReplayCountWrap{Dba: db}
	return &wrap
}

func (s *SDemoReplayCountWrap) GetMainVal(val []byte) *prototype.AccountName {
	res := &SoListDemoByReplayCount{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Owner

}

func (s *SDemoReplayCountWrap) GetSubVal(val []byte) *int64 {
	res := &SoListDemoByReplayCount{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return &res.ReplayCount

}

func (m *SoListDemoByReplayCount) OpeEncode() ([]byte, error) {
	pre := DemoReplayCountTable
	sub := m.ReplayCount

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
func (s *SDemoReplayCountWrap) ForEachByOrder(start *int64, end *int64, lastMainKey *prototype.AccountName,
	lastSubVal *int64, f func(mVal *prototype.AccountName, sVal *int64, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := DemoReplayCountTable
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
type SDemoTaglistWrap struct {
	Dba iservices.IDatabaseRW
}

func NewDemoTaglistWrap(db iservices.IDatabaseRW) *SDemoTaglistWrap {
	if db == nil {
		return nil
	}
	wrap := SDemoTaglistWrap{Dba: db}
	return &wrap
}

func (s *SDemoTaglistWrap) GetMainVal(val []byte) *prototype.AccountName {
	res := &SoListDemoByTaglist{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Owner

}

func (s *SDemoTaglistWrap) GetSubVal(val []byte) *[]string {
	res := &SoListDemoByTaglist{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return &res.Taglist

}

func (m *SoListDemoByTaglist) OpeEncode() ([]byte, error) {
	pre := DemoTaglistTable
	sub := m.Taglist

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
func (s *SDemoTaglistWrap) ForEachByOrder(start *[]string, end *[]string, lastMainKey *prototype.AccountName,
	lastSubVal *[]string, f func(mVal *prototype.AccountName, sVal *[]string, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := DemoTaglistTable
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

func (s *SoDemoWrap) update(sa *SoDemo) bool {
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

func (s *SoDemoWrap) getDemo() *SoDemo {
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

	res := &SoDemo{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoDemoWrap) updateDemo(so *SoDemo) error {
	if s.dba == nil {
		return errors.New("update fail:the db is nil")
	}

	if so == nil {
		return errors.New("update fail: the SoDemo is nil")
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

func (s *SoDemoWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := DemoOwnerRow
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

func (s *SoDemoWrap) delAllUniKeys(br bool, val *SoDemo) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delUniKeyIdx(val) {
		if br {
			return false
		} else {
			res = false
		}
	}
	if !s.delUniKeyLikeCount(val) {
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

func (s *SoDemoWrap) delUniKeysWithNames(names map[string]string, val *SoDemo) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if len(names["Idx"]) > 0 {
		if !s.delUniKeyIdx(val) {
			res = false
		}
	}
	if len(names["LikeCount"]) > 0 {
		if !s.delUniKeyLikeCount(val) {
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

func (s *SoDemoWrap) insertAllUniKeys(val *SoDemo) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoDemo fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyIdx(val) {
		return sucFields, errors.New("insert unique Field Idx fail while insert table ")
	}
	sucFields["Idx"] = "Idx"
	if !s.insertUniKeyLikeCount(val) {
		return sucFields, errors.New("insert unique Field LikeCount fail while insert table ")
	}
	sucFields["LikeCount"] = "LikeCount"
	if !s.insertUniKeyOwner(val) {
		return sucFields, errors.New("insert unique Field Owner fail while insert table ")
	}
	sucFields["Owner"] = "Owner"

	return sucFields, nil
}

func (s *SoDemoWrap) delUniKeyIdx(sa *SoDemo) bool {
	if s.dba == nil {
		return false
	}
	pre := DemoIdxUniTable
	kList := []interface{}{pre}
	if sa != nil {

		sub := sa.Idx
		kList = append(kList, sub)
	} else {
		sub := s.GetIdx()
		kList = append(kList, sub)

	}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoDemoWrap) insertUniKeyIdx(sa *SoDemo) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	pre := DemoIdxUniTable
	sub := sa.Idx
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
	val := SoUniqueDemoByIdx{}
	val.Owner = sa.Owner
	val.Idx = sa.Idx

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniDemoIdxWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniDemoIdxWrap(db iservices.IDatabaseRW) *UniDemoIdxWrap {
	if db == nil {
		return nil
	}
	wrap := UniDemoIdxWrap{Dba: db}
	return &wrap
}

func (s *UniDemoIdxWrap) UniQueryIdx(start *int64) *SoDemoWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := DemoIdxUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueDemoByIdx{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoDemoWrap(s.Dba, res.Owner)

			return wrap
		}
	}
	return nil
}

func (s *SoDemoWrap) delUniKeyLikeCount(sa *SoDemo) bool {
	if s.dba == nil {
		return false
	}
	pre := DemoLikeCountUniTable
	kList := []interface{}{pre}
	if sa != nil {

		sub := sa.LikeCount
		kList = append(kList, sub)
	} else {
		sub := s.GetLikeCount()
		kList = append(kList, sub)

	}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoDemoWrap) insertUniKeyLikeCount(sa *SoDemo) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	pre := DemoLikeCountUniTable
	sub := sa.LikeCount
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
	val := SoUniqueDemoByLikeCount{}
	val.Owner = sa.Owner
	val.LikeCount = sa.LikeCount

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniDemoLikeCountWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniDemoLikeCountWrap(db iservices.IDatabaseRW) *UniDemoLikeCountWrap {
	if db == nil {
		return nil
	}
	wrap := UniDemoLikeCountWrap{Dba: db}
	return &wrap
}

func (s *UniDemoLikeCountWrap) UniQueryLikeCount(start *int64) *SoDemoWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := DemoLikeCountUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueDemoByLikeCount{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoDemoWrap(s.Dba, res.Owner)

			return wrap
		}
	}
	return nil
}

func (s *SoDemoWrap) delUniKeyOwner(sa *SoDemo) bool {
	if s.dba == nil {
		return false
	}
	pre := DemoOwnerUniTable
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

func (s *SoDemoWrap) insertUniKeyOwner(sa *SoDemo) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	pre := DemoOwnerUniTable
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
	val := SoUniqueDemoByOwner{}
	val.Owner = sa.Owner

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type UniDemoOwnerWrap struct {
	Dba iservices.IDatabaseRW
}

func NewUniDemoOwnerWrap(db iservices.IDatabaseRW) *UniDemoOwnerWrap {
	if db == nil {
		return nil
	}
	wrap := UniDemoOwnerWrap{Dba: db}
	return &wrap
}

func (s *UniDemoOwnerWrap) UniQueryOwner(start *prototype.AccountName) *SoDemoWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := DemoOwnerUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueDemoByOwner{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoDemoWrap(s.Dba, res.Owner)

			return wrap
		}
	}
	return nil
}

func (s *SoDemoWrap) getMdFuncMap() map[string]interface{} {
	if s.mdFuncMap != nil && len(s.mdFuncMap) > 0 {
		return s.mdFuncMap
	}
	m := map[string]interface{}{}

	m["Content"] = s.mdFieldContent

	m["Idx"] = s.mdFieldIdx

	m["LikeCount"] = s.mdFieldLikeCount

	m["PostTime"] = s.mdFieldPostTime

	m["ReplayCount"] = s.mdFieldReplayCount

	m["Taglist"] = s.mdFieldTaglist

	m["Title"] = s.mdFieldTitle

	if len(m) > 0 {
		s.mdFuncMap = m
	}
	return m
}
