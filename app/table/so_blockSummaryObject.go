package table

import (
	"errors"

	"github.com/coschain/contentos-go/common/encoding/kope"
	"github.com/coschain/contentos-go/iservices"
	prototype "github.com/coschain/contentos-go/prototype"
	proto "github.com/golang/protobuf/proto"
)

////////////// SECTION Prefix Mark ///////////////
var (
	BlockSummaryObjectTable      = []byte("BlockSummaryObjectTable")
	BlockSummaryObjectIdUniTable = []byte("BlockSummaryObjectIdUniTable")
)

////////////// SECTION Wrap Define ///////////////
type SoBlockSummaryObjectWrap struct {
	dba     iservices.IDatabaseService
	mainKey *uint32
}

func NewSoBlockSummaryObjectWrap(dba iservices.IDatabaseService, key *uint32) *SoBlockSummaryObjectWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoBlockSummaryObjectWrap{dba, key}
	return result
}

func (s *SoBlockSummaryObjectWrap) CheckExist() bool {
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

func (s *SoBlockSummaryObjectWrap) Create(f func(tInfo *SoBlockSummaryObject)) error {
	val := &SoBlockSummaryObject{}
	f(val)
	if s.CheckExist() {
		return errors.New("the mainkey is already exist")
	}
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return err

	}
	resBuf, err := proto.Marshal(val)
	if err != nil {
		return err
	}
	err = s.dba.Put(keyBuf, resBuf)
	if err != nil {
		return err
	}

	// update sort list keys

	//update unique list
	if !s.insertUniKeyId(val) {
		s.delAllSortKeys()
		s.delAllUniKeys()
		s.dba.Delete(keyBuf)
		return errors.New("insert unique Field uint32 while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoBlockSummaryObjectWrap) delAllSortKeys() bool {
	if s.dba == nil {
		return false
	}
	sa := s.getBlockSummaryObject()
	if sa == nil {
		return false
	}
	res := true

	return res
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoBlockSummaryObjectWrap) RemoveBlockSummaryObject() bool {
	if s.dba == nil {
		return false
	}
	sa := s.getBlockSummaryObject()
	if sa == nil {
		return false
	}
	//delete sort list key

	//delete unique list
	if !s.delUniKeyId(sa) {
		return false
	}

	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return false
	}
	return s.dba.Delete(keyBuf) == nil
}

////////////// SECTION Members Get/Modify ///////////////
func (s *SoBlockSummaryObjectWrap) GetBlockId() *prototype.Sha256 {
	res := s.getBlockSummaryObject()

	if res == nil {
		return nil

	}
	return res.BlockId
}

func (s *SoBlockSummaryObjectWrap) MdBlockId(p *prototype.Sha256) bool {
	if s.dba == nil {
		return false
	}
	sa := s.getBlockSummaryObject()
	if sa == nil {
		return false
	}

	sa.BlockId = p
	if !s.update(sa) {
		return false
	}

	return true
}

func (s *SoBlockSummaryObjectWrap) GetId() uint32 {
	res := s.getBlockSummaryObject()

	if res == nil {
		var tmpValue uint32
		return tmpValue
	}
	return res.Id
}

/////////////// SECTION Private function ////////////////

func (s *SoBlockSummaryObjectWrap) update(sa *SoBlockSummaryObject) bool {
	if s.dba == nil {
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

func (s *SoBlockSummaryObjectWrap) getBlockSummaryObject() *SoBlockSummaryObject {
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

	res := &SoBlockSummaryObject{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoBlockSummaryObjectWrap) encodeMainKey() ([]byte, error) {
	pre := BlockSummaryObjectTable
	sub := s.mainKey
	if sub == nil {
		return nil, errors.New("the mainKey is nil")
	}
	kList := []interface{}{pre, sub}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
}

////////////// Unique Query delete/insert/query ///////////////

func (s *SoBlockSummaryObjectWrap) delAllUniKeys() bool {
	if s.dba == nil {
		return false
	}
	sa := s.getBlockSummaryObject()
	if sa == nil {
		return false
	}
	res := true
	if !s.delUniKeyId(sa) && res {
		res = false
	}

	return res
}

func (s *SoBlockSummaryObjectWrap) delUniKeyId(sa *SoBlockSummaryObject) bool {
	if s.dba == nil {
		return false
	}
	pre := BlockSummaryObjectIdUniTable
	sub := sa.Id
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoBlockSummaryObjectWrap) insertUniKeyId(sa *SoBlockSummaryObject) bool {
	if s.dba == nil {
		return false
	}
	uniWrap := UniBlockSummaryObjectIdWrap{}
	uniWrap.Dba = s.dba
	res := uniWrap.UniQueryId(&sa.Id)

	if res != nil {
		//the unique key is already exist
		return false
	}
	val := SoUniqueBlockSummaryObjectById{}
	val.Id = sa.Id

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	pre := BlockSummaryObjectIdUniTable
	sub := sa.Id
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniBlockSummaryObjectIdWrap struct {
	Dba iservices.IDatabaseService
}

func NewUniBlockSummaryObjectIdWrap(db iservices.IDatabaseService) *UniBlockSummaryObjectIdWrap {
	if db == nil {
		return nil
	}
	wrap := UniBlockSummaryObjectIdWrap{Dba: db}
	return &wrap
}

func (s *UniBlockSummaryObjectIdWrap) UniQueryId(start *uint32) *SoBlockSummaryObjectWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := BlockSummaryObjectIdUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueBlockSummaryObjectById{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoBlockSummaryObjectWrap(s.Dba, &res.Id)
			return wrap
		}
	}
	return nil
}
