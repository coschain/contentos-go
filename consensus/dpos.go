package consensus

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/common/logging"
	"github.com/coschain/contentos-go/db/blocklog"
	"github.com/coschain/contentos-go/db/forkdb"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/prototype"
	//"github.com/coschain/contentos-go/app"
)

func timeToNextSec() time.Duration {
	now := time.Now()
	ceil := now.Add(time.Millisecond * 500).Round(time.Second)
	return ceil.Sub(now)
}

type DPoS struct {
	ForkDB *forkdb.DB
	blog   blocklog.BLog

	Producers []string
	Name      string
	//producerIdx    uint64
	privKey        *prototype.PrivateKeyType
	readyToProduce bool
	prodTimer      *time.Timer
	trxCh          chan common.ISignedTransaction
	blkCh          chan common.ISignedBlock
	bootstrap      bool
	slot           uint64

	ctx  *node.ServiceContext
	ctrl iservices.IController

	stopCh chan struct{}
	wg     sync.WaitGroup
	sync.RWMutex
}

func NewDPoS(ctx *node.ServiceContext) *DPoS {
	ret := &DPoS{
		ForkDB: forkdb.NewDB(),
		//Producers: make([]*Producer, constants.ProducerNum),
		prodTimer: time.NewTimer(1 * time.Millisecond),
		trxCh:     make(chan common.ISignedTransaction, 5000),
		blkCh:     make(chan common.ISignedBlock),
		ctx:       ctx,
		stopCh:    make(chan struct{}),
	}
	ret.Name = constants.COS_INIT_MINER
	var err error
	ret.privKey, err = prototype.PrivateKeyFromWIF(constants.INITMINER_PRIKEY)
	if err != nil {
		panic(err)
	}
	return ret
}

func (d *DPoS) getController() iservices.IController {
	ctrl, err := d.ctx.Service(iservices.CTRL_SERVER_NAME)
	if err != nil {
		panic(err)
	}
	return ctrl.(iservices.IController)
}

func (d *DPoS) SetBootstrap(b bool) {
	d.bootstrap = b
	if d.bootstrap {
		d.readyToProduce = true
		//d.shuffle()
	}
}

func (d *DPoS) CurrentProducer() string {
	//d.RLock()
	//defer d.RUnlock()
	now := time.Now().Round(time.Second)
	slot := d.getSlotAtTime(now)
	return d.getScheduledProducer(slot)
}

// Called when a produce round complete, it adds new producers,
// remove unqualified producers and shuffle the block-producing order
func (d *DPoS) shuffle() {
	prods := d.ctrl.GetWitnessTopN(constants.MAX_WITNESSES)
	var seed uint64
	if !d.ForkDB.Empty() {
		seed = d.ForkDB.Head().Timestamp() << 32
	}
	prodNum := len(prods)
	for i := 0; i < prodNum; i++ {
		k := seed + uint64(i)*2695921657736338717
		k ^= k >> 12
		k ^= k << 25
		k ^= k >> 27
		k *= 2695921657736338717

		j := i + int(k%uint64(prodNum-i))
		prods[i], prods[j] = prods[j], prods[i]
	}

	d.Producers = prods
	d.ctrl.SetShuffledWitness(prods)
}

func (d *DPoS) restoreProducers() {
	d.Producers = d.ctrl.GetShuffledWitness()
}

func (d *DPoS) ActiveProducers() []string {
	//d.RLock()
	//defer d.RUnlock()
	return d.Producers
}

func (d *DPoS) Start(node *node.Node) error {
	d.ctrl = d.getController()
	d.blog.Open(node.Config().DataDir)
	d.SetBootstrap(true)

	go d.start()
	return nil
}

func (d *DPoS) scheduleProduce() bool {
	if !d.checkGenesis() {
		//logging.CLog().Info("checkGenesis failed.")
		d.prodTimer.Reset(timeToNextSec())
		return false
	}

	if !d.readyToProduce {
		if d.checkSync() {
			d.readyToProduce = true
		} else {
			d.prodTimer.Reset(timeToNextSec())
			// TODO: p2p sync
			//logging.CLog().Info("checkSync failed.")
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
	time.Sleep(4 * time.Second)

	logging.CLog().Info("DPoS started.")
	if d.bootstrap && d.ForkDB.Empty() && d.blog.Empty() {
		d.shuffle()
	} else {
		d.restoreProducers()
	}
	//d.scheduleProduce()
	for {
		select {
		case <-d.stopCh:
			break
		case b := <-d.blkCh:
			if err := d.pushBlock(b); err != nil {
				logging.CLog().Error("push block error: ", err)
			}
		//case trx := <-d.trxCh:
		// handle trx
		case <-d.prodTimer.C:
			//logging.CLog().Debug("scheduleProduce.")
			//logging.CLog().Debug("producers: ", d.Producers)
			if !d.scheduleProduce() {
				continue
			}
			d.prodTimer.Reset(1 * time.Second)
			b, err := d.generateBlock()
			if err != nil {
				logging.CLog().Error("generating block error: ", err)
				continue
			}
			logging.CLog().Debugf("generated block: <id %v> <ts %d> <prev %d>", b.Id(), b.Timestamp(), b.Previous())
			// TODO: broadcast block
			d.PushBlock(b)
		}
	}
}

func (d *DPoS) Stop() error {
	logging.CLog().Info("DPoS consensus stopped.")
	// TODO: flush mainchain in the forkDB so that it can be restored when restarted??
	close(d.stopCh)
	d.wg.Wait()
	return nil
}

func (d *DPoS) generateBlock() (common.ISignedBlock, error) {
	//logging.CLog().Debug("generateBlock.")
	ts := d.getSlotTime(d.slot)
	prev := &prototype.Sha256{}
	if !d.ForkDB.Empty() {
		prev.FromBlockID(d.ForkDB.Head().Id())
		//logging.CLog().Debug("xxxxxxxxxxxxx ", d.ForkDB.Head().Id())
	} else {
		prev.Hash = make([]byte, 32)
	}
	//logging.CLog().Debugf("generating block. <prev %v>, <ts %d>", prev.Hash, ts)
	return d.ctrl.GenerateBlock(d.Name, prev, uint32(ts), d.privKey, prototype.Skip_nothing), nil
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
		//logging.CLog().Info("checkProducingTiming failed.")
		return false
	}
	return true
}

func (d *DPoS) checkOurTurn() bool {
	producer := d.getScheduledProducer(d.slot)
	ret := strings.Compare(d.Name, producer) == 0
	if !ret {
		//logging.CLog().Info("checkProducingTiming failed.")
	}
	return ret
}

func (d *DPoS) getScheduledProducer(slot uint64) string {
	if d.ForkDB.Empty() {
		return d.Producers[0]
	}
	absSlot := (d.ForkDB.Head().Timestamp() - constants.GenesisTime) / constants.BLOCK_INTERVAL
	return d.Producers[(absSlot+slot)%uint64(len(d.Producers))]
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
	logging.CLog().Debug("pushBlock #", b.Id().BlockNum())
	//d.Lock()
	//defer d.Unlock()
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
		logging.CLog().Debug("[pushBlock]possibly detached block. prev: got %v, want %v", b.Id(), head.Id())
		// TODO: if it's detached, trigger sync
		return nil
	} else if head != nil && newHead.Previous() != head.Id() {
		d.switchFork(head.Id(), newHead.Id())
		return nil
	}

	if err := d.applyBlock(b); err != nil {
		// the block is illegal
		d.ForkDB.MarkAsIllegal(b.Id())
		d.ForkDB.Pop()
		return err
	}

	// shuffle
	if d.ForkDB.Head().Id().BlockNum()%uint64(len(d.Producers)) == 0 {
		d.shuffle()
	}

	lastCommitted := d.ForkDB.LastCommitted()
	var commitIdx uint64
	if newHead.Id().BlockNum()-lastCommitted.BlockNum() > 3/*constants.MAX_WITNESSES*2/3*/ {
		if lastCommitted == common.EmptyBlockID {
			commitIdx = 1
		} else {
			commitIdx = lastCommitted.BlockNum() + 1
		}
		b, err := d.ForkDB.FetchBlockFromMainBranch(commitIdx)
		if err != nil {
			return err
		}
		return d.commit(b)
	}
	return nil
}

func (d *DPoS) commit(b common.ISignedBlock) error {
	logging.CLog().Debug("commit block #", b.Id().BlockNum())
	// TODO: state db commit
	err := d.blog.Append(b)
	if err != nil {
		// something went really wrong if we got here
		panic(err)
	}

	d.ForkDB.Commit(b.Id())
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
		//d.popBlock()
	}
	d.popBlock(branches[0][poppedNum])

	// producers fixup
	d.restoreProducers()

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
			//d.popBlock()
		}
		d.popBlock(branches[0][poppedNum])

		// producers fixup
		d.restoreProducers()

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
	//logging.CLog().Debug("applyBlock #", b.Id().BlockNum())
	d.ctrl.PushBlock(b.(*prototype.SignedBlock), prototype.Skip_nothing)
	return nil
}

func (d *DPoS) popBlock(id common.BlockID) error {
	d.ctrl.Pop(&id)
	return nil
}

func (d *DPoS) GetHeadBlockId() common.BlockID {
	return d.ForkDB.Head().Id()
}

func (d *DPoS) GetHashes(start, end common.BlockID) []common.BlockID {
	ret := make([]common.BlockID, 10)
	length := 0
	for end != start {
		b, err := d.ForkDB.FetchBlock(end)
		if err != nil {
			return nil
		}
		ret = append(ret, end)
		end = b.Previous()
		length++
	}
	ret = append(ret, end)
	ret = ret[:length]
	for i := 0; i <= (length-1)/2; i++ {
		ret[i], ret[length-1-i] = ret[length-1-i], ret[i]
	}
	return ret
}

func (d *DPoS) FetchBlock(id common.BlockID) (common.ISignedBlock, error) {
	return d.ForkDB.FetchBlock(id)
}

func (d *DPoS) HasBlock(id common.BlockID) bool {
	_, err := d.ForkDB.FetchBlock(id)
	return err == nil
}
