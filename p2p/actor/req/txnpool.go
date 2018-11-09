package req

import (
	"time"

	"github.com/ontio/ontology-eventbus/actor"
	//"github.com/coschain/contentos-go/p2p/depend/common"
	//"github.com/coschain/contentos-go/p2p/depend/common/log"
	//"github.com/coschain/contentos-go/p2p/depend/core/types"
	//"github.com/coschain/contentos-go/p2p/depend/errors"
	p2pcommon "github.com/coschain/contentos-go/p2p/common"
)

const txnPoolReqTimeout = p2pcommon.ACTOR_TIMEOUT * time.Second

var txnPoolPid *actor.PID

func SetTxnPoolPid(txnPid *actor.PID) {
	txnPoolPid = txnPid
}

//add txn to txnpool
//func AddTransaction(transaction *types.Transaction) {
//	if txnPoolPid == nil {
//		log.Error("[p2p]net_server AddTransaction(): txnpool pid is nil")
//		return
//	}
//	txReq := &tc.TxReq{
//		Tx:         transaction,
//		Sender:     tc.NetSender,
//		TxResultCh: nil,
//	}
//	txnPoolPid.Tell(txReq)
//}

//get txn according to hash
//func GetTransaction(hash common.Uint256) (*types.Transaction, error) {
//	if txnPoolPid == nil {
//		log.Warn("[p2p]net_server tx pool pid is nil")
//		return nil, errors.NewErr("[p2p]net_server tx pool pid is nil")
//	}
//	future := txnPoolPid.RequestFuture(&tc.GetTxnReq{Hash: hash}, txnPoolReqTimeout)
//	result, err := future.Result()
//	if err != nil {
//		log.Warnf("[p2p]net_server GetTransaction error: %v\n", err)
//		return nil, err
//	}
//	return result.(tc.GetTxnRsp).Txn, nil
//}
