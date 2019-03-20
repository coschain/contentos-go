package prototype

import (
	"bytes"
	"fmt"
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/common/constants"
	"time"
)

func FetchTrxApplyResult(eb EventBus.Bus, timeout time.Duration, trx *SignedTransaction) *TransactionReceipt {
	if eb == nil || trx == nil {
		return &TransactionReceipt{Status: StatusError,
			ErrorInfo: "the trx or the event bus is nil"}
	}
	tId, err := trx.Id()
	if err != nil {
		return &TransactionReceipt{Status: StatusError,
			ErrorInfo: "Get id of new trx fail"}
	}
	rec := make(chan *TransactionReceipt, 10)
	done := make(chan bool, 10)

	handler := func(trx *SignedTransaction, result *TransactionReceipt) {
		if trx == nil {
			return
		}
		cId, err := trx.Id()
		if err != nil {
			desc := fmt.Sprintf("Get id of trx: %v fail", trx)
			rec <- &TransactionReceipt{Status: StatusError,
				ErrorInfo: desc}
			return
		}
		if bytes.Compare(tId.Hash, cId.Hash) == 0 {
			done <- true
			rec <- result
		}
	}
	eb.SubscribeAsync(constants.NoticeTrxApplied, handler, false)
	go func() {
		tOut := time.NewTimer(timeout)
		for {
			select {
			case <-done:
				eb.Unsubscribe(constants.NoticeTrxApplied, handler)
				return
			case <-tOut.C:
				result := &TransactionReceipt{Status: StatusError,
					ErrorInfo: "Apply transaction timeout when Broadcast Trx"}
				rec <- result
				eb.Unsubscribe(constants.NoticeTrxApplied, handler)
				return
			}
		}
	}()

	return <-rec
}
