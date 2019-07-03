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
	PostCreatedTable         uint32 = 3346451556
	PostCashoutBlockNumTable uint32 = 1826021466
	PostRewardsTable         uint32 = 2325142906
	PostPostIdUniTable       uint32 = 157486700
	PostAuthorCell           uint32 = 1681275280
	PostBeneficiariesCell    uint32 = 2794141504
	PostBodyCell             uint32 = 395462793
	PostCashoutBlockNumCell  uint32 = 2338008419
	PostCategoryCell         uint32 = 2849013589
	PostChildrenCell         uint32 = 3908796047
	PostCopyrightCell        uint32 = 2903094549
	PostCopyrightMemoCell    uint32 = 791964881
	PostCreatedCell          uint32 = 4199172684
	PostDappRewardsCell      uint32 = 3278808896
	PostDepthCell            uint32 = 4080627723
	PostLastPayoutCell       uint32 = 3845986349
	PostParentIdCell         uint32 = 1393772380
	PostPostIdCell           uint32 = 22700035
	PostRewardsCell          uint32 = 2822376492
	PostRootIdCell           uint32 = 784045146
	PostTagsCell             uint32 = 828203383
	PostTicketCell           uint32 = 2248685104
	PostTitleCell            uint32 = 3943450465
	PostVoteCntCell          uint32 = 2947124424
	PostWeightedVpCell       uint32 = 502117977
)

////////////// SECTION Wrap Define ///////////////
type SoPostWrap struct {
	dba      iservices.IDatabaseRW
	mainKey  *uint64
	mKeyFlag int    //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf  []byte //the buffer after the main key is encoded with prefix
	mBuf     []byte //the value after the main key is encoded
}

func NewSoPostWrap(dba iservices.IDatabaseRW, key *uint64) *SoPostWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoPostWrap{dba, key, -1, nil, nil}
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

func (s *SoPostWrap) delSortKeyCashoutBlockNum(sa *SoPost) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListPostByCashoutBlockNum{}
	if sa == nil {
		key, err := s.encodeMemKey("CashoutBlockNum")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemPostByCashoutBlockNum{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		val.CashoutBlockNum = ori.CashoutBlockNum
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

func (s *SoPostWrap) delSortKeyRewards(sa *SoPost) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListPostByRewards{}
	if sa == nil {
		key, err := s.encodeMemKey("Rewards")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemPostByRewards{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		val.Rewards = ori.Rewards
		val.PostId = *s.mainKey
	} else {
		val.Rewards = sa.Rewards
		val.PostId = sa.PostId
	}

	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoPostWrap) insertSortKeyRewards(sa *SoPost) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListPostByRewards{}
	val.PostId = sa.PostId
	val.Rewards = sa.Rewards
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
	if !s.delSortKeyRewards(val) {
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
	if !s.insertSortKeyRewards(val) {
		return errors.New("insert sort Field Rewards fail while insert table ")
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
	if err == nil {
		s.mKeyBuf = nil
		s.mKeyFlag = -1
		return true
	} else {
		return false
	}
}

////////////// SECTION Members Get/Modify ///////////////
func (s *SoPostWrap) getMemKeyPrefix(fName string) uint32 {
	if fName == "Author" {
		return PostAuthorCell
	}
	if fName == "Beneficiaries" {
		return PostBeneficiariesCell
	}
	if fName == "Body" {
		return PostBodyCell
	}
	if fName == "CashoutBlockNum" {
		return PostCashoutBlockNumCell
	}
	if fName == "Category" {
		return PostCategoryCell
	}
	if fName == "Children" {
		return PostChildrenCell
	}
	if fName == "Copyright" {
		return PostCopyrightCell
	}
	if fName == "CopyrightMemo" {
		return PostCopyrightMemoCell
	}
	if fName == "Created" {
		return PostCreatedCell
	}
	if fName == "DappRewards" {
		return PostDappRewardsCell
	}
	if fName == "Depth" {
		return PostDepthCell
	}
	if fName == "LastPayout" {
		return PostLastPayoutCell
	}
	if fName == "ParentId" {
		return PostParentIdCell
	}
	if fName == "PostId" {
		return PostPostIdCell
	}
	if fName == "Rewards" {
		return PostRewardsCell
	}
	if fName == "RootId" {
		return PostRootIdCell
	}
	if fName == "Tags" {
		return PostTagsCell
	}
	if fName == "Ticket" {
		return PostTicketCell
	}
	if fName == "Title" {
		return PostTitleCell
	}
	if fName == "VoteCnt" {
		return PostVoteCntCell
	}
	if fName == "WeightedVp" {
		return PostWeightedVpCell
	}

	return 0
}

func (s *SoPostWrap) encodeMemKey(fName string) ([]byte, error) {
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

func (s *SoPostWrap) saveAllMemKeys(tInfo *SoPost, br bool) error {
	if s.dba == nil {
		return errors.New("save member Field fail , the db is nil")
	}

	if tInfo == nil {
		return errors.New("save member Field fail , the data is nil ")
	}
	var err error = nil
	errDes := ""
	if err = s.saveMemKeyAuthor(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Author", err)
		}
	}
	if err = s.saveMemKeyBeneficiaries(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Beneficiaries", err)
		}
	}
	if err = s.saveMemKeyBody(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Body", err)
		}
	}
	if err = s.saveMemKeyCashoutBlockNum(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "CashoutBlockNum", err)
		}
	}
	if err = s.saveMemKeyCategory(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Category", err)
		}
	}
	if err = s.saveMemKeyChildren(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Children", err)
		}
	}
	if err = s.saveMemKeyCopyright(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Copyright", err)
		}
	}
	if err = s.saveMemKeyCopyrightMemo(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "CopyrightMemo", err)
		}
	}
	if err = s.saveMemKeyCreated(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Created", err)
		}
	}
	if err = s.saveMemKeyDappRewards(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "DappRewards", err)
		}
	}
	if err = s.saveMemKeyDepth(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Depth", err)
		}
	}
	if err = s.saveMemKeyLastPayout(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "LastPayout", err)
		}
	}
	if err = s.saveMemKeyParentId(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "ParentId", err)
		}
	}
	if err = s.saveMemKeyPostId(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "PostId", err)
		}
	}
	if err = s.saveMemKeyRewards(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Rewards", err)
		}
	}
	if err = s.saveMemKeyRootId(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "RootId", err)
		}
	}
	if err = s.saveMemKeyTags(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Tags", err)
		}
	}
	if err = s.saveMemKeyTicket(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Ticket", err)
		}
	}
	if err = s.saveMemKeyTitle(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "Title", err)
		}
	}
	if err = s.saveMemKeyVoteCnt(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "VoteCnt", err)
		}
	}
	if err = s.saveMemKeyWeightedVp(tInfo); err != nil {
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

func (s *SoPostWrap) delAllMemKeys(br bool, tInfo *SoPost) error {
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

func (s *SoPostWrap) delMemKey(fName string) error {
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

func (s *SoPostWrap) saveMemKeyCashoutBlockNum(tInfo *SoPost) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemPostByCashoutBlockNum{}
	val.CashoutBlockNum = tInfo.CashoutBlockNum
	key, err := s.encodeMemKey("CashoutBlockNum")
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

func (s *SoPostWrap) GetCashoutBlockNum() uint64 {
	res := true
	msg := &SoMemPostByCashoutBlockNum{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("CashoutBlockNum")
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

func (s *SoPostWrap) MdCashoutBlockNum(p uint64) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("CashoutBlockNum")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemPostByCashoutBlockNum{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoPost{}
	sa.PostId = *s.mainKey
	sa.CashoutBlockNum = ori.CashoutBlockNum

	if !s.delSortKeyCashoutBlockNum(sa) {
		return false
	}
	ori.CashoutBlockNum = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.CashoutBlockNum = p

	if !s.insertSortKeyCashoutBlockNum(sa) {
		return false
	}

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

func (s *SoPostWrap) saveMemKeyCopyright(tInfo *SoPost) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemPostByCopyright{}
	val.Copyright = tInfo.Copyright
	key, err := s.encodeMemKey("Copyright")
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

func (s *SoPostWrap) GetCopyright() uint32 {
	res := true
	msg := &SoMemPostByCopyright{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Copyright")
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
				return msg.Copyright
			}
		}
	}
	if !res {
		var tmpValue uint32
		return tmpValue
	}
	return msg.Copyright
}

func (s *SoPostWrap) MdCopyright(p uint32) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Copyright")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemPostByCopyright{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoPost{}
	sa.PostId = *s.mainKey
	sa.Copyright = ori.Copyright

	ori.Copyright = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Copyright = p

	return true
}

func (s *SoPostWrap) saveMemKeyCopyrightMemo(tInfo *SoPost) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemPostByCopyrightMemo{}
	val.CopyrightMemo = tInfo.CopyrightMemo
	key, err := s.encodeMemKey("CopyrightMemo")
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

func (s *SoPostWrap) GetCopyrightMemo() string {
	res := true
	msg := &SoMemPostByCopyrightMemo{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("CopyrightMemo")
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
				return msg.CopyrightMemo
			}
		}
	}
	if !res {
		var tmpValue string
		return tmpValue
	}
	return msg.CopyrightMemo
}

func (s *SoPostWrap) MdCopyrightMemo(p string) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("CopyrightMemo")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemPostByCopyrightMemo{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoPost{}
	sa.PostId = *s.mainKey
	sa.CopyrightMemo = ori.CopyrightMemo

	ori.CopyrightMemo = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.CopyrightMemo = p

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

func (s *SoPostWrap) saveMemKeyDappRewards(tInfo *SoPost) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemPostByDappRewards{}
	val.DappRewards = tInfo.DappRewards
	key, err := s.encodeMemKey("DappRewards")
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

func (s *SoPostWrap) GetDappRewards() *prototype.Vest {
	res := true
	msg := &SoMemPostByDappRewards{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("DappRewards")
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

func (s *SoPostWrap) MdDappRewards(p *prototype.Vest) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("DappRewards")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemPostByDappRewards{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoPost{}
	sa.PostId = *s.mainKey
	sa.DappRewards = ori.DappRewards

	ori.DappRewards = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.DappRewards = p

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

func (s *SoPostWrap) saveMemKeyRewards(tInfo *SoPost) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemPostByRewards{}
	val.Rewards = tInfo.Rewards
	key, err := s.encodeMemKey("Rewards")
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

func (s *SoPostWrap) GetRewards() *prototype.Vest {
	res := true
	msg := &SoMemPostByRewards{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Rewards")
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

func (s *SoPostWrap) MdRewards(p *prototype.Vest) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Rewards")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemPostByRewards{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoPost{}
	sa.PostId = *s.mainKey
	sa.Rewards = ori.Rewards

	if !s.delSortKeyRewards(sa) {
		return false
	}
	ori.Rewards = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Rewards = p

	if !s.insertSortKeyRewards(sa) {
		return false
	}

	return true
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
	key, err := s.encodeMemKey("Tags")
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
		key, err := s.encodeMemKey("Tags")
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
	key, err := s.encodeMemKey("Tags")
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

func (s *SoPostWrap) saveMemKeyTicket(tInfo *SoPost) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemPostByTicket{}
	val.Ticket = tInfo.Ticket
	key, err := s.encodeMemKey("Ticket")
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

func (s *SoPostWrap) GetTicket() uint32 {
	res := true
	msg := &SoMemPostByTicket{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("Ticket")
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
				return msg.Ticket
			}
		}
	}
	if !res {
		var tmpValue uint32
		return tmpValue
	}
	return msg.Ticket
}

func (s *SoPostWrap) MdTicket(p uint32) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("Ticket")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemPostByTicket{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoPost{}
	sa.PostId = *s.mainKey
	sa.Ticket = ori.Ticket

	ori.Ticket = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.Ticket = p

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

func (s *SoPostWrap) GetWeightedVp() string {
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
		var tmpValue string
		return tmpValue
	}
	return msg.WeightedVp
}

func (s *SoPostWrap) MdWeightedVp(p string) bool {
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

////////////// SECTION List Keys ///////////////
type SPostRewardsWrap struct {
	Dba iservices.IDatabaseRW
}

func NewPostRewardsWrap(db iservices.IDatabaseRW) *SPostRewardsWrap {
	if db == nil {
		return nil
	}
	wrap := SPostRewardsWrap{Dba: db}
	return &wrap
}

func (s *SPostRewardsWrap) GetMainVal(val []byte) *uint64 {
	res := &SoListPostByRewards{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return &res.PostId

}

func (s *SPostRewardsWrap) GetSubVal(val []byte) *prototype.Vest {
	res := &SoListPostByRewards{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.Rewards

}

func (m *SoListPostByRewards) OpeEncode() ([]byte, error) {
	pre := PostRewardsTable
	sub := m.Rewards
	if sub == nil {
		return nil, errors.New("the pro Rewards is nil")
	}
	sub1 := m.PostId

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
func (s *SPostRewardsWrap) ForEachByRevOrder(start *prototype.Vest, end *prototype.Vest, lastMainKey *uint64,
	lastSubVal *prototype.Vest, f func(mVal *uint64, sVal *prototype.Vest, idx uint32) bool) error {
	if s.Dba == nil {
		return errors.New("the db is nil")
	}
	if (lastSubVal != nil && lastMainKey == nil) || (lastSubVal == nil && lastMainKey != nil) {
		return errors.New("last query param error")
	}
	if f == nil {
		return nil
	}
	pre := PostRewardsTable
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
	if s.mKeyBuf != nil {
		return s.mKeyBuf, nil
	}
	pre := s.getMemKeyPrefix("PostId")
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
