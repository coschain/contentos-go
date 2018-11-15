package utils

import (
	"fmt"
	"github.com/coschain/contentos-go/p2p/msg"
	"github.com/coschain/contentos-go/prototype"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/coschain/contentos-go/p2p/depend/common/config"
	"github.com/coschain/contentos-go/p2p/depend/common/log"
	evtActor "github.com/ontio/ontology-eventbus/actor"

	msgCommon "github.com/coschain/contentos-go/p2p/common"
	"github.com/coschain/contentos-go/p2p/message/msg_pack"
	msgTypes "github.com/coschain/contentos-go/p2p/message/types"
	"github.com/coschain/contentos-go/p2p/net/protocol"
)

// AddrReqHandle handles the neighbor address request from peer
func AddrReqHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, pid *evtActor.PID, args ...interface{}) {
	log.Trace("[p2p]receive addr request message", data.Addr, data.Id)
	remotePeer := p2p.GetPeer(data.Id)
	if remotePeer == nil {
		log.Debug("[p2p]remotePeer invalid in AddrReqHandle")
		return
	}

	var addrStr []msgCommon.PeerAddr
	addrStr = p2p.GetNeighborAddrs()
	//check mask peers
	mskPeers := config.DefConfig.P2PNode.ReservedCfg.MaskPeers
	if config.DefConfig.P2PNode.ReservedPeersOnly && len(mskPeers) > 0 {
		for i := 0; i < len(addrStr); i++ {
			var ip net.IP
			ip = addrStr[i].IpAddr[:]
			address := ip.To16().String()
			for j := 0; j < len(mskPeers); j++ {
				if address == mskPeers[j] {
					addrStr = append(addrStr[:i], addrStr[i+1:]...)
					i--
					break
				}
			}
		}

	}
	msg := msgpack.NewAddrs(addrStr)
	err := p2p.Send(remotePeer, msg, false)
	if err != nil {
		log.Warn(err)
		return
	}
}

//PingHandle handle ping msg from peer
func PingHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, pid *evtActor.PID, args ...interface{}) {
	log.Trace("[p2p]receive ping message", data.Addr, data.Id)

	ping := data.Payload.(*msgTypes.Ping)
	remotePeer := p2p.GetPeer(data.Id)
	if remotePeer == nil {
		log.Debug("[p2p]remotePeer invalid in PingHandle")
		return
	}
	remotePeer.SetHeight(ping.Height)

	//height := ledger.DefLedger.GetCurrentBlockHeight()

	height := 0

	p2p.SetHeight(uint64(height))
	msg := msgpack.NewPongMsg(uint64(height))

	err := p2p.Send(remotePeer, msg, false)
	if err != nil {
		log.Warn(err)
	}
}

///PongHandle handle pong msg from peer
func PongHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, pid *evtActor.PID, args ...interface{}) {
	log.Trace("[p2p]receive pong message", data.Addr, data.Id)

	pong := data.Payload.(*msgTypes.Pong)

	remotePeer := p2p.GetPeer(data.Id)
	if remotePeer == nil {
		log.Debug("[p2p]remotePeer invalid in PongHandle")
		return
	}
	remotePeer.SetHeight(pong.Height)
}

// BlockHandle handles the block message from peer
func BlockHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, pid *evtActor.PID, args ...interface{}) {
	log.Trace("[p2p]receive block message from ", data.Addr, data.Id)

	var block = data.Payload.(*msg.SigBlkMsg)

	log.Info("receive a block")
	fmt.Printf("data:   +%v\n", block)

	if pid != nil {
		var block = data.Payload.(*msg.SigBlkMsg)

		log.Info("receive a block")
		fmt.Printf("data:   +%v\n", block)
		//input := &msgCommon.AppendBlock{
		//	FromID:    data.Id,
		//	BlockSize: data.PayloadSize,
		//	Block:     block.Blk,
		//}
		//pid.Tell(input)
	}
}

// NotFoundHandle handles the not found message from peer
func NotFoundHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, pid *evtActor.PID, args ...interface{}) {
	var notFound = data.Payload.(*msgTypes.NotFound)
	log.Debug("[p2p]receive notFound message, hash is ", notFound.Hash)
}

// TransactionHandle handles the transaction message from peer
func TransactionHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, pid *evtActor.PID, args ...interface{}) {
	log.Trace("[p2p]receive transaction message", data.Addr, data.Id)

	var trn = data.Payload.(*msg.BroadcastSigTrx)

	log.Info("receive a trx")
	fmt.Printf("data:   +%v\n", trn)

	//actor.AddTransaction(trn.Txn)
	//log.Trace("[p2p]receive Transaction message hash", trn.Txn.Hash())

}

// VersionHandle handles version handshake protocol from peer
func VersionHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, pid *evtActor.PID, args ...interface{}) {
	log.Trace("[p2p]receive version message", data.Addr, data.Id)

	version := data.Payload.(*msgTypes.Version)

	remotePeer := p2p.GetPeerFromAddr(data.Addr)
	if remotePeer == nil {
		log.Debug("[p2p]peer is not exist", data.Addr)
		//peer not exist,just remove list and return
		p2p.RemoveFromConnectingList(data.Addr)
		return
	}
	addrIp, err := msgCommon.ParseIPAddr(data.Addr)
	if err != nil {
		log.Warn(err)
		return
	}
	nodeAddr := addrIp + ":" +
		strconv.Itoa(int(version.P.SyncPort))
	if config.DefConfig.P2PNode.ReservedPeersOnly && len(config.DefConfig.P2PNode.ReservedCfg.ReservedPeers) > 0 {
		found := false
		for _, addr := range config.DefConfig.P2PNode.ReservedCfg.ReservedPeers {
			if strings.HasPrefix(data.Addr, addr) {
				log.Debug("[p2p]peer in reserved list", data.Addr)
				found = true
				break
			}
		}
		if !found {
			remotePeer.CloseSync()
			remotePeer.CloseCons()
			log.Debug("[p2p]peer not in reserved list,close", data.Addr)
			return
		}

	}

	if version.P.IsConsensus == true {
		if config.DefConfig.P2PNode.DualPortSupport == false {
			log.Warn("[p2p]consensus port not surpport", data.Addr)
			remotePeer.CloseCons()
			return
		}

		p := p2p.GetPeer(version.P.Nonce)

		if p == nil {
			log.Warn("[p2p]sync link is not exist", version.P.Nonce, data.Addr)
			remotePeer.CloseCons()
			remotePeer.CloseSync()
			return
		} else {
			//p synclink must exist,merged
			p.ConsLink = remotePeer.ConsLink
			p.ConsLink.SetID(version.P.Nonce)
			p.SetConsState(remotePeer.GetConsState())
			remotePeer = p

		}
		if version.P.Nonce == p2p.GetID() {
			log.Warn("[p2p]the node handshake with itself", data.Addr)
			p2p.SetOwnAddress(nodeAddr)
			p2p.RemoveFromInConnRecord(remotePeer.GetAddr())
			p2p.RemoveFromOutConnRecord(remotePeer.GetAddr())
			remotePeer.CloseCons()
			return
		}

		s := remotePeer.GetConsState()
		if s != msgCommon.INIT && s != msgCommon.HAND {
			log.Warnf("[p2p]unknown status to received version,%d,%s\n", s, data.Addr)
			remotePeer.CloseCons()
			return
		}

		// Todo: change the method of input parameters
		remotePeer.UpdateInfo(time.Now(), version.P.Version,
			version.P.Services, version.P.SyncPort,
			version.P.ConsPort, version.P.Nonce,
			version.P.Relay, version.P.StartHeight)

		var msg msgTypes.Message
		if s == msgCommon.INIT {
			remotePeer.SetConsState(msgCommon.HAND_SHAKE)
			//msg = msgpack.NewVersion(p2p, true, ledger.DefLedger.GetCurrentBlockHeight())

			msg = msgpack.NewVersion(p2p, true, 0)

		} else if s == msgCommon.HAND {
			remotePeer.SetConsState(msgCommon.HAND_SHAKED)
			msg = msgpack.NewVerAck(true)

		}
		err := p2p.Send(remotePeer, msg, true)
		if err != nil {
			log.Warn(err)
			return
		}
	} else {
		if version.P.Nonce == p2p.GetID() {
			p2p.RemoveFromInConnRecord(remotePeer.GetAddr())
			p2p.RemoveFromOutConnRecord(remotePeer.GetAddr())
			log.Warn("[p2p]the node handshake with itself", remotePeer.GetAddr())
			p2p.SetOwnAddress(nodeAddr)
			remotePeer.CloseSync()
			return
		}

		s := remotePeer.GetSyncState()
		if s != msgCommon.INIT && s != msgCommon.HAND {
			log.Warnf("[p2p]unknown status to received version,%d,%s\n", s, remotePeer.GetAddr())
			remotePeer.CloseSync()
			return
		}

		// Obsolete node
		p := p2p.GetPeer(version.P.Nonce)
		if p != nil {
			ipOld, err := msgCommon.ParseIPAddr(p.GetAddr())
			if err != nil {
				log.Warn("[p2p]exist peer %d ip format is wrong %s", version.P.Nonce, p.GetAddr())
				return
			}
			ipNew, err := msgCommon.ParseIPAddr(data.Addr)
			if err != nil {
				remotePeer.CloseSync()
				log.Warn("[p2p]connecting peer %d ip format is wrong %s, close", version.P.Nonce, data.Addr)
				return
			}
			if ipNew == ipOld {
				//same id and same ip
				n, ret := p2p.DelNbrNode(version.P.Nonce)
				if ret == true {
					log.Infof("[p2p]peer reconnect %d", version.P.Nonce, data.Addr)
					// Close the connection and release the node source
					n.CloseSync()
					n.CloseCons()
					if pid != nil {
						input := &msgCommon.RemovePeerID{
							ID: version.P.Nonce,
						}
						pid.Tell(input)
					}
				}
			} else {
				log.Warnf("[p2p]same peer id from different addr: %s, %s close latest one", ipOld, ipNew)
				remotePeer.CloseSync()
				return

			}
		}

		if version.P.Cap[msgCommon.HTTP_INFO_FLAG] == 0x01 {
			remotePeer.SetHttpInfoState(true)
		} else {
			remotePeer.SetHttpInfoState(false)
		}
		remotePeer.SetHttpInfoPort(version.P.HttpInfoPort)

		remotePeer.UpdateInfo(time.Now(), version.P.Version,
			version.P.Services, version.P.SyncPort,
			version.P.ConsPort, version.P.Nonce,
			version.P.Relay, version.P.StartHeight)
		remotePeer.SyncLink.SetID(version.P.Nonce)
		p2p.AddNbrNode(remotePeer)

		if pid != nil {
			input := &msgCommon.AppendPeerID{
				ID: version.P.Nonce,
			}
			pid.Tell(input)
		}

		var msg msgTypes.Message
		if s == msgCommon.INIT {
			remotePeer.SetSyncState(msgCommon.HAND_SHAKE)
			//msg = msgpack.NewVersion(p2p, false, ledger.DefLedger.GetCurrentBlockHeight())

			msg = msgpack.NewVersion(p2p, false, 0)

		} else if s == msgCommon.HAND {
			remotePeer.SetSyncState(msgCommon.HAND_SHAKED)
			msg = msgpack.NewVerAck(false)
		}
		err := p2p.Send(remotePeer, msg, false)
		if err != nil {
			log.Warn(err)
			return
		}
	}
}

// VerAckHandle handles the version ack from peer
func VerAckHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, pid *evtActor.PID, args ...interface{}) {
	log.Trace("[p2p]receive verAck message from ", data.Addr, data.Id)

	verAck := data.Payload.(*msgTypes.VerACK)
	remotePeer := p2p.GetPeer(data.Id)

	if remotePeer == nil {
		log.Warn("[p2p]nbr node is not exist", data.Id, data.Addr)
		return
	}

	if verAck.IsConsensus == true {
		if config.DefConfig.P2PNode.DualPortSupport == false {
			log.Warn("[p2p]consensus port not surpport")
			return
		}
		s := remotePeer.GetConsState()
		if s != msgCommon.HAND_SHAKE && s != msgCommon.HAND_SHAKED {
			log.Warnf("[p2p]unknown status to received verAck,state:%d,%s\n", s, data.Addr)
			return
		}

		remotePeer.SetConsState(msgCommon.ESTABLISH)
		p2p.RemoveFromConnectingList(data.Addr)
		remotePeer.SetConsConn(remotePeer.GetConsConn())

		if s == msgCommon.HAND_SHAKE {
			msg := msgpack.NewVerAck(true)
			p2p.Send(remotePeer, msg, true)
		}
	} else {
		s := remotePeer.GetSyncState()
		if s != msgCommon.HAND_SHAKE && s != msgCommon.HAND_SHAKED {
			log.Warnf("[p2p]unknown status to received verAck,state:%d,%s\n", s, data.Addr)
			return
		}

		remotePeer.SetSyncState(msgCommon.ESTABLISH)
		p2p.RemoveFromConnectingList(data.Addr)
		remotePeer.DumpInfo()

		addr := remotePeer.SyncLink.GetAddr()

		if s == msgCommon.HAND_SHAKE {
			msg := msgpack.NewVerAck(false)
			p2p.Send(remotePeer, msg, false)
		} else {
			//consensus port connect
			if config.DefConfig.P2PNode.DualPortSupport && remotePeer.GetConsPort() > 0 {
				addrIp, err := msgCommon.ParseIPAddr(addr)
				if err != nil {
					log.Warn(err)
					return
				}
				nodeConsensusAddr := addrIp + ":" +
					strconv.Itoa(int(remotePeer.GetConsPort()))
				go p2p.Connect(nodeConsensusAddr, true)
			}
		}

		msg := msgpack.NewAddrReq()
		go p2p.Send(remotePeer, msg, false)
	}

}

// AddrHandle handles the neighbor address response message from peer
func AddrHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, pid *evtActor.PID, args ...interface{}) {
	log.Trace("[p2p]handle addr message", data.Addr, data.Id)

	var msg = data.Payload.(*msgTypes.Addr)
	for _, v := range msg.NodeAddrs {
		var ip net.IP
		ip = v.IpAddr[:]
		address := ip.To16().String() + ":" + strconv.Itoa(int(v.Port))

		if v.ID == p2p.GetID() {
			continue
		}

		if p2p.NodeEstablished(v.ID) {
			continue
		}

		if ret := p2p.GetPeerFromAddr(address); ret != nil {
			continue
		}

		if v.Port == 0 {
			continue
		}
		if p2p.IsAddrFromConnecting(address) {
			continue
		}
		log.Debug("[p2p]connect ip address:", address)
		go p2p.Connect(address, false)
	}
}

// DisconnectHandle handles the disconnect events
func DisconnectHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, pid *evtActor.PID, args ...interface{}) {
	log.Debug("[p2p]receive disconnect message", data.Addr, data.Id)
	p2p.RemoveFromInConnRecord(data.Addr)
	p2p.RemoveFromOutConnRecord(data.Addr)
	remotePeer := p2p.GetPeer(data.Id)
	if remotePeer == nil {
		log.Debug("[p2p]disconnect peer is nil")
		return
	}
	p2p.RemoveFromConnectingList(data.Addr)

	if remotePeer.SyncLink.GetAddr() == data.Addr {
		p2p.RemovePeerSyncAddress(data.Addr)
		p2p.RemovePeerConsAddress(data.Addr)
		remotePeer.CloseSync()
		remotePeer.CloseCons()
	}
	if remotePeer.ConsLink.GetAddr() == data.Addr {
		p2p.RemovePeerConsAddress(data.Addr)
		remotePeer.CloseCons()
	}
}

func HashMsgHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, pid *evtActor.PID, args ...interface{}) {
	log.Trace("[p2p]receive hash message from ", data.Addr, data.Id)

	var msgdata = data.Payload.(*msg.HashMsg)
	remotePeer := p2p.GetPeerFromAddr(data.Addr)
	switch msgdata.Msgtype{
	case msg.HashMsg_broadcast_sigblk_hash:
		//if consensus do not has this hash
		var reqmsg msg.HashMsg
		reqmsg.Msgtype = msg.HashMsg_request_sigblk_by_hash
		for _, ha := range msgdata.Value {
			reqmsg.Value = append(reqmsg.Value, new(prototype.Sha256) )
			idx := len(reqmsg.Value) - 1
			*reqmsg.Value[idx] = *ha
		}
		err := p2p.Send(remotePeer, &reqmsg, false)
		if err != nil {
			log.Warn(err)
			return
		}
	case msg.HashMsg_request_sigblk_by_hash:
		fallthrough
	case msg.HashMsg_request_hash_ack:
		//for i, ha := range msgdata.Value {
		//	sigblk := get sigblk from consensus by hash
		//	msg := msgpack.NewSigBlk(sigblk)
		//	err := p2p.Send(remotePeer, msg, false)
		//	if err != nil {
		//		log.Warn(err)
		//		return
		//	}
		//}
	default:
		log.Warnf("[p2p]Unknown hash message %v", msgdata)
	}
}

func ReqHashHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, pid *evtActor.PID, args ...interface{}) {
	log.Trace("[p2p]receive request hash message from ", data.Addr, data.Id)

	//var msgdata = data.Payload.(*msg.ReqHashMsg)
	//remote_head_blk_id := msgdata.HeadBlockId

	// hashes := call consensus to get hashes

	//remotePeer := p2p.GetPeerFromAddr(data.Addr)
	//var reqmsg msg.HashMsg
	//reqmsg.Msgtype = msg.HashMsg_request_hash_ack
	//for _, ha := range hashes{
	//	reqmsg.Value = append(reqmsg.Value, new(prototype.Sha256) )
	//	idx := len(reqmsg.Value) - 1
	//	*reqmsg.Value[idx] = *ha
	//}
	//err := p2p.Send(remotePeer, &reqmsg, false)
	//if err != nil {
	//	log.Warn(err)
	//	return
	//}
}
