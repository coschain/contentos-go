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

func (w *Watcher) getOrCreateStateChangeContext(branch string, trxId string, op int, cause string) (ctx *StateChangeContext) {
	if w.currBlock == nil {
		return
	}
	if oldCtx := w.changeCtxsByBranch[branch]; oldCtx != nil {
		oldCtx.SetTrxAndOperation(trxId, op)
		oldCtx.SetCause(cause)
		return oldCtx
	}
	ctx = newBlockEffectContext(branch, trxId, op, cause)
	w.changeCtxs = append(w.changeCtxs, ctx)
	w.changeCtxsByBranch[branch] = ctx
	return
}

func (w *Watcher) GetOrCreateStateChangeContext(branch string, trxId string, op int, cause string) (ctx *StateChangeContext) {
	w.Lock()
	defer w.Unlock()

	return w.getOrCreateStateChangeContext(branch, trxId, op, cause)
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
	w.currBlockCtx = w.getOrCreateStateChangeContext(iservices.DbTrunk, "", -1, "")
	return nil
}

func (w *Watcher) EndBlock(ok bool, block *prototype.SignedBlock) error {
	var blockLog *BlockLog

	w.Lock()
	if w.currBlock == nil {
		return errors.New("no block to end")
	}
	if ok {
		w.makeLog(block)
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

func (w *Watcher) makeLog(block *prototype.SignedBlock) {
	blockId := block.Id()
	w.currBlock.BlockId = fmt.Sprintf("%x", blockId.Data)
	w.currBlock.BlockNum = blockId.BlockNum()
	w.currBlock.BlockTime = block.GetSignedHeader().GetHeader().GetTimestamp().GetUtcSeconds()
	trxId2idx := make(map[string]int)

	trxs := block.GetTransactions()
	w.currBlock.Transactions = make([]*TransactionLog, len(block.GetTransactions()))
	for i, trxWrapper := range trxs {
		ops := trxWrapper.GetSigTrx().GetTrx().GetOperations()
		opLogs := make([]*OperationLog, len(ops))
		for i, op := range ops {
			opLogs[i] = &OperationLog{
				Op: &OperationData{
					Type: prototype.GetGenericOperationName(op),
					Data: prototype.GetBaseOperation(op),
				},
				Changes: make([]*StateChange, 0, 32),
			}
		}
		trxId, _ := trxWrapper.GetSigTrx().Id()
		sId := fmt.Sprintf("%x", trxId.Hash)
		w.currBlock.Transactions[i] = &TransactionLog{
			TrxId:      sId,
			Receipt:    trxWrapper.GetReceipt(),
			Operations: opLogs,
		}
		trxId2idx[sId] = i
	}

	w.currBlock.Changes = make([]*StateChange, 0, 128)
	for _, ctx := range w.changeCtxs {
		changes := ctx.Changes()
		for _, change := range changes {
			if change.Operation >= 0 && len(change.TransactionId) >= 32 {
				if idx, ok := trxId2idx[change.TransactionId]; ok {
					change.Transaction = idx
				}
			}
			if change.Transaction >= 0 && change.Operation >= 0 {
				opLog := w.currBlock.Transactions[change.Transaction].Operations[change.Operation]
				opLog.Changes = append(opLog.Changes, &change.StateChange)
			} else {
				w.currBlock.Changes = append(w.currBlock.Changes, &change.StateChange)
			}
		}
	}
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
