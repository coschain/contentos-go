package consensus

import (
	"bytes"
	"sync"
	"time"

	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/db/forkdb"
	"github.com/coschain/contentos-go/prototype"
)

type Producer struct {
	Name   string
	PubKey *prototype.PublicKeyType
	Weight uint32
}

func (p *Producer) Produce() (common.ISignedBlock, error) {
	return nil, nil
}

type DPoS struct {
	ForkDB *forkdb.DB

	Producers []*Producer
	hostMask  []bool
	activeNum uint64
	producing bool

	slot           uint64
	currentAbsSlot uint64

	stopCh chan struct{}
	wg     sync.WaitGroup
	sync.RWMutex
}

func NewDPoS() *DPoS {
	return &DPoS{
		ForkDB:    forkdb.NewDB(),
		Producers: make([]*Producer, constants.ProducerNum),
		hostMask:  make([]bool, constants.ProducerNum),
		stopCh:    make(chan struct{}),
	}
}

func (d *DPoS) SetProducer(prod bool) {
	d.producing = prod
}

func (d *DPoS) CurrentProducer() *Producer {
	d.RLock()
	defer d.RUnlock()
	return d.Producers[0]
}

// Called when a produce round complete, it adds new producers,
// remove unqualified producers and shuffle the block-producing order
func (d *DPoS) shuffle() {}

func (d *DPoS) ActiveProducers() []*Producer {
	d.RLock()
	defer d.RUnlock()
	return d.Producers
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
			if !d.producing {
				time.Sleep(time.Second)
				continue
			}
			if !d.checkGenesis() || !d.checkSync() ||
				!d.checkProducingTiming() || !d.checkOurTurn() {
				now := time.Now()
				ceil := now.Add(time.Millisecond * 500).Round(time.Second)
				time.Sleep(ceil.Sub(now))
				continue
			}
			b, err := d.GenerateBlock()
			if err != nil {
				d.PushBlock(b)
				// TODO: broadcast block
			}
		}
	}
}

func (d *DPoS) Stop() {
	close(d.stopCh)
	d.wg.Wait()
}

func (d *DPoS) GenerateBlock() (common.ISignedBlock, error) {

	return nil, nil
}

func (d *DPoS) checkGenesis() bool {
	now := time.Now()
	genesisTime := time.Unix(constants.GenesisTime, 0)
	if now.After(genesisTime) || now.Equal(genesisTime) {
		return true
	}

	ceil := now.Round(time.Second)
	if ceil.Before(now) {
		ceil = ceil.Add(time.Second)
	}

	if ceil.Before(genesisTime) {
		//time.Sleep(ceil.Sub(now))
		return false
	}
}

func (d *DPoS) checkProducingTiming() bool {
	now := time.Now().Round(time.Second)
	d.slot = d.getSlotAtTime(now)
	if d.slot == 0 {
		// not time yet, wait till the next block producing
		// cycle comes
		//nextSlotTime := d.getSlotTime(1)
		//time.Sleep(time.Unix(int64(nextSlotTime), 0).Sub(time.Now()))
		return false
	}
	return true
}

func (d *DPoS) checkOurTurn() bool {
	idx := d.getScheduledProducer(d.slot)
	if d.hostMask[idx] == true {
		return true
	}
	return false
}

func (d *DPoS) getScheduledProducer(slot uint64) int {
	// TODO:
	return 0
}

// returns true if we're out of sync
func (d *DPoS) checkSync() bool {
	now := time.Now().Round(time.Second).Unix()
	if d.getSlotTime(1) < uint64(now) {
		//time.Sleep(time.Second)
		return false
	}
	return true
}

func (d *DPoS) getSlotTime(slot uint64) uint64 {
	return 0
}

func (d *DPoS) getSlotAtTime(t time.Time) uint64 {
	nextSlotTime := d.getSlotTime(1)
	if uint64(t.Unix()) < nextSlotTime {
		return 0
	}
	return (uint64(t.Unix())-nextSlotTime)/constants.BLOCK_INTERNAL + 1
}

func (d *DPoS) PushBlock(b common.ISignedBlock) error {
	// TODO: check signee
	head := d.ForkDB.Head()
	newHead := d.ForkDB.PushBlock(b)

	if newHead == head {
		// this implies that b is a:
		// 1. detached block or
		// 2. out of range block or
		// 3. head of a non-main branch or
		// 4. illegal block
		return nil
	} else if newHead.Previous() != head.Id() {
		// TODO: swith fork
		return nil
	}

	if err := d.applyBlock(b); err != nil {
		// the block is illegal
		d.ForkDB.MarkAsIllegal(b.Id())
		d.ForkDB.Pop()
		return err
	}

	if bytes.Equal(b.GetSignee().(*prototype.PublicKeyType).Data, d.Producers[d.activeNum-1].PubKey.Data) {
		d.shuffle()
	}
	return nil
}

func (d *DPoS) RemoveBlock(id common.BlockID) {
	d.ForkDB.Remove(id)
}

func (d *DPoS) ForkRoot(fork1, fork2 common.BlockID) common.BlockID {
	return common.BlockID{}
}

func (d *DPoS) applyBlock(b common.ISignedBlock) error {
	// TODO: state db apply
	return nil
}

func (d *DPoS) popBlock() error {
	// TODO: state db revert
	return nil
}
