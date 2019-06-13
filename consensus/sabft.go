package consensus

import (
	"io/ioutil"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
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
	bftStarted    uint32
	commitCh      chan message.Commit
	cp            *BFTCheckPoint

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
		dynasties:  NewDynasties(),
		prodTimer:  time.NewTimer(1 * time.Millisecond),
		trxCh:      make(chan func()),
		pendingCh:  make(chan func()),
		blkCh:      make(chan common.ISignedBlock),
		ctx:        ctx,
		stopCh:     make(chan struct{}),
		extLog:     lg,
		log:        lg.WithField("sabft", "on"),
		bftStarted: 0,
		commitCh:   make(chan message.Commit),
		Ticker:     &Timer{},
	}

	ret.SetBootstrap(ctx.Config().Consensus.BootStrap)
	ret.Name = ctx.Config().Consensus.LocalBpName

	ret.log.Info("[SABFT bootstrap] ", ctx.Config().Consensus.BootStrap)

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

	// When a produce round complete, it adds new producers,
	// remove unqualified producers and shuffle the block-producing order
	prods, pubKeys := sabft.ctrl.GetWitnessTopN(constants.MaxWitnessCount)
	sabft.log.Warn("GetWitnessTopN ", prods)

	var seed uint64
	if head != nil {
		seed = head.Timestamp() << 32
	}
	sabft.updateProducers(seed, prods, pubKeys)

	newDyn := sabft.makeDynasty(blockNum, prods, pubKeys, sabft.localPrivKey)
	sabft.addDynasty(newDyn)
	if atomic.LoadUint32(&sabft.bftStarted) == 0 && sabft.bft != nil{
		sabft.checkBFTRoutine()
	}
	return true, prods
}

func (sabft *SABFT) addDynasty(d *Dynasty) {
	for !sabft.dynasties.Empty() && sabft.dynasties.Front().GetValidatorNum() < 3 {
		sabft.log.Info("remove inferior dynasty: ", sabft.dynasties.Front().Seq)
		sabft.dynasties.PopFront()
	}
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

func (sabft *SABFT) checkBFTRoutine() {
	if sabft.readyToProduce && sabft.dynasties.Front().GetValidatorNum() >= 3 &&
		sabft.isValidatorName(sabft.Name) {
		if atomic.LoadUint32(&sabft.bftStarted) == 0 {
			sabft.bft.Start()
			sabft.log.Info("[SABFT] gobft started...")
			atomic.StoreUint32(&sabft.bftStarted, 1)
		}
	} else {
		if atomic.LoadUint32(&sabft.bftStarted) == 1 {
			sabft.bft.Stop()
			sabft.log.Info("[SABFT] gobft stopped...")
			atomic.StoreUint32(&sabft.bftStarted, 0)
		}
	}
}

func (sabft *SABFT) restoreProducers() {
	prods, _ := sabft.ctrl.GetShuffledWitness()
	sabft.producers = sabft.makeProducers(prods)
	sabft.log.Info("[SABFT] active producers: ", prods)
}

func (sabft *SABFT) updateProducers(seed uint64, prods []string, pubKeys []*prototype.PublicKeyType) int {
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
	sabft.ctrl.SetShuffledWitness(prods, pubKeys)

	return prodNum
}

func (sabft *SABFT) ActiveProducers() []string {
	sabft.RLock()
	defer sabft.RUnlock()

	// TODO
	return nil
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
	sabft.ctrl.SetShuffle(func(block common.ISignedBlock) (bool, []string) {
		return sabft.shuffle(block)
	})

	// reload ForkDB
	snapshotPath := cfg.ResolvePath("forkdb_snapshot")
	// TODO: fuck!! this is fugly
	var avatar []common.ISignedBlock
	for i := 0; i < 2001; i++ {
		// TODO: if the bft process falls behind too much, the number
		// TODO: of the avatar might not be sufficient

		// deep copy hell
		avatar = append(avatar, &prototype.SignedBlock{})
	}
	sabft.ForkDB.LoadSnapshot(avatar, snapshotPath, &sabft.blog)

	sabft.cp = NewBFTCheckPoint(cfg.ResolvePath("checkpoint"), sabft)

	sabft.log.Info("[SABFT] starting...")
	if sabft.bootstrap && sabft.ForkDB.Empty() && sabft.blog.Empty() {
		sabft.log.Info("[SABFT] bootstrapping...")
	}
	if !sabft.ForkDB.Empty() && !sabft.blog.Empty() {
		lc, err := sabft.cp.GetNext(sabft.ForkDB.LastCommitted().BlockNum() - 1)
		if err != nil {
			sabft.log.Error(err)
		} else {
			sabft.lastCommitted.Store(lc)
		}
	}

	sabft.restoreProducers()

	err = sabft.handleBlockSync()
	if err != nil {
		return err
	}

	if sabft.dynasties.Empty() {
		sabft.restoreDynasty()
	}

	sabft.appState = &message.AppState{
		// TODO: store last height
		LastHeight:       0,
		LastProposedData: sabft.ForkDB.LastCommitted().Data,
	}
	sabft.bft = gobft.NewCore(sabft, sabft.dynasties.Front().priv)
	//pv := newPrivValidator(sabft, sabft.localPrivKey, sabft.Name)
	//sabft.bft = gobft.NewCore(sabft, pv)
	sabft.bft.SetLogger(sabft.extLog)
	// start block generation process
	go sabft.start()

	return nil
}

func (sabft *SABFT) restoreDynasty() {
	if sabft.ForkDB.Empty() {
		// new chain, no blocks
		prods, pubKeys := sabft.ctrl.GetWitnessTopN(constants.MaxWitnessCount)
		dyn := sabft.makeDynasty(0, prods, pubKeys, sabft.localPrivKey)
		sabft.addDynasty(dyn)
	} else {
		// pop all uncommitted blocks and fix first dynasty
		lcNum := sabft.ForkDB.LastCommitted().BlockNum()
		length := sabft.ForkDB.Head().Id().BlockNum() - lcNum
		cache := make([]common.ISignedBlock, length)
		b := sabft.ForkDB.Head()
		var err error
		for i := int(length) - 1; i >= 0; i-- {
			cache[i] = b
			if i == 0 {
				break
			}
			prev := b.Previous()
			b, err = sabft.ForkDB.FetchBlock(prev)
			if err != nil {
				panic(err)
			}
		}
		sabft.popBlock(lcNum + 1)

		prods, pubKeys := sabft.ctrl.GetShuffledWitness()
		dyn := sabft.makeDynasty(lcNum, prods, pubKeys, sabft.localPrivKey)
		sabft.addDynasty(dyn)
		for i := range cache {
			if err = sabft.applyBlock(cache[i]); err != nil {
				panic(err)
			}
		}
	}
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
	sabft.ForkDB = forkdb.NewDB()
	if popNum > 1 {
		sabft.ForkDB.PushBlock(lastCommittedBlock)
		sabft.ForkDB.Commit(lastCommittedID)
	}

	sabft.log.Infof("[SABFT][checkpoint] revert to last committed block %d.", popNum-1)
}

func (sabft *SABFT) start() {
	sabft.wg.Add(1)
	defer sabft.wg.Done()

	sabft.log.Info("[SABFT] DPoS routine started")
	for {
		select {
		case <-sabft.stopCh:
			sabft.log.Debug("[SABFT] routine stopped.")
			return
		case b := <-sabft.blkCh:
			if sabft.readyToProduce && sabft.tooManyUncommittedBlocks() &&
				b.Id().BlockNum() > sabft.ForkDB.Head().Id().BlockNum() {
				sabft.log.Debugf("dropping new block %v cause we had too many uncommitted blocks", b.Id())
				return
			}
			sabft.Lock()
			err := sabft.pushBlock(b, true)
			sabft.Unlock()
			if err != nil {
				sabft.log.Error("[SABFT] pushBlock failed: ", err)
				continue
			}

			sabft.Lock()
			commit := sabft.cp.NextUncommitted()
			if commit != nil && b.Id() == ExtractBlockID(commit) {
				success := sabft.cp.Validate(commit)
				if !success {
					if !sabft.readyToProduce {
						sabft.revertToLastCheckPoint()
					}
					// remove this invalid checkpoint
					sabft.cp.RemoveNextUncommitted()
					// TODO: fetch next checkpoint from a different peer
				} else {
					// if we're a validator and the gobft falls behind, pass the commit to gobft and let it catchup
					if sabft.gobftCatchUp(commit) {
						sabft.Unlock()
						continue
					}

					// if it suits the following situation:
					// 1. we're not a validator
					// 2. gobft is far ahead due to committing missing blocks
					if err = sabft.commit(commit); err == nil {
						sabft.dynasties.Purge(ExtractBlockID(commit).BlockNum())
					}
				}
			}
			sabft.Unlock()

		case trxFn := <-sabft.trxCh:
			sabft.Lock()
			trxFn()
			sabft.Unlock()
			continue
		case commit := <-sabft.commitCh:
			// TODO: reduce critical area guarded by lock
			sabft.Lock()
			sabft.handleCommitRecords(&commit)
			sabft.Unlock()
		case pendingFn := <-sabft.pendingCh:
			pendingFn()
			continue
		case <-sabft.prodTimer.C:
			sabft.MaybeProduceBlock()
		}
	}
}

func (sabft *SABFT) Stop() error {
	sabft.log.Info("SABFT consensus stopped.")

	// stop bft process
	if atomic.LoadUint32(&sabft.bftStarted) == 1 {
		sabft.bft.Stop()
		sabft.log.Info("[SABFT] gobft stopped...")
	}

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

func (sabft *SABFT) Push(msg interface{}) {
	switch msg := msg.(type) {
	case *message.Vote:
		if atomic.LoadUint32(&sabft.bftStarted) == 1 {
			sabft.bft.RecvMsg(msg)
		}
	case *message.Commit:
		go func() {
			sabft.commitCh <- *msg
		}()
	default:
	}
}

func (sabft *SABFT) verifyCommitSig(records *message.Commit) bool {
	for i := range records.Precommits {
		//val := sabft.getValidator(records.Precommits[i].Address)
		val := sabft.dynasties.Front().GetValidatorByPubKey(records.Precommits[i].Address)
		if val == nil {
			sabft.log.Errorf("[SABFT] error while checking precommits: %s is not a validator", records.Precommits[i].Address)
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
	//sabft.log.Warn("handleCommitRecords: ", records.ProposedData, records.Address)
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

	checkPoint := records
	for checkPoint != nil {
		newID = ExtractBlockID(checkPoint)
		if !sabft.cp.IsNextCheckPoint(checkPoint) {
			return
		}
		sabft.log.Debug("reach checkpoint at ", checkPoint.ProposedData)

		// if we're a validator, pass it to gobft so that it can catch up
		if sabft.gobftCatchUp(checkPoint) {
			return
		}

		if !sabft.cp.Validate(checkPoint) {
			sabft.log.Error("validation on checkpoint failed")
			return
		}
		if _, err := sabft.ForkDB.FetchBlock(newID); err == nil {
			if err = sabft.commit(checkPoint); err == nil {
				sabft.dynasties.Purge(newID.BlockNum())
				checkPoint = sabft.cp.NextUncommitted()
				if checkPoint != nil {
					sabft.log.Debug("loop checkpoint at ", checkPoint.ProposedData)
				}
				continue
			}
		}
		break
	}
}

func (sabft *SABFT) gobftCatchUp(commit *message.Commit) bool {
	if sabft.isValidatorName(sabft.Name) && atomic.LoadUint32(&sabft.bftStarted) == 1 &&
		sabft.appState.LastHeight+1 == commit.Height() {
		sabft.log.Warn("pass commits to gobft ", commit.ProposedData)
		sabft.bft.RecvMsg(commit)
		return true
	}
	return false
}

func (sabft *SABFT) validateProducer(b common.ISignedBlock) bool {
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

func (sabft *SABFT) pushBlock(b common.ISignedBlock, applyStateDB bool) error {
	sabft.log.Debug("[SABFT] start pushBlock #", b.Id().BlockNum())
	// TODO: check signee & merkle

	//if b.Timestamp() < sabft.getSlotTime(1) {
	//	// sabft.log.Debugf("the timestamp of the new block is less than that of the head block.")
	//}

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
		sabft.log.Debugf("[SABFT][pushBlock]possibly detached block. prev: got %v, want %v", b.Id(), head.Id())
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
	case forkdb.RTOnFork:
		if newHead != head && newHead.Previous() != head.Id() {
			sabft.log.Debug("[SABFT] start to switch fork.")
			switchSuccess := sabft.switchFork(head.Id(), newHead.Id())
			if !switchSuccess {
				sabft.log.Error("[SABFT] there's an error while switching to new branch. new head", newHead.Id())
			}
		}
		return nil
	case forkdb.RTInvalid:
		return ErrInvalidBlock
	case forkdb.RTDuplicated:
		return ErrDupBlock
	case forkdb.RTSuccess:
	default:
		return ErrInternal
	}

	if applyStateDB {
		if err := sabft.applyBlock(b); err != nil {
			// the block is illegal
			sabft.ForkDB.MarkAsIllegal(b.Id())
			sabft.ForkDB.Pop()
			return err
		}
	}
	sabft.log.Debug("[SABFT] pushBlock FINISHED #", b.Id().BlockNum(), " id ", b.Id())
	return nil
}

func (sabft *SABFT) GetLastBFTCommit() interface{} {
	lastCommitted := sabft.lastCommitted.Load()

	if lastCommitted == nil {
		return nil
	}
	return lastCommitted.(*message.Commit)
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
	err := sabft.cp.Add(commitRecords)
	if err == ErrCheckPointOutOfRange || err == ErrInvalidCheckPoint {
		sabft.log.Error(err)
		return err
	}
	err = sabft.commit(commitRecords)
	if err == nil {
		sabft.dynasties.Purge(ExtractBlockID(commitRecords).BlockNum())
		sabft.checkBFTRoutine()
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

func (sabft *SABFT) commit(commitRecords *message.Commit) error {
	defer func() {
		sabft.appState.LastHeight = commitRecords.FirstPrecommit().Height
		sabft.appState.LastProposedData = commitRecords.ProposedData
	}()

	blockID := common.BlockID{
		Data: commitRecords.ProposedData,
	}

	sabft.log.Infof("[SABFT] start to commit block #%d %v %d", blockID.BlockNum(), blockID, commitRecords.FirstPrecommit().Height)
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

	blkMain, err := sabft.ForkDB.FetchBlockFromMainBranch(blockID.BlockNum())
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
		// also need to reset new head
		sabft.ForkDB.ResetHead(blockID)
		sabft.ForkDB.PurgeBranch()
	}

	blks, _, err := sabft.ForkDB.FetchBlocksSince(sabft.ForkDB.LastCommitted())
	if err != nil {
		sabft.log.Errorf("[SABFT] internal error when committing %v, err: %v", blockID, err)
		return ErrInternal
	}
	for i := range blks {
		if err = sabft.blog.Append(blks[i]); err != nil {
			sabft.log.Errorf("[SABFT] internal error when committing %v, err: %v", blockID, err)
			return ErrInternal
		}
		if blks[i] == blk {
			sabft.log.Debugf("[SABFT] committed from block #%d to #%d", blks[0].Id().BlockNum(), blk.Id().BlockNum())
			break
		}
	}

	sabft.ctrl.Commit(blockID.BlockNum())
	sabft.ForkDB.Commit(blockID)
	sabft.lastCommitted.Store(commitRecords)
	sabft.cp.Flush(blockID)

	sabft.log.Debug("[SABFT] committed block #", blockID)
	return nil
}

// GetValidator returns the validator correspond to the PubKey
func (sabft *SABFT) GetValidator(key message.PubKey) custom.IPubValidator {
	sabft.RLock()
	defer sabft.RUnlock()

	return sabft.getValidator(key)
}

func (sabft *SABFT) getValidator(key message.PubKey) custom.IPubValidator {
	valset := sabft.dynasties.Front().validators
	for i := range valset {
		if valset[i].bftPubKey == key {
			return valset[i]
		}
	}
	return nil
}

// IsValidator returns true if key is a validator
func (sabft *SABFT) IsValidator(key message.PubKey) bool {
	sabft.RLock()
	defer sabft.RUnlock()

	return sabft.isValidator(key)
}

func (sabft *SABFT) isValidator(key message.PubKey) bool {
	valset := sabft.dynasties.Front().validators
	for i := range valset {
		if valset[i].bftPubKey == key {
			return true
		}
	}
	return false
}

func (sabft *SABFT) isValidatorName(name string) bool {
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

	return int64(sabft.dynasties.Front().GetValidatorNum())
}

func (sabft *SABFT) GetCurrentProposer(round int) message.PubKey {
	sabft.RLock()
	defer sabft.RUnlock()

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

	return sabft.ForkDB.Head().Id().Data
}

// ValidateProposed validates the proposed data
func (sabft *SABFT) ValidateProposal(data message.ProposedData) bool {
	blockID := common.BlockID{
		Data: data,
	}
	blockNum := blockID.BlockNum()

	sabft.RLock()
	defer sabft.RUnlock()

	if b, err := sabft.ForkDB.FetchBlockFromMainBranch(blockNum); err != nil {
		return false
	} else if b.Id() != blockID {
		return false
	}

	return true
}

func (sabft *SABFT) GetAppState() *message.AppState {
	//sabft.RLock()
	//defer sabft.RUnlock()

	return sabft.appState
}

func (sabft *SABFT) BroadCast(msg message.ConsensusMessage) error {
	sabft.p2p.Broadcast(msg)
	return nil
}

func (sabft *SABFT) GetValidatorNum() int {
	sabft.RLock()
	defer sabft.RUnlock()

	return sabft.dynasties.Front().GetValidatorNum()
}

/********* end gobft ICommittee ***********/

func (sabft *SABFT) switchFork(old, new common.BlockID) bool {
	// TODO: validate block producer
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

func (sabft *SABFT) MaybeProduceBlock() {
	defer sabft.prodTimer.Reset(sabft.timeToNextSec())
	defer func() {

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

	sabft.log.Debugf("[SABFT] generated block: <num %d> <ts %d>", b.Id().BlockNum(), b.Timestamp())
	if err := sabft.pushBlock(b, false); err != nil {
		sabft.log.Error("[SABFT] pushBlock push generated block failed: ", err)
	}
	sabft.Unlock()

	go func() {
		time.Sleep(time.Duration(1 * time.Duration(rand.Int()%15) * time.Second / 10))
		sabft.log.Debugf("[SABFT] call p2p to broadcast sigblk %d", b.Id().BlockNum())
		sabft.p2p.Broadcast(b)
	}()
	//sabft.p2p.Broadcast(b)
}

func (sabft *SABFT) handleBlockSync() error {
	if sabft.ForkDB.Head() == nil {
		//Not need to sync
		return nil
	}
	var err error = nil
	lastCommit := sabft.ForkDB.LastCommitted().BlockNum()
	//Fetch the commit block num in db
	dbLastPushed, err := sabft.ctrl.GetLastPushedBlockNum()

	sabft.log.Debugf("[sync pushed]: progress 1: dbLastPushed: %v, %v, %v",
		dbLastPushed, lastCommit, err)

	if err != nil {
		return err
	}
	//Fetch the commit block numbers saved in block log
	commitNum := sabft.blog.Size()
	//1.sync commit blocks
	if dbLastPushed < lastCommit && commitNum > 0 && commitNum >= int64(lastCommit) {
		sabft.log.Debugf("[Reload commit] start sync lost commit blocks from block log,db commit num is: "+
			"%v,end:%v,real commit num is %v", dbLastPushed, sabft.ForkDB.Head().Id().BlockNum(), lastCommit)
		for i := int64(dbLastPushed); i < int64(lastCommit); i++ {
			blk := &prototype.SignedBlock{}
			if err := sabft.blog.ReadBlock(blk, i); err != nil {
				return err
			}
			err = sabft.ctrl.SyncCommittedBlockToDB(blk)

			if err != nil {
				sabft.log.Debugf("[Reload commit] SyncCommittedBlockToDB Failed: "+
					"%v", i)
				return err
			}
		}
	}

	dbLastPushed, err = sabft.ctrl.GetLastPushedBlockNum()
	latestNumber := sabft.ForkDB.Head().Id().BlockNum()

	sabft.log.Debugf("[sync pushed]: progress 2: dbLastPushed: %v, %v, %v",
		dbLastPushed, latestNumber, err)

	if dbLastPushed < latestNumber {
		pSli, err := sabft.FetchBlocks(dbLastPushed+1, latestNumber+1)
		if err != nil {
			return err
		}
		if len(pSli) > 0 {
			sabft.log.Debugf("[sync pushed]: start sync uncommitted blocks,start: %v,end:%v, count: %v",
				dbLastPushed+1, sabft.ForkDB.Head().Id().BlockNum(), len(pSli))
			err = sabft.ctrl.SyncPushedBlocksToDB(pSli)

			if err != nil {
				return err
			}
		}
		return nil

	} else if dbLastPushed > latestNumber {

		sabft.log.Infof("[Revert commit] start revert invalid commit to statedb: "+
			"%v,end:%v,real commit num is %v", dbLastPushed, sabft.ForkDB.Head().Id().BlockNum(), latestNumber)

		sabft.popBlock(latestNumber + 1)
		
	}

	return nil
}

func (sabft *SABFT) CheckSyncFinished() bool {
	return sabft.readyToProduce
}

func (sabft *SABFT) IsOnMainBranch(id common.BlockID) (bool, error) {
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
