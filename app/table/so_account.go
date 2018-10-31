package table


import (
	"fmt"
	"github.com/coschain/contentos-go/common/encoding"
	base "github.com/coschain/contentos-go/proto/type-proto"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/gogo/protobuf/proto"
)

var (
	mainTable 			= []byte { 0x0, 0x1 }
	createdTimeTable 	= []byte { 0x0, 0x2 }
)

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

	res, err := s.dba.Has( keyBuf )
	if err != nil {
		return false
	}

	return res
}

func (s *SoAccountWrap) CreateAccount( sa *SoAccount) bool {

	if sa == nil{
		return false
	}

	keyBuf ,err := s.encodeMainKey()

	if err != nil {
		return false
	}

	resBuf ,err := proto.Marshal( sa )
	if err != nil {
		return false
	}

	err = s.dba.Put(keyBuf, resBuf)

	if err != nil{
		return false
	}

	// update secondary keys
	if !s.insertSubKeyCreatedTime(sa){
		return false
	}

	return true
}

func (s *SoAccountWrap) deleteSubKeyCreatedTime( sa *SoAccount) bool {
	val := SKeyAccountByCreatedTime{}

	val.Name 			= sa.Name
	val.CreatedTime		= sa.CreatedTime

	key, err := encoding.Encode( &val )

	if err != nil {
		return false
	}

	return s.dba.Delete( key ) == nil

}

func (s *SoAccountWrap) insertSubKeyCreatedTime( sa *SoAccount) bool {
	val := SKeyAccountByCreatedTime{}

	val.Name 			= sa.Name
	val.CreatedTime		= sa.CreatedTime

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	key, err := encoding.Encode( &val )

	if err != nil {
		return false
	}

	fmt.Println("insertSubKeyCreatedTime: ", key)
	return s.dba.Put( key, buf) == nil

}

func (s *SoAccountWrap) RemoveAccount() bool {

	sa := s.getAccount()

	if sa == nil{
		return false
	}

	s.deleteSubKeyCreatedTime(sa)


	keyBuf ,err := s.encodeMainKey()

	if err != nil {
		return false
	}

	return s.dba.Delete(keyBuf) == nil
}

func (s *SoAccountWrap) GetAccountName() *base.AccountName {
	res := s.getAccount()

	if res == nil{
		return nil
	}
	return res.Name
}

func (s *SoAccountWrap) GetAccountCreatedTime() *base.TimePointSec {
	res := s.getAccount()

	if res == nil{
		return nil
	}
	return res.CreatedTime
}

func (s *SoAccountWrap) GetAccountCreator() *base.AccountName {
	res := s.getAccount()

	if res == nil{
		return nil
	}
	return res.Creator
}

func (s *SoAccountWrap) ModifyCreatedTime( p base.TimePointSec) bool {

	sa := s.getAccount()

	if sa == nil{
		return false
	}

	if !s.deleteSubKeyCreatedTime( sa ){
		return false
	}
	sa.CreatedTime = &p

	if !s.update(sa) {
		return false
	}

	return s.insertSubKeyCreatedTime(sa)
}

func (s *SoAccountWrap) ModifyPubKey( p base.PublicKeyType) bool {

	sa := s.getAccount()

	if sa == nil{
		return false
	}
	sa.PubKey = &p

	return s.update(sa)
}

func (m *SKeyAccountByCreatedTime) OpeEncode() ([]byte, error) {

	mainBuf, err := encoding.Encode(m.Name)
	if err != nil {
		return nil, err
	}
	subBuf, err := encoding.Encode(m.CreatedTime)
	if err != nil {
		return nil, err
	}

	return append( append(createdTimeTable, subBuf...), mainBuf...), nil
}


type SListAccountByCreatedTime struct {
	Dba    storage.Database
}

func (s *SListAccountByCreatedTime) GetMainVal( iterator storage.Iterator ) *base.AccountName {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val , err := iterator.Value()

	if err != nil{
		return nil
	}

	res := &SKeyAccountByCreatedTime{}
	err = proto.Unmarshal( val, res )

	if err != nil {
		return nil
	}

	return res.Name
}

func (s *SListAccountByCreatedTime) GetSubVal( iterator storage.Iterator ) *base.TimePointSec {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val , err := iterator.Value()

	if err != nil{
		return nil
	}

	res := &SKeyAccountByCreatedTime{}
	err = proto.Unmarshal( val, res )

	if err != nil {
		return nil
	}

	return res.CreatedTime
}

func (s *SListAccountByCreatedTime) DoList( start base.TimePointSec, end base.TimePointSec ) storage.Iterator {

	startBuf, err := encoding.Encode( &start )
	if err != nil {
		return nil
	}

	endBuf, err := encoding.Encode( &end )
	if err != nil {
		return nil
	}

	bufStartkey := append( createdTimeTable, startBuf...)
	bufEndkey   := append( createdTimeTable, endBuf...)

	//fmt.Println("find start: ", bufStartkey)
	//fmt.Println("find start: ", bufEndkey)

	iter := s.Dba.NewIterator(bufStartkey, bufEndkey)

	return iter
}

func (s *SoAccountWrap) ModifyCreator( p base.AccountName) bool {

	// modify primary key value
	// modify secondary key

	sa := s.getAccount()

	if sa == nil{
		return false
	}
	sa.Creator = &p

	return s.update(sa)
}


func (s *SoAccountWrap) update( sa *SoAccount) bool {
	buf, err := proto.Marshal(sa)
	if err != nil {
		return false
	}

	keyBuf ,err := s.encodeMainKey()
	if err != nil {
		return false
	}

	return s.dba.Put( keyBuf, buf) == nil
}

func (s *SoAccountWrap) getAccount() *SoAccount  {
	keyBuf ,err := s.encodeMainKey()

	if err != nil {
		return nil
	}

	resBuf, err := s.dba.Get( keyBuf )

	if err != nil {
		return nil
	}

	res := &SoAccount{}
	if proto.Unmarshal( resBuf, res) != nil{
		return nil
	}
	return res
}

func (s* SoAccountWrap) encodeMainKey() ([]byte, error) {
	res, err := encoding.Encode(s.mainKey)

	if err != nil {
		return nil, err
	}

	return append(mainTable,res...), nil
}
