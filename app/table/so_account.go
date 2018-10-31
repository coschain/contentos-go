package table


import (
	base "github.com/coschain/contentos-go/proto/type-proto"
	"github.com/coschain/contentos-go/db/storage"

)
func SoFetchAccountByName( dba *storage.Database, name *base.AccountName) *SoAccount {
	return nil
}


func SoCreateAccount( dba *storage.Database, sa *SoAccount) bool {
	return true
}

func (m *SoAccount) ModifyName(dba *storage.Database, name base.AccountName) bool {
	return true
}

func (m *SoAccount) ModifyCreatedTime(dba *storage.Database, name base.TimePointSec) bool {

	// modify primary key value
	// modify second key
	return true
}

func (m *SoAccount) ModifyPubKey(dba *storage.Database, name base.PublicKeyType) bool {

	// modify primary key value
	// modify second key
	return true
}

func (m *SoAccount) ModifyCreator(dba *storage.Database, name base.AccountName) bool {

	// modify primary key value
	// modify secondary key
	return true
}

func (m *SoAccount) RemoveSelf(dba *storage.Database) bool {

	// remove primary key value
	// remove all secondary key
	return true
}