package consensus

import (
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/db/forkdb"
	"github.com/coschain/contentos-go/proto/type-proto"
)

type Producer struct {
	Name   string
	Weight uint32
}

type DPoS struct {
	ForkDB    *forkdb.DB
	Porducers []*Producer
}

func (d *DPoS) CurrentProducer() *Producer {

}

func (d *DPoS) ActiveProducers() []*Producer {

}

func (d *DPoS) Start() {

}

func (d *DPoS) Stop() {

}

func (d *DPoS) GenerateBlock() error {

}

func (d *DPoS) PushTransaction(trx *prototype.SignedTransaction) {

}

func (d *DPoS) ValidateBlock(b *prototype.SignedBlock) bool {

}

func (d *DPoS) AddBlock(b *prototype.SignedBlock) error {

}

func (d *DPoS) RemoveBlock(bh common.BlockID) {

}

func (d *DPoS) ForkRoot(fork1, fork2 common.BlockID) common.BlockID {

}

func (d *DPoS) applyBlock(b *prototype.SignedBlock) error {

}

func (d *DPoS) revertBlock(height int) error {

}
