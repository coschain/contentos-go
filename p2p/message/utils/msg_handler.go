package utils

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	msgCommon "github.com/coschain/contentos-go/p2p/common"
	"github.com/coschain/contentos-go/p2p/depend/common/config"
	"github.com/coschain/contentos-go/p2p/depend/common/log"
	"github.com/coschain/contentos-go/p2p/message/msg_pack"
	msgTypes "github.com/coschain/contentos-go/p2p/message/types"
	"github.com/coschain/contentos-go/p2p/msg"
	"github.com/coschain/contentos-go/p2p/net/protocol"
	"github.com/coschain/contentos-go/prototype"
)

// AddrReqHandle handles the neighbor address request from peer
func AddrReqHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	log.Trace("[p2p]receive addr request message", data.Addr, data.Id)
	remotePeer := p2p.GetPeer(data.Id)
	if remotePeer == nil {
		log.Debug("[p2p]remotePeer invalid in AddrReqHandle")
		return
	}

	var addrStr []*msg.PeerAddr
	addrStr = p2p.GetNeighborAddrs()
	//check mask peers
	mskPeers := config.DefConfig.P2PNode.ReservedCfg.MaskPeers
	if config.DefConfig.P2PNode.ReservedPeersOnly && len(mskPeers) > 0 {
		for i := 0; i < len(addrStr); i++ {
			var ip net.IP
			ip = addrStr[i].IpAddr
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
func PingHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	log.Trace("[p2p]receive ping message", data.Addr, data.Id)

	ping := data.Payload.(*msg.TransferMsg).Msg.(*msg.TransferMsg_Msg8).Msg8

	remotePeer := p2p.GetPeer(data.Id)
	if remotePeer == nil {
		log.Debug("[p2p]remotePeer invalid in PingHandle")
		return
	}
	remotePeer.SetHeight(ping.Height)

	//s, err := p2p.GetService(iservices.CS_SERVER_NAME)
	//if err != nil {
	//	log.Info("can't get other service, service name: ", iservices.CS_SERVER_NAME)
	//	return
	//}
	//ctrl := s.(iservices.IConsensus)
	//height := ctrl.GetHeadBlockId().BlockNum()
	var height uint64 = 0
	var err error

	p2p.SetHeight(height)
	msg := msgpack.NewPongMsg(height)

	err = p2p.Send(remotePeer, msg, false)
	if err != nil {
		log.Warn(err)
	}
}

///PongHandle handle pong msg from peer
func PongHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	log.Trace("[p2p]receive pong message", data.Addr, data.Id)

	pong := data.Payload.(*msg.TransferMsg).Msg.(*msg.TransferMsg_Msg9).Msg9

	remotePeer := p2p.GetPeer(data.Id)
	if remotePeer == nil {
		log.Debug("[p2p]remotePeer invalid in PongHandle")
		return
	}
	remotePeer.SetHeight(pong.Height)
}

// BlockHandle handles the block message from peer
func BlockHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	log.Trace("[p2p]receive block message from ", data.Addr, data.Id)

	var block = data.Payload.(*msg.TransferMsg).Msg.(*msg.TransferMsg_Msg3).Msg3

	remotePeer := p2p.GetPeerFromAddr(data.Addr)

	log.Info("receive a SignedBlock msg:   ", block)

	noticer := p2p.GetNoticer()
	noticer.Publish(constants.NOTICE_HANDLE_P2P_SIGBLK, remotePeer, block.SigBlk)
}

// NotFoundHandle handles the not found message from peer
func NotFoundHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	var notFound = data.Payload.(*msg.TransferMsg).Msg.(*msg.TransferMsg_Msg12).Msg12
	log.Debug("[p2p]receive notFound message, hash is ", notFound.Hash)
}

// TransactionHandle handles the transaction message from peer
func TransactionHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	log.Trace("[p2p]receive transaction message", data.Addr, data.Id)

	var trn = data.Payload.(*msg.TransferMsg).Msg.(*msg.TransferMsg_Msg1).Msg1

	log.Info("receive a trx")
	fmt.Printf("data:   +%v\n", trn)

	noticer := p2p.GetNoticer()
	noticer.Publish(constants.NOTICE_HANDLE_P2P_SIGTRX, trn.SigTrx)
}

// VersionHandle handles version handshake protocol from peer
func VersionHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	log.Trace("[p2p]receive version message", data.Addr, data.Id)

	version := data.Payload.(*msg.TransferMsg).Msg.(*msg.TransferMsg_Msg11).Msg11

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
		strconv.Itoa(int(version.SyncPort))
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

	//service, err := p2p.GetService(iservices.CS_SERVER_NAME)
	//if err != nil {
	//	log.Info("can't get other service, service name: ", iservices.CS_SERVER_NAME)
	//	return
	//}
	//ctrl := service.(iservices.IConsensus)

	if version.IsConsensus == true {
		if config.DefConfig.P2PNode.DualPortSupport == false {
			log.Warn("[p2p]consensus port not surpport", data.Addr)
			remotePeer.CloseCons()
			return
		}

		p := p2p.GetPeer(version.Nonce)

		if p == nil {
			log.Warn("[p2p]sync link is not exist", version.Nonce, data.Addr)
			remotePeer.CloseCons()
			remotePeer.CloseSync()
			return
		} else {
			//p synclink must exist,merged
			p.ConsLink = remotePeer.ConsLink
			p.ConsLink.SetID(version.Nonce)
			p.SetConsState(remotePeer.GetConsState())
			remotePeer = p

		}
		if version.Nonce == p2p.GetID() {
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
		remotePeer.UpdateInfo(time.Now(), version.Version,
			version.Services, version.SyncPort,
			version.ConsPort, version.Nonce,
			version.Relay, version.StartHeight)

		var msg msgTypes.Message
		if s == msgCommon.INIT {
			remotePeer.SetConsState(msgCommon.HAND_SHAKE)
			//msg = msgpack.NewVersion(p2p, true, ctrl.GetHeadBlockId().BlockNum())
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
		if version.Nonce == p2p.GetID() {
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
		p := p2p.GetPeer(version.Nonce)
		if p != nil {
			ipOld, err := msgCommon.ParseIPAddr(p.GetAddr())
			if err != nil {
				log.Warn("[p2p]exist peer %d ip format is wrong %s", version.Nonce, p.GetAddr())
				return
			}
			ipNew, err := msgCommon.ParseIPAddr(data.Addr)
			if err != nil {
				remotePeer.CloseSync()
				log.Warn("[p2p]connecting peer %d ip format is wrong %s, close", version.Nonce, data.Addr)
				return
			}
			if ipNew == ipOld {
				//same id and same ip
				n, ret := p2p.DelNbrNode(version.Nonce)
				if ret == true {
					log.Infof("[p2p]peer reconnect %d", version.Nonce, data.Addr)
					// Close the connection and release the node source
					n.CloseSync()
					n.CloseCons()
				}
			} else {
				log.Warnf("[p2p]same peer id from different addr: %s, %s close latest one", ipOld, ipNew)
				remotePeer.CloseSync()
				return

			}
		}

		remotePeer.UpdateInfo(time.Now(), version.Version,
			version.Services, version.SyncPort,
			version.ConsPort, version.Nonce,
			version.Relay, version.StartHeight)
		remotePeer.SyncLink.SetID(version.Nonce)
		p2p.AddNbrNode(remotePeer)

		var msg msgTypes.Message
		if s == msgCommon.INIT {
			remotePeer.SetSyncState(msgCommon.HAND_SHAKE)
			//msg = msgpack.NewVersion(p2p, false, ctrl.GetHeadBlockId().BlockNum())
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
func VerAckHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	log.Trace("[p2p]receive verAck message from ", data.Addr, data.Id)

	verAck := data.Payload.(*msg.TransferMsg).Msg.(*msg.TransferMsg_Msg10).Msg10

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

		var blockid common.BlockID
		//service, err := p2p.GetService(iservices.CS_SERVER_NAME)
		//if err != nil {
		//	log.Info("can't get other service, service name: ", iservices.CS_SERVER_NAME)
		//	return
		//}
		//ctrl := service.(iservices.IConsensus)
		//blockid = ctrl.GetHeadBlockId()
		p2p.Trigger_sync(remotePeer, blockid)

		msg := msgpack.NewAddrReq()
		go p2p.Send(remotePeer, msg, false)
	}

}

// AddrHandle handles the neighbor address response message from peer
func AddrHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	log.Trace("[p2p]handle addr message", data.Addr, data.Id)

	var msg = data.Payload.(*msg.TransferMsg).Msg.(*msg.TransferMsg_Msg5).Msg5
	for _, v := range msg.Addr {
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
func DisconnectHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
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

func IdMsgHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	log.Trace("[p2p]receive hash message from ", data.Addr, data.Id)

	var msgdata = data.Payload.(*msg.TransferMsg).Msg.(*msg.TransferMsg_Msg2).Msg2
	remotePeer := p2p.GetPeerFromAddr(data.Addr)
	switch msgdata.Msgtype{
	case msg.IdMsg_broadcast_sigblk_id:
		log.Info("receive a msg from:    v%    data:   %v\n", data.Addr, *msgdata)
		length := len( msgdata.Value[0])
		if length > prototype.Size {
			log.Info("block id length beyond the limit ", prototype.Size)
			return
		}
		var blkId common.BlockID
		copy( blkId.Data[:], msgdata.Value[0] )

		//s, err := p2p.GetService(iservices.CS_SERVER_NAME)
		//if err != nil {
		//	log.Info("can't get other service, service name: ", iservices.CS_SERVER_NAME)
		//	return
		//}
		//ctrl := s.(iservices.IConsensus)
		//if !ctrl.HasBlock(blkId) {
		//	var reqmsg msg.IdMsg
		//	reqmsg.Msgtype = msg.IdMsg_request_sigblk_by_id
		//	var tmp []byte
		//	reqmsg.Value = append(reqmsg.Value, tmp )
		//	reqmsg.Value[0] = msgdata.Value[0]
		//
		//	err := p2p.Send(remotePeer, &reqmsg, false)
		//	if err != nil {
		//		log.Warn(err)
		//		return
		//	}
		//	log.Info("send a message to:   v%   data:   v%\n", data.Addr, reqmsg)
		//}
	case msg.IdMsg_request_sigblk_by_id:
		log.Info("receive a msg from:    v%    data:   %v\n", data.Addr, *msgdata)
		for i, id := range msgdata.Value {
			length := len( msgdata.Value[i])
			if length > prototype.Size {
				log.Info("block id length beyond the limit ", prototype.Size)
				continue
			}
			var blkId common.BlockID
			copy( blkId.Data[:], id )

			// test code create a SignedBlock
			sigBlk := new(prototype.SignedBlock)
			sigBlkHdr := new(prototype.SignedBlockHeader)
			sigBlk.SignedHeader = sigBlkHdr
			sigBlkHdr.Header = new(prototype.BlockHeader)
			sigBlkHdr.Header.Witness = new(prototype.AccountName)
			sigBlkHdr.Header.Witness.Value = "alice"

			sigBlkHdr.Header.Previous = new(prototype.Sha256)
			sigBlkHdr.Header.TransactionMerkleRoot = new(prototype.Sha256)
			sigBlkHdr.Header.Previous.Hash = make([]byte, prototype.Size)
			sigBlkHdr.Header.TransactionMerkleRoot.Hash = make([]byte, prototype.Size)

			//sigBlk := get sigblk from consensus by id

			msg := msgpack.NewSigBlk(sigBlk)
			err := p2p.Send(remotePeer, msg, false)
			if err != nil {
				log.Warn(err)
				return
			}
			log.Info("send a SignedBlock msg to   v%   data   v%\n", data.Addr, msg)
		}
	case msg.IdMsg_request_id_ack:
		log.Info("receive a msg from:    v%    data:   %v\n", data.Addr, *msgdata)
		var reqmsg msg.TransferMsg
		var reqdata msg.IdMsg
		reqdata.Msgtype = msg.IdMsg_request_sigblk_by_id
		for _, id := range msgdata.Value {
			length := len( id )
			if length > prototype.Size {
				log.Info("block id length beyond the limit ", prototype.Size)
				continue
			}
			var blkId common.BlockID
			copy( blkId.Data[:], id )

			//s, err := p2p.GetService(iservices.CS_SERVER_NAME)
			//if err != nil {
			//	log.Info("can't get other service, service name: ", iservices.CS_SERVER_NAME)
			//	return
			//}
			//ctrl := s.(iservices.IConsensus)
			//if !ctrl.HasBlock(blkId) {
			//	var tmp []byte
			//	reqdata.Value = append(reqdata.Value, tmp)
			//	idx := len(reqdata.Value) - 1
			//	reqdata.Value[idx] = id
			//}
		}
		reqmsg.Msg = &msg.TransferMsg_Msg2{Msg2:&reqdata}
		err := p2p.Send(remotePeer, &reqmsg, false)
		if err != nil {
			log.Warn(err)
			return
		}
		log.Info("send a message to:   v%   data:   v%\n", remotePeer, reqmsg)
	default:
		log.Warnf("[p2p]Unknown id message %v", msgdata)
	}
}

func ReqIdHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	log.Trace("[p2p]receive request id message from ", data.Addr, data.Id)

	var msgdata = data.Payload.(*msg.TransferMsg).Msg.(*msg.TransferMsg_Msg4).Msg4
	length := len(msgdata.HeadBlockId)
	if length > prototype.Size {
		log.Info("block id length beyond the limit ", prototype.Size)
		return
	}

	log.Info("receive a ReqIdMsg from   v%    data   v%\n", data.Addr, msgdata.HeadBlockId)

	//s, err := p2p.GetService(iservices.CS_SERVER_NAME)
	//if err != nil {
	//	log.Info("can't get other service, service name: ", iservices.CS_SERVER_NAME)
	//	return
	//}
	//ctrl := s.(iservices.IConsensus)
	//var remote_head_blk_id common.BlockID
	//copy( remote_head_blk_id.Data[:], msgdata.HeadBlockId )
	//current_head_blk_id := ctrl.GetHeadBlockId()
	//ids := ctrl.GetHashes(remote_head_blk_id, current_head_blk_id)

	remotePeer := p2p.GetPeerFromAddr(data.Addr)

	var reqmsg msg.TransferMsg
	var reqdata msg.IdMsg
	reqdata.Msgtype = msg.IdMsg_request_id_ack

	//for i, id := range ids {
	//	var tmp []byte
	//	reqdata.Value = append(reqdata.Value, tmp)
	//	reqdata.Value[i] = make([]byte, prototype.Size)
	//	reqdata.Value[i] = id.Data[:]
	//}

	reqmsg.Msg = &msg.TransferMsg_Msg2{Msg2:&reqdata}
	err := p2p.Send(remotePeer, &reqmsg, false)
	if err != nil {
		log.Warn(err)
		return
	}
}
