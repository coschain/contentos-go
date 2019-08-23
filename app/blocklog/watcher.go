package blocklog

import (
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"sync"
)

type WatcherCallback func(*BlockLog)

type Watcher struct {
	sync.RWMutex
	dbSvc              uint32
	callback           WatcherCallback
	currBlock          *BlockLog
	currBlockCtx       *StateChangeContext
	changeCtxs         []*StateChangeContext
	changeCtxsByBranch map[string]*StateChangeContext
}

func NewWatcher(dbSvcId uint32, callback WatcherCallback) (w *Watcher) {
	w = &Watcher{
		dbSvc:              dbSvcId,
		callback:           callback,
		changeCtxsByBranch: make(map[string]*StateChangeContext),
	}
	w.setupWatchers()
	return
}

func (w *Watcher) newStateChangeContext(branch string, trxId string, op int, cause string) (ctx *StateChangeContext) {
	if w.currBlock == nil {
		return
	}
	if oldCtx := w.changeCtxsByBranch[branch]; oldCtx != nil {
		return oldCtx
	}
	ctx = newBlockEffectContext(branch, trxId, op, cause)
	w.changeCtxs = append(w.changeCtxs, ctx)
	w.changeCtxsByBranch[branch] = ctx
	return
}

func (w *Watcher) NewStateChangeContext(branch string, trxId string, op int, cause string) (ctx *StateChangeContext) {
	w.Lock()
	defer w.Unlock()

	return w.newStateChangeContext(branch, trxId, op, cause)
}

func (w *Watcher) CurrentBlockContext() (ctx *StateChangeContext) {
	w.RLock()
	defer w.RUnlock()

	return w.currBlockCtx
}

func (w *Watcher) BeginBlock(blockNum uint64) error {
	w.Lock()
	defer w.Unlock()

	if w.currBlock != nil {
		return errors.New("cannot begin a block without ending previous one")
	}
	if len(w.changeCtxs) > 0 || len(w.changeCtxsByBranch) > 0 {
		return errors.New("found pending state change contexts")
	}
	w.currBlock = new(BlockLog)
	w.currBlockCtx = w.newStateChangeContext(iservices.DbTrunk, "", -1, "")
	return nil
}

func (w *Watcher) EndBlock(ok bool, block *prototype.SignedBlock) error {
	var blockLog *BlockLog

	w.Lock()
	if w.currBlock == nil {
		return errors.New("no block to end")
	}
	if ok {
		blockId := block.Id()
		w.currBlock.BlockId = fmt.Sprintf("%x", blockId.Data)
		w.currBlock.BlockNum = blockId.BlockNum()
		w.currBlock.BlockTime = block.GetSignedHeader().GetHeader().GetTimestamp().GetUtcSeconds()
		w.currBlock.Transactions = make([]*TransactionLog, len(block.GetTransactions()))
		trxId2idx := make(map[string]int)
		for i, trxWrapper := range block.GetTransactions() {
			trxId, _ := trxWrapper.GetSigTrx().Id()
			sId := fmt.Sprintf("%x", trxId.Hash)
			trxId2idx[sId] = i
			w.currBlock.Transactions[i] = &TransactionLog{
				TrxId:      sId,
				Receipt:    trxWrapper.GetReceipt(),
				Operations: trxWrapper.GetSigTrx().GetTrx().GetOperations(),
			}
		}
		var totalChanges InternalStateChangeSlice
		for _, ctx := range w.changeCtxs {
			totalChanges = append(totalChanges, ctx.Changes()...)
		}
		w.currBlock.Changes = make([]*StateChange, len(totalChanges))
		for i, c := range totalChanges {
			if idx, ok := trxId2idx[c.TransactionId]; ok {
				c.Transaction = idx
			}
			w.currBlock.Changes[i] = &c.StateChange
		}
		blockLog = w.currBlock
	}
	w.changeCtxs = w.changeCtxs[:0]
	w.changeCtxsByBranch = make(map[string]*StateChangeContext)
	w.currBlock = nil
	w.currBlockCtx = nil
	w.Unlock()

	if blockLog != nil && w.callback != nil {
		w.callback(blockLog)
	}
	return nil
}

func (w *Watcher) recordChange(branch, what string, change interface{}) {
	w.RLock()
	defer w.RUnlock()
	if ctx := w.changeCtxsByBranch[branch]; ctx != nil {
		ctx.AddChange(what, change)
	}
}

func (w *Watcher) setupWatchers() {
	for _, e := range sInterestedChanges {
		table.AddTableRecordFieldWatcher(w.dbSvc, e.record, e.primary, e.field, w.makeWatcherFunc(e.what, e.maker))
	}
}

func (w *Watcher) makeWatcherFunc(what string, changeMaker ChangeDataMaker) func(branch string, event int, key, before, after interface{}) {
	return func(branch string, event int, key, before, after interface{}) {
		w.recordChange(branch, what, changeMaker(key, before, after))
	}
}
