package table

import (
	"github.com/coschain/contentos-go/common/encoding"
	"github.com/coschain/contentos-go/db/storage"
	base "github.com/coschain/contentos-go/common/prototype"
	"github.com/gogo/protobuf/proto"
)

////////////// SECTION Prefix Mark ///////////////
var (
	mainTable        = []byte{0x1, 0x0}

	CreatedTimeTable = []byte{0x1, 1 + 0x0 }

	PubKeyTable = []byte{0x1, 1 + 0x1 }

)

////////////// SECTION Wrap Define ///////////////
type SoAccountWrap struct {
	dba 		storage.Database
	mainKey 	*base.AccountName
}

func NewSoAccountWrap(dba storage.Database, key *base.AccountName) *SoAccountWrap{
	result := &SoAccountWrap{ dba, key}
	return result
}

func (s *SoAccountWrap) CheckExist() bool {
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

func (s *SoAccountWrap) CreateAccount(sa *SoAccount) bool {

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

	if !s.insertSubKeyCreatedTime(sa) {
		return false
	}

	if !s.insertSubKeyPubKey(sa) {
		return false
	}


	return true
}

////////////// SECTION SubKeys delete/insert ///////////////


func (s *SoAccountWrap) deleteSubKeyCreatedTime(sa *SoAccount) bool {
	val := SKeyAccountByCreatedTime{}

	val.CreatedTime = sa.CreatedTime
	val.Name = sa.Name

	key, err := encoding.Encode(&val)

	if err != nil {
		return false
	}

	return s.dba.Delete(key) == nil
}


func (s *SoAccountWrap) insertSubKeyCreatedTime(sa *SoAccount) bool {
	val := SKeyAccountByCreatedTime{}

	val.Name = sa.Name
	val.CreatedTime = sa.CreatedTime

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


func (s *SoAccountWrap) deleteSubKeyPubKey(sa *SoAccount) bool {
	val := SKeyAccountByPubKey{}

	val.PubKey = sa.PubKey
	val.Name = sa.Name

	key, err := encoding.Encode(&val)

	if err != nil {
		return false
	}

	return s.dba.Delete(key) == nil
}


func (s *SoAccountWrap) insertSubKeyPubKey(sa *SoAccount) bool {
	val := SKeyAccountByPubKey{}

	val.Name = sa.Name
	val.PubKey = sa.PubKey

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




func (s *SoAccountWrap) RemoveAccount() bool {

	sa := s.getAccount()

	if sa == nil {
		return false
	}


	if !s.deleteSubKeyCreatedTime(sa) {
		return false
	}

	if !s.deleteSubKeyPubKey(sa) {
		return false
	}


	keyBuf, err := s.encodeMainKey()

	if err != nil {
		return false
	}

	return s.dba.Delete(keyBuf) == nil
}

////////////// SECTION Members Get/Modify ///////////////


func (s *SoAccountWrap) GetAccountCreatedTime() *base.TimePointSec {
	res := s.getAccount()

	if res == nil {
		return nil
	}
	return res.CreatedTime
}


func (s *SoAccountWrap) MdAccountCreatedTime(p base.TimePointSec) bool {

	sa := s.getAccount()

	if sa == nil {
		return false
	}



	if !s.deleteSubKeyCreatedTime(sa) {
		return false
	}




	sa.CreatedTime = &p
	if !s.update(sa) {
		return false
	}


	if !s.insertSubKeyCreatedTime(sa) {
		return false
	}




	return true
}


func (s *SoAccountWrap) GetAccountCreator() *base.AccountName {
	res := s.getAccount()

	if res == nil {
		return nil
	}
	return res.Creator
}


func (s *SoAccountWrap) MdAccountCreator(p base.AccountName) bool {

	sa := s.getAccount()

	if sa == nil {
		return false
	}






	sa.Creator = &p
	if !s.update(sa) {
		return false
	}





	return true
}


func (s *SoAccountWrap) GetAccountPubKey() *base.PublicKeyType {
	res := s.getAccount()

	if res == nil {
		return nil
	}
	return res.PubKey
}


func (s *SoAccountWrap) MdAccountPubKey(p base.PublicKeyType) bool {

	sa := s.getAccount()

	if sa == nil {
		return false
	}





	if !s.deleteSubKeyPubKey(sa) {
		return false
	}


	sa.PubKey = &p
	if !s.update(sa) {
		return false
	}




	if !s.insertSubKeyPubKey(sa) {
		return false
	}


	return true
}





////////////// SECTION List Keys ///////////////

func (m *SKeyAccountByCreatedTime) OpeEncode() ([]byte, error) {

	mainBuf, err := encoding.Encode(m.Name)
	if err != nil {
		return nil, err
	}
	subBuf, err := encoding.Encode(m.CreatedTime)
	if err != nil {
		return nil, err
	}

	return append(append(CreatedTimeTable, subBuf...), mainBuf...), nil
}

type SListAccountByCreatedTime struct {
	Dba storage.Database
}

func (s *SListAccountByCreatedTime) GetMainVal(iterator storage.Iterator) *base.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SKeyAccountByCreatedTime{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return res.Name
}

func (s *SListAccountByCreatedTime) GetSubVal(iterator storage.Iterator) *base.TimePointSec {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SKeyAccountByCreatedTime{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return res.CreatedTime
}

func (s *SListAccountByCreatedTime) DoList(start base.TimePointSec, end base.TimePointSec) storage.Iterator {

	startBuf, err := encoding.Encode(&start)
	if err != nil {
		return nil
	}

	endBuf, err := encoding.Encode(&end)
	if err != nil {
		return nil
	}

	bufStartkey := append(CreatedTimeTable, startBuf...)
	bufEndkey := append(CreatedTimeTable, endBuf...)

	iter := s.Dba.NewIterator(bufStartkey, bufEndkey)

	return iter
}


////////////// SECTION List Keys ///////////////

func (m *SKeyAccountByPubKey) OpeEncode() ([]byte, error) {

	mainBuf, err := encoding.Encode(m.Name)
	if err != nil {
		return nil, err
	}
	subBuf, err := encoding.Encode(m.PubKey)
	if err != nil {
		return nil, err
	}

	return append(append(PubKeyTable, subBuf...), mainBuf...), nil
}

type SListAccountByPubKey struct {
	Dba storage.Database
}

func (s *SListAccountByPubKey) GetMainVal(iterator storage.Iterator) *base.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SKeyAccountByPubKey{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return res.Name
}

func (s *SListAccountByPubKey) GetSubVal(iterator storage.Iterator) *base.PublicKeyType {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SKeyAccountByPubKey{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return res.PubKey
}

func (s *SListAccountByPubKey) DoList(start base.PublicKeyType, end base.PublicKeyType) storage.Iterator {

	startBuf, err := encoding.Encode(&start)
	if err != nil {
		return nil
	}

	endBuf, err := encoding.Encode(&end)
	if err != nil {
		return nil
	}

	bufStartkey := append(PubKeyTable, startBuf...)
	bufEndkey := append(PubKeyTable, endBuf...)

	iter := s.Dba.NewIterator(bufStartkey, bufEndkey)

	return iter
}



/////////////// SECTION Private function ////////////////

func (s *SoAccountWrap) update(sa *SoAccount) bool {
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
	res, err := encoding.Encode(s.mainKey)

	if err != nil {
		return nil, err
	}

	return append(mainTable, res...), nil
}

