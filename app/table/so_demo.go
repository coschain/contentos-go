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
	dba     iservices.IDatabaseService
	mainKey *prototype.AccountName
}

func NewSoDemoWrap(dba iservices.IDatabaseService, key *prototype.AccountName) *SoDemoWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoDemoWrap{dba, key}
	return result
}

func (s *SoDemoWrap) CheckExist() bool {
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

func (s *SoDemoWrap) Create(f func(tInfo *SoDemo)) error {
	val := &SoDemo{}
	f(val)
	if val.Owner == nil {
		return errors.New("the mainkey is nil")
	}
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
	if err = s.insertAllSortKeys(val); err != nil {
		s.delAllSortKeys(false, val)
		s.dba.Delete(keyBuf)
		return err
	}

	//update unique list
	if err = s.insertAllUniKeys(val); err != nil {
		s.delAllSortKeys(false, val)
		s.delAllUniKeys(false, val)
		s.dba.Delete(keyBuf)
		return err
	}

	return nil
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoDemoWrap) delSortKeyOwner(sa *SoDemo) bool {
	if s.dba == nil {
		return false
	}
	val := SoListDemoByOwner{}
	val.Owner = sa.Owner
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
	if s.dba == nil {
		return false
	}
	val := SoListDemoByPostTime{}
	val.PostTime = sa.PostTime
	val.Owner = sa.Owner
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
	if s.dba == nil {
		return false
	}
	val := SoListDemoByLikeCount{}
	val.LikeCount = sa.LikeCount
	val.Owner = sa.Owner
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
	if s.dba == nil {
		return false
	}
	val := SoListDemoByIdx{}
	val.Idx = sa.Idx
	val.Owner = sa.Owner
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
	if s.dba == nil {
		return false
	}
	val := SoListDemoByReplayCount{}
	val.ReplayCount = sa.ReplayCount
	val.Owner = sa.Owner
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
	if s.dba == nil {
		return false
	}
	val := SoListDemoByTaglist{}
	val.Taglist = sa.Taglist
	val.Owner = sa.Owner
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
	if val == nil {
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
	val := s.getDemo()
	if val == nil {
		return false
	}
	//delete sort list key
	if res := s.delAllSortKeys(true, val); !res {
		return false
	}

	//delete unique list
	if res := s.delAllUniKeys(true, val); !res {
		return false
	}

	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return false
	}
	return s.dba.Delete(keyBuf) == nil
}

////////////// SECTION Members Get/Modify ///////////////
func (s *SoDemoWrap) GetContent() string {
	res := s.getDemo()

	if res == nil {
		var tmpValue string
		return tmpValue
	}
	return res.Content
}

func (s *SoDemoWrap) MdContent(p string) bool {
	if s.dba == nil {
		return false
	}
	sa := s.getDemo()
	if sa == nil {
		return false
	}

	sa.Content = p
	if !s.update(sa) {
		return false
	}

	return true
}

func (s *SoDemoWrap) GetIdx() int64 {
	res := s.getDemo()

	if res == nil {
		var tmpValue int64
		return tmpValue
	}
	return res.Idx
}

func (s *SoDemoWrap) MdIdx(p int64) bool {
	if s.dba == nil {
		return false
	}
	sa := s.getDemo()
	if sa == nil {
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
	if !s.delUniKeyIdx(sa) {
		return false
	}

	if !s.delSortKeyIdx(sa) {
		return false
	}
	sa.Idx = p
	if !s.update(sa) {
		return false
	}

	if !s.insertSortKeyIdx(sa) {
		return false
	}

	if !s.insertUniKeyIdx(sa) {
		return false
	}
	return true
}

func (s *SoDemoWrap) GetLikeCount() int64 {
	res := s.getDemo()

	if res == nil {
		var tmpValue int64
		return tmpValue
	}
	return res.LikeCount
}

func (s *SoDemoWrap) MdLikeCount(p int64) bool {
	if s.dba == nil {
		return false
	}
	sa := s.getDemo()
	if sa == nil {
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
	if !s.delUniKeyLikeCount(sa) {
		return false
	}

	if !s.delSortKeyLikeCount(sa) {
		return false
	}
	sa.LikeCount = p
	if !s.update(sa) {
		return false
	}

	if !s.insertSortKeyLikeCount(sa) {
		return false
	}

	if !s.insertUniKeyLikeCount(sa) {
		return false
	}
	return true
}

func (s *SoDemoWrap) GetOwner() *prototype.AccountName {
	res := s.getDemo()

	if res == nil {
		return nil

	}
	return res.Owner
}

func (s *SoDemoWrap) GetPostTime() *prototype.TimePointSec {
	res := s.getDemo()

	if res == nil {
		return nil

	}
	return res.PostTime
}

func (s *SoDemoWrap) MdPostTime(p *prototype.TimePointSec) bool {
	if s.dba == nil {
		return false
	}
	sa := s.getDemo()
	if sa == nil {
		return false
	}

	if !s.delSortKeyPostTime(sa) {
		return false
	}
	sa.PostTime = p
	if !s.update(sa) {
		return false
	}

	if !s.insertSortKeyPostTime(sa) {
		return false
	}

	return true
}

func (s *SoDemoWrap) GetReplayCount() int64 {
	res := s.getDemo()

	if res == nil {
		var tmpValue int64
		return tmpValue
	}
	return res.ReplayCount
}

func (s *SoDemoWrap) MdReplayCount(p int64) bool {
	if s.dba == nil {
		return false
	}
	sa := s.getDemo()
	if sa == nil {
		return false
	}

	if !s.delSortKeyReplayCount(sa) {
		return false
	}
	sa.ReplayCount = p
	if !s.update(sa) {
		return false
	}

	if !s.insertSortKeyReplayCount(sa) {
		return false
	}

	return true
}

func (s *SoDemoWrap) GetTaglist() []string {
	res := s.getDemo()

	if res == nil {
		var tmpValue []string
		return tmpValue
	}
	return res.Taglist
}

func (s *SoDemoWrap) MdTaglist(p []string) bool {
	if s.dba == nil {
		return false
	}
	sa := s.getDemo()
	if sa == nil {
		return false
	}

	if !s.delSortKeyTaglist(sa) {
		return false
	}
	sa.Taglist = p
	if !s.update(sa) {
		return false
	}

	if !s.insertSortKeyTaglist(sa) {
		return false
	}

	return true
}

func (s *SoDemoWrap) GetTitle() string {
	res := s.getDemo()

	if res == nil {
		var tmpValue string
		return tmpValue
	}
	return res.Title
}

func (s *SoDemoWrap) MdTitle(p string) bool {
	if s.dba == nil {
		return false
	}
	sa := s.getDemo()
	if sa == nil {
		return false
	}

	sa.Title = p
	if !s.update(sa) {
		return false
	}

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

func (s *SDemoOwnerWrap) DelIterater(iterator iservices.IDatabaseIterator) {
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
func (s *SDemoOwnerWrap) QueryListByRevOrder(start *prototype.AccountName, end *prototype.AccountName) iservices.IDatabaseIterator {
	if s.Dba == nil {
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
		return nil
	}
	eKeyList := []interface{}{pre}
	if end != nil {
		eKeyList = append(eKeyList, end)
	}
	eBuf, cErr := kope.EncodeSlice(eKeyList)
	if cErr != nil {
		return nil
	}
	//reverse the start and end when create ReversedIterator to query by reverse order
	iter := s.Dba.NewReversedIterator(eBuf, sBuf)
	return iter
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

func (s *SDemoPostTimeWrap) DelIterater(iterator iservices.IDatabaseIterator) {
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
//start = nil  end = nil (query the db from start to end)
//start = nil (query from start the db)
//end = nil (query to the end of db)
func (s *SDemoPostTimeWrap) QueryListByOrder(start *prototype.TimePointSec, end *prototype.TimePointSec) iservices.IDatabaseIterator {
	if s.Dba == nil {
		return nil
	}
	pre := DemoPostTimeTable
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

//Query sort by reverse order
func (s *SDemoPostTimeWrap) QueryListByRevOrder(start *prototype.TimePointSec, end *prototype.TimePointSec) iservices.IDatabaseIterator {
	if s.Dba == nil {
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
		return nil
	}
	eKeyList := []interface{}{pre}
	if end != nil {
		eKeyList = append(eKeyList, end)
	}
	eBuf, cErr := kope.EncodeSlice(eKeyList)
	if cErr != nil {
		return nil
	}
	//reverse the start and end when create ReversedIterator to query by reverse order
	iter := s.Dba.NewReversedIterator(eBuf, sBuf)
	return iter
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

func (s *SDemoLikeCountWrap) DelIterater(iterator iservices.IDatabaseIterator) {
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
func (s *SDemoLikeCountWrap) QueryListByRevOrder(start *int64, end *int64) iservices.IDatabaseIterator {
	if s.Dba == nil {
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
		return nil
	}
	eKeyList := []interface{}{pre}
	if end != nil {
		eKeyList = append(eKeyList, end)
	}
	eBuf, cErr := kope.EncodeSlice(eKeyList)
	if cErr != nil {
		return nil
	}
	//reverse the start and end when create ReversedIterator to query by reverse order
	iter := s.Dba.NewReversedIterator(eBuf, sBuf)
	return iter
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

func (s *SDemoIdxWrap) DelIterater(iterator iservices.IDatabaseIterator) {
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
func (s *SDemoIdxWrap) QueryListByRevOrder(start *int64, end *int64) iservices.IDatabaseIterator {
	if s.Dba == nil {
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
		return nil
	}
	eKeyList := []interface{}{pre}
	if end != nil {
		eKeyList = append(eKeyList, end)
	}
	eBuf, cErr := kope.EncodeSlice(eKeyList)
	if cErr != nil {
		return nil
	}
	//reverse the start and end when create ReversedIterator to query by reverse order
	iter := s.Dba.NewReversedIterator(eBuf, sBuf)
	return iter
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

func (s *SDemoReplayCountWrap) DelIterater(iterator iservices.IDatabaseIterator) {
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
//start = nil  end = nil (query the db from start to end)
//start = nil (query from start the db)
//end = nil (query to the end of db)
func (s *SDemoReplayCountWrap) QueryListByOrder(start *int64, end *int64) iservices.IDatabaseIterator {
	if s.Dba == nil {
		return nil
	}
	pre := DemoReplayCountTable
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

func (s *SDemoTaglistWrap) DelIterater(iterator iservices.IDatabaseIterator) {
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
//start = nil  end = nil (query the db from start to end)
//start = nil (query from start the db)
//end = nil (query to the end of db)
func (s *SDemoTaglistWrap) QueryListByOrder(start *[]string, end *[]string) iservices.IDatabaseIterator {
	if s.Dba == nil {
		return nil
	}
	pre := DemoTaglistTable
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

func (s *SoDemoWrap) update(sa *SoDemo) bool {
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
	pre := DemoTable
	sub := s.mainKey
	if sub == nil {
		return nil, errors.New("the mainKey is nil")
	}
	kList := []interface{}{pre, sub}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
}

////////////// Unique Query delete/insert/query ///////////////

func (s *SoDemoWrap) delAllUniKeys(br bool, val *SoDemo) bool {
	if s.dba == nil {
		return false
	}
	if val == nil {
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

func (s *SoDemoWrap) insertAllUniKeys(val *SoDemo) error {
	if s.dba == nil {
		return errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert uniuqe Field fail,get the SoDemo fail ")
	}
	if !s.insertUniKeyIdx(val) {
		return errors.New("insert unique Field Idx fail while insert table ")
	}
	if !s.insertUniKeyLikeCount(val) {
		return errors.New("insert unique Field LikeCount fail while insert table ")
	}
	if !s.insertUniKeyOwner(val) {
		return errors.New("insert unique Field Owner fail while insert table ")
	}

	return nil
}

func (s *SoDemoWrap) delUniKeyIdx(sa *SoDemo) bool {
	if s.dba == nil {
		return false
	}
	pre := DemoIdxUniTable
	sub := sa.Idx
	kList := []interface{}{pre, sub}
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
	uniWrap := UniDemoIdxWrap{}
	uniWrap.Dba = s.dba
	res := uniWrap.UniQueryIdx(&sa.Idx)

	if res != nil {
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

	pre := DemoIdxUniTable
	sub := sa.Idx
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
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
	sub := sa.LikeCount
	kList := []interface{}{pre, sub}
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
	uniWrap := UniDemoLikeCountWrap{}
	uniWrap.Dba = s.dba
	res := uniWrap.UniQueryLikeCount(&sa.LikeCount)

	if res != nil {
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

	pre := DemoLikeCountUniTable
	sub := sa.LikeCount
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
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
	sub := sa.Owner
	kList := []interface{}{pre, sub}
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
	uniWrap := UniDemoOwnerWrap{}
	uniWrap.Dba = s.dba

	res := uniWrap.UniQueryOwner(sa.Owner)
	if res != nil {
		//the unique key is already exist
		return false
	}
	val := SoUniqueDemoByOwner{}
	val.Owner = sa.Owner

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	pre := DemoOwnerUniTable
	sub := sa.Owner
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
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
