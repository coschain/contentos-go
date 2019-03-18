package consensus

import (
	"errors"
	"fmt"
	"io/ioutil"
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
	"github.com/sirupsen/logrus"
	//"github.com/coschain/contentos-go/app"
)

func (d *DPoS) timeToNextSec() time.Duration {
	now := d.Ticker.Now()
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
	pendingCh      chan func()
	blkCh          chan common.ISignedBlock
	bootstrap      bool
	slot           uint64

	ctx  *node.ServiceContext
	ctrl iservices.ITrxPool
	p2p  iservices.IP2P
	log  *logrus.Logger

	Ticker TimerDriver

	stopCh chan struct{}
	wg     sync.WaitGroup
	sync.RWMutex
}

func NewDPoS(ctx *node.ServiceContext, lg *logrus.Logger) *DPoS {
	if lg == nil {
		lg = logrus.New()
		lg.SetOutput(ioutil.Discard)
	}
	ret := &DPoS{
		ForkDB:    forkdb.NewDB(),
		Producers: make([]string, 0, 1),
		prodTimer: time.NewTimer(100 * time.Millisecond),
		trxCh:     make(chan func()),
		pendingCh: make(chan func()),
		//trxRetCh:  make(chan common.ITransactionInvoice),
		blkCh:  make(chan common.ISignedBlock),
		ctx:    ctx,
		stopCh: make(chan struct{}),
		log:    lg,
		Ticker: &Timer{},
	}
	ret.SetBootstrap(ctx.Config().Consensus.BootStrap)
	ret.Name = ctx.Config().Consensus.LocalBpName
	ret.log.Info("[DPoS bootstrap] ", ctx.Config().Consensus.BootStrap)

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
	ctrl, err := d.ctx.Service(iservices.TxPoolServerName)
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
	now := d.Ticker.Now().Round(time.Second)
	slot := d.getSlotAtTime(now)
	return d.getScheduledProducer(slot)
}

func (d *DPoS) shuffle(head common.ISignedBlock) {
	if head.Id().BlockNum()%uint64(len(d.Producers)) != 0 {
		return
	}

	// When a produce round complete, it adds new producers,
	// remove unqualified producers and shuffle the block-producing order
	prods := d.ctrl.GetWitnessTopN(constants.MaxWitnessCount)
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
	d.log.Debug("[DPoS shuffle] active producers: ", d.Producers)
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
	d.ctrl.SetShuffle(func(block common.ISignedBlock) {
		d.shuffle(block)
	})

	d.log.Info("[DPoS] starting...")

	// TODO: fuck!! this is fugly
	var avatar []common.ISignedBlock
	for i := 0; i < constants.MaxWitnessCount+1; i++ {
		// deep copy hell
		avatar = append(avatar, &prototype.SignedBlock{})
	}
	d.ForkDB.LoadSnapshot(avatar, cfg.ResolvePath("forkdb_snapshot"), &d.blog)

	if d.bootstrap && d.ForkDB.Empty() && d.blog.Empty() {
		d.log.Info("[DPoS] bootstrapping...")
	}

	d.restoreProducers()

	////sync blocks to squash db
	err = d.handleBlockSync()
    if err != nil {
    	return err
	}

	go d.start()
	return nil
}

func (d *DPoS) scheduleProduce() bool {
	if !d.checkGenesis() {
		//d.log.Info("checkGenesis failed.")
		if _, ok := d.Ticker.(*Timer); ok {
			d.prodTimer.Reset(d.timeToNextSec())
		}
		//d.prodTimer.Reset(timeToNextSec())
		return false
	}

	if !d.readyToProduce {
		if d.checkSync() {
			d.readyToProduce = true
		} else {
			if _, ok := d.Ticker.(*Timer); ok {
				d.prodTimer.Reset(d.timeToNextSec())
			}
			//d.prodTimer.Reset(timeToNextSec())
			var headID common.BlockID
			if !d.ForkDB.Empty() {
				headID = d.ForkDB.Head().Id()
			}
			d.p2p.TriggerSync(headID)
			// TODO:  if we are not on the main branch, pop until the head is on main branch
			d.log.Debug("[DPoS TriggerSync]: start from ", headID.BlockNum())
			return false
		}
	}
	if !d.checkProducingTiming() || !d.checkOurTurn() {
		if _, ok := d.Ticker.(*Timer); ok {
			d.prodTimer.Reset(d.timeToNextSec())
		}
		//d.prodTimer.Reset(timeToNextSec())
		return false
	}
	return true
}

func (d *DPoS) testStart(path string) {
	// TODO: fuck!! this is fugly
	var avatar []common.ISignedBlock
	for i := 0; i < constants.MaxWitnessCount+1; i++ {
		// deep copy hell
		avatar = append(avatar, &prototype.SignedBlock{})
	}
	d.ForkDB.LoadSnapshot(avatar, path, &d.blog)

	if d.bootstrap && d.ForkDB.Empty() && d.blog.Empty() {
		d.log.Info("[DPoS] bootstrapping...")
	}
	d.restoreProducers()
	d.start()
}

func (d *DPoS) start() {
	d.wg.Add(1)
	defer d.wg.Done()

	d.log.Info("[DPoS] started")
	for {
		select {
		case <-d.stopCh:
			d.log.Debug("[DPoS] routine stopped.")
			return
		case b := <-d.blkCh:
			if err := d.pushBlock(b, true); err != nil {
				d.log.Error("[DPoS] pushBlock failed: ", err)
			}
		case trxFn := <-d.trxCh:
			trxFn()
			continue
		case pendingFn := <- d.pendingCh:
			pendingFn()
		    continue
		case <-d.prodTimer.C:
			d.MaybeProduceBlock()
		}
	}
}

func (d *DPoS) Stop() error {
	d.log.Info("DPoS consensus stopped.")
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
	//d.log.Debug("generateBlock.")
	ts := d.getSlotTime(d.slot)
	prev := &prototype.Sha256{}
	if !d.ForkDB.Empty() {
		prev.FromBlockID(d.ForkDB.Head().Id())
	} else {
		prev.Hash = make([]byte, 32)
	}
	//d.log.Debugf("generating block. <prev %v>, <ts %d>", prev.Hash, ts)
	return d.ctrl.GenerateAndApplyBlock(d.Name, prev, uint32(ts), d.privKey, prototype.Skip_nothing)
}

func (d *DPoS) checkGenesis() bool {
	now := d.Ticker.Now()
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
	now := d.Ticker.Now().Round(time.Second)
	d.slot = d.getSlotAtTime(now)
	if d.slot == 0 {
		// not time yet, wait till the next block producing
		// cycle comes
		//nextSlotTime := d.getSlotTime(1)
		//time.Sleep(time.Unix(int64(nextSlotTime), 0).Sub(time.Now()))
		//d.log.Info("checkProducingTiming failed.")
		return false
	}
	return true
}

func (d *DPoS) checkOurTurn() bool {
	producer := d.getScheduledProducer(d.slot)
	ret := strings.Compare(d.Name, producer) == 0
	if !ret {
		//d.log.Info("checkProducingTiming failed.")
	}
	return ret
}

func (d *DPoS) getScheduledProducer(slot uint64) string {
	if d.ForkDB.Empty() {
		return d.Producers[0]
	}
	absSlot := (d.ForkDB.Head().Timestamp() - constants.GenesisTime) / constants.BlockInterval
	return d.Producers[(absSlot+slot)%uint64(len(d.Producers))]
}

// returns false if we're out of sync
func (d *DPoS) checkSync() bool {
	now := d.Ticker.Now().Round(time.Second).Unix()
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
		return constants.GenesisTime + slot*constants.BlockInterval
	}

	headSlotTime := head.Timestamp() / constants.BlockInterval * constants.BlockInterval
	return headSlotTime + slot*constants.BlockInterval
}

func (d *DPoS) getSlotAtTime(t time.Time) uint64 {
	nextSlotTime := d.getSlotTime(1)
	if uint64(t.Unix()) < nextSlotTime {
		return 0
	}
	return (uint64(t.Unix())-nextSlotTime)/constants.BlockInterval + 1
}

func (d *DPoS) Push(msg interface{}) {}

func (d *DPoS) PushBlock(b common.ISignedBlock) {
	go func(blk common.ISignedBlock) {
		d.blkCh <- b
	}(b)
}

func (d *DPoS) PushTransaction(trx common.ISignedTransaction, wait bool, broadcast bool) common.ITransactionReceiptWithInfo {

	var waitChan chan common.ITransactionReceiptWithInfo

	if wait {
		waitChan = make(chan common.ITransactionReceiptWithInfo)
	}

	d.trxCh <- func() {
		ret := d.ctrl.PushTrx(trx.(*prototype.SignedTransaction))

		if wait {
			waitChan <- ret
		}
		if ret.IsSuccess() {
			//	if broadcast {
			d.log.Debug("DPoS Broadcast trx.")
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

func (d *DPoS) PushTransactionToPending(trx common.ISignedTransaction, callBack func(err error)) {
	d.pendingCh <- func(){
		err := d.ctrl.PushTrxToPending(trx.(*prototype.SignedTransaction))
		if err == nil {
			d.p2p.Broadcast(trx.(*prototype.SignedTransaction))
		}
		callBack(err)
	}
}

func (d *DPoS) pushBlock(b common.ISignedBlock, applyStateDB bool) error {
	d.log.Debug("pushBlock #", b.Id().BlockNum())
	//d.Lock()
	//defer d.Unlock()
	// TODO: check signee & merkle

	if b.Timestamp() < d.getSlotTime(1) {
		d.log.Debugf("the timestamp of the new block is less than that of the head block.")
	}

	head := d.ForkDB.Head()
	if head == nil && b.Id().BlockNum() != 1 {
		d.log.Errorf("[DPoS] the first block pushed should have number of 1, got %d", b.Id().BlockNum())
		return fmt.Errorf("invalid block number")
	}

	rc := d.ForkDB.PushBlock(b)
	newHead := d.ForkDB.Head()
	switch rc {
	case forkdb.RTDetached:
		d.log.Debugf("[DPoS][pushBlock]possibly detached block. prev: got %v, want %v", b.Id(), head.Id())
	case forkdb.RTOutOfRange:
	case forkdb.RTOnFork:
		if newHead.Previous() != head.Id() {
			d.log.Debug("[DPoS] start to switch fork.")
			d.switchFork(head.Id(), newHead.Id())
			return nil
		}
	case forkdb.RTInvalid:
		return ErrInvalidBlock
	case forkdb.RTDuplicated:
		return ErrDupBlock
	case forkdb.RTSuccess:
	default:
		return ErrInternal
	}
	/*
	newHead := d.ForkDB.PushBlock(b)
	if newHead == head {
		// this implies that b is a:
		// 1. detached block or
		// 2. out of range block or
		// 3. head of a non-main branch or
		// 4. illegal block
		d.log.Debugf("[pushBlock]possibly detached block. prev: got %v, want %v", b.Id(), head.Id())
		if b.Id().BlockNum() > head.Id().BlockNum() {
			d.p2p.FetchUnlinkedBlock(b.Previous())
		}
		return nil
	} else if head != nil && newHead.Previous() != head.Id() {
		d.log.Debug("[DPoS] start to switch fork.")
		d.switchFork(head.Id(), newHead.Id())
		return nil
	}
	*/

	if applyStateDB {
		if err := d.applyBlock(b); err != nil {
			// the block is illegal
			d.ForkDB.MarkAsIllegal(b.Id())
			d.ForkDB.Pop()
			return err
		}
	}

	lastCommitted := d.ForkDB.LastCommitted()
	//d.log.Debug("last committed: ", lastCommitted.BlockNum())
	var commitIdx uint64
	if newHead.Id().BlockNum()-lastCommitted.BlockNum() > constants.MaxWitnessCount*2/3 {
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
	d.log.Debug("commit block #", b.Id().BlockNum())
	err := d.blog.Append(b)

	if err != nil {
		// something went really wrong if we got here
		panic(err)
	}

	d.ctrl.Commit(b.Id().BlockNum())
	d.ForkDB.Commit(b.Id())
	return nil
}

func (d *DPoS) switchFork(old, new common.BlockID) {
	branches, err := d.ForkDB.FetchBranch(old, new)
	if err != nil {
		panic(err)
	}
	d.log.Debug("[DPoS][switchFork] fork branches: ", branches)
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
			d.log.Errorf("[DPoS][switchFork] applying block %v failed.", b.Id())
			errWhileSwitch = true
			// TODO: peels off this invalid branch to avoid flip-flop switch
			break
		}
	}

	// switch back
	if errWhileSwitch {
		d.log.Info("[DPoS][switchFork] switch back to original fork")
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
	//d.log.Debug("applyBlock #", b.Id().BlockNum())
	err := d.ctrl.PushBlock(b.(*prototype.SignedBlock), prototype.Skip_nothing)
	//d.log.Debugf("applyBlock #%d finished.", b.Id().BlockNum())
	return err
}

func (d *DPoS) popBlock(id common.BlockID) error {
	d.ctrl.PopBlock(id.BlockNum())
	return nil
}

func (d *DPoS) GetLastBFTCommit() (evidence interface{}) {
	return nil
}

func (d *DPoS) GetNextBFTCheckPoint(blockNum uint64) (evidence interface{}) {
	return nil
}

func (d *DPoS) GetLIB() common.BlockID {
	return d.ForkDB.LastCommitted()
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
		d.log.Debugf("[GetIDs] <from: %v, to: %v> start %v", start, end, blocks[0].Previous())
		return nil, fmt.Errorf("[DPoS GetIDs] internal error")
	}

	ret = append(ret, start)
	for i := 0; i < int(length) && i < len(blocks); i++ {
		ret = append(ret, blocks[i].Id())
	}
	//d.log.Debugf("FetchBlocksSince %v: %v", start, ret)
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

		if start == idNum+1 && b.Previous() != id {
			return nil, fmt.Errorf("blockchain doesn't have block with id %v", id)
		}

		ret = append(ret, &b)
		start++

		if start > end && b.Id() != d.ForkDB.LastCommitted() {
			// there probably is a new committed block during the execution of this process
			return nil, errors.New("ForkDB and BLog inconsistent state")
		}
	}

	blocksInForkDB, _, err := d.ForkDB.FetchBlocksSince(d.ForkDB.LastCommitted())
	if err != nil {
		return nil, err
	}
	ret = append(ret, blocksInForkDB...)
	return ret, nil
}

func (d *DPoS) FetchBlocks(from, to uint64) ([]common.ISignedBlock, error) {
	return fetchBlocks(from, to, d.ForkDB, &d.blog)
}


func (d *DPoS) ResetProdTimer(t time.Duration) {
	if !d.prodTimer.Stop() {
		<-d.prodTimer.C
	}
	d.prodTimer.Reset(t)
}

func (d *DPoS) ResetTicker(ts time.Time) {
	d.Ticker = &FakeTimer {t : ts}
}

func (d *DPoS) MaybeProduceBlock() {
	if !d.scheduleProduce() {
		return
	}

	b, err := d.generateAndApplyBlock()
	if err != nil {
		d.log.Error("[DPoS] generateAndApplyBlock error: ", err)
		return
	}
	if _, ok := d.Ticker.(*Timer); ok {
		d.prodTimer.Reset(d.timeToNextSec())
	}
	//d.prodTimer.Reset(timeToNextSec())
	d.log.Debugf("[DPoS] generated block: <num %d> <ts %d>", b.Id().BlockNum(), b.Timestamp())
	if err := d.pushBlock(b, false); err != nil {
		d.log.Error("[DPoS] pushBlock push generated block failed: ", err)
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

func (d *DPoS) handleBlockSync() error {
	if d.ForkDB.Head() == nil {
		//Not need to sync
		return nil
	}
	var err error = nil
	lastCommit := d.ForkDB.LastCommitted().BlockNum()
	//Fetch the commit block num in db
	dbCommit,err := d.ctrl.GetCommitBlockNum()
	if err != nil {
		return err
	}
	//Fetch the commit block numbers saved in block log
	commitNum := d.blog.Size()
	//1.sync commit blocks
	if dbCommit < lastCommit && commitNum > 0  && commitNum >= int64(lastCommit) {
		d.log.Debugf("[Reload commit] start sync lost commit blocks from block log,db commit num is: " +
							"%v,end:%v,real commit num is %v", dbCommit, d.ForkDB.Head().Id().BlockNum(), lastCommit)
		for i := int64(dbCommit); i < int64(lastCommit); i++ {
			blk := &prototype.SignedBlock{}
			if err := d.blog.ReadBlock(blk, i); err != nil {
				return err
			}
			err = d.ctrl.SyncCommittedBlockToDB(blk)
			if err != nil {
				return err
			}
		}
	}
	//2.sync pushed blocks
	//Fetch pushed blocks in snapshot
	pSli, _, err := d.ForkDB.FetchBlocksSince(d.ForkDB.LastCommitted())
	if err != nil {
		return err
	}
	if len(pSli) > 0 {
		d.log.Debugf("[sync pushed]: start sync lost blocks,start: %v,end:%v",
			lastCommit+1, d.ForkDB.Head().Id().BlockNum())
		err = d.ctrl.SyncPushedBlocksToDB(pSli)
	}
	
	return err
}
