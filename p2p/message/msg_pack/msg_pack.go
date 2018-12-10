package msgpack

import (
	"time"

	msgCommon "github.com/coschain/contentos-go/p2p/common"
	"github.com/coschain/contentos-go/p2p/depend/common"
	mt "github.com/coschain/contentos-go/p2p/message/types"
	"github.com/coschain/contentos-go/p2p/msg"
	"github.com/coschain/contentos-go/p2p/net/protocol"
	"github.com/coschain/contentos-go/prototype"
)

//Peer address package
func NewAddrs(nodeAddrs []msgCommon.PeerAddr) mt.Message {
	var addr mt.Addr
	addr.NodeAddrs = nodeAddrs

	return &addr
}

//Peer address request package
func NewAddrReq() mt.Message {
	var msg mt.AddrReq
	return &msg
}

//block package
func NewSigBlkIdMsg(bk *prototype.SignedBlock) mt.Message {
	var reqmsg msg.TransferMsg
	data := new(msg.IdMsg)
	var tmp []byte
	data.Msgtype = msg.IdMsg_broadcast_sigblk_id
	id := bk.Id()
	data.Value = append(data.Value, tmp)
	data.Value[0] = id.Data[:]

	reqmsg.Msg = &msg.TransferMsg_Msg2{Msg2:data}
	return &reqmsg
}

func NewSigBlk(bk *prototype.SignedBlock) mt.Message {
	var reqmsg msg.TransferMsg
	data := new(msg.SigBlkMsg)
	data.SigBlk = bk

	reqmsg.Msg = &msg.TransferMsg_Msg3{Msg3:data}
	return &reqmsg
}

//NotFound package
func NewNotFound(hash common.Uint256) mt.Message {
	var notFound mt.NotFound
	notFound.Hash = hash

	return &notFound
}

//ping msg package
func NewPingMsg(height uint64) *mt.Ping {
	var ping mt.Ping
	ping.Height = uint64(height)

	return &ping
}

//pong msg package
func NewPongMsg(height uint64) *mt.Pong {
	var pong mt.Pong
	pong.Height = uint64(height)

	return &pong
}

//Transaction package
func NewTxn(txn *prototype.SignedTransaction) mt.Message {
	var reqmsg msg.TransferMsg
	data := new(msg.BroadcastSigTrx)
	data.SigTrx = txn

	reqmsg.Msg = &msg.TransferMsg_Msg1{Msg1:data}
	return &reqmsg
}

//version ack package
func NewVerAck(isConsensus bool) mt.Message {
	var verAck mt.VerACK
	verAck.IsConsensus = isConsensus

	return &verAck
}

//Version package
func NewVersion(n p2p.P2P, isCons bool, height uint64) mt.Message {
	var version mt.Version
	version.P = mt.VersionPayload{
		Version:     n.GetVersion(),
		Services:    n.GetServices(),
		SyncPort:    n.GetSyncPort(),
		ConsPort:    n.GetConsPort(),
		Nonce:       n.GetID(),
		IsConsensus: isCons,
		StartHeight: uint64(height),
		TimeStamp:   time.Now().UnixNano(),
	}

	if n.GetRelay() {
		version.P.Relay = 1
	} else {
		version.P.Relay = 0
	}
	return &version
}
