package dandelion

import (
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/prototype"
)

type Dandelion interface {
	OpenDatabase() error

	//Sign(ops ...interface{}) (*prototype.SignedTransaction, error)

	GenerateBlock()

	GenerateBlocks(count uint32)

	SetWitness(name string, privKey *prototype.PrivateKeyType)

	// deadline
	GenerateBlockUntil(timestamp uint32)
	//
	// pass by time
	GenerateBlockFor(timestamp uint32)
	//
	////Validate()
	CreateAccount() error

	Transfer(from, to string, amount uint64, memo string) error

	Fund(name string, amount uint64) error

	GetProduced() uint32

	GetTimestamp() uint32

	GetAccount(name string) *table.SoAccountWrap

	Clean()
}
