package utils

import (
	"github.com/coschain/gobft/message"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/iservices"
	msgCommon "github.com/coschain/contentos-go/p2p/common"
	"github.com/coschain/contentos-go/p2p/message/msg_pack"
	msgTypes "github.com/coschain/contentos-go/p2p/message/types"
	"github.com/coschain/contentos-go/p2p/net/protocol"
	"github.com/coschain/contentos-go/prototype"
)

type MsgHandler struct {
	blockCache map[common.BlockID]common.ISignedBlock
	sync.Mutex

	syncPushBlock sync.Mutex
}

func NewMsgHandler() *MsgHandler  {
	blockCache := make(map[common.BlockID]common.ISignedBlock)

	return &MsgHandler{ blockCache:blockCache, syncPushBlock:sync.Mutex{} }
}

func (p *MsgHandler) popFirstBlock() common.ISignedBlock {

	var retV common.ISignedBlock = nil
	var retK = common.EmptyBlockID

	for k, v := range p.blockCache {

		if retK == common.EmptyBlockID{
			retK = k
			retV = v
		} else if k.BlockNum() < retK.BlockNum() {
			retK = k
			retV = v
		}
	}

	if retK != common.EmptyBlockID{
		delete( p.blockCache, retK)
	}

	return retV
}

// AddrReqHandle handles the neighbor address request from peer
func (p *MsgHandler)AddrReqHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	log := p2p.GetLog()
	remotePeer := p2p.GetPeer(data.Id)
	if remotePeer == nil {
		log.Error("[p2p] remotePeer invalid in AddrReqHandle")
		return
	}

	var addrStr []*msgTypes.PeerAddr
	addrStr = p2p.GetNeighborAddrs()
	//check mask peers
	ctx := p2p.GetContex()
	if ctx == nil {
		log.Error("[p2p] ctx invalid in AddrReqHandle")
		return
	}
	mskPeers := ctx.Config().P2P.ReservedCfg.MaskPeers
	if ctx.Config().P2P.ReservedPeersOnly && len(mskPeers) > 0 {
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
		log.Error("[p2p] send message error: ", err)
		return
	}
}

//PingHandle handle ping msg from peer
func (p *MsgHandler)PingHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	log := p2p.GetLog()

	var raw = data.Payload.(*msgTypes.TransferMsg)
	ping := raw.Msg.(*msgTypes.TransferMsg_Msg8).Msg8

	remotePeer := p2p.GetPeer(data.Id)
	if remotePeer == nil {
		log.Error("[p2p] remotePeer invalid in PingHandle")
		return
	}
	remotePeer.SetHeight(ping.Height)

	//s, err := p2p.GetService(iservices.ConsensusServerName)
	//if err != nil {
	//	panic(err)
	//}
	//ctrl := s.(iservices.IConsensus)
	//height := ctrl.GetHeadBlockId().BlockNum()
	var height uint64 = 0

	p2p.SetHeight(height)
	reqmsg := msgpack.NewPongMsg(height)

	err := p2p.Send(remotePeer, reqmsg, false)
	if err != nil {
		log.Error("[p2p] send message error: ", err)
	}
}

///PongHandle handle pong msg from peer
func (p *MsgHandler)PongHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	var raw = data.Payload.(*msgTypes.TransferMsg)
	pong := raw.Msg.(*msgTypes.TransferMsg_Msg9).Msg9

	log := p2p.GetLog()

	remotePeer := p2p.GetPeer(data.Id)
	if remotePeer == nil {
		log.Error("[p2p] remotePeer invalid in PongHandle")
		return
	}
	remotePeer.SetHeight(pong.Height)
}

// BlockHandle handles the block message from peer

func (p *MsgHandler)BlockSyncHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {

	var raw = data.Payload.(*msgTypes.TransferMsg)
	var block = raw.Msg.(*msgTypes.TransferMsg_Msg3).Msg3

	log := p2p.GetLog()
	log.Info("[p2p] receive a SignedBlock msg, block number :   ", block.SigBlk.Id().BlockNum())
	blkNum := block.SigBlk.Id().BlockNum()

	remotePeer := p2p.GetPeerFromAddr(data.Addr)
	if remotePeer == nil {
		log.Error("[p2p] peer is not exist: ", data.Addr)
		return
	}
	remotePeer.SetLastSeenBlkNum(blkNum)

	p.Lock()
	p.blockCache[ block.SigBlk.Id() ] = block.SigBlk
	p.Unlock()

	s, err := p2p.GetService(iservices.ConsensusServerName)
	if err != nil {
		log.Error("[p2p] can't get other service, service name: ", iservices.ConsensusServerName)
		return
	}
	ctrl := s.(iservices.IConsensus)

	go func() {
		p.blockHandle(ctrl)
	}()
}

func (p *MsgHandler)blockHandle(ctrl iservices.IConsensus) {

	p.syncPushBlock.Lock()
	defer p.syncPushBlock.Unlock()

	p.Lock()
	block := p.popFirstBlock()
	p.Unlock()

	if block != nil{
		if ctrl.HasBlock(block.Id()) {
			return
		}
		ctrl.PushBlock(block)
	}
}

// TransactionHandle handles the transaction message from peer
func (p *MsgHandler)TransactionHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	var raw = data.Payload.(*msgTypes.TransferMsg)
	var trn = raw.Msg.(*msgTypes.TransferMsg_Msg1).Msg1

	log := p2p.GetLog()
	//log.Info("receive a SignedTransaction msg: ", trn)

	id, _ := trn.SigTrx.Id()
	remotePeer := p2p.GetPeerFromAddr(data.Addr)
	if remotePeer == nil {
		log.Error("[p2p] peer is not exist: ", data.Addr)
		return
	}
	if remotePeer.HasTrx(id.Hash) {
		//log.Info("[p2p] we alerady have this transaction, transaction hash: ", id.Hash)
		return
	}
	remotePeer.RecordTrxCache(id.Hash)

	s, err := p2p.GetService(iservices.ConsensusServerName)
	if err != nil {
		log.Error("[p2p] can't get other service, service name: ", iservices.ConsensusServerName)
		return
	}
	ctrl := s.(iservices.IConsensus)
	go func() {
		_ = ctrl.PushTransactionToPending(trn.SigTrx)
	}()
}

// VersionHandle handles version handshake protocol from peer
func (p *MsgHandler)VersionHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	var raw = data.Payload.(*msgTypes.TransferMsg)
	version := raw.Msg.(*msgTypes.TransferMsg_Msg11).Msg11

	log := p2p.GetLog()

	remotePeer := p2p.GetPeerFromAddr(data.Addr)
	if remotePeer == nil {
		log.Error("[p2p] peer is not exist: ", data.Addr)
		//peer not exist,just remove list and return
		p2p.RemoveFromConnectingList(data.Addr)
		return
	}
	addrIp, err := msgCommon.ParseIPAddr(data.Addr)
	if err != nil {
		log.Error("[p2p] can't parse IP address: ", err)
		return
	}
	nodeAddr := addrIp + ":" + strconv.Itoa(int(version.SyncPort))
	ctx := p2p.GetContex()
	if ctx == nil {
		log.Error("[p2p] ctx invalid in VersionHandle")
		return
	}
	if ctx.Config().P2P.ReservedPeersOnly && len(ctx.Config().P2P.ReservedCfg.ReservedPeers) > 0 {
		found := false
		for _, addr := range ctx.Config().P2P.ReservedCfg.ReservedPeers {
			if strings.HasPrefix(data.Addr, addr) {
				log.Debug("[p2p] peer in reserved list: ", data.Addr)
				found = true
				break
			}
		}
		if !found {
			remotePeer.CloseSync()
			remotePeer.CloseCons()
			log.Debug("[p2p] peer not in reserved list, close ", data.Addr)
			return
		}
	}

	//service, err := p2p.GetService(iservices.ConsensusServerName)
	//if err != nil {
	//	log.Error("[p2p] can't get other service, service name: ", iservices.ConsensusServerName)
	//	return
	//}
	//ctrl := service.(iservices.IConsensus)

	if version.IsConsensus == true {
		if ctx.Config().P2P.DualPortSupport == false {
			log.Warn("[p2p] consensus port not surpport ", data.Addr)
			remotePeer.CloseCons()
			return
		}

		p := p2p.GetPeer(version.Nonce)

		if p == nil {
			log.Warn("[p2p] sync link is not exist: ", version.Nonce, data.Addr)
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
			log.Warn("[p2p] the node handshake with itself ", data.Addr)
			p2p.SetOwnAddress(nodeAddr)
			p2p.RemoveFromInConnRecord(remotePeer.GetAddr())
			p2p.RemoveFromOutConnRecord(remotePeer.GetAddr())
			remotePeer.CloseCons()
			return
		}

		s := remotePeer.GetConsState()
		if s != msgCommon.INIT && s != msgCommon.HAND {
			log.Warnf("[p2p] unknown status to received version,%d,%s\n", s, data.Addr)
			remotePeer.CloseCons()
			return
		}

		// Todo: change the method of input parameters
		remotePeer.UpdateInfo(time.Now(), version.Version,
			version.Services, version.SyncPort,
			version.ConsPort, version.Nonce,
			version.Relay, version.StartHeight, version.RunningCodeVersion)

		var msg msgTypes.Message
		if s == msgCommon.INIT {
			remotePeer.SetConsState(msgCommon.HAND_SHAKE)
			//msg = msgpack.NewVersion(p2p, true, ctrl.GetHeadBlockId().BlockNum())
			msg = msgpack.NewVersion(p2p, true, uint64(0), ctx.Config().P2P.RunningCodeVersion )
		} else if s == msgCommon.HAND {
			remotePeer.SetConsState(msgCommon.HAND_SHAKED)
			msg = msgpack.NewVerAck(true)

		}
		err := p2p.Send(remotePeer, msg, true)
		if err != nil {
			log.Error("[p2p] send message error: ", err)
			return
		}
	} else {
		if version.Nonce == p2p.GetID() {
			p2p.RemoveFromInConnRecord(remotePeer.GetAddr())
			p2p.RemoveFromOutConnRecord(remotePeer.GetAddr())
			log.Warn("[p2p] the node handshake with itself: ", remotePeer.GetAddr())
			p2p.SetOwnAddress(nodeAddr)
			remotePeer.CloseSync()
			return
		}

		s := remotePeer.GetSyncState()
		if s != msgCommon.INIT && s != msgCommon.HAND {
			log.Warnf("[p2p] unknown status to received version,%d,%s\n", s, remotePeer.GetAddr())
			remotePeer.CloseSync()
			return
		}

		// Obsolete node
		p := p2p.GetPeer(version.Nonce)
		if p != nil {
			ipOld, err := msgCommon.ParseIPAddr(p.GetAddr())
			if err != nil {
				log.Warnf("[p2p] exist peer %d ip format is wrong %s", version.Nonce, p.GetAddr())
				return
			}
			ipNew, err := msgCommon.ParseIPAddr(data.Addr)
			if err != nil {
				remotePeer.CloseSync()
				log.Warnf("[p2p] connecting peer %d ip format is wrong %s, close", version.Nonce, data.Addr)
				return
			}
			if ipNew == ipOld {
				//same id and same ip
				n, ret := p2p.DelNbrNode(p)
				if ret == true {
					log.Infof("[p2p] peer reconnect %d, %s ", version.Nonce, data.Addr)
					// Close the connection and release the node source
					n.CloseSync()
					n.CloseCons()
				}
			} else {
				log.Warnf("[p2p] same peer id from different addr: %s, %s close latest one", ipOld, ipNew)
				remotePeer.CloseSync()
				return

			}
		}

		remotePeer.UpdateInfo(time.Now(), version.Version,
			version.Services, version.SyncPort,
			version.ConsPort, version.Nonce,
			version.Relay, version.StartHeight, version.RunningCodeVersion)
		remotePeer.SyncLink.SetID(version.Nonce)
		p2p.AddNbrNode(remotePeer)

		var msg msgTypes.Message
		if s == msgCommon.INIT {
			remotePeer.SetSyncState(msgCommon.HAND_SHAKE)
			//msg = msgpack.NewVersion(p2p, false, ctrl.GetHeadBlockId().BlockNum())
			msg = msgpack.NewVersion(p2p, false, uint64(0), ctx.Config().P2P.RunningCodeVersion )
		} else if s == msgCommon.HAND {
			remotePeer.SetSyncState(msgCommon.HAND_SHAKED)
			msg = msgpack.NewVerAck(false)
		}
		err := p2p.Send(remotePeer, msg, false)
		if err != nil {
			log.Error("[p2p] send message error: ", err)
			return
		}
	}
}

// VerAckHandle handles the version ack from peer
func (p *MsgHandler)VerAckHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	var raw = data.Payload.(*msgTypes.TransferMsg)
	verAck := raw.Msg.(*msgTypes.TransferMsg_Msg10).Msg10

	log := p2p.GetLog()

	remotePeer := p2p.GetPeer(data.Id)
	if remotePeer == nil {
		log.Error("[p2p] nbr node is not exist ", data.Id, " ", data.Addr)
		return
	}
	ctx := p2p.GetContex()
	if ctx == nil {
		log.Error("[p2p] ctx invalid in VerAckHandle")
		return
	}

	if verAck.IsConsensus == true {
		if ctx.Config().P2P.DualPortSupport == false {
			log.Warn("[p2p] consensus port not surpport")
			return
		}
		s := remotePeer.GetConsState()
		if s != msgCommon.HAND_SHAKE && s != msgCommon.HAND_SHAKED {
			log.Warnf("[p2p] unknown status to received verAck,state:%d,%s\n", s, data.Addr)
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
			log.Warnf("[p2p] unknown status to received verAck,state:%d,%s\n", s, data.Addr)
			return
		}

		remotePeer.SetSyncState(msgCommon.ESTABLISH)
		p2p.RemoveFromConnectingList(data.Addr)
		remotePeer.DumpInfo(log)

		addr := remotePeer.SyncLink.GetAddr()

		if s == msgCommon.HAND_SHAKE {
			msg := msgpack.NewVerAck(false)
			p2p.Send(remotePeer, msg, false)
		} else {
			//consensus port connect
			if ctx.Config().P2P.DualPortSupport && remotePeer.GetConsPort() > 0 {
				addrIp, err := msgCommon.ParseIPAddr(addr)
				if err != nil {
					log.Error("[p2p] can't parse IP address: ", err)
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
func (p *MsgHandler)AddrHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	var raw = data.Payload.(*msgTypes.TransferMsg)
	var msgdata = raw.Msg.(*msgTypes.TransferMsg_Msg5).Msg5

	log := p2p.GetLog()

	for _, v := range msgdata.Addr {
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
		log.Info("[p2p] connect ip address:", address)
		go p2p.Connect(address, false)
	}
}

// DisconnectHandle handles the disconnect events
func (p *MsgHandler)DisconnectHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	log := p2p.GetLog()
	log.Info("[p2p] receive disconnect message ", data.Addr, " ", data.Id)

	p2p.RemoveFromInConnRecord(data.Addr)
	p2p.RemoveFromOutConnRecord(data.Addr)
	remotePeer := p2p.GetPeer(data.Id)
	if remotePeer == nil {
		log.Warn("[p2p] disconnect peer is nil")
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

func (p *MsgHandler)IdMsgHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	var raw = data.Payload.(*msgTypes.TransferMsg)
	var msgdata = raw.Msg.(*msgTypes.TransferMsg_Msg2).Msg2

	log := p2p.GetLog()

	remotePeer := p2p.GetPeerFromAddr(data.Addr)

	if remotePeer == nil {
		log.Error("[p2p] remotePeer invalid in IdMsgHandle")
		return
	}

	switch msgdata.Msgtype {
	case msgTypes.IdMsg_broadcast_sigblk_id:
		//log.Infof("receive a msg from:    v%    data:   %v\n", data.Addr, *msgdata)
		length := len(msgdata.Value[0])
		if length > prototype.Size {
			log.Error("[p2p] block id length beyond the limit ", prototype.Size)
			return
		}
		var blkId common.BlockID
		copy(blkId.Data[:], msgdata.Value[0])

		s, err := p2p.GetService(iservices.ConsensusServerName)
		if err != nil {
			log.Info("[p2p] can't get other service, service name: ", iservices.ConsensusServerName)
			return
		}
		ctrl := s.(iservices.IConsensus)
		if !ctrl.HasBlock(blkId) {
			var reqmsg msgTypes.TransferMsg
			reqdata := new(msgTypes.IdMsg)
			reqdata.Msgtype = msgTypes.IdMsg_request_sigblk_by_id
			var tmp []byte
			reqdata.Value = append(reqdata.Value, tmp)
			reqdata.Value[0] = msgdata.Value[0]

			reqmsg.Msg = &msgTypes.TransferMsg_Msg2{Msg2:reqdata}

			err := p2p.Send(remotePeer, &reqmsg, false)
			if err != nil {
				log.Error("[p2p] send message error: ", err)
				return
			}
			//log.Infof("send a message to:   v%   data:   v%\n", data.Addr, reqmsg)
		}
	case msgTypes.IdMsg_request_sigblk_by_id:
		//log.Infof("receive a msg from:    v%    data:   %v\n", data.Addr, *msgdata)
		if !remotePeer.LockBusy() {
			return
		} else {
			defer remotePeer.UnlockBusy()
		}

		var startBlockId common.BlockID
		s, err := p2p.GetService(iservices.ConsensusServerName)
		if err != nil {
			log.Error("[p2p] can't get other service, service name: ", iservices.ConsensusServerName)
			return
		}
		ctrl := s.(iservices.IConsensus)

		for i, id := range msgdata.Value {
			length := len(msgdata.Value[i])
			if length > prototype.Size {
				log.Info("[p2p] block id length beyond the limit ", prototype.Size)
				continue
			}
			var blkId common.BlockID
			copy(blkId.Data[:], id)
			if blkId.BlockNum() == 0 {
				continue
			}


			IsigBlk, err := ctrl.FetchBlock(blkId)
			if err != nil {
				log.Error("[p2p] can't get IsigBlk from consensus, block number: ", blkId.BlockNum(), " error: ", err)
				return
			}
			sigBlk := IsigBlk.(*prototype.SignedBlock)

			copy(startBlockId.Data[:], id)

			commitEvidence := ctrl.GetNextBFTCheckPoint(sigBlk.Id().BlockNum()-1)
			if commitEvidence != nil {
				bftCommit := &msgTypes.ConsMsg {
					MsgData: commitEvidence.(*message.Commit),
				}
				err = p2p.Send(remotePeer, bftCommit, false)
				if err != nil {
					log.Error("[p2p] send message error: ", err)
					return
				}
				log.Info("[p2p] send checkpoint message, start block number: ", startBlockId.BlockNum())
			}

			msg := msgpack.NewSigBlk(sigBlk)
			err = p2p.Send(remotePeer, msg, false)
			if err != nil {
				log.Error("[p2p] send message error: ", err)
				return
			}
			//log.Infof("send a SignedBlock msg to   v%   data   v%\n", data.Addr, msg)
		}


	case msgTypes.IdMsg_request_id_ack:
		//log.Infof("receive a msg from:    v%    data:   %v\n", data.Addr, *msgdata)
		var reqmsg msgTypes.TransferMsg
		reqdata := new(msgTypes.IdMsg)
		reqdata.Msgtype = msgTypes.IdMsg_request_sigblk_by_id
		for _, id := range msgdata.Value {
			length := len(id)
			if length > prototype.Size {
				log.Warn("[p2p] block id length beyond the limit ", prototype.Size)
				continue
			}
			var blkId common.BlockID
			copy(blkId.Data[:], id)

			s, err := p2p.GetService(iservices.ConsensusServerName)
			if err != nil {
				log.Error("[p2p] can't get other service, service name: ", iservices.ConsensusServerName)
				return
			}

			p.Lock()
			_, existInCache := p.blockCache[blkId]
			p.Unlock()

			if existInCache{
				continue
			}

			ctrl := s.(iservices.IConsensus)
			if !ctrl.HasBlock(blkId) {
				var tmp []byte
				reqdata.Value = append(reqdata.Value, tmp)
				idx := len(reqdata.Value) - 1
				reqdata.Value[idx] = id
			}
		}
		if len(reqdata.Value) == 0 {
			log.Info("[p2p] no block need to request")
			return
		}
		reqmsg.Msg = &msgTypes.TransferMsg_Msg2{Msg2:reqdata}
		err := p2p.Send(remotePeer, &reqmsg, false)
		if err != nil {
			log.Error("[p2p] send message error: ", err)
			return
		}
		//log.Infof("send a message to:   v%   data:   v%\n", remotePeer, reqmsg)
	default:
		log.Warnf("[p2p] Unknown id message %v", msgdata)
	}
}

func (p *MsgHandler)ReqIdHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	var raw = data.Payload.(*msgTypes.TransferMsg)
	var msgdata = raw.Msg.(*msgTypes.TransferMsg_Msg4).Msg4
	remotePeer := p2p.GetPeerFromAddr(data.Addr)

	log := p2p.GetLog()

	if remotePeer == nil {
		log.Error("[p2p] remotePeer invalid in ReqIdHandle")
		return
	}

	if !remotePeer.LockBusy() {
		return
	} else {
		defer remotePeer.UnlockBusy()
	}


	length := len(msgdata.HeadBlockId)
	if length > prototype.Size {
		log.Error("[p2p] block id length beyond the limit ", prototype.Size)
		return
	}

	//log.Info("receive a ReqIdMsg from   v%    data   v%\n", data.Addr, msgdata.HeadBlockId)

	s, err := p2p.GetService(iservices.ConsensusServerName)
	if err != nil {
		log.Error("[p2p] can't get other service, service name: ", iservices.ConsensusServerName)
		return
	}
	ctrl := s.(iservices.IConsensus)
	var remote_head_blk_id common.BlockID
	copy(remote_head_blk_id.Data[:], msgdata.HeadBlockId)
	current_head_blk_id := ctrl.GetHeadBlockId()

	start := remote_head_blk_id.BlockNum()
	end := current_head_blk_id.BlockNum()

	if start >= end {
		log.Debug("[p2p] no need to get ids, remote head block num: ", start, " current head block num: ", end)
		return
	}

	if end - start > msgCommon.MAX_ID_LENGTH {
		end = start + msgCommon.MAX_ID_LENGTH
	}

	log.Debug("[p2p] sync start num: ", start, " end num: ", end)

	beginTime := time.Now()
	blockList, err := ctrl.FetchBlocks(start, end)
	if err != nil {
		log.Error("[p2p] can't fetch blocks from consessus, start number: ", start, " end number: ",end, " error: ", err )
		return
	}
	endTime := time.Now()
	log.Debug("[p2p] consensus FetchBlocks cost time, start: ", beginTime, " end: ", endTime)
	if len(blockList) == 0 {
		log.Debug("[p2p] we have same blocks, no need to request from me")
		return
	}

	var ids []common.BlockID
	for i:=0;i<len(blockList);i++ {
		ids = append(ids, blockList[i].Id())
	}

	var reqmsg msgTypes.TransferMsg
	var idlength int
	reqdata := new(msgTypes.IdMsg)
	reqdata.Msgtype = msgTypes.IdMsg_request_id_ack

	if len(ids) <= msgCommon.MAX_ID_LENGTH {
		idlength = len(ids)
	} else {
		idlength = msgCommon.MAX_ID_LENGTH
	}

	for i:=0;i<idlength;i++ {
		var tmp []byte
		reqdata.Value = append(reqdata.Value, tmp)
		reqdata.Value[i] = make([]byte, prototype.Size)
		reqdata.Value[i] = ids[i].Data[:]
	}

	reqmsg.Msg = &msgTypes.TransferMsg_Msg2{Msg2:reqdata}
	err = p2p.Send(remotePeer, &reqmsg, false)
	if err != nil {
		log.Error("[p2p] send message error: ", err)
		return
	}
	//log.Info("[p2p] send a message to:   v%   data:   v%\n", remotePeer, reqmsg)

	commitEvidence := ctrl.GetNextBFTCheckPoint(remote_head_blk_id.BlockNum())
	if commitEvidence != nil {
		bftCommit := &msgTypes.ConsMsg {
			MsgData: commitEvidence.(*message.Commit),
		}
		err = p2p.Send(remotePeer, bftCommit, false)
		if err != nil {
			log.Error("[p2p] send message error: ", err)
			return
		}
	}
}

func (p *MsgHandler)ConsMsgHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	var msgdata = data.Payload.(*msgTypes.ConsMsg)

	log := p2p.GetLog()

	s, err := p2p.GetService(iservices.ConsensusServerName)
	if err != nil {
		log.Error("[p2p] can't get other service, service name: ", iservices.ConsensusServerName)
		return
	}
	ctrl := s.(iservices.IConsensus)

	log.Info("receive a consensus message, message data: ", msgdata)

	ctrl.Push(msgdata.MsgData)
}

func (p *MsgHandler) RequestCheckpointBatchHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	var raw = data.Payload.(*msgTypes.TransferMsg)
	var msgdata = raw.Msg.(*msgTypes.TransferMsg_Msg12).Msg12

	log := p2p.GetLog()

	remotePeer := p2p.GetPeerFromAddr(data.Addr)

	if remotePeer == nil {
		log.Error("[p2p] remotePeer invalid in RequestCheckpointBatchHandle")
		return
	}

	if !remotePeer.LockBusyFetchingCP() {
		log.Info("processing your former request, ignore this one")
		return
	} else {
		defer remotePeer.UnlockBusyFetchingCP()
	}


	log.Info("start checkpoint number: ", msgdata.Start, " end checkpoint number: ", msgdata.End)

	//s, err := p2p.GetService(iservices.ConsensusServerName)
	//if err != nil {
	//	log.Error("[p2p] can't get other service, service name: ", iservices.ConsensusServerName)
	//	return
	//}
	//ctrl := s.(iservices.IConsensus)

	startNum := msgdata.Start
	endNum := msgdata.End

	if endNum - startNum > msgCommon.MAX_ID_LENGTH {
		endNum = startNum + msgCommon.MAX_ID_LENGTH
	}
}