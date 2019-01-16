package consensus

import (
	"fmt"
	"github.com/sirupsen/logrus"
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
	"github.com/coschain/gobft"
	"github.com/coschain/gobft/custom"
	"github.com/coschain/gobft/message"
	"github.com/sasha-s/go-deadlock"
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
	startBFTCh    chan struct{}
	bftStarted    bool

	readyToProduce bool
	prodTimer      *time.Timer
	trxCh          chan func()
	blkCh          chan common.ISignedBlock
	bootstrap      bool
	slot           uint64

	ctx  *node.ServiceContext
	ctrl iservices.ITrxPool
	p2p  iservices.IP2P
	log  *logrus.Logger

	stopCh chan struct{}
	wg     sync.WaitGroup
	deadlock.RWMutex
}

func NewSABFT(ctx *node.ServiceContext, lg *logrus.Logger) *SABFT {
	if lg == nil {
		lg = logrus.New()
		lg.SetOutput(ioutil.Discard)
	}
	ret := &SABFT{
		ForkDB:     forkdb.NewDB(),
		validators: make([]*publicValidator, 0, 1),
		prodTimer:  time.NewTimer(1 * time.Millisecond),
		trxCh:      make(chan func()),
		blkCh:      make(chan common.ISignedBlock),
		ctx:        ctx,
		stopCh:     make(chan struct{}),
		startBFTCh: make(chan struct{}),
		log:        lg,
	}

	ret.SetBootstrap(ctx.Config().Consensus.BootStrap)
	ret.Name = ctx.Config().Consensus.LocalBpName
	ret.priv = &privateValidator{
		sab:  ret,
		name: ret.Name,
	}
	ret.bft = gobft.NewCore(ret, ret.priv)
	ret.bft.SetLogLevel(4)
	ret.log.Info("[SABFT bootstrap] ", ctx.Config().Consensus.BootStrap)
	ret.appState = &message.AppState{
		LastHeight:       0,
		LastProposedData: message.NilData,
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
	if head.Id().BlockNum()%uint64(len(sabft.validators)) != 0 {
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
	validatorNames := ""
	for i := range sabft.validators {
		validatorNames += sabft.validators[i].accountName + " "
	}
	sabft.log.Debug("[SABFT shuffle] active producers: ", validatorNames)
	sabft.ctrl.SetShuffledWitness(prods)

	sabft.suffledID = head.Id()

	if prodNum >= 4 {
		if !sabft.bftStarted {
			sabft.bftStarted = true
			go func() {
				sabft.startBFTCh <- struct{}{}
			}()
		}
	}
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
	for i := 0; i < constants.MAX_WITNESSES+1; i++ {
		// TODO: if the bft process falls behind too much, the number
		// TODO: of the avatar might not be sufficient

		// deep copy hell
		avatar = append(avatar, &prototype.SignedBlock{})
	}
	sabft.ForkDB.LoadSnapshot(avatar, snapshotPath, &sabft.blog)

	sabft.log.Info("[SABFT] starting...")
	if sabft.bootstrap && sabft.ForkDB.Empty() && sabft.blog.Empty() {
		sabft.log.Info("[SABFT] bootstrapping...")
	}
	sabft.restoreProducers()

	sabft.syncDataToSquashDB()

	// start block generation process
	go sabft.start()

	return nil
}

func (sabft *SABFT) scheduleProduce() bool {
	if !sabft.checkGenesis() {
		//sabft.log.Info("checkGenesis failed.")
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
			sabft.log.Debug("[SABFT TriggerSync]: start from ", headID.BlockNum())
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

	sabft.log.Info("[SABFT] DPoS routine started")
	for {
		select {
		case <-sabft.startBFTCh:
			sabft.log.Info("sabft gobft starting...")
			sabft.bft.Start()
			sabft.log.Info("sabft gobft started...")
		case <-sabft.stopCh:
			sabft.log.Debug("[SABFT] routine stopped.")
			return
		case b := <-sabft.blkCh:
			sabft.Lock()
			if err := sabft.pushBlock(b, true); err != nil {
				sabft.log.Error("[SABFT] pushBlock failed: ", err)
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
				sabft.log.Error("[SABFT] generateAndApplyBlock error: ", err)
				continue
			}
			sabft.prodTimer.Reset(timeToNextSec())
			sabft.log.Debugf("[SABFT] generated block: <num %d> <ts %d>", b.Id().BlockNum(), b.Timestamp())
			if err := sabft.pushBlock(b, false); err != nil {
				sabft.log.Error("[SABFT] pushBlock push generated block failed: ", err)
			}
			sabft.Unlock()

			sabft.p2p.Broadcast(b)
		}
	}
}

func (sabft *SABFT) Stop() error {
	sabft.log.Info("SABFT consensus stopped.")

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
	//sabft.log.Debug("generateBlock.")
	ts := sabft.getSlotTime(sabft.slot)
	prev := &prototype.Sha256{}
	if !sabft.ForkDB.Empty() {
		prev.FromBlockID(sabft.ForkDB.Head().Id())
	} else {
		prev.Hash = make([]byte, 32)
	}
	//sabft.log.Debugf("generating block. <prev %v>, <ts %d>", prev.Hash, ts)
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
		//sabft.log.Info("checkProducingTiming failed.")
		return false
	}
	return true
}

func (sabft *SABFT) checkOurTurn() bool {
	producer := sabft.getScheduledProducer(sabft.slot)
	ret := strings.Compare(sabft.Name, producer) == 0
	if !ret {
		//sabft.log.Info("checkProducingTiming failed.")
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
	case *message.Vote:
		sabft.RLock()
		if sabft.bftStarted {
			sabft.bft.RecvMsg(msg)
		}
		sabft.RUnlock()
	case *message.Commit:
		sabft.handleCommitRecords(msg)
	default:
	}
}

func (sabft *SABFT) handleCommitRecords(records *message.Commit) {
	sabft.log.Warn("handleCommitRecords: ", records.ProposedData, records.Address)
	if err := records.ValidateBasic(); err != nil {
		sabft.log.Error(err)
	}

	sabft.RLock()
	if sabft.lastCommitted != nil {
		oldID := common.BlockID{
			Data: sabft.lastCommitted.ProposedData,
		}
		newID := common.BlockID{
			Data: records.ProposedData,
		}
		if oldID.BlockNum() >= newID.BlockNum() {
			sabft.RUnlock()
			return
		}
	}
	sabft.RUnlock()

	// check signature
	for i := range records.Precommits {
		val := sabft.GetValidator(records.Precommits[i].Address)
		if !val.VerifySig(records.Precommits[i].Digest(), records.Precommits[i].Signature) {
			sabft.log.Error("handleCommitRecords precommits verification failed")
			return
		}
	}
	val := sabft.GetValidator(records.Address)
	if !val.VerifySig(records.Digest(), records.Signature) {
		sabft.log.Error("handleCommitRecords verification failed")
		return
	}

	sabft.Lock()
	defer sabft.Unlock()

	sabft.lastCommitted = records
	sabft.appState = &message.AppState{
		LastHeight:       records.FirstPrecommit().Height,
		LastProposedData: records.ProposedData,
	}
	sabft.log.Info("handleCommitRecords height = ", sabft.appState.LastHeight)
	// TODO: if the gobft haven't reach +2/3, push records to bft core??
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
			sabft.log.Debug("SABFT Broadcast trx.")
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
	sabft.log.Debug("pushBlock #", b.Id().BlockNum())
	// TODO: check signee & merkle

	if b.Timestamp() < sabft.getSlotTime(1) {
		sabft.log.Debugf("the timestamp of the new block is less than that of the head block.")
	}

	head := sabft.ForkDB.Head()
	if head == nil && b.Id().BlockNum() != 1 {
		sabft.log.Errorf("[SABFT] the first block pushed should have number of 1, got %d", b.Id().BlockNum())
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
			sabft.log.Debugf("[SABFT][pushBlock]possibly detached block. prev: got %v, want %v", b.Id(), head.Id())
			sabft.p2p.TriggerSync(head.Id())
		}
		return nil
	} else if head != nil && newHead.Previous() != head.Id() {
		sabft.log.Debug("[SABFT] start to switch fork.")
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
	sabft.log.Debug("pushBlock FINISHED #", b.Id().BlockNum())
	return nil
}

func (sabft *SABFT) GetLastBFTCommit() (evidence interface{}) {
	sabft.RLock()
	defer sabft.RUnlock()

	if sabft.lastCommitted == nil {
		return nil
	}
	return sabft.lastCommitted
}

/********* implements gobft ICommittee ***********/
// All the methods below will be called by gobft

// Commit sets b as the last irreversible block
func (sabft *SABFT) Commit(commitRecords *message.Commit) error {
	sabft.Lock()
	defer sabft.Unlock()

	blockID := common.BlockID{
		Data: commitRecords.ProposedData,
	}
	sabft.log.Debug("[SABFT] commit block #", blockID)

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
		if switchErr {
			panic("there's an error while switching to committed block")
			// TODO: just discard current commit process, not panic
		}
		// also need to reset new head
		sabft.ForkDB.ResetHead(blockID)
	}

	blks, _, err := sabft.ForkDB.FetchBlocksSince(sabft.ForkDB.LastCommitted())
	for i := range blks {
		if err = sabft.blog.Append(blks[i]); err != nil {
			panic(err)
		}
		if blks[i] == blk {
			break
		}
	}

	sabft.ctrl.Commit(blockID.BlockNum())

	sabft.ForkDB.Commit(blockID)

	sabft.appState.LastHeight = commitRecords.FirstPrecommit().Height
	sabft.appState.LastProposedData = commitRecords.ProposedData

	if commitRecords.Signature != nil {
		sabft.lastCommitted = commitRecords
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
	sabft.p2p.Broadcast(msg)
	return nil
}

func (sabft *SABFT) GetValidatorNum() int {
	sabft.RLock()
	defer sabft.RUnlock()

	return len(sabft.validators)
}

/********* end gobft ICommittee ***********/

func (sabft *SABFT) switchFork(old, new common.BlockID) bool {
	branches, err := sabft.ForkDB.FetchBranch(old, new)
	if err != nil {
		panic(err)
	}
	sabft.log.Debug("[SABFT][switchFork] fork branches: ", branches)
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
			sabft.log.Errorf("[SABFT][switchFork] applying block %v failed.", b.Id())
			errWhileSwitch = true
			// TODO: peels off this invalid branch to avoid flip-flop switch
			break
		}
	}

	// switch back
	if errWhileSwitch {
		sabft.log.Info("[SABFT][switchFork] switch back to original fork")
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
	//sabft.log.Debug("applyBlock #", b.Id().BlockNum())
	err := sabft.ctrl.PushBlock(b.(*prototype.SignedBlock), prototype.Skip_nothing)
	//sabft.log.Debugf("applyBlock #%d finished.", b.Id().BlockNum())
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
		//for ii := range blocks {
		//	sabft.log.Warn(blocks[ii].Id())
		//}
		sabft.log.Warnf("[GetIDs] <from: %v, to: %v> start %v", start, end, blocks[0].Previous())
		return nil, fmt.Errorf("[SABFT GetIDs] internal error")
	}

	ret = append(ret, start)
	for i := 0; i < int(length) && i < len(blocks); i++ {
		ret = append(ret, blocks[i].Id())
	}
	//sabft.log.Debugf("FetchBlocksSince %v: %v", start, ret)
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
	//sabft.log.Errorf("fetch from blog: from %d to %d", start, end)
	for start <= end {
		b := &prototype.SignedBlock{}
		if err := sabft.blog.ReadBlock(b, int64(start-1)); err != nil {
			return nil, err
		}

		if start == idNum && b.Id() != id {
			return nil, fmt.Errorf("blockchain doesn't have block with id %v", id)
		}

		ret = append(ret, b)
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

func (sabft *SABFT) getSquashDB() (iservices.IDatabaseService, error) {
	ctrl, err := sabft.ctx.Service(iservices.DbServerName)
	if err != nil {
		return nil,err
	}
	return ctrl.(iservices.IDatabaseService),nil
}

func (sabft *SABFT) syncDataToSquashDB() {
	if sabft.ForkDB.Head() == nil {
		return
	}

	db,err := sabft.getSquashDB()
	if err != nil {
		sabft.log.Debug("[sync Block]: Failed to get squash db")
		return
	}

	//Fetch the latest commit block number in squash db
	num,err := db.GetCommitNum()
	if err != nil {
		sabft.log.Debug("[sync Block]: Failed to get latest commit block number")
		return
	}

	//Fetch the real commit block number in chain
	realNum := sabft.ForkDB.LastCommitted().BlockNum()
	//Fetch the head block number in chain
	headNum := sabft.ForkDB.Head().Id().BlockNum()

	//syncNum is the start number need to fetch from ForkDB
	syncNum := num

	//Reload lost commit blocks
	if num == 0 && realNum > 0 {
		commitBlk := sabft.reloadCommitBlocks(&sabft.blog)
		cnt := len(commitBlk)
		if cnt > 0  {
			//Scene 1: The block data in squash db has been deleted,so need sync lost blocks to squash db
			//Because ForkDB will not continue to store commit blocks,so we need load all commit blocks from block log,
			//meanWhile add block to squash db,so the start value is 0
			start := 0
			if start >= cnt {
				sabft.log.Errorf("[Reload commit] start index %v out range of reload" +
					" block count %v",start,cnt)
			}else {
				sabft.log.Debugf("[Reload commit] start sync lost commit blocks from block log,start: " +
					"%v,end:%v,real commit num is %v", start, headNum, realNum)
				for i := start; i < cnt; i++ {
					blk := commitBlk[i]
					sabft.log.Debugf("[Reload commit] push block,blockNum is: " +
						"%v", blk.(*prototype.SignedBlock).Id().BlockNum())
					err = sabft.ctrl.PushBlock(blk.(*prototype.SignedBlock),prototype.Skip_nothing)
					if err != nil {
						desc := fmt.Sprintf("[Reload commit] push the block which num is %v fail,error " +
							"is %s", i, err)
						panic(desc)
					}
				}
			}
			syncNum = uint64(cnt)
		}else {
			sabft.log.Errorf("[Sync commit] Failed to get lost commit blocks from block log," +
				"start: %v," + "end:%v,real commit num is %v", syncNum+1, headNum, realNum)
		}

	}

	//1.Synchronous all the lost blocks from DorkDB to squash db
	if headNum > syncNum {
		sabft.log.Debugf("[sync pushed]: start sync lost blocks,start: %v,end:%v,real commit num " +
			"is %v", syncNum+1, headNum, realNum)
		for i := syncNum+1; i <= headNum ; i++ {
			blk,err := sabft.ForkDB.FetchBlockFromMainBranch(i)
			if err != nil {
				desc := fmt.Sprintf("[sync pushed]: Failed to fetch the pushed block by num %v,the " +
					"error is %s", i, err)
				panic(desc)
			}
			sabft.log.Debugf("[sync pushed]: sync block,num: %v,blockNum: %v",
				i, blk.(*prototype.SignedBlock).Id().BlockNum())
			err = sabft.ctrl.PushBlock(blk.(*prototype.SignedBlock),prototype.Skip_nothing)
			if err != nil {
				desc := fmt.Sprintf("[sync pushed]: push the block which num is %v fail,error is %s", i, err)
				panic(desc)
			}
		}

	}

	//2.Synchronous the lost commit blocks to squash db
	if realNum > num {
		//Need synchronous commit
		sabft.log.Debugf("[sync commit] start sync commit block num %v ",realNum)
		sabft.ctrl.Commit(realNum)
	}

}

//Load commit block from block log when Replay
func (sabft *SABFT) reloadCommitBlocks(blog *blocklog.BLog) []common.ISignedBlock {
	if blog == nil {
		return nil
	}
	if !blog.Empty() {
		var (
			i int64
			blkSli  = make([]common.ISignedBlock,blog.Size())
		)
		for i = 0; i < blog.Size(); i++ {
			blk := &prototype.SignedBlock{}
			if err := blog.ReadBlock(blk,i); err != nil {
				panic(err)
			}
			blkSli[i] = blk
		}
		return blkSli
	}
	return nil
}
