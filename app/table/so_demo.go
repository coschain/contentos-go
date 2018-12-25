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
	DemoTable             = []byte("DemoTable")
	DemoOwnerTable        = []byte("DemoOwnerTable")
	DemoPostTimeTable     = []byte("DemoPostTimeTable")
	DemoLikeCountTable    = []byte("DemoLikeCountTable")
	DemoIdxTable          = []byte("DemoIdxTable")
	DemoReplayCountTable  = []byte("DemoReplayCountTable")
	DemoTaglistTable      = []byte("DemoTaglistTable")
	DemoIdxUniTable       = []byte("DemoIdxUniTable")
	DemoLikeCountUniTable = []byte("DemoLikeCountUniTable")
	DemoOwnerUniTable     = []byte("DemoOwnerUniTable")
)

////////////// SECTION Wrap Define ///////////////
type SoDemoWrap struct {
	dba      iservices.IDatabaseService
	mainKey  *prototype.AccountName
	mKeyFlag int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf  []byte //the buffer after the main key is encoded
	mBuf     []byte //the value after the main key is encoded
}

func NewSoDemoWrap(dba iservices.IDatabaseService, key *prototype.AccountName) *SoDemoWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoDemoWrap{
		dba,
		key,
		-1,
		nil,
		nil,
	}
	return result
}

func (s *SoDemoWrap) CheckExist() bool {
	if s.dba == nil {
		return false
	}
	if s.mKeyFlag != -1 {
		//f you have already obtained the existence status of the primary key, use it directly
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
	if sucNames, err := s.insertAllUniKeys(val); err != nil {
		s.delAllSortKeys(false, val)
		s.delUniKeysWithNames(sucNames, val)
		s.dba.Delete(keyBuf)
		s.delAllMemKeys(false, val)
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

func (s *SoDemoWrap) encodeMemKey(fName string) ([]byte, error) {
	if len(fName) < 1 || s.mainKey == nil {
		return nil, errors.New("field name or main key is empty")
	}
	pre := "Demo" + fName + "cell"
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

func (so *SoDemoWrap) saveAllMemKeys(tInfo *SoDemo, br bool) error {
	if so.dba == nil {
		return errors.New("save member Field fail , the db is nil")
	}

	if tInfo == nil {
		return errors.New("save member Field fail , the data is nil ")
	}
	var err error = nil
	errDes := ""
	if err = so.saveMemKeyContent(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Content", err)
		}
	}
	if err = so.saveMemKeyIdx(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Idx", err)
		}
	}
	if err = so.saveMemKeyLikeCount(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "LikeCount", err)
		}
	}
	if err = so.saveMemKeyOwner(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Owner", err)
		}
	}
	if err = so.saveMemKeyPostTime(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "PostTime", err)
		}
	}
	if err = so.saveMemKeyReplayCount(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "ReplayCount", err)
		}
	}
	if err = so.saveMemKeyTaglist(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Taglist", err)
		}
	}
	if err = so.saveMemKeyTitle(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Title", err)
		}
	}

	if len(errDes) > 0 {
		return errors.New(errDes)
	}
	return err
}

func (so *SoDemoWrap) delAllMemKeys(br bool, tInfo *SoDemo) error {
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

func (so *SoDemoWrap) delMemKey(fName string) error {
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

func (s *SoDemoWrap) delSortKeyOwner(sa *SoDemo) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListDemoByOwner{}
	if sa == nil {
		key, err := s.encodeMemKey("Owner")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemDemoByOwner{}
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
		key, err := s.encodeMemKey("PostTime")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemDemoByPostTime{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		val.PostTime = ori.PostTime
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
		key, err := s.encodeMemKey("LikeCount")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemDemoByLikeCount{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		val.LikeCount = ori.LikeCount
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
		key, err := s.encodeMemKey("Idx")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemDemoByIdx{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		val.Idx = ori.Idx
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
		key, err := s.encodeMemKey("ReplayCount")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemDemoByReplayCount{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		val.ReplayCount = ori.ReplayCount
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
		key, err := s.encodeMemKey("Taglist")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemDemoByTaglist{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		val.Taglist = ori.Taglist
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
	if !s.delSortKeyOwner(val) {
		if br {
			return false
		} else {
			res = false
		}
	}
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
	if !s.insertSortKeyOwner(val) {
		return errors.New("insert sort Field Owner fail while insert table ")
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
	val := &SoDemo{}
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
func (s *SoDemoWrap) saveMemKeyContent(tInfo *SoDemo) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemDemoByContent{}
	val.Content = tInfo.Content
	key, err := s.encodeMemKey("Content")
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

func (s *SoDemoWrap) GetContent() string {
	res := true
	msg := &SoMemDemoByContent{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Content")
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

func (s *SoDemoWrap) MdContent(p string) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Content")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemDemoByContent{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoDemo{}
	sa.Owner = s.mainKey

	sa.Content = ori.Content

	ori.Content = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Content = p

	return true
}

func (s *SoDemoWrap) saveMemKeyIdx(tInfo *SoDemo) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemDemoByIdx{}
	val.Idx = tInfo.Idx
	key, err := s.encodeMemKey("Idx")
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

func (s *SoDemoWrap) GetIdx() int64 {
	res := true
	msg := &SoMemDemoByIdx{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Idx")
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

func (s *SoDemoWrap) MdIdx(p int64) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Idx")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemDemoByIdx{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoDemo{}
	sa.Owner = s.mainKey

	sa.Idx = ori.Idx
	//judge the unique value if is exist
	uniWrap := UniDemoIdxWrap{}
	uniWrap.Dba = s.dba
	res := uniWrap.UniQueryIdx(&p)
	if res != nil {
		//the unique value to be modified is already exist
		return false
	}
	if !s.delUniKeyIdx(sa) {
		return false
	}

	if !s.delSortKeyIdx(sa) {
		return false
	}
	ori.Idx = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Idx = p

	if !s.insertSortKeyIdx(sa) {
		return false
	}

	if !s.insertUniKeyIdx(sa) {
		return false
	}
	return true
}

func (s *SoDemoWrap) saveMemKeyLikeCount(tInfo *SoDemo) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemDemoByLikeCount{}
	val.LikeCount = tInfo.LikeCount
	key, err := s.encodeMemKey("LikeCount")
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

func (s *SoDemoWrap) GetLikeCount() int64 {
	res := true
	msg := &SoMemDemoByLikeCount{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("LikeCount")
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

func (s *SoDemoWrap) MdLikeCount(p int64) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("LikeCount")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemDemoByLikeCount{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoDemo{}
	sa.Owner = s.mainKey

	sa.LikeCount = ori.LikeCount
	//judge the unique value if is exist
	uniWrap := UniDemoLikeCountWrap{}
	uniWrap.Dba = s.dba
	res := uniWrap.UniQueryLikeCount(&p)
	if res != nil {
		//the unique value to be modified is already exist
		return false
	}
	if !s.delUniKeyLikeCount(sa) {
		return false
	}

	if !s.delSortKeyLikeCount(sa) {
		return false
	}
	ori.LikeCount = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.LikeCount = p

	if !s.insertSortKeyLikeCount(sa) {
		return false
	}

	if !s.insertUniKeyLikeCount(sa) {
		return false
	}
	return true
}

func (s *SoDemoWrap) saveMemKeyOwner(tInfo *SoDemo) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemDemoByOwner{}
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

func (s *SoDemoWrap) GetOwner() *prototype.AccountName {
	res := true
	msg := &SoMemDemoByOwner{}
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

func (s *SoDemoWrap) saveMemKeyPostTime(tInfo *SoDemo) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemDemoByPostTime{}
	val.PostTime = tInfo.PostTime
	key, err := s.encodeMemKey("PostTime")
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

func (s *SoDemoWrap) GetPostTime() *prototype.TimePointSec {
	res := true
	msg := &SoMemDemoByPostTime{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("PostTime")
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

func (s *SoDemoWrap) MdPostTime(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("PostTime")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemDemoByPostTime{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoDemo{}
	sa.Owner = s.mainKey

	sa.PostTime = ori.PostTime

	if !s.delSortKeyPostTime(sa) {
		return false
	}
	ori.PostTime = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.PostTime = p

	if !s.insertSortKeyPostTime(sa) {
		return false
	}

	return true
}

func (s *SoDemoWrap) saveMemKeyReplayCount(tInfo *SoDemo) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemDemoByReplayCount{}
	val.ReplayCount = tInfo.ReplayCount
	key, err := s.encodeMemKey("ReplayCount")
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

func (s *SoDemoWrap) GetReplayCount() int64 {
	res := true
	msg := &SoMemDemoByReplayCount{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("ReplayCount")
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

func (s *SoDemoWrap) MdReplayCount(p int64) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("ReplayCount")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemDemoByReplayCount{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoDemo{}
	sa.Owner = s.mainKey

	sa.ReplayCount = ori.ReplayCount

	if !s.delSortKeyReplayCount(sa) {
		return false
	}
	ori.ReplayCount = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.ReplayCount = p

	if !s.insertSortKeyReplayCount(sa) {
		return false
	}

	return true
}

func (s *SoDemoWrap) saveMemKeyTaglist(tInfo *SoDemo) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemDemoByTaglist{}
	val.Taglist = tInfo.Taglist
	key, err := s.encodeMemKey("Taglist")
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

func (s *SoDemoWrap) GetTaglist() []string {
	res := true
	msg := &SoMemDemoByTaglist{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Taglist")
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

func (s *SoDemoWrap) MdTaglist(p []string) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Taglist")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemDemoByTaglist{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoDemo{}
	sa.Owner = s.mainKey

	sa.Taglist = ori.Taglist

	if !s.delSortKeyTaglist(sa) {
		return false
	}
	ori.Taglist = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Taglist = p

	if !s.insertSortKeyTaglist(sa) {
		return false
	}

	return true
}

func (s *SoDemoWrap) saveMemKeyTitle(tInfo *SoDemo) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemDemoByTitle{}
	val.Title = tInfo.Title
	key, err := s.encodeMemKey("Title")
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

func (s *SoDemoWrap) GetTitle() string {
	res := true
	msg := &SoMemDemoByTitle{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Title")
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

func (s *SoDemoWrap) MdTitle(p string) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Title")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemDemoByTitle{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoDemo{}
	sa.Owner = s.mainKey

	sa.Title = ori.Title

	ori.Title = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Title = p

	return true
}

////////////// SECTION List Keys ///////////////
type SDemoOwnerWrap struct {
	Dba iservices.IDatabaseService
}

func NewDemoOwnerWrap(db iservices.IDatabaseService) *SDemoOwnerWrap {
	if db == nil {
		return nil
	}
	wrap := SDemoOwnerWrap{Dba: db}
	return &wrap
}

func (s *SDemoOwnerWrap) DelIterator(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SDemoOwnerWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListDemoByOwner{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Owner

}

func (s *SDemoOwnerWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListDemoByOwner{}
	err = proto.Unmarshal(val, res)
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

//Query sort by reverse order
//
//f: callback for each traversal , primary 、sub key、idx(the number of times it has been iterated)
//as arguments to the callback function
//if the return value of f is true,continue iterating until the end iteration;
//otherwise stop iteration immediately
//
func (s *SDemoOwnerWrap) ForEachByRevOrder(start *prototype.AccountName, end *prototype.AccountName,
	f func(mVal *prototype.AccountName, sVal *prototype.AccountName, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if f == nil {
		return nil
	}
	pre := DemoOwnerTable
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

////////////// SECTION List Keys ///////////////
type SDemoPostTimeWrap struct {
	Dba iservices.IDatabaseService
}

func NewDemoPostTimeWrap(db iservices.IDatabaseService) *SDemoPostTimeWrap {
	if db == nil {
		return nil
	}
	wrap := SDemoPostTimeWrap{Dba: db}
	return &wrap
}

func (s *SDemoPostTimeWrap) DelIterator(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SDemoPostTimeWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListDemoByPostTime{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Owner

}

func (s *SDemoPostTimeWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.TimePointSec {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListDemoByPostTime{}
	err = proto.Unmarshal(val, res)
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

//Query sort by order
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
func (s *SDemoPostTimeWrap) ForEachByOrder(start *prototype.TimePointSec, end *prototype.TimePointSec,
	f func(mVal *prototype.AccountName, sVal *prototype.TimePointSec, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if f == nil {
		return nil
	}
	pre := DemoPostTimeTable
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

//Query sort by reverse order
//
//f: callback for each traversal , primary 、sub key、idx(the number of times it has been iterated)
//as arguments to the callback function
//if the return value of f is true,continue iterating until the end iteration;
//otherwise stop iteration immediately
//
func (s *SDemoPostTimeWrap) ForEachByRevOrder(start *prototype.TimePointSec, end *prototype.TimePointSec,
	f func(mVal *prototype.AccountName, sVal *prototype.TimePointSec, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if f == nil {
		return nil
	}
	pre := DemoPostTimeTable
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

////////////// SECTION List Keys ///////////////
type SDemoLikeCountWrap struct {
	Dba iservices.IDatabaseService
}

func NewDemoLikeCountWrap(db iservices.IDatabaseService) *SDemoLikeCountWrap {
	if db == nil {
		return nil
	}
	wrap := SDemoLikeCountWrap{Dba: db}
	return &wrap
}

func (s *SDemoLikeCountWrap) DelIterator(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SDemoLikeCountWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListDemoByLikeCount{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Owner

}

func (s *SDemoLikeCountWrap) GetSubVal(iterator iservices.IDatabaseIterator) *int64 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListDemoByLikeCount{}
	err = proto.Unmarshal(val, res)
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

//Query sort by reverse order
//
//f: callback for each traversal , primary 、sub key、idx(the number of times it has been iterated)
//as arguments to the callback function
//if the return value of f is true,continue iterating until the end iteration;
//otherwise stop iteration immediately
//
func (s *SDemoLikeCountWrap) ForEachByRevOrder(start *int64, end *int64,
	f func(mVal *prototype.AccountName, sVal *int64, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if f == nil {
		return nil
	}
	pre := DemoLikeCountTable
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

////////////// SECTION List Keys ///////////////
type SDemoIdxWrap struct {
	Dba iservices.IDatabaseService
}

func NewDemoIdxWrap(db iservices.IDatabaseService) *SDemoIdxWrap {
	if db == nil {
		return nil
	}
	wrap := SDemoIdxWrap{Dba: db}
	return &wrap
}

func (s *SDemoIdxWrap) DelIterator(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SDemoIdxWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListDemoByIdx{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Owner

}

func (s *SDemoIdxWrap) GetSubVal(iterator iservices.IDatabaseIterator) *int64 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListDemoByIdx{}
	err = proto.Unmarshal(val, res)
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

//Query sort by reverse order
//
//f: callback for each traversal , primary 、sub key、idx(the number of times it has been iterated)
//as arguments to the callback function
//if the return value of f is true,continue iterating until the end iteration;
//otherwise stop iteration immediately
//
func (s *SDemoIdxWrap) ForEachByRevOrder(start *int64, end *int64,
	f func(mVal *prototype.AccountName, sVal *int64, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if f == nil {
		return nil
	}
	pre := DemoIdxTable
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

////////////// SECTION List Keys ///////////////
type SDemoReplayCountWrap struct {
	Dba iservices.IDatabaseService
}

func NewDemoReplayCountWrap(db iservices.IDatabaseService) *SDemoReplayCountWrap {
	if db == nil {
		return nil
	}
	wrap := SDemoReplayCountWrap{Dba: db}
	return &wrap
}

func (s *SDemoReplayCountWrap) DelIterator(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SDemoReplayCountWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListDemoByReplayCount{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Owner

}

func (s *SDemoReplayCountWrap) GetSubVal(iterator iservices.IDatabaseIterator) *int64 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListDemoByReplayCount{}
	err = proto.Unmarshal(val, res)
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

//Query sort by order
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
func (s *SDemoReplayCountWrap) ForEachByOrder(start *int64, end *int64,
	f func(mVal *prototype.AccountName, sVal *int64, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if f == nil {
		return nil
	}
	pre := DemoReplayCountTable
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
type SDemoTaglistWrap struct {
	Dba iservices.IDatabaseService
}

func NewDemoTaglistWrap(db iservices.IDatabaseService) *SDemoTaglistWrap {
	if db == nil {
		return nil
	}
	wrap := SDemoTaglistWrap{Dba: db}
	return &wrap
}

func (s *SDemoTaglistWrap) DelIterator(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SDemoTaglistWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListDemoByTaglist{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.Owner

}

func (s *SDemoTaglistWrap) GetSubVal(iterator iservices.IDatabaseIterator) *[]string {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListDemoByTaglist{}
	err = proto.Unmarshal(val, res)
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

//Query sort by order
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
func (s *SDemoTaglistWrap) ForEachByOrder(start *[]string, end *[]string,
	f func(mVal *prototype.AccountName, sVal *[]string, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if f == nil {
		return nil
	}
	pre := DemoTaglistTable
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

func (s *SoDemoWrap) encodeMainKey() ([]byte, error) {
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := "Demo" + "Owner" + "cell"
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
		key, err := s.encodeMemKey("Idx")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemDemoByIdx{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		sub := ori.Idx
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
	Dba iservices.IDatabaseService
}

func NewUniDemoIdxWrap(db iservices.IDatabaseService) *UniDemoIdxWrap {
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
		key, err := s.encodeMemKey("LikeCount")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemDemoByLikeCount{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		sub := ori.LikeCount
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
	Dba iservices.IDatabaseService
}

func NewUniDemoLikeCountWrap(db iservices.IDatabaseService) *UniDemoLikeCountWrap {
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
		key, err := s.encodeMemKey("Owner")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemDemoByOwner{}
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
	Dba iservices.IDatabaseService
}

func NewUniDemoOwnerWrap(db iservices.IDatabaseService) *UniDemoOwnerWrap {
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
