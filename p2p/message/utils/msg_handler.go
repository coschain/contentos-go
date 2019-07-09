package utils

import (
	"github.com/coschain/gobft/message"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/iservices"
	msgCommon "github.com/coschain/contentos-go/p2p/common"
	"github.com/coschain/contentos-go/p2p/message/msg_pack"
	msgTypes "github.com/coschain/contentos-go/p2p/message/types"
	"github.com/coschain/contentos-go/p2p/net/protocol"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/p2p/peer"
)

type MsgHandler struct {
	blockCache map[common.BlockID]common.ISignedBlock
	sync.Mutex

	syncPushBlock sync.Mutex
}

func NewMsgHandler() *MsgHandler {
	blockCache := make(map[common.BlockID]common.ISignedBlock)

	return &MsgHandler{blockCache: blockCache, syncPushBlock: sync.Mutex{}}
}

func (p *MsgHandler) popFirstBlock() common.ISignedBlock {

	var retV common.ISignedBlock = nil
	var retK = common.EmptyBlockID

	for k, v := range p.blockCache {

		if retK == common.EmptyBlockID {
			retK = k
			retV = v
		} else if k.BlockNum() < retK.BlockNum() {
			retK = k
			retV = v
		}
	}

	if retK != common.EmptyBlockID {
		delete(p.blockCache, retK)
	}

	return retV
}

// AddrReqHandle handles the neighbor address request from peer
func (p *MsgHandler) AddrReqHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
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
func (p *MsgHandler) PingHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
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
func (p *MsgHandler) PongHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
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

func (p *MsgHandler) BlockSyncHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {

	var raw = data.Payload.(*msgTypes.TransferMsg)
	var block = raw.Msg.(*msgTypes.TransferMsg_Msg3).Msg3

	log := p2p.GetLog()
	log.Info("[p2p] receive a SignedBlock msg, block number :   ", block.SigBlk.Id().BlockNum())
	blkNum := block.SigBlk.Id().BlockNum()

	remotePeer := p2p.GetPeer(data.Id)
	if remotePeer == nil {
		log.Error("[p2p] peer is not exist: ", data.Addr)
		return
	}
	remotePeer.SetLastSeenBlkNum(blkNum)

	p.Lock()
	p.blockCache[block.SigBlk.Id()] = block.SigBlk
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

	go func(){
		maybeTriggerFetch(p2p, log, remotePeer, block)
	}()
}

func (p *MsgHandler) blockHandle(ctrl iservices.IConsensus) {

	p.syncPushBlock.Lock()
	defer p.syncPushBlock.Unlock()

	p.Lock()
	block := p.popFirstBlock()
	p.Unlock()

	if block != nil {
		if ctrl.HasBlock(block.Id()) {
			return
		}
		ctrl.PushBlock(block)
	}
}

func maybeTriggerFetch(p2p p2p.P2P, lg *logrus.Logger, remotePeer *peer.Peer, sigBlkMsg *msgTypes.SigBlkMsg) {
	if !sigBlkMsg.NeedTriggerFetch {
		//lg.Info("no need to trigger fetch block batch")
		return
	}

	remotePeer.OutOfRangeState.Lock()
	defer remotePeer.OutOfRangeState.Unlock()

	if len(remotePeer.OutOfRangeState.KeyPointIDList) == 0 {
		lg.Error("remotePeer OutOfRangeState KeyPointIDList length should not be 0")
	} else if len(remotePeer.OutOfRangeState.KeyPointIDList) == 1 {
		remotePeer.OutOfRangeState.KeyPointIDList = remotePeer.OutOfRangeState.KeyPointIDList[:0]
		lg.Info("all gap blocks fetch over")
	} else {
		length := len(remotePeer.OutOfRangeState.KeyPointIDList)
		startId := remotePeer.OutOfRangeState.KeyPointIDList[length-1]
		endId := remotePeer.OutOfRangeState.KeyPointIDList[length-2]

		var endBlockID common.BlockID
		copy(endBlockID.Data[:], endId)
		if endBlockID.BlockNum() != sigBlkMsg.SigBlk.Id().BlockNum() {
			lg.Warn("receive a fake sigblk msg")
			return
		}

		remotePeer.OutOfRangeState.KeyPointIDList = remotePeer.OutOfRangeState.KeyPointIDList[0:length-1]

		msg := msgpack.NewRequestBlockBatch(startId, endId)
		err := p2p.Send(remotePeer, msg, false)
		if err != nil {
			lg.Error("[p2p] send message error: ", err)
		}
	}
}

// TransactionHandle handles the transaction message from peer
func (p *MsgHandler) TransactionHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	var raw = data.Payload.(*msgTypes.TransferMsg)
	var trn = raw.Msg.(*msgTypes.TransferMsg_Msg1).Msg1

	log := p2p.GetLog()
	//log.Info("receive a SignedTransaction msg: ", trn)

	id, _ := trn.SigTrx.Id()
	remotePeer := p2p.GetPeer(data.Id)
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
func (p *MsgHandler) VersionHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
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
			msg = msgpack.NewVersion(p2p, true, uint64(0), ctx.Config().P2P.RunningCodeVersion)
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
			msg = msgpack.NewVersion(p2p, false, uint64(0), ctx.Config().P2P.RunningCodeVersion)
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
func (p *MsgHandler) VerAckHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
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
func (p *MsgHandler) AddrHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
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
func (p *MsgHandler) DisconnectHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
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

func (p *MsgHandler) IdMsgHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	var raw = data.Payload.(*msgTypes.TransferMsg)
	var msgdata = raw.Msg.(*msgTypes.TransferMsg_Msg2).Msg2

	log := p2p.GetLog()

	remotePeer := p2p.GetPeer(data.Id)

	if remotePeer == nil {
		log.Error("[p2p] remotePeer invalid in IdMsgHandle")
		return
	}

	s, err := p2p.GetService(iservices.ConsensusServerName)
	if err != nil {
		log.Error("[p2p] can't get other service, service name: ", iservices.ConsensusServerName)
		return
	}
	ctrl := s.(iservices.IConsensus)

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

		log.Info("[p2p] receive a broadcast_sigblk_id msg block number ", blkId.BlockNum())
		start := time.Now()

		if !ctrl.HasBlock(blkId) {
			var reqmsg msgTypes.TransferMsg
			reqdata := new(msgTypes.IdMsg)
			reqdata.Msgtype = msgTypes.IdMsg_request_sigblk_by_id
			var tmp []byte
			reqdata.Value = append(reqdata.Value, tmp)
			reqdata.Value[0] = msgdata.Value[0]

			reqmsg.Msg = &msgTypes.TransferMsg_Msg2{Msg2: reqdata}

			log.Infof("[p2p] send a request_sigblk_by_id msg block number %d cost time %v", blkId.BlockNum(), time.Now().Sub(start))
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

			log.Info("[p2p] receive a request_sigblk_by_id msg block number ", blkId.BlockNum())
			start := time.Now()

			IsigBlk, err := ctrl.FetchBlock(blkId)
			if err != nil {
				log.Error("[p2p] can't get IsigBlk from consensus, block number: ", blkId.BlockNum(), " error: ", err)
				return
			}
			sigBlk := IsigBlk.(*prototype.SignedBlock)

			msg := msgpack.NewSigBlk(sigBlk, false)
			log.Infof("[p2p] send a sigblk block number %d cost time %v", blkId.BlockNum(), time.Now().Sub(start))
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

			p.Lock()
			_, existInCache := p.blockCache[blkId]
			p.Unlock()

			if existInCache {
				continue
			}

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
		reqmsg.Msg = &msgTypes.TransferMsg_Msg2{Msg2: reqdata}
		err := p2p.Send(remotePeer, &reqmsg, false)
		if err != nil {
			log.Error("[p2p] send message error: ", err)
			return
		}
		//log.Infof("send a message to:   v%   data:   v%\n", remotePeer, reqmsg)
	case msgTypes.IdMsg_detect_former_ids:
		for idx, id := range msgdata.Value {
			var blkId common.BlockID
			copy(blkId.Data[:], id)

			//p.Lock()
			//_, existInCache := p.blockCache[blkId]
			//p.Unlock()
			//
			//if existInCache {
			//	continue
			//}

			if !ctrl.HasBlock(blkId) {
				if idx == 0 {
					remotePeer.OutOfRangeState.Lock()
					remotePeer.OutOfRangeState.KeyPointIDList = append(remotePeer.OutOfRangeState.KeyPointIDList, id)
					remotePeer.OutOfRangeState.Unlock()

					msg := msgpack.NewDetectFormerIds(id)
					err = p2p.Send(remotePeer, msg, false)
					if err != nil {
						log.Error("[p2p] send message error: ", err)
					}
					return
				} else {
					remotePeer.OutOfRangeState.Lock()
					length := len(remotePeer.OutOfRangeState.KeyPointIDList)
					endId := remotePeer.OutOfRangeState.KeyPointIDList[length-1]
					remotePeer.OutOfRangeState.Unlock()

					msg := msgpack.NewRequestBlockBatch(msgdata.Value[idx-1], endId)
					err = p2p.Send(remotePeer, msg, false)
					if err != nil {
						log.Error("[p2p] send message error: ", err)
					}
					return
				}
			}

			if idx == len(msgdata.Value)-1 {
				remotePeer.OutOfRangeState.Lock()
				length := len(remotePeer.OutOfRangeState.KeyPointIDList)
				endId := remotePeer.OutOfRangeState.KeyPointIDList[length-1]
				remotePeer.OutOfRangeState.Unlock()

				msg := msgpack.NewRequestBlockBatch(msgdata.Value[idx], endId)
				err = p2p.Send(remotePeer, msg, false)
				if err != nil {
					log.Error("[p2p] send message error: ", err)
					return
				}
			}
		}
	default:
		log.Warnf("[p2p] Unknown id message %v", msgdata)
	}
}

func (p *MsgHandler) ReqIdHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	var raw = data.Payload.(*msgTypes.TransferMsg)
	var msgdata = raw.Msg.(*msgTypes.TransferMsg_Msg4).Msg4
	remotePeer := p2p.GetPeer(data.Id)

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

	var blocksSize int
	var ids []common.BlockID
	stopFetchBlock := false
	batchStart := start
	blockCount := 0
	for {
		if batchStart > end {
			break
		}
		batchEnd := batchStart + msgCommon.BATCH_LENGTH
		if batchEnd > end {
			batchEnd = end
		}

		beginTime := time.Now()
		blockList, err := ctrl.FetchBlocks(batchStart, batchEnd)
		if err != nil {
			log.Error("[p2p] can't fetch blocks from consessus, start number: ", start, " end number: ", end, " error: ", err)
			return
		}
		endTime := time.Now()
		if len(blockList) == 0 {
			log.Errorf("[p2p] consensus can't fetch blocks from %d to %d", batchStart, batchEnd)
			return
		}
		log.Debugf("[p2p] consensus fetch block batch from %d to %d, cost time %v", batchStart, batchEnd, endTime.Sub(beginTime))

		for i := 0; i < len(blockList); i++ {
			sigBlk := blockList[i].(*prototype.SignedBlock)
			if blocksSize + sigBlk.GetBlockSize() <= msgCommon.BLOCKS_SIZE_LIMIT && blockCount + 1 <= msgCommon.MAX_BLOCK_COUNT {
				ids = append(ids, blockList[i].Id())
				blocksSize += sigBlk.GetBlockSize()
				blockCount++
			} else {
				stopFetchBlock = true
				break
			}
		}

		if stopFetchBlock {
			break
		}
		batchStart = batchEnd + 1
	}

	if len(ids) == 0 {
		log.Warn("[p2p] fetch no block from consensus, maybe one block is too big")
		return
	}

	var reqmsg msgTypes.TransferMsg
	reqdata := new(msgTypes.IdMsg)
	reqdata.Msgtype = msgTypes.IdMsg_request_id_ack

	for i := 0; i < len(ids); i++ {
		var tmp []byte
		reqdata.Value = append(reqdata.Value, tmp)
		reqdata.Value[i] = make([]byte, prototype.Size)
		reqdata.Value[i] = ids[i].Data[:]
	}

	reqmsg.Msg = &msgTypes.TransferMsg_Msg2{Msg2: reqdata}
	err = p2p.Send(remotePeer, &reqmsg, false)
	if err != nil {
		log.Error("[p2p] send message error: ", err)
		return
	}
	//log.Info("[p2p] send a message to:   v%   data:   v%\n", remotePeer, reqmsg)
}

func (p *MsgHandler) ConsMsgHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	var msgdata = data.Payload.(*msgTypes.ConsMsg)

	log := p2p.GetLog()
	remotePeer := p2p.GetPeer(data.Id)
	if remotePeer == nil {
		log.Error("[p2p] remotePeer invalid in ConsMsgHandle")
		return
	}

	if msgdata.Bcast == 1 {
		hash := msgdata.Hash()
		if remotePeer.HasConsensusMsg(hash) {
			//log.Info("[p2p] we alerady have this consensus msg, msg hash: ", hash)
			return
		}
		//log.Info("receive a consensus hash: ", hash)
		remotePeer.RecordConsensusMsg(hash)
	}

	s, err := p2p.GetService(iservices.ConsensusServerName)
	if err != nil {
		log.Error("[p2p] can't get other service, service name: ", iservices.ConsensusServerName)
		return
	}
	ctrl := s.(iservices.IConsensus)

	//log.Info("receive a consensus message, message data: ", msgdata)

	ctrl.Push(msgdata.MsgData)

	if msgdata.Bcast == 1 {
		//log.Info("forward broadcast consensus msg")
		p2p.Broadcast(msgdata, false)
	}
}

func (p *MsgHandler) RequestCheckpointBatchHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	var raw = data.Payload.(*msgTypes.TransferMsg)
	var msgdata = raw.Msg.(*msgTypes.TransferMsg_Msg12).Msg12

	log := p2p.GetLog()

	remotePeer := p2p.GetPeer(data.Id)

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

	s, err := p2p.GetService(iservices.ConsensusServerName)
	if err != nil {
		log.Error("[p2p] can't get other service, service name: ", iservices.ConsensusServerName)
		return
	}
	ctrl := s.(iservices.IConsensus)

	startNum := msgdata.Start
	endNum := msgdata.End
	if endNum-startNum > msgCommon.BATCH_LENGTH {
		endNum = startNum + msgCommon.BATCH_LENGTH
	}
	log.Infof("RequestCheckpointBatchHandle from %d to %d", startNum, endNum)
	for {
		if startNum >= endNum {
			return
		}
		cp := ctrl.GetNextBFTCheckPoint(startNum)
		if cp == nil {
			return
		}
		bftCommitCP := &msgTypes.ConsMsg{
			MsgData: cp.(*message.Commit),
		}
		err = p2p.Send(remotePeer, bftCommitCP, false)
		if err != nil {
			log.Error("[p2p] send message error: ", err)
			return
		}
		log.Debug("sending cp ", ExtractBlockID(cp.(*message.Commit)).BlockNum())
		startNum = ExtractBlockID(cp.(*message.Commit)).BlockNum()
	}
}

func ExtractBlockID(commit *message.Commit) common.BlockID {
	return common.BlockID{
		Data: commit.ProposedData,
	}
}

func (p *MsgHandler) FetchOutOfRangeHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	var raw = data.Payload.(*msgTypes.TransferMsg)
	var msgdata = raw.Msg.(*msgTypes.TransferMsg_Msg13).Msg13

	log := p2p.GetLog()

	remotePeer := p2p.GetPeer(data.Id)
	if remotePeer == nil {
		log.Error("[p2p] remotePeer invalid in FetchOutOfRangeHandle")
		return
	}

	var startID, targetID common.BlockID
	copy(startID.Data[:], msgdata.StartId)
	copy(targetID.Data[:], msgdata.TargetId)

	if startID.BlockNum() >= targetID.BlockNum() {
		log.Debug("[p2p] no need to call FetchOutOfRangeHandle method, start num: ", startID.BlockNum(), " target num: ", targetID.BlockNum() )
		clearMsg := msgpack.NewClearOutOfRangeState()
		p2p.Send(remotePeer, clearMsg, false)
		return
	}
	log.Info("request out-of-range ids, start number: ", startID.BlockNum(), " end number: ", targetID.BlockNum())

	s, err := p2p.GetService(iservices.ConsensusServerName)
	if err != nil {
		log.Error("[p2p] can't get other service, service name: ", iservices.ConsensusServerName)
		clearMsg := msgpack.NewClearOutOfRangeState()
		p2p.Send(remotePeer, clearMsg, false)
		return
	}
	ctrl := s.(iservices.IConsensus)

	ret1, err := ctrl.IsOnMainBranch(startID)
	if err != nil {
		log.Error("can not check whether startID on main branch, ", err)
		ret1 = false
	}

	ret2, err := ctrl.IsOnMainBranch(targetID)
	if err != nil {
		log.Error("can not check whether targetID on main branch, ", err)
		ret2 = false
	}

	ret2 = false

	if ret1 && ret2 {
		log.Info("fetch out of range blocks from main branch")
		startNum := startID.BlockNum()
		endNum := targetID.BlockNum()

		var blockList []common.ISignedBlock
		batchStart := startNum
		blocksSize := 0
		blockCount := 0
		stopFetchBlock := false
		for {
			if batchStart > endNum {
				break
			}
			batchEnd := batchStart + msgCommon.BATCH_LENGTH
			if batchEnd > endNum {
				batchEnd = endNum
			}

			blockBatchList, err := ctrl.FetchBlocks(batchStart, batchEnd)
			if err != nil {
				log.Error("[p2p] can't fetch blocks from consessus, start number: ", batchStart, " end number: ", batchEnd, " error: ", err)
				clearMsg := msgpack.NewClearOutOfRangeState()
				p2p.Send(remotePeer, clearMsg, false)
				return
			}
			if len(blockBatchList) == 0 {
				log.Debug("[p2p] we have same blocks, no need to request from me")
				clearMsg := msgpack.NewClearOutOfRangeState()
				p2p.Send(remotePeer, clearMsg, false)
				return
			}

			for i := 0; i < len(blockBatchList); i++ {
				sigBlk := blockBatchList[i].(*prototype.SignedBlock)
				if blocksSize + sigBlk.GetBlockSize() <= msgCommon.BLOCKS_SIZE_LIMIT && blockCount + 1 <= msgCommon.MAX_BLOCK_COUNT {
					blockList = append(blockList, blockBatchList[i])
					blocksSize += sigBlk.GetBlockSize()
					blockCount++
				} else {
					stopFetchBlock = true
					break
				}
			}

			if stopFetchBlock {
				break
			}
			batchStart = batchEnd + 1
		}

		if len(blockList) == 0 {
			log.Warn("[p2p] fetch no block from consensus, maybe one block is too big")
			return
		}

		for i:=0;i<len(blockList);i++ {
			sigBlk := blockList[i].(*prototype.SignedBlock)

			var msg msgTypes.Message
			if i == len(blockList) - 1 {
				msg = msgpack.NewSigBlk(sigBlk, true)
			} else {
				msg = msgpack.NewSigBlk(sigBlk, false)
			}

			err = p2p.Send(remotePeer, msg, false)
			if err != nil {
				log.Error("[p2p] send message error: ", err)
				return
			}
		}
	} else {
		log.Info("fetch out of range blocks from lateral branch")
		count := 0
		blkId := targetID
		var IDList [][]byte

		for {
			IDList = append(IDList, blkId.Data[:])
			count++
			if count == msgCommon.BATCH_LENGTH || blkId == startID {
				break
			}

			IsigBlk, err := ctrl.FetchBlock(blkId)
			if err != nil {
				log.Error("[p2p] can't get IsigBlk from consensus, block number: ", blkId.BlockNum(), " error: ", err)
				clearMsg := msgpack.NewClearOutOfRangeState()
				p2p.Send(remotePeer, clearMsg, false)
				return
			}
			blkId = IsigBlk.Previous()
		}

		var reqmsg msgTypes.TransferMsg
		reqdata := new(msgTypes.IdMsg)
		reqdata.Msgtype = msgTypes.IdMsg_detect_former_ids

		for i:=len(IDList)-1;i>=0;i-- {
			reqdata.Value = append(reqdata.Value, IDList[i])
		}

		reqmsg.Msg = &msgTypes.TransferMsg_Msg2{Msg2: reqdata}
		err = p2p.Send(remotePeer, &reqmsg, false)
		if err != nil {
			log.Error("[p2p] send message error: ", err)
			return
		}

	}
}

func (p *MsgHandler) RequestBlockBatchHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	var raw = data.Payload.(*msgTypes.TransferMsg)
	var msgdata = raw.Msg.(*msgTypes.TransferMsg_Msg14).Msg14

	log := p2p.GetLog()

	remotePeer := p2p.GetPeer(data.Id)
	if remotePeer == nil {
		log.Error("[p2p] remotePeer invalid in RequestBlockBatchHandle")
		return
	}

	var startID, endID common.BlockID
	copy(startID.Data[:], msgdata.StartId)
	copy(endID.Data[:], msgdata.EndId)

	if endID.BlockNum() - startID.BlockNum() > msgCommon.BATCH_LENGTH {
		log.Error("[p2p] block batch length beyond limit ", msgCommon.BATCH_LENGTH)
		clearMsg := msgpack.NewClearOutOfRangeState()
		p2p.Send(remotePeer, clearMsg, false)
		return
	}

	blkId := endID
	var IsigBlkList []common.ISignedBlock

	s, err := p2p.GetService(iservices.ConsensusServerName)
	if err != nil {
		log.Error("[p2p] can't get other service, service name: ", iservices.ConsensusServerName)
		clearMsg := msgpack.NewClearOutOfRangeState()
		p2p.Send(remotePeer, clearMsg, false)
		return
	}
	ctrl := s.(iservices.IConsensus)

	for {
		if blkId == startID {
			break
		}
		IsigBlk, err := ctrl.FetchBlock(blkId)
		if err != nil {
			log.Error("[p2p] can't get IsigBlk from consensus, block number: ", blkId.BlockNum(), " error: ", err)
			clearMsg := msgpack.NewClearOutOfRangeState()
			p2p.Send(remotePeer, clearMsg, false)
			return
		}
		IsigBlkList = append(IsigBlkList, IsigBlk)

		blkId = IsigBlk.Previous()
	}

	if len(IsigBlkList) == 0 {
		log.Error("[p2p] get no batch block")
		clearMsg := msgpack.NewClearOutOfRangeState()
		p2p.Send(remotePeer, clearMsg, false)
		return
	}

	for i:=len(IsigBlkList)-1;i>=0;i-- {
		sigBlk := IsigBlkList[i].(*prototype.SignedBlock)

		var msg msgTypes.Message
		if i == 0 {
			msg = msgpack.NewSigBlk(sigBlk, true)
		} else {
			msg = msgpack.NewSigBlk(sigBlk, false)
		}

		err = p2p.Send(remotePeer, msg, false)
		if err != nil {
			log.Error("[p2p] send message error: ", err)
			return
		}
	}
}

func (p *MsgHandler) DetectFormerIdsHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	var raw = data.Payload.(*msgTypes.TransferMsg)
	var msgdata = raw.Msg.(*msgTypes.TransferMsg_Msg15).Msg15

	log := p2p.GetLog()

	remotePeer := p2p.GetPeer(data.Id)
	if remotePeer == nil {
		log.Error("[p2p] remotePeer invalid in DetectFormerIdsHandle")
		return
	}

	s, err := p2p.GetService(iservices.ConsensusServerName)
	if err != nil {
		log.Error("[p2p] can't get other service, service name: ", iservices.ConsensusServerName)
		clearMsg := msgpack.NewClearOutOfRangeState()
		p2p.Send(remotePeer, clearMsg, false)
		return
	}
	ctrl := s.(iservices.IConsensus)

	var endID common.BlockID
	copy(endID.Data[:], msgdata.EndId)
	blkId := endID

	count := 0
	var reqmsg msgTypes.TransferMsg
	reqdata := new(msgTypes.IdMsg)
	reqdata.Msgtype = msgTypes.IdMsg_detect_former_ids

	for {
		reqdata.Value = append(reqdata.Value, blkId.Data[:])
		count++
		if count == msgCommon.BATCH_LENGTH {
			break
		}

		IsigBlk, err := ctrl.FetchBlock(blkId)
		if err != nil {
			log.Error("[p2p] can't get IsigBlk from consensus, block number: ", blkId.BlockNum(), " error: ", err)
			clearMsg := msgpack.NewClearOutOfRangeState()
			p2p.Send(remotePeer, clearMsg, false)
			return
		}
		blkId = IsigBlk.Previous()
	}

	reqmsg.Msg = &msgTypes.TransferMsg_Msg2{Msg2: reqdata}
	err = p2p.Send(remotePeer, &reqmsg, false)
	if err != nil {
		log.Error("[p2p] send message error: ", err)
		return
	}
}

func (p *MsgHandler) ClearOutOfRangeStateHandle(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	log := p2p.GetLog()

	remotePeer := p2p.GetPeer(data.Id)
	if remotePeer == nil {
		log.Error("[p2p] remotePeer invalid in ClearOutOfRangeStateHandle")
		return
	}

	log.Info("clear local peer OutOfRangeState")

	remotePeer.OutOfRangeState.Lock()
	remotePeer.OutOfRangeState.KeyPointIDList = remotePeer.OutOfRangeState.KeyPointIDList[:0]
	remotePeer.OutOfRangeState.Unlock()
}
