package prototype

import (
	"bytes"
	"fmt"
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/common/constants"
	"sync"
	"time"
)

func FetchTrxApplyResult(eb EventBus.Bus, timeout time.Duration, trx *SignedTransaction) *TransactionReceiptWithInfo {
	if eb == nil || trx == nil {
     	return  &TransactionReceiptWithInfo{Status:StatusError,
			 ErrorInfo:"the trx or the event bus is nil"}
	 }
     tId,err := trx.Id()
     if err != nil {
     	return &TransactionReceiptWithInfo{Status:StatusError,
			ErrorInfo:"Get id of new trx fail"}
	 }
	rec := make(chan *TransactionReceiptWithInfo,10)
    done := make(chan bool,10)

	handler := func(trx *SignedTransaction, result *TransactionReceiptWithInfo, blockNum uint64) {
		  if trx == nil {
			  return
		  }
     	  cId,err := trx.Id()
     	  if err != nil {
     	  	 desc := fmt.Sprintf("Get id of trx: %v fail",trx)
     	  	  rec <-  &TransactionReceiptWithInfo{Status:StatusError,
				  ErrorInfo:desc}
			  return
		  }
     	  if bytes.Compare(tId.Hash, cId.Hash) == 0 {
     	  	 done <- true
     	  	 rec <- result
		  }
	 }
	eb.SubscribeAsync(constants.NoticeTrxApplied,handler, false)
	go func() {
	 	tOut := time.NewTimer(timeout)
	 	for {
			select {
	 		    case <- done:
					eb.Unsubscribe(constants.NoticeTrxApplied, handler)
					return
			    case <- tOut.C:
					result :=  &TransactionReceiptWithInfo{Status:StatusError,
						ErrorInfo:"Apply transaction timeout when Broadcast Trx"}
					rec <- result
					eb.Unsubscribe(constants.NoticeTrxApplied, handler)
					return
			}
		}
	 }()

     return <- rec
}

func FetchTrxFinalResult(eb EventBus.Bus, timeout time.Duration, trx *SignedTransaction) (receipt *TransactionReceiptWithInfo, finality bool) {
	if eb == nil || trx == nil {
		return  &TransactionReceiptWithInfo{Status:StatusError, ErrorInfo:"the trx or the event bus is nil"}, true
	}
	const (
		sInit int32 = iota
		sWaitingBlockApplied
		sWaitingBlockCommitted
		sDone
	)
	var (
		sig = trx.Signature.Sig
		block uint64
		s = sInit
		done = make(chan struct{})
		lock sync.Mutex
	)

	safeDone := func() {
		defer func() { recover() }()
		s = sDone
		close(done)
	}
	trxAppliedHandler := func(trx *SignedTransaction, result *TransactionReceiptWithInfo, blockNum uint64) {
		lock.Lock()
		if s == sInit && trx != nil && trx.Signature != nil && bytes.Compare(sig, trx.Signature.Sig) == 0 {
			receipt, block = result, blockNum
			if receipt == nil {
				receipt = &TransactionReceiptWithInfo{Status:StatusError, ErrorInfo:"unexpected nil receipt"}
			}
			if receipt.Status == StatusError {
				safeDone()
			} else {
				s = sWaitingBlockApplied
			}
		}
		lock.Unlock()
	}
	blockAppliedHandler := func(b *SignedBlock) {
		lock.Lock()
		if s == sWaitingBlockApplied && b.Id().BlockNum() == block {
			s = sWaitingBlockCommitted
		}
		lock.Unlock()
	}
	blockApplyFailedHandler := func(b *SignedBlock) {
		lock.Lock()
		if s == sWaitingBlockApplied && b.Id().BlockNum() == block {
			s = sInit
		}
		lock.Unlock()
	}
	blockGenerationFailedHandler := func(blockNum uint64) {
		lock.Lock()
		if s == sWaitingBlockApplied && blockNum == block {
			s = sInit
		}
		lock.Unlock()
	}
	blockRevertedHandler := func(blockNum uint64) {
		lock.Lock()
		if s == sWaitingBlockCommitted && blockNum <= block {
			s = sInit
		}
		lock.Unlock()
	}
	blockCommittedHandler := func(blockNum uint64) {
		lock.Lock()
		if s == sWaitingBlockCommitted && blockNum >= block {
			safeDone()
		}
		lock.Unlock()
	}
	_ = eb.Subscribe(constants.NoticeTrxApplied, trxAppliedHandler)
	_ = eb.Subscribe(constants.NoticeBlockApplied, blockAppliedHandler)
	_ = eb.Subscribe(constants.NoticeBlockApplyFailed, blockApplyFailedHandler)
	_ = eb.Subscribe(constants.NoticeBlockGenerationFailed, blockGenerationFailedHandler)
	_ = eb.Subscribe(constants.NoticeBlockRevert, blockRevertedHandler)
	_ = eb.Subscribe(constants.NoticeBlockCommit, blockCommittedHandler)
	t := time.NewTimer(timeout)
	select {
		case <-done:
			t.Stop()
		case <-t.C:
			break
	}
	_ = eb.Unsubscribe(constants.NoticeTrxApplied, trxAppliedHandler)
	_ = eb.Unsubscribe(constants.NoticeBlockApplied, blockAppliedHandler)
	_ = eb.Unsubscribe(constants.NoticeBlockApplyFailed, blockApplyFailedHandler)
	_ = eb.Unsubscribe(constants.NoticeBlockGenerationFailed, blockGenerationFailedHandler)
	_ = eb.Unsubscribe(constants.NoticeBlockRevert, blockRevertedHandler)
	_ = eb.Unsubscribe(constants.NoticeBlockCommit, blockCommittedHandler)

	if receipt == nil {
		return &TransactionReceiptWithInfo{Status:StatusError, ErrorInfo:fmt.Sprintf("no receipt in %v", timeout)}, false
	}
	return receipt, s == sDone
}
