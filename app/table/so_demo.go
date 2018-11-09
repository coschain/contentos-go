package table

import (
	"bytes"

	"github.com/coschain/contentos-go/common/encoding"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/gogo/protobuf/proto"
)

////////////// SECTION Prefix Mark ///////////////
var (
	DemoTable = []byte("DemoTable")

	DemoPostTimeTable       = []byte("DemoPostTimeTable")
	DemoPostTimeRevOrdTable = []byte("DemoPostTimeRevOrdTable")

	DemoReplayCountTable       = []byte("DemoReplayCountTable")
	DemoReplayCountRevOrdTable = []byte("DemoReplayCountRevOrdTable")

	DemoIdxTable = []byte("DemoIdxTable")

	DemoLikeCountTable = []byte("DemoLikeCountTable")

	DemoOwnerTable = []byte("DemoOwnerTable")
)

////////////// SECTION Wrap Define ///////////////
type SoDemoWrap struct {
	dba     storage.Database
	mainKey *string
}

func NewSoDemoWrap(dba storage.Database, key *string) *SoDemoWrap {
	result := &SoDemoWrap{dba, key}
	return result
}

func (s *SoDemoWrap) CheckExist() bool {
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

func (s *SoDemoWrap) CreateDemo(sa *SoDemo) bool {

	if sa == nil {
		return false
	}

	keyBuf, err := s.encodeMainKey()

	if err != nil {
		return false
	}
	resBuf, err := proto.Marshal(sa)
	if err != nil {
		return false
	}
	err = s.dba.Put(keyBuf, resBuf)
	if err != nil {
		return false
	}

	// update sort list keys

	if !s.insertSortKeyPostTime(sa) {
		return false
	}

	if !s.insertSortKeyReplayCount(sa) {
		return false
	}

	//update unique list
	if !s.insertUniKeyIdx(sa) {
		return false
	}
	if !s.insertUniKeyLikeCount(sa) {
		return false
	}
	if !s.insertUniKeyOwner(sa) {
		return false
	}

	return true
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoDemoWrap) delSortKeyPostTime(sa *SoDemo) bool {
	val := SoListDemoByPostTime{}

	val.PostTime = sa.PostTime
	val.Owner = sa.Owner

	subBuf, err := encoding.Encode(sa.PostTime)
	if err != nil {
		return false
	}
	ordKey := append(DemoPostTimeTable, subBuf...)
	revOrdBuf := append(DemoPostTimeRevOrdTable, subBuf...)
	revOrdKey, revErr := encoding.Complement(revOrdBuf, err)
	if revErr != nil {
		return false
	}
	ordErr := s.dba.Delete(ordKey)
	revOrdErr := s.dba.Delete(revOrdKey)
	if ordErr == nil && revOrdErr == nil {
		return true
	} else {
		return false
	}
}

func (s *SoDemoWrap) insertSortKeyPostTime(sa *SoDemo) bool {
	val := SoListDemoByPostTime{}
	val.Owner = sa.Owner
	val.PostTime = sa.PostTime
	buf, err := proto.Marshal(&val)
	if err != nil {
		return false
	}

	subBuf, err := encoding.Encode(sa.PostTime)
	if err != nil {
		return false
	}
	ordKey := append(DemoPostTimeTable, subBuf...)
	revOrdBuf := append(DemoPostTimeRevOrdTable, subBuf...)
	revOrdKey, revErr := encoding.Complement(revOrdBuf, err)
	if revErr != nil {
		return false
	}
	ordErr := s.dba.Put(ordKey, buf)
	revOrdErr := s.dba.Put(revOrdKey, buf)
	if ordErr == nil && revOrdErr == nil {
		return true
	} else {
		return false
	}
}

func (s *SoDemoWrap) delSortKeyReplayCount(sa *SoDemo) bool {
	val := SoListDemoByReplayCount{}

	val.ReplayCount = sa.ReplayCount
	val.Owner = sa.Owner

	subBuf, err := encoding.Encode(sa.ReplayCount)
	if err != nil {
		return false
	}
	ordKey := append(DemoReplayCountTable, subBuf...)
	revOrdBuf := append(DemoReplayCountRevOrdTable, subBuf...)
	revOrdKey, revErr := encoding.Complement(revOrdBuf, err)
	if revErr != nil {
		return false
	}
	ordErr := s.dba.Delete(ordKey)
	revOrdErr := s.dba.Delete(revOrdKey)
	if ordErr == nil && revOrdErr == nil {
		return true
	} else {
		return false
	}
}

func (s *SoDemoWrap) insertSortKeyReplayCount(sa *SoDemo) bool {
	val := SoListDemoByReplayCount{}
	val.Owner = sa.Owner
	val.ReplayCount = sa.ReplayCount
	buf, err := proto.Marshal(&val)
	if err != nil {
		return false
	}

	subBuf, err := encoding.Encode(sa.ReplayCount)
	if err != nil {
		return false
	}
	ordKey := append(DemoReplayCountTable, subBuf...)
	revOrdBuf := append(DemoReplayCountRevOrdTable, subBuf...)
	revOrdKey, revErr := encoding.Complement(revOrdBuf, err)
	if revErr != nil {
		return false
	}
	ordErr := s.dba.Put(ordKey, buf)
	revOrdErr := s.dba.Put(revOrdKey, buf)
	if ordErr == nil && revOrdErr == nil {
		return true
	} else {
		return false
	}
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoDemoWrap) RemoveDemo() bool {

	sa := s.getDemo()

	if sa == nil {
		return false
	}

	//delete sort list key

	if !s.delSortKeyPostTime(sa) {
		return false
	}

	if !s.delSortKeyReplayCount(sa) {
		return false
	}

	//delete unique list

	if !s.delUniKeyIdx(sa) {
		return false
	}

	if !s.delUniKeyLikeCount(sa) {
		return false
	}

	if !s.delUniKeyOwner(sa) {
		return false
	}

	keyBuf, err := s.encodeMainKey()

	if err != nil {
		return false
	}

	return s.dba.Delete(keyBuf) == nil
}

////////////// SECTION Members Get/Modify ///////////////

func (s *SoDemoWrap) GetContent() *string {
	res := s.getDemo()
	if res == nil {
		return nil
	}
	return &res.Content
}

func (s *SoDemoWrap) MdContent(p string) bool {

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

func (s *SoDemoWrap) GetIdx() *int64 {
	res := s.getDemo()
	if res == nil {
		return nil
	}
	return &res.Idx
}

func (s *SoDemoWrap) MdIdx(p int64) bool {

	sa := s.getDemo()

	if sa == nil {
		return false
	}
	//judge the unique value if is exist
	uniWrap := UniDemoIdxWrap{}
	res := uniWrap.UniQueryIdx(&sa.Idx)
	if res != nil {
		//the unique value to be modified is already exist
		return false
	}
	if !s.delUniKeyIdx(sa) {
		return false
	}

	sa.Idx = p

	if !s.update(sa) {
		return false
	}

	if !s.insertUniKeyIdx(sa) {
		return false
	}

	return true
}

func (s *SoDemoWrap) GetLikeCount() *int64 {
	res := s.getDemo()
	if res == nil {
		return nil
	}
	return &res.LikeCount
}

func (s *SoDemoWrap) MdLikeCount(p int64) bool {

	sa := s.getDemo()

	if sa == nil {
		return false
	}

	//judge the unique value if is exist
	uniWrap := UniDemoLikeCountWrap{}
	res := uniWrap.UniQueryLikeCount(&sa.LikeCount)
	if res != nil {
		//the unique value to be modified is already exist
		return false
	}
	if !s.delUniKeyLikeCount(sa) {
		return false
	}

	sa.LikeCount = p

	if !s.update(sa) {
		return false
	}

	if !s.insertUniKeyLikeCount(sa) {
		return false
	}

	return true
}

func (s *SoDemoWrap) GetOwner() *string {
	res := s.getDemo()
	if res == nil {
		return nil
	}
	return &res.Owner
}

func (s *SoDemoWrap) GetPostTime() *uint32 {
	res := s.getDemo()
	if res == nil {
		return nil
	}
	return &res.PostTime
}

func (s *SoDemoWrap) MdPostTime(p uint32) bool {

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

func (s *SoDemoWrap) GetReplayCount() *int64 {
	res := s.getDemo()
	if res == nil {
		return nil
	}
	return &res.ReplayCount
}

func (s *SoDemoWrap) MdReplayCount(p int64) bool {

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

func (s *SoDemoWrap) GetTaglist() *string {
	res := s.getDemo()
	if res == nil {
		return nil
	}
	return &res.Taglist
}

func (s *SoDemoWrap) MdTaglist(p string) bool {

	sa := s.getDemo()

	if sa == nil {
		return false
	}

	sa.Taglist = p

	if !s.update(sa) {
		return false
	}

	return true
}

func (s *SoDemoWrap) GetTitle() *string {
	res := s.getDemo()
	if res == nil {
		return nil
	}
	return &res.Title
}

func (s *SoDemoWrap) MdTitle(p string) bool {

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

func (m *SoListDemoByPostTime) OpeEncode() ([]byte, []byte, error) {

	mainBuf, err := encoding.Encode(m.Owner)
	if err != nil {
		return nil, nil, err
	}
	subBuf, err := encoding.Encode(m.PostTime)
	if err != nil {
		return nil, nil, err
	}
	ordKey := append(append(DemoPostTimeTable, subBuf...), mainBuf...)
	revOrdBuf := append(append(DemoPostTimeRevOrdTable, subBuf...), mainBuf...)
	revSubKey, revErr := encoding.Complement(revOrdBuf, err)
	if revErr != nil {
		return nil, nil, revErr
	}
	return ordKey, revSubKey, nil
}

type SDemoPostTimeWrap struct {
	Dba storage.Database
}

func (s *SDemoPostTimeWrap) GetMainVal(iterator storage.Iterator) *string {
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

	return &res.Owner

}

func (s *SDemoPostTimeWrap) GetSubVal(iterator storage.Iterator) *uint32 {
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

	return &res.PostTime

}

//Query by sort
//sort by reverse order: the encoded value of start greater than end
//sort by order: the encoded value of start less or equal  end
func (s *SDemoPostTimeWrap) QueryList(start uint32, end uint32) storage.Iterator {

	startBuf, err := encoding.Encode(&start)
	if err != nil {
		return nil
	}
	endBuf, err := encoding.Encode(&end)
	if err != nil {
		return nil
	}
	bufStartkey := append(DemoPostTimeTable, startBuf...)
	bufEndkey := append(DemoPostTimeTable, endBuf...)
	if bytes.Compare(startBuf, endBuf) > 1 {
		//reverse order
		rBufStart, rErr := encoding.Complement(bufStartkey, err)
		if rErr != nil {
			return nil
		}
		rBufEnd, rErr := encoding.Complement(bufEndkey, err)
		if rErr != nil {
			return nil
		}
		iter := s.Dba.NewIterator(rBufStart, rBufEnd)
		return iter
	} else {
		iter := s.Dba.NewIterator(bufStartkey, bufEndkey)
		return iter
	}
}

////////////// SECTION List Keys ///////////////

func (m *SoListDemoByReplayCount) OpeEncode() ([]byte, []byte, error) {

	mainBuf, err := encoding.Encode(m.Owner)
	if err != nil {
		return nil, nil, err
	}
	subBuf, err := encoding.Encode(m.ReplayCount)
	if err != nil {
		return nil, nil, err
	}
	ordKey := append(append(DemoReplayCountTable, subBuf...), mainBuf...)
	revOrdBuf := append(append(DemoReplayCountRevOrdTable, subBuf...), mainBuf...)
	revSubKey, revErr := encoding.Complement(revOrdBuf, err)
	if revErr != nil {
		return nil, nil, revErr
	}
	return ordKey, revSubKey, nil
}

type SDemoReplayCountWrap struct {
	Dba storage.Database
}

func (s *SDemoReplayCountWrap) GetMainVal(iterator storage.Iterator) *string {
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

	return &res.Owner

}

func (s *SDemoReplayCountWrap) GetSubVal(iterator storage.Iterator) *int64 {
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

//Query by sort
//sort by reverse order: the encoded value of start greater than end
//sort by order: the encoded value of start less or equal  end
func (s *SDemoReplayCountWrap) QueryList(start int64, end int64) storage.Iterator {

	startBuf, err := encoding.Encode(&start)
	if err != nil {
		return nil
	}
	endBuf, err := encoding.Encode(&end)
	if err != nil {
		return nil
	}
	bufStartkey := append(DemoReplayCountTable, startBuf...)
	bufEndkey := append(DemoReplayCountTable, endBuf...)
	if bytes.Compare(startBuf, endBuf) > 1 {
		//reverse order
		rBufStart, rErr := encoding.Complement(bufStartkey, err)
		if rErr != nil {
			return nil
		}
		rBufEnd, rErr := encoding.Complement(bufEndkey, err)
		if rErr != nil {
			return nil
		}
		iter := s.Dba.NewIterator(rBufStart, rBufEnd)
		return iter
	} else {
		iter := s.Dba.NewIterator(bufStartkey, bufEndkey)
		return iter
	}
}

/////////////// SECTION Private function ////////////////

func (s *SoDemoWrap) update(sa *SoDemo) bool {
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
	res, err := encoding.Encode(s.mainKey)

	if err != nil {
		return nil, err
	}

	return append(DemoTable, res...), nil
}

////////////// Unique Query delete/insert/query ///////////////

func (s *SoDemoWrap) delUniKeyIdx(sa *SoDemo) bool {
	val := SoUniqueDemoByIdx{}

	val.Idx = sa.Idx
	val.Owner = sa.Owner

	key, err := encoding.Encode(sa.Idx)

	if err != nil {
		return false
	}

	return s.dba.Delete(append(DemoIdxTable, key...)) == nil
}

func (s *SoDemoWrap) insertUniKeyIdx(sa *SoDemo) bool {
	uniWrap := UniDemoIdxWrap{}

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

	key, err := encoding.Encode(sa.Idx)

	if err != nil {
		return false
	}
	return s.dba.Put(append(DemoIdxTable, key...), buf) == nil

}

type UniDemoIdxWrap struct {
	Dba storage.Database
}

func (s *UniDemoIdxWrap) UniQueryIdx(start *int64) *SoDemoWrap {

	startBuf, err := encoding.Encode(start)
	if err != nil {
		return nil
	}
	bufStartkey := append(DemoIdxTable, startBuf...)
	bufEndkey := bufStartkey
	iter := s.Dba.NewIterator(bufStartkey, bufEndkey)
	val, err := iter.Value()
	if err != nil {
		return nil
	}
	res := &SoUniqueDemoByIdx{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	wrap := NewSoDemoWrap(s.Dba, &res.Owner)

	return wrap
}

func (s *SoDemoWrap) delUniKeyLikeCount(sa *SoDemo) bool {
	val := SoUniqueDemoByLikeCount{}

	val.LikeCount = sa.LikeCount
	val.Owner = sa.Owner

	key, err := encoding.Encode(sa.LikeCount)

	if err != nil {
		return false
	}

	return s.dba.Delete(append(DemoLikeCountTable, key...)) == nil
}

func (s *SoDemoWrap) insertUniKeyLikeCount(sa *SoDemo) bool {
	uniWrap := UniDemoLikeCountWrap{}

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

	key, err := encoding.Encode(sa.LikeCount)

	if err != nil {
		return false
	}
	return s.dba.Put(append(DemoLikeCountTable, key...), buf) == nil

}

type UniDemoLikeCountWrap struct {
	Dba storage.Database
}

func (s *UniDemoLikeCountWrap) UniQueryLikeCount(start *int64) *SoDemoWrap {

	startBuf, err := encoding.Encode(start)
	if err != nil {
		return nil
	}
	bufStartkey := append(DemoLikeCountTable, startBuf...)
	bufEndkey := bufStartkey
	iter := s.Dba.NewIterator(bufStartkey, bufEndkey)
	val, err := iter.Value()
	if err != nil {
		return nil
	}
	res := &SoUniqueDemoByLikeCount{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	wrap := NewSoDemoWrap(s.Dba, &res.Owner)

	return wrap
}

func (s *SoDemoWrap) delUniKeyOwner(sa *SoDemo) bool {
	val := SoUniqueDemoByOwner{}

	val.Owner = sa.Owner
	val.Owner = sa.Owner

	key, err := encoding.Encode(sa.Owner)

	if err != nil {
		return false
	}

	return s.dba.Delete(append(DemoOwnerTable, key...)) == nil
}

func (s *SoDemoWrap) insertUniKeyOwner(sa *SoDemo) bool {
	uniWrap := UniDemoOwnerWrap{}

	res := uniWrap.UniQueryOwner(&sa.Owner)

	if res != nil {
		//the unique key is already exist
		return false
	}

	val := SoUniqueDemoByOwner{}

	val.Owner = sa.Owner
	val.Owner = sa.Owner

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	key, err := encoding.Encode(sa.Owner)

	if err != nil {
		return false
	}
	return s.dba.Put(append(DemoOwnerTable, key...), buf) == nil

}

type UniDemoOwnerWrap struct {
	Dba storage.Database
}

func (s *UniDemoOwnerWrap) UniQueryOwner(start *string) *SoDemoWrap {

	startBuf, err := encoding.Encode(start)
	if err != nil {
		return nil
	}
	bufStartkey := append(DemoOwnerTable, startBuf...)
	bufEndkey := bufStartkey
	iter := s.Dba.NewIterator(bufStartkey, bufEndkey)
	val, err := iter.Value()
	if err != nil {
		return nil
	}
	res := &SoUniqueDemoByOwner{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	wrap := NewSoDemoWrap(s.Dba, &res.Owner)

	return wrap
}
