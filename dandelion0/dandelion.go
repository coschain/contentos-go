package dandelion0

import (
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/coschain/contentos-go/prototype"
)

type Dandelion interface {
	OpenDatabase() error

	//Sign(ops ...interface{}) (*prototype.SignedTransaction, error)

	GenerateBlock()

	GenerateBlocks(count uint32)

	// deadline
	GenerateBlockUntil(timestamp uint32)
	//
	// pass by time
	GenerateBlockFor(timestamp uint32)
	//

	Sign(privKeyStr string, ops ...interface{}) (*prototype.SignedTransaction, error)

	////Validate()
	CreateAccount() error

	Transfer(from, to string, amount uint64, memo string) error

	Fund(name string, amount uint64) error

	GetDB() *storage.DatabaseService

	GetProduced() uint32

	GetAccount(name string) *table.SoAccountWrap

	GeneralPrivKey() string

	Clean()
}
