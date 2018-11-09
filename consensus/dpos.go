package consensus

import (
	"sync"

	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/db/forkdb"
)

type Producer struct {
	Name   string
	Weight uint32
}

type DPoS struct {
	ForkDB    *forkdb.DB
	Porducers []*Producer

	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewDPoS() *DPoS {
	return &DPoS{
		ForkDB:    forkdb.NewDB(),
		Porducers: make([]*Producer, 21),
		stopCh:    make(chan struct{}),
	}
}

func (d *DPoS) CurrentProducer() *Producer {
	return d.Porducers[0]
}

func (d *DPoS) ActiveProducers() []*Producer {
	return d.Porducers
}

func (d *DPoS) Start() {
	go d.start()
}

func (d *DPoS) start() {
	d.wg.Add(1)
	defer d.wg.Done()
	for {
		select {
		case <-d.stopCh:
			break
		default:
			// TODO: try to generate block
		}
	}
}

func (d *DPoS) Stop() {
	close(d.stopCh)
	d.wg.Wait()
}

func (d *DPoS) GenerateBlock() error {
	return nil
}

func (d *DPoS) PushTransaction(trx common.SignedTransactionIF) {

}

func (d *DPoS) ValidateBlock(b common.SignedBlockIF) bool {
	return true
}

func (d *DPoS) AddBlock(b common.SignedBlockIF) error {
	return nil
}

func (d *DPoS) RemoveBlock(bh common.BlockID) {
	d.ForkDB.Remove(bh)
}

func (d *DPoS) ForkRoot(fork1, fork2 common.BlockID) common.BlockID {
	return common.BlockID{}
}

func (d *DPoS) applyBlock(b common.SignedBlockIF) error {
	// TODO: state db apply
	return nil
}

func (d *DPoS) revertBlock(height int) error {
	// TODO: state db revert
	return nil
}
