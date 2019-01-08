package consensus

import (
	"fmt"
	"github.com/coschain/gobft"
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
	"github.com/coschain/gobft/custom"
	"github.com/coschain/gobft/message"
)

/********* implements gobft IPubValidator ***********/

type publicValidator struct {
	sab         *SABFT
	accountName string
}

func (pv *publicValidator) VerifySig(digest, signature []byte) bool {
	// Warning: DO NOT remove the lock unless you know what you're doing
	pv.sab.RLock()
	defer pv.sab.RUnlock()

	acc := &prototype.AccountName{
		Value: pv.accountName,
	}
	return pv.sab.ctrl.VerifySig(acc, digest, signature)
}

func (pv *publicValidator) GetPubKey() message.PubKey {
	return message.PubKey(pv.accountName)
}

func (pv *publicValidator) GetVotingPower() int64 {
	return 1
}

func (pv *publicValidator) SetVotingPower(int64) {

}

/********* end gobft IPubValidator ***********/

/********* implements gobft IPrivValidator ***********/

type privateValidator struct {
	sab     *SABFT
	privKey *prototype.PrivateKeyType
	name    string
}

func (pv *privateValidator) Sign(digest []byte) []byte {
	// Warning: DO NOT remove the lock unless you know what you're doing
	pv.sab.RLock()
	defer pv.sab.RUnlock()

	return pv.sab.ctrl.Sign(pv.privKey, digest)
}

func (pv *privateValidator) GetPubKey() message.PubKey {
	return message.PubKey(pv.name)
}

/********* end gobft IPrivValidator ***********/

// SABFT: self-adaptive BFT
// It generates blocks in the same manner of DPoS and adopts bft
// to achieve fast block confirmation. It's self adaptive in a way
// that it can adjust the frequency of bft process based on the
// load of the network.
type SABFT struct {
	node   *node.Node
	ForkDB *forkdb.DB
	blog   blocklog.BLog

	Name string

	validators    []*publicValidator
	priv          *privateValidator
	bft           *gobft.Core
	lastCommitted *message.Commit
	suffledID     common.BlockID
	appState      *message.AppState

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

func NewSABFT(ctx *node.ServiceContext) *SABFT {
	logService, err := ctx.Service(iservices.LogServerName)
	if err != nil {
		panic(err)
	}
	ret := &SABFT{
		ForkDB:     forkdb.NewDB(),
		validators: make([]*publicValidator, 0, 1),
		prodTimer:  time.NewTimer(1 * time.Millisecond),
		trxCh:      make(chan func()),
		blkCh:      make(chan common.ISignedBlock),
		ctx:        ctx,
		stopCh:     make(chan struct{}),
		log:        logService.(iservices.ILog),
	}

	ret.bft = gobft.NewCore(ret, ret.priv)
	ret.SetBootstrap(ctx.Config().Consensus.BootStrap)
	ret.Name = ctx.Config().Consensus.LocalBpName
	ret.log.GetLog().Info("[SABFT bootstrap] ", ctx.Config().Consensus.BootStrap)
	ret.appState = &message.AppState{
		LastHeight:       0,
		LastProposedData: message.NilData,
	}

	ret.priv = &privateValidator{
		sab:  ret,
		name: ret.Name,
	}
	privateKey := ctx.Config().Consensus.LocalBpPrivateKey
	if len(privateKey) > 0 {
		var err error
		ret.priv.privKey, err = prototype.PrivateKeyFromWIF(privateKey)
		if err != nil {
			panic(err)
		}
	}
	return ret
}

func (sabft *SABFT) getController() iservices.ITrxPool {
	ctrl, err := sabft.ctx.Service(iservices.TxPoolServerName)
	if err != nil {
		panic(err)
	}
	return ctrl.(iservices.ITrxPool)
}

func (sabft *SABFT) SetBootstrap(b bool) {
	sabft.bootstrap = b
	if sabft.bootstrap {
		sabft.readyToProduce = true
	}
}

func (sabft *SABFT) CurrentProducer() string {
	sabft.RLock()
	defer sabft.RUnlock()

	now := time.Now().Round(time.Second)
	slot := sabft.getSlotAtTime(now)
	return sabft.getScheduledProducer(slot)
}

func (sabft *SABFT) makeValidators(names []string) []*publicValidator {
	ret := make([]*publicValidator, len(names))
	for i := range ret {
		ret[i] = &publicValidator{
			sab:         sabft,
			accountName: names[i],
		}
	}
	return ret
}

func (sabft *SABFT) shuffle(head common.ISignedBlock) {
	if !sabft.ForkDB.Empty() && sabft.ForkDB.Head().Id().BlockNum()%uint64(len(sabft.validators)) != 0 {
		return
	}

	// When a produce round complete, it adds new producers,
	// remove unqualified producers and shuffle the block-producing order
	prods := sabft.ctrl.GetWitnessTopN(constants.MAX_WITNESSES)
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

	sabft.validators = sabft.makeValidators(prods)
	sabft.log.GetLog().Debug("[SABFT shuffle] active producers: ", sabft.validators)
	sabft.ctrl.SetShuffledWitness(prods)

	sabft.suffledID = head.Id()
}

func (sabft *SABFT) restoreProducers() {
	sabft.validators = sabft.makeValidators(sabft.ctrl.GetShuffledWitness())
}

func (sabft *SABFT) ActiveProducers() []string {
	sabft.RLock()
	defer sabft.RUnlock()

	ret := make([]string, 0, constants.MAX_WITNESSES)
	for i := range sabft.validators {
		ret = append(ret, sabft.validators[i].accountName)
	}
	return ret
}

func (sabft *SABFT) Start(node *node.Node) error {
	sabft.ctrl = sabft.getController()
	p2p, err := sabft.ctx.Service(iservices.P2PServerName)
	if err != nil {
		panic(err)
	}
	sabft.p2p = p2p.(iservices.IP2P)
	cfg := sabft.ctx.Config()
	sabft.blog.Open(cfg.ResolvePath("blog"))
	sabft.ctrl.SetShuffle(func(block common.ISignedBlock) {
		sabft.shuffle(block)
	})

	// reload ForkDB
	snapshotPath := cfg.ResolvePath("forkdb_snapshot")
	// TODO: fuck!! this is fugly
	var avatar []common.ISignedBlock
	for i := 0; i < constants.MAX_WITNESSES; i++ {
		// deep copy hell
		avatar = append(avatar, &prototype.SignedBlock{})
	}
	sabft.ForkDB.LoadSnapshot(avatar, snapshotPath)

	sabft.log.GetLog().Info("[SABFT] starting...")
	if sabft.bootstrap && sabft.ForkDB.Empty() && sabft.blog.Empty() {
		sabft.log.GetLog().Info("[SABFT] bootstrapping...")
	}
	sabft.restoreProducers()

	// start block generation process
	go sabft.start()

	// start the bft process
	sabft.bft.Start()
	return nil
}

func (sabft *SABFT) scheduleProduce() bool {
	if !sabft.checkGenesis() {
		//sabft.log.GetLog().Info("checkGenesis failed.")
		sabft.prodTimer.Reset(timeToNextSec())
		return false
	}

	if !sabft.readyToProduce {
		if sabft.checkSync() {
			sabft.readyToProduce = true
		} else {
			sabft.prodTimer.Reset(timeToNextSec())
			var headID common.BlockID
			if !sabft.ForkDB.Empty() {
				headID = sabft.ForkDB.Head().Id()
			}
			sabft.p2p.TriggerSync(headID)
			// TODO:  if we are not on the main branch, pop until the head is on main branch
			sabft.log.GetLog().Debug("[SABFT TriggerSync]: start from ", headID.BlockNum())
			return false
		}
	}
	if !sabft.checkProducingTiming() || !sabft.checkOurTurn() {
		sabft.prodTimer.Reset(timeToNextSec())
		return false
	}
	return true
}

func (sabft *SABFT) start() {
	sabft.wg.Add(1)
	defer sabft.wg.Done()

	sabft.log.GetLog().Info("[SABFT] DPoS routine started")
	for {
		select {
		case <-sabft.stopCh:
			sabft.log.GetLog().Debug("[SABFT] routine stopped.")
			return
		case b := <-sabft.blkCh:
			sabft.Lock()
			if err := sabft.pushBlock(b, true); err != nil {
				sabft.log.GetLog().Error("[SABFT] pushBlock failed: ", err)
			}
			sabft.Unlock()
		case trxFn := <-sabft.trxCh:
			sabft.Lock()
			trxFn()
			sabft.Unlock()
			continue
		case <-sabft.prodTimer.C:
			sabft.RLock()
			if !sabft.scheduleProduce() {
				sabft.RUnlock()
				continue
			}
			sabft.RUnlock()

			sabft.Lock()
			b, err := sabft.generateAndApplyBlock()

			if err != nil {
				sabft.log.GetLog().Error("[SABFT] generateAndApplyBlock error: ", err)
				continue
			}
			sabft.prodTimer.Reset(timeToNextSec())
			sabft.log.GetLog().Debugf("[SABFT] generated block: <num %d> <ts %d>", b.Id().BlockNum(), b.Timestamp())
			if err := sabft.pushBlock(b, false); err != nil {
				sabft.log.GetLog().Error("[SABFT] pushBlock push generated block failed: ", err)
			}
			sabft.Unlock()

			sabft.p2p.Broadcast(b)
		}
	}
}

func (sabft *SABFT) Stop() error {
	sabft.log.GetLog().Info("SABFT consensus stopped.")

	// stop bft process
	sabft.bft.Stop()

	// restore uncommitted forkdb
	cfg := sabft.ctx.Config()
	snapshotPath := cfg.ResolvePath("forkdb_snapshot")
	sabft.ForkDB.Snapshot(snapshotPath)
	sabft.prodTimer.Stop()
	close(sabft.stopCh)
	sabft.wg.Wait()
	return nil
}

func (sabft *SABFT) generateAndApplyBlock() (common.ISignedBlock, error) {
	//sabft.log.GetLog().Debug("generateBlock.")
	ts := sabft.getSlotTime(sabft.slot)
	prev := &prototype.Sha256{}
	if !sabft.ForkDB.Empty() {
		prev.FromBlockID(sabft.ForkDB.Head().Id())
	} else {
		prev.Hash = make([]byte, 32)
	}
	//sabft.log.GetLog().Debugf("generating block. <prev %v>, <ts %d>", prev.Hash, ts)
	return sabft.ctrl.GenerateAndApplyBlock(sabft.Name, prev, uint32(ts), sabft.priv.privKey, prototype.Skip_nothing)
}

func (sabft *SABFT) checkGenesis() bool {
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
func (sabft *SABFT) checkProducingTiming() bool {
	now := time.Now().Round(time.Second)
	sabft.slot = sabft.getSlotAtTime(now)
	if sabft.slot == 0 {
		// not time yet, wait till the next block producing
		// cycle comes
		//nextSlotTime := sabft.getSlotTime(1)
		//time.Sleep(time.Unix(int64(nextSlotTime), 0).Sub(time.Now()))
		//sabft.log.GetLog().Info("checkProducingTiming failed.")
		return false
	}
	return true
}

func (sabft *SABFT) checkOurTurn() bool {
	producer := sabft.getScheduledProducer(sabft.slot)
	ret := strings.Compare(sabft.Name, producer) == 0
	if !ret {
		//sabft.log.GetLog().Info("checkProducingTiming failed.")
	}
	return ret
}

func (sabft *SABFT) getScheduledProducer(slot uint64) string {
	if sabft.ForkDB.Empty() {
		return sabft.validators[0].accountName
	}
	absSlot := (sabft.ForkDB.Head().Timestamp() - constants.GenesisTime) / constants.BLOCK_INTERVAL
	return sabft.validators[(absSlot+slot)%uint64(len(sabft.validators))].accountName
}

// returns false if we're out of sync
func (sabft *SABFT) checkSync() bool {
	now := time.Now().Round(time.Second).Unix()
	if sabft.getSlotTime(1) < uint64(now) {
		//time.Sleep(time.Second)
		return false
	}
	return true
}

func (sabft *SABFT) getSlotTime(slot uint64) uint64 {
	if slot == 0 {
		return 0
	}
	head := sabft.ForkDB.Head()
	if head == nil {
		return constants.GenesisTime + slot*constants.BLOCK_INTERVAL
	}

	headSlotTime := head.Timestamp() / constants.BLOCK_INTERVAL * constants.BLOCK_INTERVAL
	return headSlotTime + slot*constants.BLOCK_INTERVAL
}

func (sabft *SABFT) getSlotAtTime(t time.Time) uint64 {
	nextSlotTime := sabft.getSlotTime(1)
	if uint64(t.Unix()) < nextSlotTime {
		return 0
	}
	return (uint64(t.Unix())-nextSlotTime)/constants.BLOCK_INTERVAL + 1
}

func (sabft *SABFT) PushBlock(b common.ISignedBlock) {
	go func(blk common.ISignedBlock) {
		sabft.blkCh <- b
	}(b)
}

func (sabft *SABFT) Push(msg interface{}) {
	switch msg := msg.(type) {
	case message.ConsensusMessage:
		sabft.bft.RecvMsg(msg)
	default:
	}
}

func (sabft *SABFT) PushTransaction(trx common.ISignedTransaction, wait bool, broadcast bool) common.ITransactionReceiptWithInfo {

	var waitChan chan common.ITransactionReceiptWithInfo

	if wait {
		waitChan = make(chan common.ITransactionReceiptWithInfo)
	}

	sabft.trxCh <- func() {
		ret := sabft.ctrl.PushTrx(trx.(*prototype.SignedTransaction))

		if wait {
			waitChan <- ret
		}
		if ret.IsSuccess() {
			//	if broadcast {
			sabft.log.GetLog().Debug("SABFT Broadcast trx.")
			sabft.p2p.Broadcast(trx.(*prototype.SignedTransaction))
			//	}
		}
	}
	if wait {
		return <-waitChan
	} else {
		return nil
	}
}

func (sabft *SABFT) pushBlock(b common.ISignedBlock, applyStateDB bool) error {
	sabft.log.GetLog().Debug("pushBlock #", b.Id().BlockNum())
	// TODO: check signee & merkle

	if b.Timestamp() < sabft.getSlotTime(1) {
		sabft.log.GetLog().Debugf("the timestamp of the new block is less than that of the head block.")
	}

	head := sabft.ForkDB.Head()
	if head == nil && b.Id().BlockNum() != 1 {
		sabft.log.GetLog().Errorf("[SABFT] the first block pushed should have number of 1, got %d", b.Id().BlockNum())
		return fmt.Errorf("invalid block number")
	}

	newHead := sabft.ForkDB.PushBlock(b)
	if newHead == head {
		// this implies that b is a:
		// 1. detached block or
		// 2. out of range block or
		// 3. head of a non-main branch or
		// 4. illegal block

		if b.Id().BlockNum() > head.Id().BlockNum() {
			sabft.log.GetLog().Debugf("[SABFT][pushBlock]possibly detached block. prev: got %v, want %v", b.Id(), head.Id())
			sabft.p2p.TriggerSync(head.Id())
		}
		return nil
	} else if head != nil && newHead.Previous() != head.Id() {
		sabft.log.GetLog().Debug("[SABFT] start to switch fork.")
		sabft.switchFork(head.Id(), newHead.Id())
		return nil
	}

	if applyStateDB {
		if err := sabft.applyBlock(b); err != nil {
			// the block is illegal
			sabft.ForkDB.MarkAsIllegal(b.Id())
			sabft.ForkDB.Pop()
			return err
		}
	}
	return nil
}

/********* implements gobft ICommittee ***********/
// All the methods below will be called by gobft

// Commit sets b as the last irreversible block
func (sabft *SABFT) Commit(data message.ProposedData, commitRecords *message.Commit) error {
	sabft.Lock()
	defer sabft.Unlock()

	blockID := common.BlockID{
		Data: data,
	}
	sabft.log.GetLog().Debug("[SABFT] commit block #", blockID)

	// if we're committing a block we don't have
	blk, err := sabft.ForkDB.FetchBlock(blockID)
	if err != nil {
		panic(err)
	}

	// if blockID points to a block that is not on the current
	// longest chain, switch fork first
	blkMain, err := sabft.ForkDB.FetchBlockFromMainBranch(blockID.BlockNum())
	if err != nil {
		panic(err)
	}
	if blkMain.Id() != blockID {
		switchErr := sabft.switchFork(sabft.ForkDB.Head().Id(), blockID)
		if switchErr == true {
			panic("there's an error while switching to committed block")
		}
		// also need to reset new head
		sabft.ForkDB.ResetHead(blockID)
	}

	sabft.ctrl.Commit(blockID.BlockNum())

	if err = sabft.blog.Append(blk); err != nil {
		panic(err)
	}

	sabft.ForkDB.Commit(blockID)

	sabft.appState.LastHeight++
	sabft.appState.LastProposedData = data

	if commitRecords != nil {
		sabft.lastCommitted = commitRecords
		sabft.BroadCast(commitRecords)
		//sabft.appState.LastCommitTime = commitRecords.CommitTime
	}

	return nil
}

// GetValidator returns the validator correspond to the PubKey
func (sabft *SABFT) GetValidator(key message.PubKey) custom.IPubValidator {
	sabft.RLock()
	defer sabft.RUnlock()

	for i := range sabft.validators {
		if sabft.validators[i].accountName == string(key) {
			return sabft.validators[i]
		}
	}
	return nil
}

// IsValidator returns true if key is a validator
func (sabft *SABFT) IsValidator(key message.PubKey) bool {
	sabft.RLock()
	defer sabft.RUnlock()

	for i := range sabft.validators {
		if sabft.validators[i].accountName == string(key) {
			return true
		}
	}
	return false
}

func (sabft *SABFT) TotalVotingPower() int64 {
	sabft.RLock()
	defer sabft.RUnlock()

	return int64(len(sabft.validators))
}

func (sabft *SABFT) GetCurrentProposer(round int) message.PubKey {
	sabft.RLock()
	defer sabft.RUnlock()

	cnt := len(sabft.validators)
	return message.PubKey(sabft.validators[round%cnt].accountName)
}

// DecidesProposal decides what will be proposed if this validator is the current proposer.
func (sabft *SABFT) DecidesProposal() message.ProposedData {
	sabft.RLock()
	defer sabft.RUnlock()

	if sabft.ForkDB.Empty() {
		return message.NilData
	}

	return sabft.ForkDB.Head().Id().Data
}

// ValidateProposed validates the proposed data
func (sabft *SABFT) ValidateProposal(data message.ProposedData) bool {
	blockID := common.BlockID{
		Data: data,
	}
	sabft.RLock()
	defer sabft.RUnlock()

	if sabft.lastCommitted != nil {
		committedID := common.BlockID{
			Data: sabft.lastCommitted.Precommits[0].Proposed,
		}
		if committedID.BlockNum() >= blockID.BlockNum() {
			return false
		}
	}
	if _, err := sabft.ForkDB.FetchBlockFromMainBranch(blockID.BlockNum()); err != nil {
		return false
	}
	return true
}

func (sabft *SABFT) GetAppState() *message.AppState {
	sabft.RLock()
	defer sabft.RUnlock()

	return sabft.appState
}

// BroadCast sends msg to other validators
func (sabft *SABFT) BroadCast(msg message.ConsensusMessage) error {
	sabft.RLock()
	defer sabft.RUnlock()

	sabft.p2p.Broadcast(msg)
	return nil
}

/********* end gobft ICommittee ***********/

func (sabft *SABFT) switchFork(old, new common.BlockID) bool {
	branches, err := sabft.ForkDB.FetchBranch(old, new)
	if err != nil {
		panic(err)
	}
	sabft.log.GetLog().Debug("[SABFT][switchFork] fork branches: ", branches)
	poppedNum := len(branches[0]) - 1
	sabft.popBlock(branches[0][poppedNum])

	// producers fixup
	sabft.restoreProducers()

	appendedNum := len(branches[1]) - 1
	errWhileSwitch := false
	var newBranchIdx int
	for newBranchIdx = appendedNum - 1; newBranchIdx >= 0; newBranchIdx-- {
		b, err := sabft.ForkDB.FetchBlock(branches[1][newBranchIdx])
		if err != nil {
			panic(err)
		}
		if sabft.applyBlock(b) != nil {
			sabft.log.GetLog().Errorf("[SABFT][switchFork] applying block %v failed.", b.Id())
			errWhileSwitch = true
			// TODO: peels off this invalid branch to avoid flip-flop switch
			break
		}
	}

	// switch back
	if errWhileSwitch {
		sabft.log.GetLog().Info("[SABFT][switchFork] switch back to original fork")
		sabft.popBlock(branches[0][poppedNum])

		// producers fixup
		sabft.restoreProducers()

		for i := poppedNum - 1; i >= 0; i-- {
			b, err := sabft.ForkDB.FetchBlock(branches[0][i])
			if err != nil {
				panic(err)
			}
			sabft.applyBlock(b)
		}

		// restore the good old head of ForkDB
		sabft.ForkDB.ResetHead(branches[0][0])
	}

	return errWhileSwitch
}

func (sabft *SABFT) applyBlock(b common.ISignedBlock) error {
	//sabft.log.GetLog().Debug("applyBlock #", b.Id().BlockNum())
	err := sabft.ctrl.PushBlock(b.(*prototype.SignedBlock), prototype.Skip_nothing)
	//sabft.log.GetLog().Debugf("applyBlock #%d finished.", b.Id().BlockNum())
	return err
}

func (sabft *SABFT) popBlock(id common.BlockID) error {
	sabft.ctrl.PopBlockTo(id.BlockNum())
	return nil
}

func (sabft *SABFT) GetHeadBlockId() common.BlockID {
	if sabft.ForkDB.Empty() {
		return common.EmptyBlockID
	}
	return sabft.ForkDB.Head().Id()
}

func (sabft *SABFT) GetIDs(start, end common.BlockID) ([]common.BlockID, error) {
	blocks, err := sabft.FetchBlocksSince(start)
	if err != nil {
		return nil, err
	}

	if len(blocks) == 0 {
		return nil, nil
	}

	length := end.BlockNum() - start.BlockNum() + 1
	ret := make([]common.BlockID, 0, length)
	if start != blocks[0].Previous() {
		sabft.log.GetLog().Debugf("[GetIDs] <from: %v, to: %v> start %v", start, end, blocks[0].Previous())
		return nil, fmt.Errorf("[SABFT GetIDs] internal error")
	}

	ret = append(ret, start)
	for i := 0; i < int(length) && i < len(blocks); i++ {
		ret = append(ret, blocks[i].Id())
	}
	//sabft.log.GetLog().Debugf("FetchBlocksSince %v: %v", start, ret)
	return ret, nil
}

func (sabft *SABFT) FetchBlock(id common.BlockID) (common.ISignedBlock, error) {
	if b, err := sabft.ForkDB.FetchBlock(id); err == nil {
		return b, nil
	}

	var b prototype.SignedBlock
	if err := sabft.blog.ReadBlock(&b, int64(id.BlockNum())-1); err == nil {
		if b.Id() == id {
			return &b, nil
		}
	}

	return nil, fmt.Errorf("[SABFT FetchBlock] block with id %v doesn't exist", id)
}

func (sabft *SABFT) HasBlock(id common.BlockID) bool {
	if _, err := sabft.ForkDB.FetchBlock(id); err == nil {
		return true
	}

	var b prototype.SignedBlock
	if err := sabft.blog.ReadBlock(&b, int64(id.BlockNum())-1); err == nil {
		if b.Id() == id {
			return true
		}
	}

	return false
}

func (sabft *SABFT) FetchBlocksSince(id common.BlockID) ([]common.ISignedBlock, error) {
	length := int64(sabft.ForkDB.Head().Id().BlockNum()) - int64(id.BlockNum())
	if length < 1 {
		return nil, nil
	}

	if id.BlockNum() >= sabft.ForkDB.LastCommitted().BlockNum() {
		blocks, _, err := sabft.ForkDB.FetchBlocksSince(id)
		return blocks, err
	}

	ret := make([]common.ISignedBlock, 0, length)
	idNum := id.BlockNum()
	start := idNum + 1

	end := uint64(sabft.blog.Size())
	for start <= end {
		var b prototype.SignedBlock
		if err := sabft.blog.ReadBlock(&b, int64(start-1)); err != nil {
			return nil, err
		}

		if start == idNum && b.Id() != id {
			return nil, fmt.Errorf("blockchain doesn't have block with id %v", id)
		}

		ret = append(ret, &b)
		start++

		if start > end && b.Id() != sabft.ForkDB.LastCommitted() {
			panic("ForkDB and BLog inconsistent state")
		}
	}

	blocksInForkDB, _, err := sabft.ForkDB.FetchBlocksSince(sabft.ForkDB.LastCommitted())
	if err != nil {
		return nil, err
	}
	ret = append(ret, blocksInForkDB...)
	return ret, nil
}
