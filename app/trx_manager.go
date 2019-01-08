package app

import (
	"errors"
	"fmt"
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
	"sync"
	"time"
)

var (
	mgr *trxManager
	once sync.Once
)

type trxManager struct {
	eb EventBus.Bus
	trxMap map[*prototype.SignedTransaction]chan *prototype.TransactionReceiptWithInfo
	lock *sync.RWMutex
}

//Get the singleton trxManager
func GetTrxMgrInstance() *trxManager {
	once.Do(func() {
		 mgr = &trxManager{
		 	eb:EventBus.New(),
		 	trxMap: map[*prototype.SignedTransaction]chan *prototype.TransactionReceiptWithInfo{},
		 	lock: new(sync.RWMutex),
		 }
	})
	return mgr
}

//register new notification
func (mgr *trxManager) RegisterNotification(name string, fn interface{}) error {
	if len(name) < 1 {
		return errors.New("register fail: the notification name is empty")
	}
	return mgr.eb.Subscribe(name, fn)
}

//unregister the exist notification
func (mgr *trxManager) UnRegisterNotification (name string, fn interface{}) error {
	if len(name) < 1 {
		return errors.New("unRegister fail: the notification name is empty")
	}
	return mgr.eb.Unsubscribe(name, fn)
}

//send notification of trx application result
func (mgr *trxManager) NotifyTrxApplyResult(trx *prototype.SignedTransaction, res bool,
	receipt *prototype.TransactionReceiptWithInfo)()  {
	mgr.eb.Publish(constants.NOTICE_TRX_APLLY_RESULT, trx, res, receipt)
}

func (mgr *trxManager) PushNewTrx(trx *prototype.SignedTransaction,
	callBack func(*grpcpb.BroadcastTrxResponse, error)) *prototype.TransactionReceiptWithInfo{
    if trx == nil {
		return &prototype.TransactionReceiptWithInfo{Status:prototype.StatusError,
			ErrorInfo:"the trx is nil "}
	}
	rec :=  make(chan *prototype.TransactionReceiptWithInfo)
	//if callBack != nil {
		res := make(chan *prototype.TransactionReceiptWithInfo)
		mgr.lock.Lock()
		mgr.trxMap[trx] = res
	    mgr.lock.Unlock()

    	if !mgr.eb.HasCallback(constants.NOTICE_TRX_APLLY_RESULT) {
			err := mgr.RegisterNotification(constants.NOTICE_TRX_APLLY_RESULT, mgr.handleApplyResult)
			if err != nil {
				desc := "register notification fail"
				fmt.Println(desc)
				callBack(nil,errors.New(desc))
				return &prototype.TransactionReceiptWithInfo{Status:prototype.StatusError,
					ErrorInfo:"Fail to register apply notification "}
			}
		}
		go func() {
			tOut := time.NewTimer(30*time.Second)
			for{
				select {
				    case result := <- res:
				    	//fmt.Printf("get result ,the trx is %p \n",trx)
						callBack(&grpcpb.BroadcastTrxResponse{Invoice:result},nil)
				    	mgr.DeleteTrx(trx)
				        rec <- result
						return
					case <- tOut.C:
						//handle trx apply time out
						result :=  &prototype.TransactionReceiptWithInfo{Status:prototype.StatusError,
							ErrorInfo:"Apply transaction timeout when BroadcastTrx "}
						callBack(&grpcpb.BroadcastTrxResponse{Invoice:result}, nil)
						mgr.DeleteTrx(trx)
						rec <- result
						//fmt.Printf("Apply transaction time out \n")
						return
					}
			}
		}()
	//}else {
	//	return nil
	//}
    return <-rec
}

func (mgr *trxManager) DeleteTrx(trx *prototype.SignedTransaction) error {
	exi := mgr.judgeIsTrxExist(trx)
	if exi {
		mgr.lock.Lock()
		delete(mgr.trxMap,trx)
		mgr.lock.Unlock()
		return nil
	}
	return errors.New("the trx is not exist")
}

func (mgr *trxManager) judgeIsTrxExist(trx *prototype.SignedTransaction) bool {
	if trx == nil {
		return false
	}
	mgr.lock.RLock()
	_,ok := mgr.trxMap[trx]
	mgr.lock.RUnlock()
	return ok
}

func (mgr *trxManager) handleApplyResult(trx *prototype.SignedTransaction, res bool,
	rec *prototype.TransactionReceiptWithInfo)  {
	exi := mgr.judgeIsTrxExist(trx)
	if exi {
		//fmt.Printf("receive trx %p \n",trx)
		mgr.lock.RLock()
		result := mgr.trxMap[trx]
		mgr.lock.RUnlock()
		result <- rec
	}

}