package consensus

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/db/blocklog"
	"github.com/coschain/contentos-go/db/forkdb"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/p2p/peer"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/gobft"
	"github.com/coschain/gobft/custom"
	"github.com/coschain/gobft/message"
	"github.com/sasha-s/go-deadlock"
	"github.com/sirupsen/logrus"
)

// SABFT: self-adaptive BFT
// It generates blocks in the same manner of DPoS and adopts bft
// to achieve fast block confirmation. It's self adaptive in a way
// that it can adjust the frequency of bft process based on the
// load of the blockchain and network traffic.
type SABFT struct {
	node   *node.Node
	ForkDB *forkdb.DB
	blog   blocklog.BLog

	Name         string
	localPrivKey *prototype.PrivateKeyType

	dynasties     *Dynasties
	bft           *gobft.Core
	lastCommitted atomic.Value
	appState      *message.AppState
	commitCh      chan message.Commit
	cp            *BFTCheckPoint
	noticer       EventBus.Bus

	producers      []*Producer
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

	extLog *logrus.Logger
	log    *logrus.Entry

	Ticker TimerDriver

	stopCh        chan struct{}
	inStartOrStop uint32
	wg            sync.WaitGroup
	deadlock.RWMutex

	hook map[string]func(args ...interface{})

	mockSignal     bool
	mockMalicious  bool
	maliciousBlock map[common.BlockID]common.ISignedBlock
}

func NewSABFT(ctx *node.ServiceContext, lg *logrus.Logger) *SABFT {
	if lg == nil {
		lg = logrus.New()
		lg.SetOutput(ioutil.Discard)
	}
	ret := &SABFT{
		ForkDB:         forkdb.NewDB(lg),
		dynasties:      NewDynasties(),
		prodTimer:      time.NewTimer(1 * time.Millisecond),
		trxCh:          make(chan func()),
		pendingCh:      make(chan func()),
		blkCh:          make(chan common.ISignedBlock, 1000),
		ctx:            ctx,
		stopCh:         make(chan struct{}),
		extLog:         lg,
		log:            lg.WithField("sabft", "on"),
		commitCh:       make(chan message.Commit, 100),
		Ticker:         &Timer{},
		hook:           make(map[string]func(args ...interface{})),
		maliciousBlock: make(map[common.BlockID]common.ISignedBlock),
	}

	ret.SetBootstrap(ctx.Config().Consensus.BootStrap)
	ret.Name = ctx.Config().Consensus.LocalBpName

	ret.log.Info("[SABFT bootstrap] ", ctx.Config().Consensus.BootStrap)
	deadlock.Opts.DeadlockTimeout = time.Second * 1000

	privateKey := ctx.Config().Consensus.LocalBpPrivateKey
	if len(privateKey) > 0 {
		var err error
		ret.localPrivKey, err = prototype.PrivateKeyFromWIF(privateKey)
		if err != nil {
			panic(err)
		}
	}
	return ret
}

func (sabft *SABFT) GetName() string {
	return sabft.Name
}

func (sabft *SABFT) timeToNextSec() time.Duration {
	now := sabft.Ticker.Now()
	ceil := now.Add(time.Millisecond * 500).Round(time.Second)
	return ceil.Sub(now)
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

	now := sabft.Ticker.Now().Round(time.Second)
	slot := sabft.getSlotAtTime(now)
	return sabft.getScheduledProducer(slot)
}

func (sabft *SABFT) makeProducers(names []string) []*Producer {
	ret := make([]*Producer, len(names))
	for i := range names {
		ret[i] = &Producer{
			//sab:         sabft,
			accountName: names[i],
		}
	}
	return ret
}

func (sabft *SABFT) shuffle(head common.ISignedBlock) (bool, []string) {
	blockNum := head.Id().BlockNum()
	if blockNum%constants.BlockProdRepetition != 0 ||
		blockNum/constants.BlockProdRepetition%uint64(len(sabft.producers)) != 0 {
		return false, []string{}
	}

	_ = sabft.ctrl.PreShuffle()

	// When a produce round complete, it adds new producers,
	// remove unqualified producers and shuffle the block-producing order
	prods, pubKeys := sabft.ctrl.GetBlockProducerTopN(constants.MaxBlockProducerCount)

	var seed uint64
	if head != nil {
		seed = head.Timestamp() << 32
	}
	sabft.updateProducers(seed, prods, pubKeys, blockNum)
	newDyn := sabft.makeDynasty(blockNum, prods, pubKeys, sabft.localPrivKey)
	sabft.addDynasty(newDyn)

	return true, prods
}

func (sabft *SABFT) addDynasty(d *Dynasty) {
	sabft.log.Info("add dynasty: ", d.Seq)
	sabft.dynasties.PushBack(d)
}

func (sabft *SABFT) makeDynasty(seq uint64, prods []string,
	keys []*prototype.PublicKeyType, pk *prototype.PrivateKeyType) *Dynasty {
	pubVS := make([]*publicValidator, len(prods))
	for i := range pubVS {
		pubVS[i] = newPubValidator(sabft, keys[i], prods[i])
	}
	pV := newPrivValidator(sabft, sabft.localPrivKey, sabft.Name)
	return NewDynasty(seq, pubVS, pV)
}

func (sabft *SABFT) restoreProducers() {
	prods, _, _ := sabft.ctrl.GetShuffledBpList()
	sabft.producers = sabft.makeProducers(prods)
	sabft.log.Info("[SABFT] active producers: ", prods)
}

func (sabft *SABFT) updateProducers(seed uint64, prods []string, pubKeys []*prototype.PublicKeyType, seq uint64) int {
	prodNum := len(prods)
	for i := 0; i < prodNum; i++ {
		k := seed + uint64(i)*2695921657736338717
		k ^= k >> 12
		k ^= k << 25
		k ^= k >> 27
		k *= 2695921657736338717

		j := i + int(k%uint64(prodNum-i))
		prods[i], prods[j] = prods[j], prods[i]
		pubKeys[i], pubKeys[j] = pubKeys[j], pubKeys[i]
	}

	sabft.producers = sabft.makeProducers(prods)
	validatorNames := ""
	for i := range sabft.producers {
		validatorNames += sabft.producers[i].accountName + " "
	}
	sabft.log.Debug("[SABFT shuffle] active producers: ", validatorNames)
	sabft.ctrl.SetShuffledBpList(prods, pubKeys, seq)

	return prodNum
}

func (sabft *SABFT) ActiveProducers() []string {
	sabft.RLock()
	defer sabft.RUnlock()

	// TODO
	return nil
}

func (sabft *SABFT) ActiveValidators() []string {
	if sabft.dynasties.Empty() {
		e := fmt.Sprintf("empty dynasty in %s", sabft.Name)
		panic(e)
	}
	valset := sabft.dynasties.Front().validators
	v := make([]string, len(valset))
	for i := range v {
		v[i] = valset[i].accountName
	}
	return v
}

func (sabft *SABFT) Start(node *node.Node) error {
	if !atomic.CompareAndSwapUint32(&sabft.inStartOrStop, 0, 1) {
		return fmt.Errorf("consensus in the process of start or stop")
	}
	defer atomic.StoreUint32(&sabft.inStartOrStop, 0)

	sabft.ctrl = sabft.getController()
	p2p, err := sabft.ctx.Service(iservices.P2PServerName)
	if err != nil {
		panic(err)
	}
	sabft.noticer = node.EvBus
	sabft.p2p = p2p.(iservices.IP2P)
	cfg := sabft.ctx.Config()
	if err = sabft.blog.Open(cfg.ResolvePath("blog")); err != nil {
		panic(err)
	}
	sabft.ctrl.SetShuffle(func(block common.ISignedBlock) (bool, []string) {
		return sabft.shuffle(block)
	})

	sabft.stateFixup(&cfg)

	sabft.bft = gobft.NewCore(sabft, sabft.dynasties.Front().priv)
	//pv := newPrivValidator(sabft, sabft.localPrivKey, sabft.Name)
	//sabft.bft = gobft.NewCore(sabft, pv)
	sabft.bft.SetLogger(sabft.extLog)
	sabft.bft.SetName(sabft.Name)

	sabft.log.Info("[SABFT] starting...")
	if sabft.bootstrap && sabft.ForkDB.Empty() && sabft.blog.Empty() {
		sabft.log.Info("[SABFT] bootstrapping...")
	}
	// start block generation process
	go sabft.start()

	return nil
}

func (sabft *SABFT) stateFixup(cfg *node.Config) {
	// reload ForkDB
	snapshotPath := cfg.ResolvePath(constants.ForkDBSnapshot)
	sabft.ForkDB.LoadSnapshot(reflect.TypeOf(prototype.SignedBlock{}), snapshotPath, &sabft.blog)
	/**** at this point, blog and forkdb is consistent ****/

	sabft.cp = NewBFTCheckPoint(cfg.ResolvePath(constants.CheckPoint), sabft)
	if !sabft.ForkDB.Empty() && !sabft.blog.Empty() {
		lc, err := sabft.cp.GetNext(sabft.ForkDB.LastCommitted().BlockNum() - 1)
		if err != nil {
			sabft.log.Error(err)
		} else {
			sabft.lastCommitted.Store(lc)
		}
	}
	// restore gobft state
	k := sabft.ForkDB.LastCommitted().BlockNum()
	if k > 0 {
		k--
	}
	lastCommit, err := sabft.cp.GetNext(k)
	var lh int64
	if err == nil {
		lh = lastCommit.Height()
	}
	sabft.appState = &message.AppState{
		LastHeight:       lh,
		LastProposedData: sabft.ForkDB.LastCommitted().Data,
	}
	sabft.log.Warn("last bft height ", lh)

	if err := sabft.databaseFixup(cfg); err != nil {
		panic(err)
	}
}

func (sabft *SABFT) restoreDynasty() {
	//prods, pubKeys := sabft.ctrl.GetBlockProducerTopN(constants.MaxBlockProducerCount)
	prods, pubKeys, seq := sabft.ctrl.GetShuffledBpList()
	//sabft.log.Warn("ssssssssss ", prods)
	dyn := sabft.makeDynasty(seq, prods, pubKeys, sabft.localPrivKey)
	sabft.log.Info("restoring dynasty ", dyn.String())
	sabft.addDynasty(dyn)
}

func (sabft *SABFT) tooManyUncommittedBlocks() bool {
	if sabft.ForkDB.Empty() {
		return false
	}
	headNum := sabft.ForkDB.Head().Id().BlockNum()
	lastCommittedNum := sabft.ForkDB.LastCommitted().BlockNum()
	if headNum-lastCommittedNum > constants.MaxUncommittedBlockNum {
		return true
	}
	return false
}

func (sabft *SABFT) scheduleProduce() bool {
	if !sabft.checkGenesis() {
		//sabft.log.Info("checkGenesis failed.")
		return false
	}

	if !sabft.readyToProduce {
		if sabft.checkSync() {
			sabft.readyToProduce = true
			sabft.log.Debugf("head block id: %d, timestamp %v", sabft.ForkDB.Head().Id().BlockNum(), time.Unix(int64(sabft.ForkDB.Head().Timestamp()), 0))
		} else {
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
		return false
	}
	return true
}

func (sabft *SABFT) revertToLastCheckPoint() {
	lastCommittedID := sabft.ForkDB.LastCommitted()
	popNum := lastCommittedID.BlockNum() + 1
	sabft.popBlock(popNum)

	var lastCommittedBlock common.ISignedBlock = nil
	var err error
	if popNum > 1 {
		lastCommittedBlock, err = sabft.ForkDB.FetchBlock(lastCommittedID)
		if err != nil {
			panic(err)
		}
	}
	sabft.ForkDB = forkdb.NewDB(sabft.extLog)
	if popNum > 1 {
		sabft.ForkDB.PushBlock(lastCommittedBlock)
		sabft.ForkDB.Commit(lastCommittedID)
	}

	sabft.log.Infof("[SABFT][checkpoint] revert to last committed block %d.", popNum-1)
}

func (sabft *SABFT) resetResource() {
	sabft.trxCh = make(chan func())
	sabft.pendingCh = make(chan func())
	sabft.blkCh = make(chan common.ISignedBlock, 100)
	sabft.stopCh = make(chan struct{})
	sabft.commitCh = make(chan message.Commit, 100)
}

func (sabft *SABFT) start() {
	sabft.resetResource()

	sabft.wg.Add(1)
	defer sabft.wg.Done()

	sabft.log.Info("[SABFT] DPoS routine started")
	for {
		select {
		case <-sabft.stopCh:
			sabft.log.Debug("[SABFT] routine stopped.")
			return
		case b := <-sabft.blkCh:
			//sabft.log.Debug("handling block ", b.Id().BlockNum())
			if sabft.readyToProduce && sabft.tooManyUncommittedBlocks() &&
				b.Id().BlockNum() > sabft.ForkDB.Head().Id().BlockNum() {
				sabft.log.Debugf("dropping new block %v cause we had too many uncommitted blocks", b.Id())
				continue
				//return
			}
			sabft.Lock()
			err := sabft.pushBlock(b, true)
			sabft.Unlock()
			if err != nil {
				sabft.log.Error("[SABFT] pushBlock failed: ", err)
				continue
			}

			sabft.Lock()
			sabft.tryCommit(b)
			sabft.Unlock()

		case trxFn := <-sabft.trxCh:
			sabft.Lock()
			trxFn()
			sabft.Unlock()
			continue
		case commit := <-sabft.commitCh:
			// TODO: reduce critical area guarded by lock
			sabft.handleCommitRecords(&commit)
		case pendingFn := <-sabft.pendingCh:
			pendingFn()
			continue
		case <-sabft.prodTimer.C:
			sabft.MaybeProduceBlock()
		}
	}
}

func (sabft *SABFT) tryCommit(b common.ISignedBlock) {
	commit := sabft.cp.NextUncommitted()
	if commit != nil && b.Id() == ExtractBlockID(commit) {
		success := sabft.verifyCommitSig(commit)
		if !success {
			if !sabft.readyToProduce {
				sabft.revertToLastCheckPoint()
			}
			// remove this invalid checkpoint
			sabft.cp.RemoveNextUncommitted()
			// TODO: fetch next checkpoint from a different peer
		} else {
			sabft.log.Debugf("new block %d on an existed checkpoint", b.Id().BlockNum())
			// if we're a validator and the gobft falls behind, pass the commit to gobft and let it catchup
			if sabft.gobftCatchUp(commit) {
				return
			}

			// if it suits the following situation:
			// 1. we're not a validator
			// 2. gobft is far ahead due to committing missing blocks
			if err := sabft.commit(commit); err != nil {
				sabft.log.Error(err)
			}
		}
	}
}

func (sabft *SABFT) checkBFTRoutine() {
	if sabft.readyToProduce && sabft.isValidatorName(sabft.Name) {
		//sabft.log.Infof("[SABFT] Starting gobft")
		if err := sabft.bft.Start(); err == nil {
			sabft.log.Infof("[SABFT] gobft started at height %d", sabft.appState.LastHeight)
		}
	} else {
		//sabft.log.Info("[SABFT] Stopping gobft")
		if err := sabft.bft.Stop(); err == nil {
			sabft.log.Info("[SABFT] gobft stopped")
		}
	}
}

func (sabft *SABFT) Stop() error {
	if !atomic.CompareAndSwapUint32(&sabft.inStartOrStop, 0, 1) {
		return fmt.Errorf("consensus in the process of start or stop")
	}
	defer atomic.StoreUint32(&sabft.inStartOrStop, 0)

	sabft.log.Info("[SABFT] Stopping SABFT consensus")

	close(sabft.stopCh)
	sabft.readyToProduce = false
	sabft.wg.Wait()

	// stop bft process
	if err := sabft.bft.Stop(); err != nil {
		sabft.log.Info("[SABFT] gobft stopped...")
	} else {
		sabft.log.Info(err)
	}

	// restore uncommitted forkdb
	cfg := sabft.ctx.Config()
	snapshotPath := cfg.ResolvePath("forkdb_snapshot")
	sabft.ForkDB.Snapshot(snapshotPath)

	sabft.prodTimer.Stop()
	sabft.cp.Close()
	sabft.dynasties.Clear()
	sabft.log.Info("[SABFT] SABFT consensus stopped")
	return nil
}

func (sabft *SABFT) generateAndApplyBlock() (common.ISignedBlock, error) {
	sabft.log.Debug("start generateBlock.")
	ts := sabft.getSlotTime(sabft.slot)
	prev := &prototype.Sha256{}
	if !sabft.ForkDB.Empty() {
		prev.FromBlockID(sabft.ForkDB.Head().Id())
	} else {
		prev.Hash = make([]byte, 32)
	}
	sabft.log.Debugf("generating block. <prev %v>, <ts %d>", prev.Hash, ts)
	b, err := sabft.ctrl.GenerateAndApplyBlock(sabft.Name, prev,
		uint32(ts), sabft.dynasties.Back().priv.privKey, prototype.Skip_nothing)
	return b, err
}

func (sabft *SABFT) checkGenesis() bool {
	now := sabft.Ticker.Now()
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
	now := sabft.Ticker.Now().Round(time.Second)
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
		return sabft.producers[0].accountName
	}
	absSlot := (sabft.ForkDB.Head().Timestamp() - constants.GenesisTime) / constants.BlockInterval
	return sabft.producers[(absSlot+slot)/constants.BlockProdRepetition%uint64(len(sabft.producers))].accountName
}

// returns false if we're out of sync
func (sabft *SABFT) checkSync() bool {
	now := sabft.Ticker.Now().Round(time.Second).Unix()
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
		return constants.GenesisTime + slot*constants.BlockInterval
	}

	headSlotTime := head.Timestamp() / constants.BlockInterval * constants.BlockInterval
	return headSlotTime + slot*constants.BlockInterval
}

func (sabft *SABFT) getSlotAtTime(t time.Time) uint64 {
	nextSlotTime := sabft.getSlotTime(1)
	if uint64(t.Unix()) < nextSlotTime {
		return 0
	}
	return (uint64(t.Unix())-nextSlotTime)/constants.BlockInterval + 1
}

func (sabft *SABFT) PushBlock(b common.ISignedBlock) {
	sabft.log.Debug("[SABFT] recv block from p2p: ", b.Id().BlockNum())
	sabft.blkCh <- b
}

func (sabft *SABFT) Push(msg interface{}, p common.IPeer) {
	switch m := msg.(type) {
	case *message.Vote:
		sabft.bft.RecvMsg(m, p.(*peer.Peer))
	case *message.FetchVotesReq:
		sabft.bft.RecvMsg(m, p.(*peer.Peer))
	case *message.FetchVotesRsp:
		sabft.bft.RecvMsg(m, p.(*peer.Peer))
	case *message.Commit:
		sabft.commitCh <- *m
	default:
	}
}

func (sabft *SABFT) verifyCommitSig(records *message.Commit) bool {
	if sabft.dynasties.Empty() {
		e := fmt.Sprintf("empty dynasty in %s", sabft.Name)
		panic(e)
	}

	valNum := sabft.dynasties.Front().GetValidatorNum()
	precommitNum := len(records.Precommits)
	if precommitNum <= valNum*2/3 {
		sabft.log.Errorf("insufficient precommits in Commit %d ** %d: %v", valNum, precommitNum, records)
		return false
	}

	for i := range records.Precommits {
		val := sabft.dynasties.Front().GetValidatorByPubKey(records.Precommits[i].Address)
		if val == nil {
			sabft.log.Errorf("[SABFT] error while checking precommits: %s is not a validator, current Dynasty: %s",
				records.Precommits[i].Address, sabft.dynasties.Front().String())
			return false
		}
		v := val.VerifySig(records.Precommits[i].Digest(), records.Precommits[i].Signature)
		if !v {
			sabft.log.Error("[SABFT] precommits verification failed")
			return false
		}
	}
	val := sabft.dynasties.Front().GetValidatorByPubKey(records.Address)
	if val == nil {
		sabft.log.Errorf("[SABFT] error while checking commits. %s is not a validator", string(records.Address))
		return false
	}
	v := val.VerifySig(records.Digest(), records.Signature)
	if !v {
		sabft.log.Error("[SABFT] verification failed")
		return false
	}
	return true
}

func (sabft *SABFT) checkCommittedAlready(id common.BlockID) bool {
	lastCommitted := sabft.lastCommitted.Load()
	if lastCommitted != nil {
		oldID := common.BlockID{
			Data: lastCommitted.(*message.Commit).ProposedData,
		}
		if oldID.BlockNum() >= id.BlockNum() {
			return true
		}
	}
	return false
}

func (sabft *SABFT) handleCommitRecords(records *message.Commit) {
	sabft.Lock()
	defer sabft.Unlock()

	if records == nil {
		return
	}

	// make sure we haven't committed it already
	newID := ExtractBlockID(records)
	if sabft.checkCommittedAlready(newID) {
		return
	}

	// TODO: handle multiple cp at same height
	err := sabft.cp.Add(records)
	if err != nil {
		return
	}

	sabft.loopCommit(records)
}

func (sabft *SABFT) loopCommit(commit *message.Commit) {
	checkPoint := commit
	for checkPoint != nil {
		newID := ExtractBlockID(checkPoint)
		if !sabft.cp.IsNextCheckPoint(checkPoint) {
			sabft.log.Warn("cp check IsNextCheckPoint failed")
			return
		}
		sabft.log.Debug("reach checkpoint at ", checkPoint)

		// if we're a validator, pass it to gobft so that it can catch up
		if sabft.gobftCatchUp(checkPoint) {
			return
		}

		if !sabft.verifyCommitSig(checkPoint) {
			sabft.log.Error("validation on checkpoint failed, remove it")
			sabft.cp.Remove(checkPoint)
			return
		}
		if _, err := sabft.ForkDB.FetchBlock(newID); err == nil {
			if err = sabft.commit(checkPoint); err == nil {
				checkPoint = sabft.cp.NextUncommitted()
				if checkPoint != nil {
					sabft.log.Debug("loop checkpoint at ", checkPoint.ProposedData)
				} else {
					sabft.log.Warn("NextUncommitted is nil")
				}
				continue
			} else {
				sabft.log.Warn("commit cp error ", err)
			}
		} else {
			sabft.log.Warn("forkdb can't fetch newID: ", newID.BlockNum(), " ", err)
		}
		break
	}
}

func (sabft *SABFT) gobftCatchUp(commit *message.Commit) bool {
	if atomic.LoadUint32(&sabft.inStartOrStop) == 0 &&
		!sabft.dynasties.Empty() &&
		sabft.isValidatorName(sabft.Name) &&
		sabft.appState.LastProposedData == commit.Prev {
		sabft.log.Warn("pass commits to gobft ", commit.ProposedData)
		if err := sabft.bft.RecvMsg(commit, nil); err == nil {
			return true
		}
	}
	return false
}

func (sabft *SABFT) validateProducer(b common.ISignedBlock) bool {
	head := sabft.ForkDB.Head()
	slotTime := sabft.getSlotTime(sabft.getSlotAtTime(time.Now()))
	if head.Timestamp() >= b.Timestamp() || b.Timestamp() > slotTime {
		return false
	}

	slot := sabft.getSlotAtTime(time.Unix(int64(b.Timestamp()), 0))
	validProducer := sabft.getScheduledProducer(slot)
	producer, err := b.GetSignee()
	if err != nil {
		sabft.log.Error(err)
		return false
	}
	pubKey := producer.(*prototype.PublicKeyType)
	res := sabft.ctrl.ValidateAddress(validProducer, pubKey)
	if !res {
		if !sabft.ForkDB.Empty() && b.Id().BlockNum() == sabft.ForkDB.Head().Id().BlockNum()+1 {
			sabft.log.Errorf("block %v's valid producer should be %s, but the block's pub_key is %s",
				b.Id(), validProducer, pubKey.ToWIF())
		}
	}
	return res
}

func (sabft *SABFT) PushTransactionToPending(trx common.ISignedTransaction) error {

	if !sabft.readyToProduce {
		return ErrConsensusNotReady
	}

	chanError := make(chan error)
	go func() {
		err := sabft.ctrl.PushTrxToPending(trx.(*prototype.SignedTransaction))
		if err == nil {
			go sabft.p2p.Broadcast(trx.(*prototype.SignedTransaction))
		}
		chanError <- err
	}()

	return <-chanError
}

func (sabft *SABFT) pushMaliciousBlock(b common.ISignedBlock) {
	if !sabft.mockMalicious {
		return
	}
	sabft.maliciousBlock[b.Id()] = b
}

func (sabft *SABFT) pushBlock(b common.ISignedBlock, applyStateDB bool) error {
	sabft.log.Debug("[SABFT] start pushBlock #", b.Id().BlockNum())
	var headNum uint64
	head := sabft.ForkDB.Head()
	if head != nil {
		headNum = head.Id().BlockNum()
	}
	newID := b.Id()
	newNum := newID.BlockNum()

	if newNum > headNum+1 {

		//if sabft.readyToProduce {
		//	sabft.p2p.FetchUnlinkedBlock(b.Previous())
		//	sabft.log.Debug("[SABFT TriggerSync]: out-of range from ", b.Previous().BlockNum())
		//}

		if sabft.readyToProduce {
			if !sabft.checkSync() {
				//sabft.readyToProduce = false

				var headID common.BlockID
				if !sabft.ForkDB.Empty() {
					headID = sabft.ForkDB.Head().Id()
				}
				sabft.p2p.FetchOutOfRange(headID, b.Id())

				sabft.log.Debug("[SABFT TriggerSync]: out-of range from ", headID.BlockNum())
			}
		}

		return ErrBlockOutOfScope
	}

	if head != nil && b.Previous() == head.Id() && applyStateDB {
		if !sabft.validateProducer(b) {
			return ErrInvalidProducer
		}
	}

	if head == nil && newNum != 1 {
		sabft.log.Errorf("[SABFT] the first block pushed should have number of 1, got %d", b.Id().BlockNum())
		return ErrInvalidBlockNum
	}

	rc := sabft.ForkDB.PushBlock(b)
	newHead := sabft.ForkDB.Head()
	switch rc {
	case forkdb.RTDetached:
		sabft.log.Debugf("[SABFT][pushBlock]possibly detached block. prev: got %v, want %v", b.Previous(), head.Id())
		tailId, errTail := sabft.ForkDB.FetchUnlinkBlockTail()
		if sabft.HasBlock(*tailId) {
			panic("GOT unlinked but exist")
		}

		if errTail == nil {
			sabft.p2p.FetchUnlinkedBlock(*tailId)
			sabft.log.Debug("[SABFT TriggerSync]: pre-start from ", tailId.BlockNum())
		} else {
			sabft.log.Debug("[SABFT TriggerSync]: not found:", errTail)
		}
		return nil
	case forkdb.RTOutOfRange:
		if b.Id().BlockNum() <= sabft.ForkDB.LastCommitted().BlockNum() {
			sabft.log.Warnf("[SABFT]: RTOutOfRange: %v, committed: %v", b.Previous(),
				sabft.ForkDB.LastCommitted())
			return nil
		}
		sabft.p2p.FetchUnlinkedBlock(b.Previous())
		sabft.log.Debug("[SABFT TriggerSync]: out-of range2 from ", b.Previous().BlockNum())
		return ErrBlockOutOfScope
	case forkdb.RTInvalid:
		return ErrInvalidBlock
	case forkdb.RTDuplicated:
		return ErrDupBlock
	case forkdb.RTPushedOnFork:
		sabft.log.Debugf("[SABFT] block %d pushed on fork branch", newNum)
		if newHead != head && newHead.Previous() != head.Id() {
			sabft.log.Debug("[SABFT] start to switch fork.")
			switchSuccess := sabft.switchFork(head.Id(), newHead.Id())
			if !switchSuccess {
				sabft.log.Error("[SABFT] there's an error while switching to new branch. new head", newHead.Id())
			} else {
				sabft.log.Info("[SABFT] switch fork success, new head", newHead)
			}
		}
	case forkdb.RTPushedOnMain:
		if applyStateDB {
			if err := sabft.applyBlock(b); err != nil {
				// the block is illegal
				sabft.ForkDB.MarkAsIllegal(b.Id())
				sabft.ForkDB.Pop()
				return err
			}
		}
		sabft.log.Debug("[SABFT] pushBlock FINISHED #", b.Id().BlockNum(), " id ", b.Id())
		if sabft.mockMalicious && !applyStateDB {
			return nil
		}
		sabft.p2p.Broadcast(b)
	default:
		return ErrInternal
	}
	if f, exist := sabft.hook["branches"]; exist {
		f()
	}
	return nil
}

func (sabft *SABFT) GetLastBFTCommit() interface{} {
	lastCommitted := sabft.lastCommitted.Load()

	if lastCommitted == nil {
		return nil
	}
	return lastCommitted.(*message.Commit)
}

func (sabft *SABFT) GetBFTCommitInfo(num uint64) interface{} {
	if sabft.cp == nil {
		return nil
	}
	if num < 1 {
		num = 1
	}

	c, err := sabft.cp.GetNext(num-1)
	if err != nil {
		return nil
	}
	return c
}

func (sabft *SABFT) GetNextBFTCheckPoint(blockNum uint64) interface{} {
	//sabft.RLock()
	//defer sabft.RUnlock()

	commit, err := sabft.cp.GetNext(blockNum)
	if err != nil {
		sabft.log.Error(err)
		return nil
	}
	return commit
}

func (sabft *SABFT) GetLIB() common.BlockID {
	lastCommitted := sabft.lastCommitted.Load()
	if lastCommitted == nil {
		return common.EmptyBlockID
	}
	return common.BlockID{
		Data: lastCommitted.(*message.Commit).ProposedData,
	}
}

/********* implements gobft ICommittee ***********/
// All the methods below will be called by gobft

// Commit sets b as the last irreversible block
func (sabft *SABFT) Commit(commitRecords *message.Commit) error {
	sabft.Lock()
	defer sabft.Unlock()

	sabft.log.Info("[SABFT] try to commit ", commitRecords)
	if !sabft.verifyCommitSig(commitRecords) {
		sabft.updateAppState(commitRecords)
		return ErrInvalidCheckPoint
	}

	err := sabft.cp.Add(commitRecords)
	if err == ErrCheckPointOutOfRange || err == ErrInvalidCheckPoint {
		sabft.log.Error(err)
		sabft.updateAppState(commitRecords)
		return err
	}
	err = sabft.commit(commitRecords)
	if err == nil {
		// try to catchup if falls behind
		checkPoint := sabft.cp.NextUncommitted()
		if checkPoint != nil {
			if !sabft.gobftCatchUp(checkPoint) {
				sabft.loopCommit(checkPoint)
			}
		}
		return nil
	}
	if err == ErrCommitted {
		// do nothing
	} else if err == ErrCommittingNonExistBlock {
		// wait for the block to arrive
	} else {
		panic(err)
	}

	return err
}

func (sabft *SABFT) updateAppState(commit *message.Commit) {
	if sabft.appState.LastHeight+1 == commit.FirstPrecommit().Height {
		sabft.appState.LastHeight++
		sabft.appState.LastProposedData = commit.ProposedData
		sabft.log.Debugf("[SABFT] gobft LastHeight %d", sabft.appState.LastHeight)
	}
}

func (sabft *SABFT) commit(commitRecords *message.Commit) error {
	defer func() {
		sabft.updateAppState(commitRecords)
		sabft.log.Debug("current dyn ", sabft.dynasties.Front().String())

		// TODO: check if checkpoint has been skipped
	}()

	blockID := common.BlockID{
		Data: commitRecords.ProposedData,
	}
	blockNum := blockID.BlockNum()

	sabft.log.Infof("[SABFT] start to commit block #%d %v %d", blockNum, blockID, commitRecords.FirstPrecommit().Height)
	// if we're committing a block we don't have
	blk, err := sabft.ForkDB.FetchBlock(blockID)
	if err != nil {
		// we're falling behind, just wait for next commit
		sabft.log.Error("[SABFT] committing a missing block", blockID)
		return ErrCommittingNonExistBlock
	}

	if sabft.ForkDB.LastCommitted() == blockID {
		return ErrCommitted
	}

	blkMain, err := sabft.ForkDB.FetchBlockFromMainBranch(blockNum)
	if err != nil {
		sabft.log.Errorf("[SABFT] internal error when committing %v, err: %v", blockID, err)
		return ErrInternal
	}
	if blkMain.Id() != blockID {
		sabft.log.Error("[SABFT] committing a forked block", blockID, " main:", blkMain.Id())
		switchSuccess := sabft.switchFork(sabft.ForkDB.Head().Id(), blockID)
		if !switchSuccess {
			return ErrSwitchFork
		}
		sabft.log.Debug("fork switch success during commit. new head ", blockID)
		return nil
	}

	blks, _, err := sabft.ForkDB.FetchBlocksSince(sabft.ForkDB.LastCommitted())
	if err != nil {
		sabft.log.Errorf("[SABFT] internal error when committing %v, err: %v", blockID, err)
		return ErrInternal
	}
	commitCount := 0
	for i := range blks {
		if err = sabft.blog.Append(blks[i]); err != nil {
			sabft.log.Errorf("[SABFT] internal error when committing %v, err: %v", blockID, err)
			return ErrInternal
		}
		commitCount++
		if blks[i] == blk {
			sabft.log.Debugf("[SABFT] committed from block #%d to #%d", blks[0].Id().BlockNum(), blk.Id().BlockNum())
			break
		}
	}

	sabft.noticer.Publish(constants.NoticeLibChange, blks[:commitCount])
	sabft.ctrl.Commit(blockNum)
	sabft.ForkDB.Commit(blockID)
	sabft.lastCommitted.Store(commitRecords)
	sabft.cp.Flush(blockID)
	sabft.dynasties.Purge(blockNum)

	sabft.log.Debug("[SABFT] committed block #", blockID)
	if f, exist := sabft.hook["commit"]; exist {
		f(blk)
	}
	return nil
}

// GetValidator returns the validator correspond to the PubKey
func (sabft *SABFT) GetValidator(key message.PubKey) custom.IPubValidator {
	sabft.RLock()
	defer sabft.RUnlock()

	return sabft.getValidator(key)
}

func (sabft *SABFT) getValidator(key message.PubKey) custom.IPubValidator {
	if sabft.dynasties.Empty() {
		e := fmt.Sprintf("empty dynasty in %s", sabft.Name)
		panic(e)
	}

	valset := sabft.dynasties.Front().validators
	for i := range valset {
		if valset[i].bftPubKey == key {
			return valset[i]
		}
	}
	sabft.log.Errorf("cannot get validator for %v, current dyn %s", key, sabft.dynasties.Front().String())
	return nil
}

// IsValidator returns true if key is a validator
func (sabft *SABFT) IsValidator(key message.PubKey) bool {
	sabft.RLock()
	defer sabft.RUnlock()

	return sabft.isValidator(key)
}

func (sabft *SABFT) isValidator(key message.PubKey) bool {
	if sabft.dynasties.Empty() {
		e := fmt.Sprintf("empty dynasty in %s", sabft.Name)
		panic(e)
	}

	valset := sabft.dynasties.Front().validators
	for i := range valset {
		if valset[i].bftPubKey == key {
			return true
		}
	}
	return false
}

func (sabft *SABFT) isValidatorName(name string) bool {
	if sabft.dynasties.Empty() {
		e := fmt.Sprintf("empty dynasty in %s", sabft.Name)
		panic(e)
	}

	valset := sabft.dynasties.Front().validators
	for i := range valset {
		if valset[i].accountName == name {
			return true
		}
	}
	return false
}

func (sabft *SABFT) TotalVotingPower() int64 {
	sabft.RLock()
	defer sabft.RUnlock()

	if sabft.dynasties.Empty() {
		e := fmt.Sprintf("empty dynasty in %s", sabft.Name)
		panic(e)
	}
	return int64(sabft.dynasties.Front().GetValidatorNum())
}

func (sabft *SABFT) GetCurrentProposer(round int) message.PubKey {
	sabft.RLock()
	defer sabft.RUnlock()

	if sabft.dynasties.Empty() {
		e := fmt.Sprintf("empty dynasty in %s", sabft.Name)
		panic(e)
	}
	dyn := sabft.dynasties.Front()
	cnt := dyn.GetValidatorNum()
	return message.PubKey(dyn.validators[round%cnt].bftPubKey)
}

// DecidesProposal decides what will be proposed if this validator is the current proposer.
func (sabft *SABFT) DecidesProposal() message.ProposedData {
	sabft.RLock()
	defer sabft.RUnlock()

	if sabft.ForkDB.Empty() {
		return message.NilData
	}

	lc := sabft.ForkDB.LastCommitted().BlockNum()
	if sabft.ForkDB.Head().Id().BlockNum()-lc > constants.MaxMarginStep {
		b, err := sabft.ForkDB.FetchBlockFromMainBranch(lc + constants.MaxMarginStep)
		if err != nil {
			return message.NilData
		}
		return b.Id().Data
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

	if blockID.BlockNum() <= sabft.ForkDB.LastCommitted().BlockNum() {
		return false
	}

	if _, err := sabft.ForkDB.FetchBlock(blockID); err != nil {
		return false
	}

	return true
}

func (sabft *SABFT) GetAppState() *message.AppState {
	//sabft.RLock()
	//defer sabft.RUnlock()

	return sabft.appState
}

func (sabft *SABFT) GetValidatorNum() int {
	sabft.RLock()
	defer sabft.RUnlock()

	if sabft.dynasties.Empty() {
		e := fmt.Sprintf("empty dynasty in %s", sabft.Name)
		panic(e)
	}
	return sabft.dynasties.Front().GetValidatorNum()
}

func (sabft *SABFT) GetValidatorList() []message.PubKey {
	return nil
}

func (sabft *SABFT) GetCommitHistory(height int64) *message.Commit {
	c, _ := sabft.cp.GetIth(uint64(height))
	return c
}

/********* end gobft ICommittee ***********/

func (sabft *SABFT) BroadCast(msg message.ConsensusMessage) error {
	sabft.p2p.Broadcast(msg)
	return nil
}

func (sabft *SABFT) Send(msg message.ConsensusMessage, p custom.IPeer) error {
	if p == nil {
		sabft.p2p.RandomSend(msg)
	} else {
		sabft.p2p.SendToPeer(p.(*peer.Peer), msg)
	}
	return nil
}

func (sabft *SABFT) switchFork(old, new common.BlockID) bool {
	branches, err := sabft.ForkDB.FetchBranch(old, new)
	if err != nil {
		panic(err)
	}
	sabft.log.Debug("[SABFT][switchFork] fork branches: ", branches)
	poppedNum := len(branches[0]) - 1
	sabft.popBlock(branches[0][poppedNum-1].BlockNum())

	appendedNum := len(branches[1]) - 1
	errWhileSwitch := false
	var newBranchIdx int
	for newBranchIdx = appendedNum - 1; newBranchIdx >= 0; newBranchIdx-- {
		b, err := sabft.ForkDB.FetchBlock(branches[1][newBranchIdx])
		if err != nil {
			panic(err)
		}
		if !sabft.validateProducer(b) || sabft.applyBlock(b) != nil {
			sabft.log.Errorf("[SABFT][switchFork] applying block %v failed.", b.Id())
			errWhileSwitch = true
			// TODO: peels off this invalid branch to avoid flip-flop switch
			break
		}
	}

	// switch back
	if errWhileSwitch {
		sabft.log.Info("[SABFT][switchFork] switch back to original fork")
		sabft.popBlock(branches[0][poppedNum-1].BlockNum())

		for i := poppedNum - 1; i >= 0; i-- {
			b, err := sabft.ForkDB.FetchBlock(branches[0][i])
			if err != nil {
				panic(err)
			}
			if err := sabft.applyBlock(b); err != nil {
				panic(err)
			}
		}

		// restore the good old head of ForkDB
		sabft.ForkDB.ResetHead(branches[0][0])
		return false
	}

	// also need to reset new head in case new branch is shorter
	sabft.ForkDB.ResetHead(new)
	sabft.ForkDB.PurgeBranch()

	// handle checkpoints on new branch
	if next := sabft.cp.NextUncommitted(); next != nil {
		sabft.loopCommit(next)
	}

	if f, exist := sabft.hook["switch_fork"]; exist {
		f()
	}
	return true
}

func (sabft *SABFT) applyBlock(b common.ISignedBlock) error {
	//sabft.log.Debug("applyBlock #", b.Id().BlockNum())
	err := sabft.ctrl.PushBlock(b.(*prototype.SignedBlock), prototype.Skip_nothing)
	//sabft.log.Debugf("applyBlock #%d finished.", b.Id().BlockNum())
	return err
}

func (sabft *SABFT) popBlock(num uint64) error {
	sabft.ctrl.PopBlock(num)
	// producers fixup
	sabft.restoreProducers()

	sabft.dynasties.PopAfter(num)
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
		return nil, ErrInternal
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

	if v, exist := sabft.maliciousBlock[id]; exist {
		return v, nil
	}

	sabft.log.Errorf("[SABFT FetchBlock] block with id %v doesn't exist", id)
	return nil, ErrBlockNotExist
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

func (sabft *SABFT) FetchBlocks(from, to uint64) ([]common.ISignedBlock, error) {
	return fetchBlocks(from, to, sabft.ForkDB, &sabft.blog)
}

func (sabft *SABFT) IsCommitted(id common.BlockID) bool {
	blockNum := id.BlockNum()
	b := &prototype.SignedBlock{}
	err := sabft.blog.ReadBlock(b, int64(blockNum)-1)
	if err != nil {
		return false
	}
	return b.Id() == id
}

// return blocks in the range of (id, max(headID, id+1024))
func (sabft *SABFT) FetchBlocksSince(id common.BlockID) ([]common.ISignedBlock, error) {
	if sabft.ForkDB.Empty() {
		return nil, ErrEmptyForkDB
	}
	length := int64(sabft.ForkDB.Head().Id().BlockNum()) - int64(id.BlockNum())
	if length < 1 {
		return nil, nil
	}

	lastCommitted := sabft.ForkDB.LastCommitted()

	if id.BlockNum() >= lastCommitted.BlockNum() {
		blocks, _, err := sabft.ForkDB.FetchBlocksSince(id)
		if err != nil {
			// there probably is a new committed block during the execution of this process, just try again
			return nil, ErrForkDBChanged
		}
		return blocks, err
	}

	ret := make([]common.ISignedBlock, 0, length)
	idNum := id.BlockNum()
	start := idNum + 1
	blocksInForkDB, _, err := sabft.ForkDB.FetchBlocksSince(lastCommitted)
	if err != nil {
		// there probably is a new committed block during the execution of this process, just try again
		return nil, ErrForkDBChanged
	}
	end := lastCommitted.BlockNum()

	for start <= end {
		b := &prototype.SignedBlock{}
		if err := sabft.blog.ReadBlock(b, int64(start-1)); err != nil {
			return nil, err
		}

		if start == idNum+1 && b.Previous() != id {
			sabft.log.Errorf("blockchain doesn't have block with id %v", id)
			return nil, ErrBlockNotExist
		}

		ret = append(ret, b)
		start++
	}

	ret = append(ret, blocksInForkDB...)
	return ret, nil
}

func (sabft *SABFT) ResetProdTimer(t time.Duration) {
	if !sabft.prodTimer.Stop() {
		<-sabft.prodTimer.C
	}
	sabft.prodTimer.Reset(t)
}

func (sabft *SABFT) ResetTicker(ts time.Time) {
	sabft.Ticker = &FakeTimer{t: ts}
}

func (sabft *SABFT) fetchCheckPoint() {
	var from, to uint64
	sabft.RLock()
	if sabft.cp.HasDanglingCheckPoint() {
		// fetch missing checkpoints
		from, to = sabft.cp.MissingRange()
	}
	sabft.RUnlock()

	if to == 0 {
		if !sabft.ForkDB.Empty() {
			headNum := sabft.ForkDB.Head().Id().BlockNum()
			lcNum := sabft.ForkDB.LastCommitted().BlockNum()
			if headNum-lcNum > constants.MaxUncommittedBlockNum/10 {
				from, to = lcNum, headNum
			}
		}
	}
	if to != 0 {
		go sabft.p2p.RequestCheckpoint(from, to)
	}

	if !sabft.readyToProduce && !sabft.ForkDB.Empty() {
		headNum := sabft.ForkDB.Head().Id().BlockNum()
		lcNum := sabft.ForkDB.LastCommitted().BlockNum()
		if headNum > lcNum {
			go sabft.p2p.RequestCheckpoint(lcNum, headNum)
		}
	}
}

func (sabft *SABFT) MaybeProduceBlock() {
	defer func() {
		sabft.prodTimer.Reset(sabft.timeToNextSec())
		sabft.fetchCheckPoint()
		sabft.checkBFTRoutine()
	}()

	sabft.RLock()
	if !sabft.scheduleProduce() {
		sabft.RUnlock()
		return
	}
	sabft.RUnlock()

	if sabft.tooManyUncommittedBlocks() {
		sabft.log.Debugf("stop generating new block cause we had too many uncommitted blocks")
		return
	}

	sabft.Lock()
	b, err := sabft.generateAndApplyBlock()

	if err != nil {
		sabft.log.Error("[SABFT] generateAndApplyBlock error: ", err)
		sabft.Unlock()
		return
	}
	sabft.log.Debugf("[SABFT] generated block: <num %d> <ts %d> <%d>", b.Id().BlockNum(), b.Timestamp(), b.Id())

	if err := sabft.pushBlock(b, false); err != nil {
		sabft.log.Error("[SABFT] pushBlock push generated block failed: ", err)
	}
	sabft.Unlock()

	if f, exist := sabft.hook["generate_block"]; exist {
		f()
	}
	if sabft.mockMalicious {
		dup := &prototype.SignedBlock{}
		s, _ := b.Marshall()
		dup.Unmarshall(s)
		sig := dup.SignedHeader.BlockProducerSignature.Sig
		sig[0] = 0x01
		sig[2] = 0x01
		sig[7] = 0x01
		sabft.log.Warnf("signature of block #%d manipulated %v", dup.Id().BlockNum(), dup.Id())
		sabft.pushMaliciousBlock(dup)
		sabft.p2p.Broadcast(dup)
		return
	}
	sabft.p2p.Broadcast(b)
}

func (sabft *SABFT) databaseFixup(cfg *node.Config) error {
	dbFinal, err := sabft.ctrl.GetFinalizedNum()
	if err != nil {
		panic(err)
	}
	dbHead, err := sabft.ctrl.GetHeadBlockNum()
	if err != nil {
		return err
	}

	lastCommit := sabft.ForkDB.LastCommitted().BlockNum()
	sabft.log.Debugf("[DB fixup]: progress 1: dbHead: %v, forkdb lastCommitted %v", dbHead, lastCommit)

	if dbFinal > lastCommit {
		return fmt.Errorf("state db finalized block is ahead of blog, please remove %s and restart", cfg.ResolvePath("db"))
	}

	if dbHead > lastCommit {
		sabft.log.Debugf("[DB fixup]: popping state db: current head %d, to %v", dbHead, lastCommit)
		if err = sabft.popBlock(lastCommit+1); err != nil {
			panic(err)
		}
	} else if dbHead < lastCommit {
		sabft.log.Debugf("[DB fixup from blog] database last commit: %v, blog head: %v, forkdb head: %v",
			dbHead, lastCommit, sabft.ForkDB.Head().Id().BlockNum())
		sabft.restoreProducers()
		for i := int64(dbHead); i < int64(lastCommit); i++ {
			blk := &prototype.SignedBlock{}
			if err := sabft.blog.ReadBlock(blk, i); err != nil {
				return err
			}
			if err = sabft.ctrl.PushBlock(blk,
				prototype.Skip_block_check&
					prototype.Skip_block_signatures&
					prototype.Skip_transaction_signatures); err != nil {
				sabft.log.Errorf("[DB fixup from blog] PushBlock #%d Failed", i)
				return err
			}
			if cr, err := sabft.cp.GetNext(uint64(i)); err != nil {
				panic(err)
			} else {
				//sabft.log.Warnf("%d got record %v", i, cr)
				blockID := ConvertToBlockID(cr.ProposedData)
				blockNum := blockID.BlockNum()
				if blockID == blk.Id() {
					sabft.ctrl.Commit(blockNum)
					sabft.dynasties.Purge(blockNum)
				}
			}

			sabft.noticer.Publish(constants.NoticeLibChange, []common.ISignedBlock{blk})
		}
	}
	sabft.restoreProducers()
	sabft.restoreDynasty()

	if sabft.ForkDB.Empty() {
		return nil
	}
	dbHead, _ = sabft.ctrl.GetHeadBlockNum()
	headNum := sabft.ForkDB.Head().Id().BlockNum()
	sabft.log.Debugf("[DB fixup]: progress 2: dbHead: %v, %v", dbHead, headNum)

	if dbHead < headNum {
		blocks, err := sabft.ForkDB.FetchBlocksFromMainBranch(dbHead + 1)
		if err != nil {
			return err
		}
		sabft.log.Debugf("[DB fixup from forkdb]: start pushing uncommitted blocks, start: %v, end:%v, count: %v",
			dbHead+1, sabft.ForkDB.Head().Id().BlockNum(), len(blocks))
		for i := range blocks {
			if err = sabft.ctrl.PushBlock(blocks[i].(*prototype.SignedBlock),
				prototype.Skip_block_check&
					prototype.Skip_block_signatures&
					prototype.Skip_transaction_signatures); err != nil {
				sabft.log.Errorf("[DB fixup from forkdb] PushBlock #%d Failed", i)
				return err
			}
		}
	}

	return nil
}

func (sabft *SABFT) CheckSyncFinished() bool {
	return sabft.readyToProduce
}

func (sabft *SABFT) IsOnMainBranch(id common.BlockID) (bool, error) {
	if sabft.mockSignal {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx := r.Intn(6)
		if idx&1 == 1 {
			return false, nil
		}
	}

	blockNum := id.BlockNum()

	lastCommittedNum := sabft.ForkDB.LastCommitted().BlockNum()
	headNum := sabft.ForkDB.Head().Id().BlockNum()

	if blockNum > headNum {
		return false, nil
	}

	if blockNum > lastCommittedNum {
		blk, err := sabft.ForkDB.FetchBlockFromMainBranch(blockNum)
		if err != nil {
			return false, err
		}
		return blk.Id() == id, nil
	} else {
		b := &prototype.SignedBlock{}
		err := sabft.blog.ReadBlock(b, int64(blockNum-1))
		if err != nil {
			return false, err
		}
		return b.Id() == id, nil
	}

	return false, nil
}

func (sabft *SABFT) SetHook(key string, f func(args ...interface{})) {
	sabft.hook[key] = f
}

func (sabft *SABFT) EnableMockSignal() {
	sabft.mockSignal = true
}

func (sabft *SABFT) MockMaliciousBehaviour(b bool) {
	sabft.mockMalicious = true
}
