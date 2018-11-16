package consensus

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/db/forkdb"
	"github.com/coschain/contentos-go/node"
)

func waitTilNextSec() {
	now := time.Now()
	ceil := now.Add(time.Millisecond * 500).Round(time.Second)
	time.Sleep(ceil.Sub(now))
}

func timeToNextSec() time.Duration {
	now := time.Now()
	ceil := now.Add(time.Millisecond * 500).Round(time.Second)
	return ceil.Sub(now)
}

type Producer struct {
	Name string
	//PubKey *prototype.PublicKeyType
	Weight uint32
}

func (p *Producer) Produce(timestamp uint64, prev common.BlockID) (common.ISignedBlock, error) {
	return nil, nil
}

type DPoS struct {
	ForkDB *forkdb.DB

	Producers      []*Producer
	hostMask       []bool
	producerIdx    uint64
	readyToProduce bool
	prodTimer      *time.Timer
	trxCh          chan common.ISignedTransaction
	blkCh          chan common.ISignedBlock

	bootstrap bool

	slot uint64
	//currentAbsSlot uint64

	//head common.ISignedBlock

	stopCh chan struct{}
	wg     sync.WaitGroup
	sync.RWMutex
}

func NewDPoS() *DPoS {
	return &DPoS{
		ForkDB:    forkdb.NewDB(),
		Producers: make([]*Producer, constants.ProducerNum),
		hostMask:  make([]bool, constants.ProducerNum),
		prodTimer: time.NewTimer(10 * time.Second),
		trxCh:     make(chan common.ISignedTransaction, 5000),
		blkCh:     make(chan common.ISignedBlock),

		stopCh: make(chan struct{}),
	}
}

func (d *DPoS) SetBootstrap(b bool) {
	d.bootstrap = b
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

func (d *DPoS) Start(node *node.Node) error {
	go d.start()
	return nil
}

func (d *DPoS) scheduleProduce() bool {
	if !d.checkGenesis() {
		d.prodTimer.Reset(timeToNextSec())
		return false
	}
	if !d.readyToProduce {
		if d.checkSync() {
			d.readyToProduce = true
		} else {
			d.prodTimer.Reset(timeToNextSec())
			return false
		}
	}
	if !d.checkProducingTiming() || !d.checkOurTurn() {
		d.prodTimer.Reset(timeToNextSec())
		return false
	}
	return true
}

func (d *DPoS) start() {
	d.wg.Add(1)
	defer d.wg.Done()
	d.scheduleProduce()
	for {
		select {
		case <-d.stopCh:
			break
		case b := <-d.blkCh:
			d.pushBlock(b)
		//case trx := <-d.trxCh:
		// handle trx
		case <-d.prodTimer.C:
			if !d.scheduleProduce() {
				continue
			}
			b, err := d.generateBlock()
			if err != nil {
				// TODO: log
				continue
			}
			// TODO: broadcast block
			d.PushBlock(b)
		}
	}
}

func (d *DPoS) Stop() error {
	close(d.stopCh)
	d.wg.Wait()
	return nil
}

func (d *DPoS) generateBlock() (common.ISignedBlock, error) {
	ts := d.getSlotTime(d.slot)
	prev := d.ForkDB.Head().Id()
	return d.Producers[d.producerIdx].Produce(ts, prev)
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

	return true
}

// this'll only be called by the start routine,
// no need to lock
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
		d.producerIdx = idx
		return true
	}
	return false
}

func (d *DPoS) getScheduledProducer(slot uint64) uint64 {
	absSlot := (d.ForkDB.Head().Timestamp() - constants.GenesisTime) / constants.BLOCK_INTERVAL
	return (absSlot + slot) % uint64(len(d.Producers))
}

// returns false if we're out of sync
func (d *DPoS) checkSync() bool {
	now := time.Now().Round(time.Second).Unix()
	if d.getSlotTime(1) < uint64(now) {
		//time.Sleep(time.Second)
		return false
	}
	return true
}

func (d *DPoS) getSlotTime(slot uint64) uint64 {
	if slot == 0 {
		return 0
	}
	head := d.ForkDB.Head()
	if head == nil {
		return constants.GenesisTime + slot*constants.BLOCK_INTERVAL
	}

	headSlotTime := head.Timestamp() / constants.BLOCK_INTERVAL * constants.BLOCK_INTERVAL
	return headSlotTime + slot*constants.BLOCK_INTERVAL
}

func (d *DPoS) getSlotAtTime(t time.Time) uint64 {
	nextSlotTime := d.getSlotTime(1)
	if uint64(t.Unix()) < nextSlotTime {
		return 0
	}
	return (uint64(t.Unix())-nextSlotTime)/constants.BLOCK_INTERVAL + 1
}

func (d *DPoS) PushBlock(b common.ISignedBlock) {
	go func(blk common.ISignedBlock) {
		d.blkCh <- b
	}(b)
}

func (d *DPoS) PushTransaction(trx common.ISignedTransaction) {
	d.trxCh <- trx
}

func (d *DPoS) pushBlock(b common.ISignedBlock) error {
	// TODO: check signee & merkle
	if b.Timestamp() < d.getSlotTime(1) {
		return errors.New("the timestamp of the new block is less than that of the head block")
	}
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
		d.switchFork(head.Id(), newHead.Id())
		return nil
	}

	if err := d.applyBlock(b); err != nil {
		// the block is illegal
		d.ForkDB.MarkAsIllegal(b.Id())
		d.ForkDB.Pop()
		return err
	}

	// TODO:
	//if bytes.Equal(b.GetSignee().(*prototype.PublicKeyType).Data, d.Producers[len(d.Producers)-1].PubKey.Data) {
	//	d.shuffle()
	//}
	return nil
}

func (d *DPoS) switchFork(old, new common.BlockID) {
	branches, err := d.ForkDB.FetchBranch(old, new)
	if err != nil {
		panic(err)
	}
	poppedNum := len(branches[0]) - 1
	for i := 0; i < poppedNum; i++ {
		d.ForkDB.Pop()
		d.popBlock()
	}
	if d.ForkDB.Head().Id() != branches[0][poppedNum] {
		errStr := fmt.Sprintf("[ForkDB][switchFork] pop to root block with id: %d, num: %d",
			d.ForkDB.Head().Id(), d.ForkDB.Head().Id().BlockNum())
		panic(errStr)
	}
	appendedNum := len(branches[1]) - 1
	errWhileSwitch := false
	var newBranchIdx int
	for newBranchIdx := appendedNum - 1; newBranchIdx >= 0; newBranchIdx-- {
		b, err := d.ForkDB.FetchBlock(branches[1][newBranchIdx])
		if err != nil {
			panic(err)
		}
		if d.pushBlock(b) != nil {
			errWhileSwitch = true
			// TODO: peels off this invalid branch to avoid flip-flop switch
			break
		}
	}

	// switch back
	if errWhileSwitch {
		for i := newBranchIdx + 1; i < appendedNum; i++ {
			d.ForkDB.Pop()
			d.popBlock()
		}
		for i := poppedNum - 1; i >= 0; i-- {
			b, err := d.ForkDB.FetchBlock(branches[0][i])
			if err != nil {
				panic(err)
			}
			d.PushBlock(b)
		}
	}
}

func (d *DPoS) applyBlock(b common.ISignedBlock) error {
	// TODO: state db apply
	return nil
}

func (d *DPoS) popBlock() error {
	// TODO: state db revert
	return nil
}
