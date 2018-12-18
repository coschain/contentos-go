package consensus

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
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
	iservices.IConsensus
	node   *node.Node
	ForkDB *forkdb.DB
	blog   blocklog.BLog

	Producers []string
	Name      string
	//producerIdx    uint64
	privKey        *prototype.PrivateKeyType
	readyToProduce bool
	prodTimer      *time.Timer
	trxCh          chan func()
	blkCh          chan common.ISignedBlock
	bootstrap      bool
	slot           uint64

	ctx  *node.ServiceContext
	ctrl iservices.ITrxPool
	p2p  iservices.IP2P
	log  iservices.ILog

	stopCh chan struct{}
	wg     sync.WaitGroup
	sync.RWMutex
}

func NewDPoS(ctx *node.ServiceContext) *DPoS {
	logService, err := ctx.Service(iservices.LogServerName)
	if err != nil {
		panic(err)
	}
	ret := &DPoS{
		ForkDB:    forkdb.NewDB(ctx),
		Producers: make([]string, 0, 1),
		prodTimer: time.NewTimer(1 * time.Millisecond),
		trxCh:     make(chan func()),
		//trxRetCh:  make(chan common.ITransactionInvoice),
		blkCh:  make(chan common.ISignedBlock),
		ctx:    ctx,
		stopCh: make(chan struct{}),
		log:    logService.(iservices.ILog),
	}
	ret.SetBootstrap(ctx.Config().Consensus.BootStrap)
	ret.Name = ctx.Config().Consensus.LocalBpName
	ret.log.GetLog().Info("[DPoS bootstrap] ", ctx.Config().Consensus.BootStrap)

	privateKey := ctx.Config().Consensus.LocalBpPrivateKey
	if len(privateKey) > 0 {
		var err error
		ret.privKey, err = prototype.PrivateKeyFromWIF(ctx.Config().Consensus.LocalBpPrivateKey)
		if err != nil {
			panic(err)
		}
	}
	return ret
}

func (d *DPoS) getController() iservices.ITrxPool {
	ctrl, err := d.ctx.Service(iservices.ControlServerName)
	if err != nil {
		panic(err)
	}
	return ctrl.(iservices.ITrxPool)
}

func (d *DPoS) SetBootstrap(b bool) {
	d.bootstrap = b
	if d.bootstrap {
		d.readyToProduce = true
	}
}

func (d *DPoS) CurrentProducer() string {
	//d.RLock()
	//defer d.RUnlock()
	now := time.Now().Round(time.Second)
	slot := d.getSlotAtTime(now)
	return d.getScheduledProducer(slot)
}

func (d *DPoS) shuffle(head common.ISignedBlock) {
	if !d.ForkDB.Empty() && d.ForkDB.Head().Id().BlockNum()%uint64(len(d.Producers)) != 0 {
		return
	}

	// When a produce round complete, it adds new producers,
	// remove unqualified producers and shuffle the block-producing order
	prods := d.ctrl.GetWitnessTopN(constants.MAX_WITNESSES)
	var seed uint64
	if head != nil {
		seed = head.Timestamp() << 32
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
	d.log.GetLog().Debug("[DPoS shuffle] active producers: ", d.Producers)
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
	p2p, err := d.ctx.Service(iservices.P2PServerName)
	if err != nil {
		panic(err)
	}
	d.p2p = p2p.(iservices.IP2P)
	cfg := d.ctx.Config()
	d.blog.Open(cfg.ResolvePath("blog"))
	forkdbPath := cfg.ResolvePath("forkdb_snapshot")
	d.ctrl.SetShuffle(func(block common.ISignedBlock) {
		d.shuffle(block)
	})
	go d.start(forkdbPath)
	return nil
}

func (d *DPoS) scheduleProduce() bool {
	if !d.checkGenesis() {
		//d.log.GetLog().Info("checkGenesis failed.")
		d.prodTimer.Reset(timeToNextSec())
		return false
	}

	if !d.readyToProduce {
		if d.checkSync() {
			d.readyToProduce = true
		} else {
			d.prodTimer.Reset(timeToNextSec())
			var headID common.BlockID
			if !d.ForkDB.Empty() {
				headID = d.ForkDB.Head().Id()
			}
			d.p2p.TriggerSync(headID)
			// TODO:  if we are not on the main branch, pop until the head is on main branch
			d.log.GetLog().Debug("[DPoS TriggerSync]: start from ", headID.BlockNum())
			return false
		}
	}
	if !d.checkProducingTiming() || !d.checkOurTurn() {
		d.prodTimer.Reset(timeToNextSec())
		return false
	}
	return true
}

func (d *DPoS) start(snapshotPath string) {
	d.wg.Add(1)
	defer d.wg.Done()
	time.Sleep(4 * time.Second)
	d.log.GetLog().Info("[DPoS] starting...")

	// TODO: fuck!! this is fugly
	var avatar []common.ISignedBlock
	for i := 0; i < constants.MAX_WITNESSES; i++ {
		// deep copy hell
		avatar = append(avatar, &prototype.SignedBlock{})
	}
	//cfg := d.ctx.Config()
	//d.ForkDB.LoadSnapshot(avatar, cfg.ResolvePath("forkdb_snapshot"))
	d.ForkDB.LoadSnapshot(avatar, snapshotPath)

	if d.bootstrap && d.ForkDB.Empty() && d.blog.Empty() {
		d.log.GetLog().Info("[DPoS] bootstrapping...")
	}
	d.restoreProducers()

	d.log.GetLog().Info("[DPoS] started")
	for {
		select {
		case <-d.stopCh:
			d.log.GetLog().Debug("[DPoS] routine stopped.")
			return
		case b := <-d.blkCh:
			if err := d.pushBlock(b, true); err != nil {
				d.log.GetLog().Error("[DPoS] pushBlock failed: ", err)
			}
		case trx_fn := <-d.trxCh:
			trx_fn()
			continue
		case <-d.prodTimer.C:
			if !d.scheduleProduce() {
				continue
			}

			b, err := d.generateAndApplyBlock()
			if err != nil {
				d.log.GetLog().Error("[DPoS] generateAndApplyBlock error: ", err)
				continue
			}
			d.prodTimer.Reset(timeToNextSec())
			d.log.GetLog().Debugf("[DPoS] generated block: <num %d> <ts %d>", b.Id().BlockNum(), b.Timestamp())
			if err := d.pushBlock(b, false); err != nil {
				d.log.GetLog().Error("[DPoS] pushBlock push generated block failed: ", err)
			}

			// broadcast block
			//if b.Id().BlockNum() % 10 == 0 {
			//	go func() {
			//		time.Sleep(4*time.Second)
			//		d.p2p.Broadcast(b)
			//	}()
			//} else {
			//	d.p2p.Broadcast(b)
			//}
			d.p2p.Broadcast(b)
		}
	}
}

func (d *DPoS) Stop() error {
	d.log.GetLog().Info("DPoS consensus stopped.")
	// restore uncommitted forkdb
	cfg := d.ctx.Config()
	snapshotPath := cfg.ResolvePath("forkdb_snapshot")
	return d.stop(snapshotPath)
}

func (d *DPoS) stop(snapshotPath string) error {
	d.ForkDB.Snapshot(snapshotPath)
	d.prodTimer.Stop()
	close(d.stopCh)
	d.wg.Wait()
	return nil
}

func (d *DPoS) generateAndApplyBlock() (common.ISignedBlock, error) {
	//d.log.GetLog().Debug("generateBlock.")
	ts := d.getSlotTime(d.slot)
	prev := &prototype.Sha256{}
	if !d.ForkDB.Empty() {
		prev.FromBlockID(d.ForkDB.Head().Id())
	} else {
		prev.Hash = make([]byte, 32)
	}
	//d.log.GetLog().Debugf("generating block. <prev %v>, <ts %d>", prev.Hash, ts)
	return d.ctrl.GenerateAndApplyBlock(d.Name, prev, uint32(ts), d.privKey, prototype.Skip_nothing)
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
		//d.log.GetLog().Info("checkProducingTiming failed.")
		return false
	}
	return true
}

func (d *DPoS) checkOurTurn() bool {
	producer := d.getScheduledProducer(d.slot)
	ret := strings.Compare(d.Name, producer) == 0
	if !ret {
		//d.log.GetLog().Info("checkProducingTiming failed.")
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

func (d *DPoS) PushTransaction(trx common.ISignedTransaction, wait bool, broadcast bool) common.ITransactionInvoice {

	var waitChan chan common.ITransactionInvoice

	if wait {
		waitChan = make(chan common.ITransactionInvoice)
	}

	d.trxCh <- func() {
		ret := d.ctrl.PushTrx(trx.(*prototype.SignedTransaction))

		if wait {
			waitChan <- ret
		}
		if ret.IsSuccess() {
			//	if broadcast {
			d.log.GetLog().Debug("DPoS Broadcast trx.")
			d.p2p.Broadcast(trx.(*prototype.SignedTransaction))
			//	}
		}
	}
	if wait {
		return <-waitChan
	} else {
		return nil
	}
}

func (d *DPoS) pushBlock(b common.ISignedBlock, applyStateDB bool) error {
	d.log.GetLog().Debug("pushBlock #", b.Id().BlockNum())
	//d.Lock()
	//defer d.Unlock()
	// TODO: check signee & merkle

	if b.Timestamp() < d.getSlotTime(1) {
		d.log.GetLog().Debugf("the timestamp of the new block is less than that of the head block.")
	}

	head := d.ForkDB.Head()
	if head == nil && b.Id().BlockNum() != 1 {
		d.log.GetLog().Errorf("[DPoS] the first block pushed should have number of 1, got %d", b.Id().BlockNum())
		return fmt.Errorf("invalid block number")
	}

	newHead := d.ForkDB.PushBlock(b)
	if newHead == head {
		// this implies that b is a:
		// 1. detached block or
		// 2. out of range block or
		// 3. head of a non-main branch or
		// 4. illegal block
		d.log.GetLog().Debugf("[pushBlock]possibly detached block. prev: got %v, want %v", b.Id(), head.Id())
		if b.Id().BlockNum() > head.Id().BlockNum() {
			d.p2p.TriggerSync(head.Id())
		}
		return nil
	} else if head != nil && newHead.Previous() != head.Id() {
		d.log.GetLog().Debug("[DPoS] start to switch fork.")
		d.switchFork(head.Id(), newHead.Id())
		return nil
	}

	if applyStateDB {
		if err := d.applyBlock(b); err != nil {
			// the block is illegal
			d.ForkDB.MarkAsIllegal(b.Id())
			d.ForkDB.Pop()
			return err
		}
	}

	lastCommitted := d.ForkDB.LastCommitted()
	//d.log.GetLog().Debug("last committed: ", lastCommitted.BlockNum())
	var commitIdx uint64
	if newHead.Id().BlockNum()-lastCommitted.BlockNum() > 3 /*constants.MAX_WITNESSES*2/3*/ {
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
	d.log.GetLog().Debug("commit block #", b.Id().BlockNum())

	d.ctrl.Commit(b.Id().BlockNum())

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
	d.log.GetLog().Debug("[DPoS][switchFork] fork branches: ", branches)
	poppedNum := len(branches[0]) - 1
	d.popBlock(branches[0][poppedNum])

	// producers fixup
	d.restoreProducers()

	appendedNum := len(branches[1]) - 1
	errWhileSwitch := false
	var newBranchIdx int
	for newBranchIdx = appendedNum - 1; newBranchIdx >= 0; newBranchIdx-- {
		b, err := d.ForkDB.FetchBlock(branches[1][newBranchIdx])
		if err != nil {
			panic(err)
		}
		if d.applyBlock(b) != nil {
			d.log.GetLog().Errorf("[DPoS][switchFork] applying block %v failed.", b.Id())
			errWhileSwitch = true
			// TODO: peels off this invalid branch to avoid flip-flop switch
			break
		}
	}

	// switch back
	if errWhileSwitch {
		d.log.GetLog().Info("[DPoS][switchFork] switch back to original fork")
		d.popBlock(branches[0][poppedNum])

		// producers fixup
		d.restoreProducers()

		for i := poppedNum - 1; i >= 0; i-- {
			b, err := d.ForkDB.FetchBlock(branches[0][i])
			if err != nil {
				panic(err)
			}
			d.applyBlock(b)
		}

		// restore the good old head of ForkDB
		d.ForkDB.ResetHead(branches[0][0])
	}
}

func (d *DPoS) applyBlock(b common.ISignedBlock) error {
	//d.log.GetLog().Debug("applyBlock #", b.Id().BlockNum())
	err := d.ctrl.PushBlock(b.(*prototype.SignedBlock), prototype.Skip_nothing)
	//d.log.GetLog().Debugf("applyBlock #%d finished.", b.Id().BlockNum())
	return err
}

func (d *DPoS) popBlock(id common.BlockID) error {
	d.ctrl.PopBlockTo(id.BlockNum())
	return nil
}

func (d *DPoS) GetHeadBlockId() common.BlockID {
	if d.ForkDB.Empty() {
		return common.EmptyBlockID
	}
	return d.ForkDB.Head().Id()
}

func (d *DPoS) GetIDs(start, end common.BlockID) ([]common.BlockID, error) {
	blocks, err := d.FetchBlocksSince(start)
	if err != nil {
		return nil, err
	}

	if len(blocks) == 0 {
		return nil, nil
	}

	length := end.BlockNum() - start.BlockNum() + 1
	ret := make([]common.BlockID, 0, length)
	if start != blocks[0].Previous() {
		d.log.GetLog().Debugf("[GetIDs] <from: %v, to: %v> start %v", start, end, blocks[0].Previous())
		return nil, fmt.Errorf("[DPoS GetIDs] internal error")
	}

	ret = append(ret, start)
	for i := 0; i < int(length) && i < len(blocks); i++ {
		ret = append(ret, blocks[i].Id())
	}
	//d.log.GetLog().Debugf("FetchBlocksSince %v: %v", start, ret)
	return ret, nil
}

func (d *DPoS) FetchBlock(id common.BlockID) (common.ISignedBlock, error) {
	if b, err := d.ForkDB.FetchBlock(id); err == nil {
		return b, nil
	}

	var b prototype.SignedBlock
	if err := d.blog.ReadBlock(&b, int64(id.BlockNum())-1); err == nil {
		if b.Id() == id {
			return &b, nil
		}
	}

	return nil, fmt.Errorf("[DPoS FetchBlock] block with id %v doesn't exist", id)
}

func (d *DPoS) HasBlock(id common.BlockID) bool {
	if _, err := d.ForkDB.FetchBlock(id); err == nil {
		return true
	}

	var b prototype.SignedBlock
	if err := d.blog.ReadBlock(&b, int64(id.BlockNum())-1); err == nil {
		if b.Id() == id {
			return true
		}
	}

	return false
}

func (d *DPoS) FetchBlocksSince(id common.BlockID) ([]common.ISignedBlock, error) {
	length := int64(d.ForkDB.Head().Id().BlockNum()) - int64(id.BlockNum())
	if length < 1 {
		return nil, nil
	}

	if id.BlockNum() >= d.ForkDB.LastCommitted().BlockNum() {
		blocks, _, err := d.ForkDB.FetchBlocksSince(id)
		return blocks, err
	}

	ret := make([]common.ISignedBlock, 0, length)
	idNum := id.BlockNum()
	start := idNum + 1

	end := uint64(d.blog.Size())
	for start <= end {
		var b prototype.SignedBlock
		if err := d.blog.ReadBlock(&b, int64(start-1)); err != nil {
			return nil, err
		}

		if start == idNum && b.Id() != id {
			return nil, fmt.Errorf("blockchain doesn't have block with id %v", id)
		}

		ret = append(ret, &b)
		start++

		if start > end && b.Id() != d.ForkDB.LastCommitted() {
			panic("ForkDB and BLog inconsistent state")
		}
	}

	blocksInForkDB, _, err := d.ForkDB.FetchBlocksSince(d.ForkDB.LastCommitted())
	if err != nil {
		return nil, err
	}
	ret = append(ret, blocksInForkDB...)
	return ret, nil
}
