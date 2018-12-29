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
	//"github.com/coschain/gobft"
	"github.com/coschain/gobft/message"
	"github.com/coschain/gobft/custom"
)

// SABFT: self-adaptive BFT
// It generates blocks in the same manner of DPoS and adopts bft
// to achieve fast block confirmation. It's self adaptive in a way
// that it can adjust the frequency of bft process based on the
// load of the network.
type SABFT struct {
	iservices.IConsensus
	node   *node.Node
	ForkDB *forkdb.DB
	blog   blocklog.BLog

	Producers []string
	Name      string

	privKey        *prototype.PrivateKeyType
	readyToProduce bool
	prodTimer      *time.Timer
	trxCh          chan func()
	blkCh          chan common.ISignedBlock
	bootstrap      bool
	slot           uint64

	lastCommitted common.BlockID
	suffledID     common.BlockID

	ctx  *node.ServiceContext
	ctrl iservices.ITrxPool
	p2p  iservices.IP2P
	log  iservices.ILog

	stopCh chan struct{}
	wg     sync.WaitGroup
	sync.RWMutex
}

func NewSABFT(ctx *node.ServiceContext) *SABFT {
	logService, err := ctx.Service(iservices.LogServerName)
	if err != nil {
		panic(err)
	}
	ret := &SABFT{
		ForkDB:    forkdb.NewDB(),
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
	ret.log.GetLog().Info("[SABFT bootstrap] ", ctx.Config().Consensus.BootStrap)

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

func (d *SABFT) getController() iservices.ITrxPool {
	ctrl, err := d.ctx.Service(iservices.TxPoolServerName)
	if err != nil {
		panic(err)
	}
	return ctrl.(iservices.ITrxPool)
}

func (d *SABFT) SetBootstrap(b bool) {
	d.bootstrap = b
	if d.bootstrap {
		d.readyToProduce = true
	}
}

func (d *SABFT) CurrentProducer() string {
	//d.RLock()
	//defer d.RUnlock()
	now := time.Now().Round(time.Second)
	slot := d.getSlotAtTime(now)
	return d.getScheduledProducer(slot)
}

func (d *SABFT) shuffle(head common.ISignedBlock) {
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
	d.log.GetLog().Debug("[SABFT shuffle] active producers: ", d.Producers)
	d.ctrl.SetShuffledWitness(prods)

	d.suffledID = head.Id()
}

func (d *SABFT) restoreProducers() {
	d.Producers = d.ctrl.GetShuffledWitness()
}

func (d *SABFT) ActiveProducers() []string {
	//d.RLock()
	//defer d.RUnlock()
	return d.Producers
}

func (d *SABFT) Start(node *node.Node) error {
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
	// TODO: start bft
	return nil
}

func (d *SABFT) scheduleProduce() bool {
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
			d.log.GetLog().Debug("[SABFT TriggerSync]: start from ", headID.BlockNum())
			return false
		}
	}
	if !d.checkProducingTiming() || !d.checkOurTurn() {
		d.prodTimer.Reset(timeToNextSec())
		return false
	}
	return true
}

func (d *SABFT) start(snapshotPath string) {
	d.wg.Add(1)
	defer d.wg.Done()
	time.Sleep(4 * time.Second)
	d.log.GetLog().Info("[SABFT] starting...")

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
		d.log.GetLog().Info("[SABFT] bootstrapping...")
	}
	d.restoreProducers()

	d.log.GetLog().Info("[SABFT] started")
	for {
		select {
		case <-d.stopCh:
			d.log.GetLog().Debug("[SABFT] routine stopped.")
			return
		case b := <-d.blkCh:
			if err := d.pushBlock(b, true); err != nil {
				d.log.GetLog().Error("[SABFT] pushBlock failed: ", err)
			}
		case trxFn := <-d.trxCh:
			trxFn()
			continue
		case <-d.prodTimer.C:
			if !d.scheduleProduce() {
				continue
			}

			b, err := d.generateAndApplyBlock()
			if err != nil {
				d.log.GetLog().Error("[SABFT] generateAndApplyBlock error: ", err)
				continue
			}
			d.prodTimer.Reset(timeToNextSec())
			d.log.GetLog().Debugf("[SABFT] generated block: <num %d> <ts %d>", b.Id().BlockNum(), b.Timestamp())
			if err := d.pushBlock(b, false); err != nil {
				d.log.GetLog().Error("[SABFT] pushBlock push generated block failed: ", err)
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

func (d *SABFT) Stop() error {
	// TODO: stop bft
	d.log.GetLog().Info("SABFT consensus stopped.")
	// restore uncommitted forkdb
	cfg := d.ctx.Config()
	snapshotPath := cfg.ResolvePath("forkdb_snapshot")
	return d.stop(snapshotPath)
}

func (d *SABFT) stop(snapshotPath string) error {
	d.ForkDB.Snapshot(snapshotPath)
	d.prodTimer.Stop()
	close(d.stopCh)
	d.wg.Wait()
	return nil
}

func (d *SABFT) generateAndApplyBlock() (common.ISignedBlock, error) {
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

func (d *SABFT) checkGenesis() bool {
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
func (d *SABFT) checkProducingTiming() bool {
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

func (d *SABFT) checkOurTurn() bool {
	producer := d.getScheduledProducer(d.slot)
	ret := strings.Compare(d.Name, producer) == 0
	if !ret {
		//d.log.GetLog().Info("checkProducingTiming failed.")
	}
	return ret
}

func (d *SABFT) getScheduledProducer(slot uint64) string {
	if d.ForkDB.Empty() {
		return d.Producers[0]
	}
	absSlot := (d.ForkDB.Head().Timestamp() - constants.GenesisTime) / constants.BLOCK_INTERVAL
	return d.Producers[(absSlot+slot)%uint64(len(d.Producers))]
}

// returns false if we're out of sync
func (d *SABFT) checkSync() bool {
	now := time.Now().Round(time.Second).Unix()
	if d.getSlotTime(1) < uint64(now) {
		//time.Sleep(time.Second)
		return false
	}
	return true
}

func (d *SABFT) getSlotTime(slot uint64) uint64 {
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

func (d *SABFT) getSlotAtTime(t time.Time) uint64 {
	nextSlotTime := d.getSlotTime(1)
	if uint64(t.Unix()) < nextSlotTime {
		return 0
	}
	return (uint64(t.Unix())-nextSlotTime)/constants.BLOCK_INTERVAL + 1
}

func (d *SABFT) PushBlock(b common.ISignedBlock) {
	go func(blk common.ISignedBlock) {
		d.blkCh <- b
	}(b)
}

func (d *SABFT) PushTransaction(trx common.ISignedTransaction, wait bool, broadcast bool) common.ITransactionReceiptWithInfo {

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
			d.log.GetLog().Debug("SABFT Broadcast trx.")
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

func (d *SABFT) pushBlock(b common.ISignedBlock, applyStateDB bool) error {
	d.log.GetLog().Debug("pushBlock #", b.Id().BlockNum())
	//d.Lock()
	//defer d.Unlock()
	// TODO: check signee & merkle

	if b.Timestamp() < d.getSlotTime(1) {
		d.log.GetLog().Debugf("the timestamp of the new block is less than that of the head block.")
	}

	head := d.ForkDB.Head()
	if head == nil && b.Id().BlockNum() != 1 {
		d.log.GetLog().Errorf("[SABFT] the first block pushed should have number of 1, got %d", b.Id().BlockNum())
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
		d.log.GetLog().Debug("[SABFT] start to switch fork.")
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
	return nil
}

/********* implements gobft ICommittee ***********/

// Commit sets b as the last irreversible block
func (d *SABFT) Commit(b message.ProposedData) error {
	d.log.GetLog().Debug("[SABFT] commit block #", b)
	return nil
}

// GetValidator returns the validator correspond to the PubKey
func (d *SABFT) GetValidator(key message.PubKey) custom.IPubValidator {
	return nil
}

// IsValidator returns true if key is a validator
func (d *SABFT) IsValidator(key message.PubKey) bool {
	return true
}

func (d *SABFT) TotalVotingPower() int64 {
	return 0
}

func (d *SABFT) GetCurrentProposer(round int) message.PubKey {
	return ""
}

// DecidesProposal decides what will be proposed if this validator is the current proposer.
func (d *SABFT) DecidesProposal() message.ProposedData {
	return message.NilData
}

// ValidateProposed validates the proposed data
func (d *SABFT) ValidateProposed(data message.ProposedData) bool {
	return true
}

func (d *SABFT) GetAppState() *message.AppState {
	return nil
}

// BroadCast sends msg to other validators
func (d *SABFT) BroadCast(msg message.ConsensusMessage) error {
	return nil
}

/********* end gobft ICommittee ***********/

func (d *SABFT) switchFork(old, new common.BlockID) {
	// TODO: what about bft process???
	branches, err := d.ForkDB.FetchBranch(old, new)
	if err != nil {
		panic(err)
	}
	d.log.GetLog().Debug("[SABFT][switchFork] fork branches: ", branches)
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
			d.log.GetLog().Errorf("[SABFT][switchFork] applying block %v failed.", b.Id())
			errWhileSwitch = true
			// TODO: peels off this invalid branch to avoid flip-flop switch
			break
		}
	}

	// switch back
	if errWhileSwitch {
		d.log.GetLog().Info("[SABFT][switchFork] switch back to original fork")
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

func (d *SABFT) applyBlock(b common.ISignedBlock) error {
	//d.log.GetLog().Debug("applyBlock #", b.Id().BlockNum())
	err := d.ctrl.PushBlock(b.(*prototype.SignedBlock), prototype.Skip_nothing)
	//d.log.GetLog().Debugf("applyBlock #%d finished.", b.Id().BlockNum())
	return err
}

func (d *SABFT) popBlock(id common.BlockID) error {
	d.ctrl.PopBlockTo(id.BlockNum())
	return nil
}

func (d *SABFT) GetHeadBlockId() common.BlockID {
	if d.ForkDB.Empty() {
		return common.EmptyBlockID
	}
	return d.ForkDB.Head().Id()
}

func (d *SABFT) GetIDs(start, end common.BlockID) ([]common.BlockID, error) {
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
		return nil, fmt.Errorf("[SABFT GetIDs] internal error")
	}

	ret = append(ret, start)
	for i := 0; i < int(length) && i < len(blocks); i++ {
		ret = append(ret, blocks[i].Id())
	}
	//d.log.GetLog().Debugf("FetchBlocksSince %v: %v", start, ret)
	return ret, nil
}

func (d *SABFT) FetchBlock(id common.BlockID) (common.ISignedBlock, error) {
	if b, err := d.ForkDB.FetchBlock(id); err == nil {
		return b, nil
	}

	var b prototype.SignedBlock
	if err := d.blog.ReadBlock(&b, int64(id.BlockNum())-1); err == nil {
		if b.Id() == id {
			return &b, nil
		}
	}

	return nil, fmt.Errorf("[SABFT FetchBlock] block with id %v doesn't exist", id)
}

func (d *SABFT) HasBlock(id common.BlockID) bool {
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

func (d *SABFT) FetchBlocksSince(id common.BlockID) ([]common.ISignedBlock, error) {
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
