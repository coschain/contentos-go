package msgpack

import (
	"time"

	"github.com/coschain/contentos-go/p2p/depend/common/log"
	mt "github.com/coschain/contentos-go/p2p/message/types"
	"github.com/coschain/contentos-go/p2p/msg"
	"github.com/coschain/contentos-go/p2p/net/protocol"
	"github.com/coschain/contentos-go/prototype"
)

//Peer address package
func NewAddrs(nodeAddrs []*msg.PeerAddr) mt.Message {
	log.Trace()
	var addr msg.TransferMsg
	data := new(msg.Address)
	data.Addr = nodeAddrs
	addr.Msg = &msg.TransferMsg_Msg5{Msg5:data}

	return &addr
}

//Peer address request package
func NewAddrReq() mt.Message {
	log.Trace()
	var getAddr msg.TransferMsg
	getAddr.Msg = &msg.TransferMsg_Msg6{}
	return &getAddr
}

//block package
func NewSigBlkIdMsg(bk *prototype.SignedBlock) mt.Message {
	log.Trace()
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
	log.Trace()
	var reqmsg msg.TransferMsg

	blk := new(msg.SigBlkMsg)
	blk.SigBlk = bk

	reqmsg.Msg = &msg.TransferMsg_Msg3{Msg3:blk}

	return &reqmsg
}

//NotFound package
func NewNotFound(hash [32]byte) mt.Message {
	log.Trace()
	var reqmsg msg.TransferMsg

	notFound := new(msg.NotFound)
	notFound.Hash = hash[:]

	reqmsg.Msg = &msg.TransferMsg_Msg12{Msg12:notFound}

	return &reqmsg
}

//ping msg package
func NewPingMsg(height uint64) mt.Message {
	log.Trace()
	var reqmsg msg.TransferMsg

	ping := new(msg.Ping)
	ping.Height = height

	reqmsg.Msg = &msg.TransferMsg_Msg8{Msg8:ping}

	return &reqmsg
}

//pong msg package
func NewPongMsg(height uint64) mt.Message {
	log.Trace()
	var reqmsg msg.TransferMsg

	pong := new(msg.Pong)
	pong.Height = height

	reqmsg.Msg = &msg.TransferMsg_Msg9{Msg9:pong}

	return &reqmsg
}

//Transaction package
func NewTxn(txn *prototype.SignedTransaction) mt.Message {
	log.Trace()
	var reqmsg msg.TransferMsg

	trn := new(msg.BroadcastSigTrx)
	trn.SigTrx = txn

	reqmsg.Msg = &msg.TransferMsg_Msg1{Msg1:trn}

	return &reqmsg
}

//version ack package
func NewVerAck(isConsensus bool) mt.Message {
	log.Trace()
	var reqmsg msg.TransferMsg

	verAck := new(msg.VerAck)
	verAck.IsConsensus = isConsensus

	reqmsg.Msg = &msg.TransferMsg_Msg10{Msg10:verAck}

	return &reqmsg
}

//Version package
func NewVersion(n p2p.P2P, isCons bool, height uint64) mt.Message {
	log.Trace()
	var reqmsg msg.TransferMsg

	version := &msg.Version{
		Version:      n.GetVersion(),
		Services:     n.GetServices(),
		SyncPort:     n.GetSyncPort(),
		ConsPort:     n.GetConsPort(),
		Nonce:        n.GetID(),
		IsConsensus:  isCons,
		StartHeight:  uint64(height),
		Timestamp:    time.Now().UnixNano(),
	}
	if n.GetRelay() {
		version.Relay = 1
	} else {
		version.Relay = 0
	}

	reqmsg.Msg = &msg.TransferMsg_Msg11{Msg11:version}
	return &reqmsg
}
