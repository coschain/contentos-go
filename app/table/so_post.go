package table

import (
	"github.com/coschain/contentos-go/common/encoding"
	"github.com/coschain/contentos-go/db/storage"
	base "github.com/coschain/contentos-go/common/prototype"
	"github.com/gogo/protobuf/proto"
)

////////////// SECTION Prefix Mark ///////////////
var (
	postTable        = []byte{0x2, 0x0}

	NameTable = []byte{0x2, 1 + 0x0 }

	PostTimeTable = []byte{0x2, 1 + 0x1 }

)

////////////// SECTION Wrap Define ///////////////
type SoPostWrap struct {
	dba 		storage.Database
	mainKey 	*uint32
}

func NewSoPostWrap(dba storage.Database, key *uint32) *SoPostWrap{
	result := &SoPostWrap{ dba, key}
	return result
}

func (s *SoPostWrap) CheckExist() bool {
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

func (s *SoPostWrap) CreatePost(sa *SoPost) bool {

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

	// update secondary keys

	if !s.insertSubKeyName(sa) {
		return false
	}

	if !s.insertSubKeyPostTime(sa) {
		return false
	}


	return true
}

////////////// SECTION SubKeys delete/insert ///////////////


func (s *SoPostWrap) deleteSubKeyName(sa *SoPost) bool {
	val := SKeyPostByName{}

	val.Name = sa.Name
	val.Idx = sa.Idx

	key, err := encoding.Encode(&val)

	if err != nil {
		return false
	}

	return s.dba.Delete(key) == nil
}


func (s *SoPostWrap) insertSubKeyName(sa *SoPost) bool {
	val := SKeyPostByName{}

	val.Idx = sa.Idx
	val.Name = sa.Name

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	key, err := encoding.Encode(&val)

	if err != nil {
		return false
	}
	return s.dba.Put(key, buf) == nil

}


func (s *SoPostWrap) deleteSubKeyPostTime(sa *SoPost) bool {
	val := SKeyPostByPostTime{}

	val.PostTime = sa.PostTime
	val.Idx = sa.Idx

	key, err := encoding.Encode(&val)

	if err != nil {
		return false
	}

	return s.dba.Delete(key) == nil
}


func (s *SoPostWrap) insertSubKeyPostTime(sa *SoPost) bool {
	val := SKeyPostByPostTime{}

	val.Idx = sa.Idx
	val.PostTime = sa.PostTime

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	key, err := encoding.Encode(&val)

	if err != nil {
		return false
	}
	return s.dba.Put(key, buf) == nil

}




func (s *SoPostWrap) RemovePost() bool {

	sa := s.getPost()

	if sa == nil {
		return false
	}


	if !s.deleteSubKeyName(sa) {
		return false
	}


	if !s.deleteSubKeyPostTime(sa) {
		return false
	}


	keyBuf, err := s.encodeMainKey()

	if err != nil {
		return false
	}

	return s.dba.Delete(keyBuf) == nil
}

////////////// SECTION Members Get/Modify ///////////////


func (s *SoPostWrap) GetPostContent() string {
	res := s.getPost()

	if res == nil {
		return ""
	}
	return res.Content
}


func (s *SoPostWrap) MdPostContent(p string) bool {

	sa := s.getPost()

	if sa == nil {
		return false
	}






	sa.Content = p
	if !s.update(sa) {
		return false
	}





	return true
}


func (s *SoPostWrap) GetPostLikeCount() uint32 {
	res := s.getPost()

	if res == nil {
		return 0
	}
	return res.LikeCount
}


func (s *SoPostWrap) MdPostLikeCount(p uint32) bool {

	sa := s.getPost()

	if sa == nil {
		return false
	}






	sa.LikeCount = p
	if !s.update(sa) {
		return false
	}





	return true
}


func (s *SoPostWrap) GetPostName() *base.AccountName {
	res := s.getPost()

	if res == nil {
		return nil
	}
	return res.Name
}


func (s *SoPostWrap) MdPostName(p base.AccountName) bool {

	sa := s.getPost()

	if sa == nil {
		return false
	}



	if !s.deleteSubKeyName(sa) {
		return false
	}




	sa.Name = &p
	if !s.update(sa) {
		return false
	}


	if !s.insertSubKeyName(sa) {
		return false
	}




	return true
}


func (s *SoPostWrap) GetPostPostTime() *base.TimePointSec {
	res := s.getPost()

	if res == nil {
		return nil
	}
	return res.PostTime
}


func (s *SoPostWrap) MdPostPostTime(p base.TimePointSec) bool {

	sa := s.getPost()

	if sa == nil {
		return false
	}





	if !s.deleteSubKeyPostTime(sa) {
		return false
	}


	sa.PostTime = &p
	if !s.update(sa) {
		return false
	}




	if !s.insertSubKeyPostTime(sa) {
		return false
	}


	return true
}





////////////// SECTION List Keys ///////////////

func (m *SKeyPostByName) OpeEncode() ([]byte, error) {

	mainBuf, err := encoding.Encode(m.Idx)
	if err != nil {
		return nil, err
	}
	subBuf, err := encoding.Encode(m.Name)
	if err != nil {
		return nil, err
	}

	return append(append(NameTable, subBuf...), mainBuf...), nil
}

type SListPostByName struct {
	Dba storage.Database
}

func (s *SListPostByName) GetMainVal(iterator storage.Iterator) *uint32 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SKeyPostByName{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return &res.Idx
}

func (s *SListPostByName) GetSubVal(iterator storage.Iterator) *base.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SKeyPostByName{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return res.Name
}

func (s *SListPostByName) DoList(start base.AccountName, end base.AccountName) storage.Iterator {

	startBuf, err := encoding.Encode(&start)
	if err != nil {
		return nil
	}

	endBuf, err := encoding.Encode(&end)
	if err != nil {
		return nil
	}

	bufStartkey := append(NameTable, startBuf...)
	bufEndkey := append(NameTable, endBuf...)

	iter := s.Dba.NewIterator(bufStartkey, bufEndkey)

	return iter
}


////////////// SECTION List Keys ///////////////

func (m *SKeyPostByPostTime) OpeEncode() ([]byte, error) {

	mainBuf, err := encoding.Encode(m.Idx)
	if err != nil {
		return nil, err
	}
	subBuf, err := encoding.Encode(m.PostTime)
	if err != nil {
		return nil, err
	}

	return append(append(PostTimeTable, subBuf...), mainBuf...), nil
}

type SListPostByPostTime struct {
	Dba storage.Database
}

func (s *SListPostByPostTime) GetMainVal(iterator storage.Iterator) *uint32 {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SKeyPostByPostTime{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return &res.Idx
}

func (s *SListPostByPostTime) GetSubVal(iterator storage.Iterator) *base.TimePointSec {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SKeyPostByPostTime{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return res.PostTime
}

func (s *SListPostByPostTime) DoList(start base.TimePointSec, end base.TimePointSec) storage.Iterator {

	startBuf, err := encoding.Encode(&start)
	if err != nil {
		return nil
	}

	endBuf, err := encoding.Encode(&end)
	if err != nil {
		return nil
	}

	bufStartkey := append(PostTimeTable, startBuf...)
	bufEndkey := append(PostTimeTable, endBuf...)

	iter := s.Dba.NewIterator(bufStartkey, bufEndkey)

	return iter
}



/////////////// SECTION Private function ////////////////

func (s *SoPostWrap) update(sa *SoPost) bool {
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
	res, err := encoding.Encode(s.mainKey)

	if err != nil {
		return nil, err
	}

	return append(postTable, res...), nil
}
