package table


import (
	"github.com/coschain/contentos-go/common/encoding"
	base "github.com/coschain/contentos-go/proto/type-proto"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/gogo/protobuf/proto"
)

var (
	markTable = []byte { 0x0, 0x1 }
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

	return err == nil
}

func (s *SoAccountWrap) RemoveAccount() bool {
	return true
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

	// modify primary key value
	// modify second key
	sa := s.getAccount()

	if sa == nil{
		return false
	}
	sa.CreatedTime = &p

	return s.update(sa)
}

func (s *SoAccountWrap) ModifyPubKey( p base.PublicKeyType) bool {

	// modify primary key value
	// modify second key
	sa := s.getAccount()

	if sa == nil{
		return false
	}
	sa.PubKey = &p

	return s.update(sa)
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

	return append(markTable,res...), nil
}