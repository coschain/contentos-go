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
	PostTable          = []byte("PostTable")
	PostCreatedTable   = []byte("PostCreatedTable")
	PostPostIdUniTable = []byte("PostPostIdUniTable")
)

////////////// SECTION Wrap Define ///////////////
type SoPostWrap struct {
	dba     iservices.IDatabaseService
	mainKey *uint64
}

func NewSoPostWrap(dba iservices.IDatabaseService, key *uint64) *SoPostWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoPostWrap{dba, key}
	return result
}

func (s *SoPostWrap) CheckExist() bool {
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

func (s *SoPostWrap) encodeMemKey(fName string) ([]byte, error) {
	if len(fName) < 1 || s.mainKey == nil {
		return nil, errors.New("field name or main key is empty")
	}
	pre := "Post" + fName + "cell"
	kList := []interface{}{pre, s.mainKey}
	key, err := kope.EncodeSlice(kList)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (so *SoPostWrap) saveAllMemKeys(tInfo *SoPost, br bool) error {
	if so.dba == nil {
		return errors.New("save member Field fail , the db is nil")
	}

	if tInfo == nil {
		return errors.New("save member Field fail , the data is nil ")
	}
	var err error = nil
	errDes := ""
	if err = so.saveMemKeyAuthor(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Author", err)
		}
	}
	if err = so.saveMemKeyBeneficiaries(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Beneficiaries", err)
		}
	}
	if err = so.saveMemKeyBody(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Body", err)
		}
	}
	if err = so.saveMemKeyCashoutTime(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "CashoutTime", err)
		}
	}
	if err = so.saveMemKeyCategory(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Category", err)
		}
	}
	if err = so.saveMemKeyChildren(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Children", err)
		}
	}
	if err = so.saveMemKeyCreated(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Created", err)
		}
	}
	if err = so.saveMemKeyDepth(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Depth", err)
		}
	}
	if err = so.saveMemKeyLastPayout(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "LastPayout", err)
		}
	}
	if err = so.saveMemKeyParentId(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "ParentId", err)
		}
	}
	if err = so.saveMemKeyPostId(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "PostId", err)
		}
	}
	if err = so.saveMemKeyRootId(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "RootId", err)
		}
	}
	if err = so.saveMemKeyTags(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Tags				", err)
		}
	}
	if err = so.saveMemKeyTitle(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Title", err)
		}
	}
	if err = so.saveMemKeyVoteCnt(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "VoteCnt", err)
		}
	}
	if err = so.saveMemKeyWeightedVp(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "WeightedVp", err)
		}
	}

	if len(errDes) > 0 {
		return errors.New(errDes)
	}
	return err
}

func (so *SoPostWrap) delAllMemKeys(br bool, tInfo *SoPost) error {
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

func (so *SoPostWrap) delMemKey(fName string) error {
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

func (s *SoPostWrap) delSortKeyCreated(sa *SoPost) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListPostByCreated{}
	if sa == nil {
		key, err := s.encodeMemKey("Created")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemPostByCreated{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		val.Created = ori.Created
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

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoPostWrap) RemovePost() bool {
	if s.dba == nil {
		return false
	}
	val := &SoPost{}
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
func (s *SoPostWrap) saveMemKeyAuthor(tInfo *SoPost) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemPostByAuthor{}
	val.Author = tInfo.Author
	key, err := s.encodeMemKey("Author")
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

func (s *SoPostWrap) GetAuthor() *prototype.AccountName {
	res := true
	msg := &SoMemPostByAuthor{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Author")
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

func (s *SoPostWrap) MdAuthor(p *prototype.AccountName) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Author")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemPostByAuthor{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoPost{}
	sa.PostId = *s.mainKey
	sa.Author = ori.Author

	ori.Author = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Author = p

	return true
}

func (s *SoPostWrap) saveMemKeyBeneficiaries(tInfo *SoPost) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemPostByBeneficiaries{}
	val.Beneficiaries = tInfo.Beneficiaries
	key, err := s.encodeMemKey("Beneficiaries")
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

func (s *SoPostWrap) GetBeneficiaries() []*prototype.BeneficiaryRouteType {
	res := true
	msg := &SoMemPostByBeneficiaries{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Beneficiaries")
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

func (s *SoPostWrap) MdBeneficiaries(p []*prototype.BeneficiaryRouteType) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Beneficiaries")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemPostByBeneficiaries{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoPost{}
	sa.PostId = *s.mainKey
	sa.Beneficiaries = ori.Beneficiaries

	ori.Beneficiaries = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Beneficiaries = p

	return true
}

func (s *SoPostWrap) saveMemKeyBody(tInfo *SoPost) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemPostByBody{}
	val.Body = tInfo.Body
	key, err := s.encodeMemKey("Body")
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

func (s *SoPostWrap) GetBody() string {
	res := true
	msg := &SoMemPostByBody{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Body")
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

func (s *SoPostWrap) MdBody(p string) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Body")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemPostByBody{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoPost{}
	sa.PostId = *s.mainKey
	sa.Body = ori.Body

	ori.Body = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Body = p

	return true
}

func (s *SoPostWrap) saveMemKeyCashoutTime(tInfo *SoPost) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemPostByCashoutTime{}
	val.CashoutTime = tInfo.CashoutTime
	key, err := s.encodeMemKey("CashoutTime")
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

func (s *SoPostWrap) GetCashoutTime() *prototype.TimePointSec {
	res := true
	msg := &SoMemPostByCashoutTime{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("CashoutTime")
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
				return msg.CashoutTime
			}
		}
	}
	if !res {
		return nil

	}
	return msg.CashoutTime
}

func (s *SoPostWrap) MdCashoutTime(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("CashoutTime")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemPostByCashoutTime{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoPost{}
	sa.PostId = *s.mainKey
	sa.CashoutTime = ori.CashoutTime

	ori.CashoutTime = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.CashoutTime = p

	return true
}

func (s *SoPostWrap) saveMemKeyCategory(tInfo *SoPost) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemPostByCategory{}
	val.Category = tInfo.Category
	key, err := s.encodeMemKey("Category")
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

func (s *SoPostWrap) GetCategory() string {
	res := true
	msg := &SoMemPostByCategory{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Category")
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

func (s *SoPostWrap) MdCategory(p string) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Category")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemPostByCategory{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoPost{}
	sa.PostId = *s.mainKey
	sa.Category = ori.Category

	ori.Category = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Category = p

	return true
}

func (s *SoPostWrap) saveMemKeyChildren(tInfo *SoPost) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemPostByChildren{}
	val.Children = tInfo.Children
	key, err := s.encodeMemKey("Children")
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

func (s *SoPostWrap) GetChildren() uint32 {
	res := true
	msg := &SoMemPostByChildren{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Children")
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

func (s *SoPostWrap) MdChildren(p uint32) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Children")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemPostByChildren{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoPost{}
	sa.PostId = *s.mainKey
	sa.Children = ori.Children

	ori.Children = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Children = p

	return true
}

func (s *SoPostWrap) saveMemKeyCreated(tInfo *SoPost) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemPostByCreated{}
	val.Created = tInfo.Created
	key, err := s.encodeMemKey("Created")
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

func (s *SoPostWrap) GetCreated() *prototype.TimePointSec {
	res := true
	msg := &SoMemPostByCreated{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Created")
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

func (s *SoPostWrap) MdCreated(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Created")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemPostByCreated{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoPost{}
	sa.PostId = *s.mainKey
	sa.Created = ori.Created

	if !s.delSortKeyCreated(sa) {
		return false
	}
	ori.Created = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Created = p

	if !s.insertSortKeyCreated(sa) {
		return false
	}

	return true
}

func (s *SoPostWrap) saveMemKeyDepth(tInfo *SoPost) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemPostByDepth{}
	val.Depth = tInfo.Depth
	key, err := s.encodeMemKey("Depth")
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

func (s *SoPostWrap) GetDepth() uint32 {
	res := true
	msg := &SoMemPostByDepth{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Depth")
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

func (s *SoPostWrap) MdDepth(p uint32) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Depth")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemPostByDepth{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoPost{}
	sa.PostId = *s.mainKey
	sa.Depth = ori.Depth

	ori.Depth = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Depth = p

	return true
}

func (s *SoPostWrap) saveMemKeyLastPayout(tInfo *SoPost) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemPostByLastPayout{}
	val.LastPayout = tInfo.LastPayout
	key, err := s.encodeMemKey("LastPayout")
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

func (s *SoPostWrap) GetLastPayout() *prototype.TimePointSec {
	res := true
	msg := &SoMemPostByLastPayout{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("LastPayout")
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

func (s *SoPostWrap) MdLastPayout(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("LastPayout")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemPostByLastPayout{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoPost{}
	sa.PostId = *s.mainKey
	sa.LastPayout = ori.LastPayout

	ori.LastPayout = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.LastPayout = p

	return true
}

func (s *SoPostWrap) saveMemKeyParentId(tInfo *SoPost) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemPostByParentId{}
	val.ParentId = tInfo.ParentId
	key, err := s.encodeMemKey("ParentId")
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

func (s *SoPostWrap) GetParentId() uint64 {
	res := true
	msg := &SoMemPostByParentId{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("ParentId")
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

func (s *SoPostWrap) MdParentId(p uint64) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("ParentId")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemPostByParentId{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoPost{}
	sa.PostId = *s.mainKey
	sa.ParentId = ori.ParentId

	ori.ParentId = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.ParentId = p

	return true
}

func (s *SoPostWrap) saveMemKeyPostId(tInfo *SoPost) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemPostByPostId{}
	val.PostId = tInfo.PostId
	key, err := s.encodeMemKey("PostId")
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

func (s *SoPostWrap) GetPostId() uint64 {
	res := true
	msg := &SoMemPostByPostId{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("PostId")
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

func (s *SoPostWrap) saveMemKeyRootId(tInfo *SoPost) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemPostByRootId{}
	val.RootId = tInfo.RootId
	key, err := s.encodeMemKey("RootId")
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

func (s *SoPostWrap) GetRootId() uint64 {
	res := true
	msg := &SoMemPostByRootId{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("RootId")
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

func (s *SoPostWrap) MdRootId(p uint64) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("RootId")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemPostByRootId{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoPost{}
	sa.PostId = *s.mainKey
	sa.RootId = ori.RootId

	ori.RootId = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.RootId = p

	return true
}

func (s *SoPostWrap) saveMemKeyTags(tInfo *SoPost) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemPostByTags{}
	val.Tags = tInfo.Tags
	key, err := s.encodeMemKey("Tags				")
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

func (s *SoPostWrap) GetTags() []string {
	res := true
	msg := &SoMemPostByTags{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Tags				")
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

func (s *SoPostWrap) MdTags(p []string) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Tags				")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemPostByTags{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoPost{}
	sa.PostId = *s.mainKey
	sa.Tags = ori.Tags

	ori.Tags = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Tags = p

	return true
}

func (s *SoPostWrap) saveMemKeyTitle(tInfo *SoPost) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemPostByTitle{}
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

func (s *SoPostWrap) GetTitle() string {
	res := true
	msg := &SoMemPostByTitle{}
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

func (s *SoPostWrap) MdTitle(p string) bool {
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
	ori := &SoMemPostByTitle{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoPost{}
	sa.PostId = *s.mainKey
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

func (s *SoPostWrap) saveMemKeyVoteCnt(tInfo *SoPost) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemPostByVoteCnt{}
	val.VoteCnt = tInfo.VoteCnt
	key, err := s.encodeMemKey("VoteCnt")
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

func (s *SoPostWrap) GetVoteCnt() uint64 {
	res := true
	msg := &SoMemPostByVoteCnt{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("VoteCnt")
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

func (s *SoPostWrap) MdVoteCnt(p uint64) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("VoteCnt")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemPostByVoteCnt{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoPost{}
	sa.PostId = *s.mainKey
	sa.VoteCnt = ori.VoteCnt

	ori.VoteCnt = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.VoteCnt = p

	return true
}

func (s *SoPostWrap) saveMemKeyWeightedVp(tInfo *SoPost) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemPostByWeightedVp{}
	val.WeightedVp = tInfo.WeightedVp
	key, err := s.encodeMemKey("WeightedVp")
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

func (s *SoPostWrap) GetWeightedVp() uint64 {
	res := true
	msg := &SoMemPostByWeightedVp{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("WeightedVp")
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

func (s *SoPostWrap) MdWeightedVp(p uint64) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("WeightedVp")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemPostByWeightedVp{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoPost{}
	sa.PostId = *s.mainKey
	sa.WeightedVp = ori.WeightedVp

	ori.WeightedVp = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.WeightedVp = p

	return true
}

////////////// SECTION List Keys ///////////////
type SPostCreatedWrap struct {
	Dba iservices.IDatabaseService
}

func NewPostCreatedWrap(db iservices.IDatabaseService) *SPostCreatedWrap {
	if db == nil {
		return nil
	}
	wrap := SPostCreatedWrap{Dba: db}
	return &wrap
}

func (s *SPostCreatedWrap) DelIterator(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SPostCreatedWrap) GetMainVal(iterator iservices.IDatabaseIterator) *uint64 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListPostByCreated{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return &res.PostId

}

func (s *SPostCreatedWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.TimePointSec {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListPostByCreated{}
	err = proto.Unmarshal(val, res)
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

//
//Query sort by order
//start = nil  end = nil (query the db from start to end)
//start = nil (query from start the db)
//end = nil (query to the end of db)
//maxCount: represent the maximum amount of data you want to get，if the maxCount is greater than or equal to
//the total count of data in result,traverse all data;otherwise traverse part of the data
//f: callback for each traversal , primary and sub key as arguments to the callback function
//
func (s *SPostCreatedWrap) QueryListByOrder(start *prototype.TimePointSec, end *prototype.TimePointSec, maxCount uint32,
	f func(mVal *uint64, sVal *prototype.TimePointSec)) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if f == nil || maxCount < 1 {
		return nil
	}
	pre := PostCreatedTable
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
	for idx < maxCount && iterator.Next() {
		idx++
		f(s.GetMainVal(iterator), s.GetSubVal(iterator))
	}
	s.DelIterator(iterator)
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

func (s *SoPostWrap) encodeMainKey() ([]byte, error) {
	pre := "Post" + "PostId" + "cell"
	sub := s.mainKey
	if sub == nil {
		return nil, errors.New("the mainKey is nil")
	}
	kList := []interface{}{pre, sub}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
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
		key, err := s.encodeMemKey("PostId")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemPostByPostId{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		sub := ori.PostId
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
	uniWrap := UniPostPostIdWrap{}
	uniWrap.Dba = s.dba
	res := uniWrap.UniQueryPostId(&sa.PostId)

	if res != nil {
		//the unique key is already exist
		return false
	}
	val := SoUniquePostByPostId{}
	val.PostId = sa.PostId

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	pre := PostPostIdUniTable
	sub := sa.PostId
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniPostPostIdWrap struct {
	Dba iservices.IDatabaseService
}

func NewUniPostPostIdWrap(db iservices.IDatabaseService) *UniPostPostIdWrap {
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
