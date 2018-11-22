package msgpack

import (
	"time"

	msgCommon "github.com/coschain/contentos-go/p2p/common"
	"github.com/coschain/contentos-go/p2p/depend/common"
	"github.com/coschain/contentos-go/p2p/depend/common/log"
	mt "github.com/coschain/contentos-go/p2p/message/types"
	"github.com/coschain/contentos-go/p2p/msg"
	"github.com/coschain/contentos-go/p2p/net/protocol"
	"github.com/coschain/contentos-go/prototype"
)

//Peer address package
func NewAddrs(nodeAddrs []msgCommon.PeerAddr) mt.Message {
	log.Trace()
	var addr mt.Addr
	addr.NodeAddrs = nodeAddrs

	return &addr
}

//Peer address request package
func NewAddrReq() mt.Message {
	log.Trace()
	var msg mt.AddrReq
	return &msg
}

//block package
func NewSigBlkIdMsg(bk *prototype.SignedBlock) mt.Message {
	log.Trace()
	var reqmsg msg.IdMsg
	var tmp []byte
	reqmsg.Msgtype = msg.IdMsg_broadcast_sigblk_id
	id := bk.Id()
	reqmsg.Value = append(reqmsg.Value, tmp)
	reqmsg.Value[0] = id.Data[:]

	return &reqmsg
}

func NewSigBlk(bk *prototype.SignedBlock) mt.Message {
	log.Trace()
	var blk msg.SigBlkMsg
	blk.SigBlk = new(prototype.SignedBlock)
	blk.SigBlk = bk

	return &blk
}

//NotFound package
func NewNotFound(hash common.Uint256) mt.Message {
	log.Trace()
	var notFound mt.NotFound
	notFound.Hash = hash

	return &notFound
}

//ping msg package
func NewPingMsg(height uint64) *mt.Ping {
	log.Trace()
	var ping mt.Ping
	ping.Height = uint64(height)

	return &ping
}

//pong msg package
func NewPongMsg(height uint64) *mt.Pong {
	log.Trace()
	var pong mt.Pong
	pong.Height = uint64(height)

	return &pong
}

//Transaction package
func NewTxn(txn *prototype.SignedTransaction) mt.Message {
	log.Trace()
	var trn msg.BroadcastSigTrx
	trn.SigTrx = txn

	return &trn
}

//version ack package
func NewVerAck(isConsensus bool) mt.Message {
	log.Trace()
	var verAck mt.VerACK
	verAck.IsConsensus = isConsensus

	return &verAck
}

//Version package
func NewVersion(n p2p.P2P, isCons bool, height uint32) mt.Message {
	log.Trace()
	var version mt.Version
	version.P = mt.VersionPayload{
		Version:      n.GetVersion(),
		Services:     n.GetServices(),
		SyncPort:     n.GetSyncPort(),
		ConsPort:     n.GetConsPort(),
		Nonce:        n.GetID(),
		IsConsensus:  isCons,
		StartHeight:  uint64(height),
		TimeStamp:    time.Now().UnixNano(),
	}

	if n.GetRelay() {
		version.P.Relay = 1
	} else {
		version.P.Relay = 0
	}
	return &version
}
