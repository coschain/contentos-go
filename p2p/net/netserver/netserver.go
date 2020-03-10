package netserver

import (
	"errors"
	common2 "github.com/coschain/contentos-go/common"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/p2p/common"
	"github.com/coschain/contentos-go/p2p/message/msg_pack"
	"github.com/coschain/contentos-go/p2p/message/types"
	"github.com/coschain/contentos-go/p2p/net/protocol"
	"github.com/coschain/contentos-go/p2p/peer"
	"github.com/sirupsen/logrus"
)

//NewNetServer return the net object in p2p
func NewNetServer(ctx *node.ServiceContext, lg *logrus.Logger) p2p.P2P {
	n := &NetServer{
		ctx:      ctx,
		log:      lg,
		SyncChan: make(chan *types.MsgPayload, common.CHAN_CAPABILITY),
		ConsChan: make(chan *types.MsgPayload, common.CHAN_CAPABILITY),
		NetworkMagic: common2.GetChainIdByName(ctx.Config().ChainId),
		msgCache: common.NewHashCache(common.DefaultHashCacheMaxCount * 50),
	}

	//n.PeerAddrMap.PeerSyncAddress = make(map[string]*peer.Peer)
	//n.PeerAddrMap.PeerConsAddress = make(map[string]*peer.Peer)
	//
	//n.init()

	return n
}

//NetServer represent all the actions in net layer
type NetServer struct {
	ctx          *node.ServiceContext
	log          *logrus.Logger
	base         peer.PeerCom
	synclistener net.Listener
	conslistener net.Listener
	SyncChan     chan *types.MsgPayload
	ConsChan     chan *types.MsgPayload
	ConnectingNodes
	PeerAddrMap
	Np            *peer.NbrPeers
	connectLock   sync.Mutex
	inConnRecord  InConnectionRecord
	outConnRecord OutConnectionRecord
	OwnAddress    string //network`s own address(ip : sync port),which get from version check
	NetworkMagic  uint32

	msgCache	 *common.HashCache

	startUpComplete bool
}

//InConnectionRecord include all addr connected
type InConnectionRecord struct {
	sync.RWMutex
	InConnectingAddrs []string
}

//OutConnectionRecord include all addr accepted
type OutConnectionRecord struct {
	sync.RWMutex
	OutConnectingAddrs []string
}

//ConnectingNodes include all addr in connecting state
type ConnectingNodes struct {
	sync.RWMutex
	ConnectingAddrs []string
}

//PeerAddrMap include all addr-peer list
type PeerAddrMap struct {
	sync.RWMutex
	PeerSyncAddress map[string]*peer.Peer
	PeerConsAddress map[string]*peer.Peer
}

//init initializes attribute of network server
func (this *NetServer) init() error {
	this.base.SetVersion(common.PROTOCOL_VERSION)

	if this.ctx.Config().P2P.EnableConsensus {
		this.base.SetServices(uint64(common.VERIFY_NODE))
	} else {
		this.base.SetServices(uint64(common.SERVICE_NODE))
	}

	if this.ctx.Config().P2P.NodePort == 0 {
		this.log.Error("[p2p] link port invalid")
		return errors.New("[p2p] invalid link port")
	}

	this.base.SetSyncPort ( uint32 ( this.ctx.Config().P2P.NodePort ) )

	if this.ctx.Config().P2P.DualPortSupport {
		if this.ctx.Config().P2P.NodeConsensusPort == 0 {
			this.log.Error("[p2p] consensus port invalid")
			return errors.New("[p2p] invalid consensus port")
		}

		this.base.SetConsPort ( uint32 ( this.ctx.Config().P2P.NodeConsensusPort ) )
	} else {
		this.base.SetConsPort(0)
	}

	this.base.SetRelay(true)

	rand.Seed(time.Now().UnixNano())
	id := rand.Uint64()

	this.base.SetID(id)

	this.log.Infof("[p2p] init peer ID to %d", this.base.GetID())
	this.Np = &peer.NbrPeers{Log:this.log}
	this.Np.Init()

	this.startUpComplete = true
	this.log.Info("net start ", len(this.inConnRecord.InConnectingAddrs), " ", len(this.outConnRecord.OutConnectingAddrs))

	return nil
}

func (this *NetServer) CheckStartUpFinished() bool {
	return this.startUpComplete
}

//InitListen start listening on the config port
func (this *NetServer) Start() {
	this.PeerAddrMap.PeerSyncAddress = make(map[string]*peer.Peer)
	this.PeerAddrMap.PeerConsAddress = make(map[string]*peer.Peer)
	this.inConnRecord.InConnectingAddrs = make([]string, 0)
	this.outConnRecord.OutConnectingAddrs = make([]string, 0)
	this.ConnectingAddrs = make([]string, 0)

	this.init()

	this.startListening()
}

//GetVersion return self peer`s version
func (this *NetServer) GetVersion() uint32 {
	return this.base.GetVersion()
}

//GetId return peer`s id
func (this *NetServer) GetID() uint64 {
	return this.base.GetID()
}

// SetHeight sets the local's height
func (this *NetServer) SetHeight(height uint64) {
	this.base.SetHeight(height)
}

// GetHeight return peer's heigh
func (this *NetServer) GetHeight() uint64 {
	return this.base.GetHeight()
}

//GetTime return the last contact time of self peer
func (this *NetServer) GetTime() int64 {
	t := time.Now()
	return t.UnixNano()
}

//GetServices return the service state of self peer
func (this *NetServer) GetServices() uint64 {
	return this.base.GetServices()
}

//GetSyncPort return the sync port
func (this *NetServer) GetSyncPort() uint32 {
	return this.base.GetSyncPort()
}

//GetConsPort return the cons port
func (this *NetServer) GetConsPort() uint32 {
	return this.base.GetConsPort()
}

//GetRelay return whether net module can relay msg
func (this *NetServer) GetRelay() bool {
	return this.base.GetRelay()
}

// GetPeer returns a peer with the peer id
func (this *NetServer) GetPeer(id uint64) *peer.Peer {
	return this.Np.GetPeer(id)
}

//return nbr peers collection
func (this *NetServer) GetNp() *peer.NbrPeers {
	return this.Np
}

//GetNeighborAddrs return all the nbr peer`s addr
func (this *NetServer) GetNeighborAddrs() []*types.PeerAddr {
	return this.Np.GetNeighborAddrs()
}

//GetConnectionCnt return the total number of valid connections
func (this *NetServer) GetConnectionCnt() uint32 {
	return this.Np.GetNbrNodeCnt()
}

//AddNbrNode add peer to nbr peer list
func (this *NetServer) AddNbrNode(remotePeer *peer.Peer) {
	this.Np.AddNbrNode(remotePeer)
}

//DelNbrNode delete nbr peer
func (this *NetServer) DelNbrNode(p *peer.Peer) (*peer.Peer, bool) {
	return this.Np.DelNbrNode(p)
}

//GetNeighbors return all nbr peer
func (this *NetServer) GetNeighbors() []*peer.Peer {
	return this.Np.GetNeighbors()
}

//NodeEstablished return whether a peer is establish with self according to id
func (this *NetServer) NodeEstablished(id uint64) bool {
	return this.Np.NodeEstablished(id)
}

func (this *NetServer) Broadcast(msg types.Message, isConsensus bool) {
	this.Np.Broadcast(msg, isConsensus, this.NetworkMagic)
}

//GetMsgChan return sync or consensus channel when msgrouter need msg input
func (this *NetServer) GetMsgChan(isConsensus bool) chan *types.MsgPayload {
	if isConsensus {
		return this.ConsChan
	} else {
		return this.SyncChan
	}
}

//Tx send data buf to peer
func (this *NetServer) Send(p *peer.Peer, msg types.Message, isConsensus bool) error {
	if p != nil {
		if this.ctx.Config().P2P.DualPortSupport == false {
			return p.Send(msg, false, this.NetworkMagic)
		}
		return p.Send(msg, isConsensus, this.NetworkMagic)
	}
	this.log.Warn("[p2p] send to a invalid peer")
	return errors.New("[p2p] send to a invalid peer")
}

//IsPeerEstablished return the establise state of given peer`s id
func (this *NetServer) IsPeerEstablished(p *peer.Peer) bool {
	if p != nil {
		return this.Np.NodeEstablished(p.GetID())
	}
	return false
}

//Connect used to connect net address under sync or cons mode
func (this *NetServer) Connect(addr string, isConsensus bool) error {
	this.connectLock.Lock()
	if added := this.AddOutConnectingList(addr); added == false {
		this.log.Debug("[p2p] node exist in connecting list ", addr)
		this.connectLock.Unlock()
		return nil
	}
	this.connectLock.Unlock()

	if this.IsAddrInOutConnRecord(addr) {
		this.RemoveFromConnectingList(addr)
		this.log.Debugf("[p2p] Address: %s Consensus: %v is in OutConnectionRecord,", addr, isConsensus)
		return nil
	}
	if this.IsOwnAddress(addr) {
		this.RemoveFromConnectingList(addr)
		return nil
	}
	if !this.AddrValid(addr) {
		this.RemoveFromConnectingList(addr)
		return nil
	}

	this.connectLock.Lock()
	connCount := uint(this.GetOutConnRecordLen())
	if connCount >= this.ctx.Config().P2P.MaxConnOutBound {
		this.log.Warnf("[p2p] Connect: out connections(%d) reach the max limit(%d)", connCount,
			this.ctx.Config().P2P.MaxConnOutBound)
		this.RemoveFromConnectingList(addr)
		this.connectLock.Unlock()
		return errors.New("[p2p] connect: out connections reach the max limit")
	}
	this.connectLock.Unlock()

	if this.IsNbrPeerAddr(addr, isConsensus) {
		this.RemoveFromConnectingList(addr)
		return nil
	}
	//this.connectLock.Lock()
	//if added := this.AddOutConnectingList(addr); added == false {
	//	this.log.Debug("[p2p] node exist in connecting list ", addr)
	//	this.connectLock.Unlock()
	//	return nil
	//}
	//this.connectLock.Unlock()

	isTls := this.ctx.Config().P2P.IsTLS
	var conn net.Conn
	var err error
	var remotePeer *peer.Peer
	if isTls {
		conn, err = TLSDial(addr, this.ctx.Config().P2P.CertPath, this.ctx.Config().P2P.KeyPath, this.ctx.Config().P2P.CAPath)
		if err != nil {
			this.log.Debugf("[p2p] connect %s failed:%s", addr, err.Error())
			this.RemoveFromConnectingList(addr)
			return err
		}
		if conn.RemoteAddr() == nil {
			this.log.Debug("conn.RemoteAddr() return nil")
			this.RemoveFromConnectingList(addr)
			return errors.New("conn.RemoteAddr() return nil")
		}
	} else {
		conn, err = nonTLSDial(addr)
		if err != nil {
			this.log.Debugf("[p2p] connect %s failed:%s", addr, err.Error())
			this.RemoveFromConnectingList(addr)
			return err
		}
		if conn.RemoteAddr() == nil {
			this.log.Debug("conn.RemoteAddr() return nil")
			this.RemoveFromConnectingList(addr)
			return errors.New("conn.RemoteAddr() return nil")
		}
	}

	addr = conn.RemoteAddr().String()
	this.log.Debugf("[p2p] peer %s connect with %s with %s",
		conn.LocalAddr().String(), conn.RemoteAddr().String(),
		conn.RemoteAddr().Network())

	if !isConsensus {
		this.AddOutConnRecord(addr)
		remotePeer = peer.NewPeer(this.log)
		this.AddPeerSyncAddress(addr, remotePeer)
		remotePeer.SyncLink.SetAddr(addr)
		remotePeer.SyncLink.SetConn(conn)
		remotePeer.AttachSyncChan(this.SyncChan)
		go remotePeer.SyncLink.Rx(this.NetworkMagic)
		go remotePeer.SyncLink.Tx(this.NetworkMagic)
		remotePeer.SetSyncState(common.HAND)

	} else {
		remotePeer = peer.NewPeer(this.log) //would merge with a exist peer in versionhandle
		this.AddPeerConsAddress(addr, remotePeer)
		remotePeer.ConsLink.SetAddr(addr)
		remotePeer.ConsLink.SetConn(conn)
		remotePeer.AttachConsChan(this.ConsChan)
		go remotePeer.ConsLink.Rx(this.NetworkMagic)
		go remotePeer.SyncLink.Tx(this.NetworkMagic)
		remotePeer.SetConsState(common.HAND)
	}

	//service, err := this.GetService(iservices.ConsensusServerName)
	//if err != nil {
	//	this.log.Error("[p2p] can't get other service, service name: ", iservices.ConsensusServerName)
	//	return err
	//}
	//ctrl := service.(iservices.IConsensus)
	//version := msgpack.NewVersion(this, isConsensus, ctrl.GetHeadBlockId().BlockNum())
	version := msgpack.NewVersion(this, isConsensus, uint64(0), this.ctx.Config().P2P.RunningCodeVersion )
	err = remotePeer.Send(version, isConsensus, this.NetworkMagic)
	if err != nil {
		if !isConsensus {
			this.RemoveFromOutConnRecord(addr)
			this.RemoveFromConnectingList(addr)
			this.RemovePeerSyncAddress(addr)
			remotePeer.CloseSync()
		}
		this.log.Error("[p2p] send message error: ", err)
		return err
	}
	return nil
}

//Halt stop all net layer logic
func (this *NetServer) Halt() {
	peers := this.Np.GetNeighbors()
	for _, p := range peers {
		p.CloseSync()
		p.CloseCons()
	}
	if this.synclistener != nil {
		this.synclistener.Close()
	}
	if this.conslistener != nil {
		this.conslistener.Close()
	}
	this.startUpComplete = false
}

//establishing the connection to remote peers and listening for inbound peers
func (this *NetServer) startListening() error {

	syncPort := this.base.GetSyncPort()
	//consPort := this.base.GetConsPort()

	if syncPort == 0 {
		this.log.Error("[p2p] sync port invalid")
		return errors.New("[p2p] sync port invalid")
	}

	err := this.startSyncListening(syncPort)
	if err != nil {
		this.log.Error("[p2p] start sync listening fail")
		return err
	}

	//consensus
	//if this.ctx.Config().P2P.DualPortSupport == false {
	//	this.log.Debug("[p2p] dual port mode not supported,keep single link")
	//	return nil
	//}
	//if consPort == 0 || consPort == syncPort {
	//	//still work
	//	this.log.Warn("[p2p] consensus port invalid,keep single link")
	//} else {
	//	err = this.startConsListening(consPort)
	//	if err != nil {
	//		return err
	//	}
	//}
	return nil
}

// startSyncListening starts a sync listener on the port for the inbound peer
func (this *NetServer) startSyncListening(port uint32) error {
	var err error
	this.synclistener, err = createListener(port,
											this.ctx.Config().P2P.IsTLS,
											this.ctx.Config().P2P.CertPath,
											this.ctx.Config().P2P.KeyPath,
											this.ctx.Config().P2P.CAPath)
	if err != nil {
		this.log.Error("[p2p] failed to create sync listener ", err)
		return errors.New("[p2p] failed to create sync listener")
	}

	go this.startSyncAccept(this.synclistener)
	this.log.Infof("[p2p] start listen on sync port %d", port)
	return nil
}

// startConsListening starts a sync listener on the port for the inbound peer
func (this *NetServer) startConsListening(port uint32) error {
	var err error
	this.conslistener, err = createListener(port,
											this.ctx.Config().P2P.IsTLS,
											this.ctx.Config().P2P.CertPath,
											this.ctx.Config().P2P.KeyPath,
											this.ctx.Config().P2P.CAPath)
	if err != nil {
		this.log.Error("[p2p] failed to create cons listener")
		return errors.New("[p2p] failed to create cons listener")
	}

	go this.startConsAccept(this.conslistener)
	this.log.Infof("[p2p] Start listen on consensus port %d", port)
	return nil
}

//startSyncAccept accepts the sync connection from the inbound peer
func (this *NetServer) startSyncAccept(listener net.Listener) {
	for {
		conn, err := listener.Accept()

		if err != nil {
			this.log.Error("[p2p] error accepting ", err.Error())
			return
		}

		if conn.RemoteAddr() == nil {
			this.log.Debug("conn.RemoteAddr() return nil")
			conn.Close()
			continue
		}

		this.log.Debug("[p2p] remote sync node connect with ",
			conn.RemoteAddr(), conn.LocalAddr())
		if !this.AddrValid(conn.RemoteAddr().String()) {
			this.log.Warnf("[p2p] remote %s not in reserved list, close it ", conn.RemoteAddr())
			conn.Close()
			continue
		}

		if this.IsAddrInInConnRecord(conn.RemoteAddr().String()) {
			conn.Close()
			continue
		}

		syncAddrCount := uint(this.GetInConnRecordLen())
		if syncAddrCount >= this.ctx.Config().P2P.MaxConnInBound {
			this.log.Warnf("[p2p] SyncAccept: total connections(%d) reach the max limit(%d), conn closed",
				syncAddrCount, this.ctx.Config().P2P.MaxConnInBound)
			conn.Close()
			continue
		}

		remoteIp, err := common.ParseIPAddr(conn.RemoteAddr().String())
		if err != nil {
			this.log.Warn("[p2p] parse ip error ", err.Error())
			conn.Close()
			continue
		}
		connNum := this.GetIpCountInInConnRecord(remoteIp)
		if connNum >= this.ctx.Config().P2P.MaxConnInBoundForSingleIP {
			this.log.Warnf("[p2p] SyncAccept: connections(%d) with ip(%s) has reach the max limit(%d), "+
				"conn closed", connNum, remoteIp, this.ctx.Config().P2P.MaxConnInBoundForSingleIP)
			conn.Close()
			continue
		}

		remotePeer := peer.NewPeer(this.log)
		addr := conn.RemoteAddr().String()
		this.AddInConnRecord(addr)

		this.AddPeerSyncAddress(addr, remotePeer)

		remotePeer.SyncLink.SetAddr(addr)
		remotePeer.SyncLink.SetConn(conn)
		remotePeer.AttachSyncChan(this.SyncChan)
		go remotePeer.SyncLink.Rx(this.NetworkMagic)
		go remotePeer.SyncLink.Tx(this.NetworkMagic)
	}
}

//startConsAccept accepts the consensus connnection from the inbound peer
func (this *NetServer) startConsAccept(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			this.log.Error("[p2p] error accepting ", err.Error())
			return
		}

		if conn.RemoteAddr() == nil {
			this.log.Debug("conn.RemoteAddr() return nil")
			conn.Close()
			continue
		}

		this.log.Debug("[p2p] remote cons node connect with ",
			conn.RemoteAddr(), conn.LocalAddr())
		if !this.AddrValid(conn.RemoteAddr().String()) {
			this.log.Warnf("[p2p] remote %s not in reserved list, close it ", conn.RemoteAddr())
			conn.Close()
			continue
		}

		remoteIp, err := common.ParseIPAddr(conn.RemoteAddr().String())
		if err != nil {
			this.log.Warn("[p2p] parse ip error ", err.Error())
			conn.Close()
			continue
		}
		if !this.IsIPInInConnRecord(remoteIp) {
			conn.Close()
			continue
		}

		remotePeer := peer.NewPeer(this.log)
		addr := conn.RemoteAddr().String()
		this.AddPeerConsAddress(addr, remotePeer)

		remotePeer.ConsLink.SetAddr(addr)
		remotePeer.ConsLink.SetConn(conn)
		remotePeer.AttachConsChan(this.ConsChan)
		go remotePeer.ConsLink.Rx(this.NetworkMagic)
		go remotePeer.SyncLink.Tx(this.NetworkMagic)
	}
}

//record the peer which is going to be dialed and sent version message but not in establish state
func (this *NetServer) AddOutConnectingList(addr string) (added bool) {
	this.ConnectingNodes.Lock()
	defer this.ConnectingNodes.Unlock()
	for _, a := range this.ConnectingAddrs {
		if strings.Compare(a, addr) == 0 {
			return false
		}
	}
	this.log.Debug("[p2p] add to out connecting list ", addr)
	this.ConnectingAddrs = append(this.ConnectingAddrs, addr)
	return true
}

//Remove the peer from connecting list if the connection is established
func (this *NetServer) RemoveFromConnectingList(addr string) {
	this.ConnectingNodes.Lock()
	defer this.ConnectingNodes.Unlock()
	addrs := this.ConnectingAddrs[:0]
	for _, a := range this.ConnectingAddrs {
		if a != addr {
			addrs = append(addrs, a)
		}
	}
	this.log.Debug("[p2p] remove from out connecting list ", addr)
	this.ConnectingAddrs = addrs
}

//record the peer which is going to be dialed and sent version message but not in establish state
func (this *NetServer) GetOutConnectingListLen() (count uint) {
	this.ConnectingNodes.RLock()
	defer this.ConnectingNodes.RUnlock()
	return uint(len(this.ConnectingAddrs))
}

//check  peer from connecting list
func (this *NetServer) IsAddrFromConnecting(addr string) bool {
	this.ConnectingNodes.Lock()
	defer this.ConnectingNodes.Unlock()
	for _, a := range this.ConnectingAddrs {
		if strings.Compare(a, addr) == 0 {
			return true
		}
	}
	return false
}

//find exist peer from addr map
func (this *NetServer) GetPeerFromAddr(addr string) *peer.Peer {
	var p *peer.Peer
	this.PeerAddrMap.RLock()
	defer this.PeerAddrMap.RUnlock()

	p, ok := this.PeerSyncAddress[addr]
	if ok {
		return p
	}
	p, ok = this.PeerConsAddress[addr]
	if ok {
		return p
	}
	return nil
}

//IsNbrPeerAddr return result whether the address is under connecting
func (this *NetServer) IsNbrPeerAddr(addr string, isConsensus bool) bool {
	var addrNew string
	this.Np.RLock()
	defer this.Np.RUnlock()
	for _, p := range this.Np.List {
		if p.GetSyncState() == common.HAND || p.GetSyncState() == common.HAND_SHAKE ||
			p.GetSyncState() == common.ESTABLISH {
			if isConsensus {
				addrNew = p.ConsLink.GetAddr()
			} else {
				addrNew = p.SyncLink.GetAddr()
			}
			if strings.Compare(addrNew, addr) == 0 {
				return true
			}
		}
	}
	return false
}

//AddPeerSyncAddress add sync addr to peer-addr map
func (this *NetServer) AddPeerSyncAddress(addr string, p *peer.Peer) {
	this.PeerAddrMap.Lock()
	defer this.PeerAddrMap.Unlock()
	this.log.Debugf("[p2p] AddPeerSyncAddress %s", addr)
	this.PeerSyncAddress[addr] = p
}

//AddPeerConsAddress add cons addr to peer-addr map
func (this *NetServer) AddPeerConsAddress(addr string, p *peer.Peer) {
	this.PeerAddrMap.Lock()
	defer this.PeerAddrMap.Unlock()
	this.log.Debugf("[p2p] AddPeerConsAddress %s", addr)
	this.PeerConsAddress[addr] = p
}

//RemovePeerSyncAddress remove sync addr from peer-addr map
func (this *NetServer) RemovePeerSyncAddress(addr string) {
	this.PeerAddrMap.Lock()
	defer this.PeerAddrMap.Unlock()
	if _, ok := this.PeerSyncAddress[addr]; ok {
		delete(this.PeerSyncAddress, addr)
		this.log.Debugf("[p2p] delete Sync Address %s", addr)
	}
}

//RemovePeerConsAddress remove cons addr from peer-addr map
func (this *NetServer) RemovePeerConsAddress(addr string) {
	this.PeerAddrMap.Lock()
	defer this.PeerAddrMap.Unlock()
	if _, ok := this.PeerConsAddress[addr]; ok {
		delete(this.PeerConsAddress, addr)
		this.log.Debugf("[p2p] delete Cons Address %s", addr)
	}
}

//GetPeerSyncAddressCount return length of cons addr from peer-addr map
func (this *NetServer) GetPeerSyncAddressCount() (count uint) {
	this.PeerAddrMap.RLock()
	defer this.PeerAddrMap.RUnlock()
	return uint(len(this.PeerSyncAddress))
}

//AddInConnRecord add in connection to inConnRecord
func (this *NetServer) AddInConnRecord(addr string) {
	this.inConnRecord.Lock()
	defer this.inConnRecord.Unlock()
	for _, a := range this.inConnRecord.InConnectingAddrs {
		if strings.Compare(a, addr) == 0 {
			return
		}
	}
	this.inConnRecord.InConnectingAddrs = append(this.inConnRecord.InConnectingAddrs, addr)
	this.log.Debugf("[p2p] add in record  %s", addr)
}

//IsAddrInInConnRecord return result whether addr is in inConnRecordList
func (this *NetServer) IsAddrInInConnRecord(addr string) bool {
	this.inConnRecord.RLock()
	defer this.inConnRecord.RUnlock()
	for _, a := range this.inConnRecord.InConnectingAddrs {
		if strings.Compare(a, addr) == 0 {
			return true
		}
	}
	return false
}

//IsIPInInConnRecord return result whether the IP is in inConnRecordList
func (this *NetServer) IsIPInInConnRecord(ip string) bool {
	this.inConnRecord.RLock()
	defer this.inConnRecord.RUnlock()
	var ipRecord string
	for _, addr := range this.inConnRecord.InConnectingAddrs {
		ipRecord, _ = common.ParseIPAddr(addr)
		if 0 == strings.Compare(ipRecord, ip) {
			return true
		}
	}
	return false
}

//RemoveInConnRecord remove in connection from inConnRecordList
func (this *NetServer) RemoveFromInConnRecord(addr string) {
	this.inConnRecord.Lock()
	defer this.inConnRecord.Unlock()
	addrs := []string{}
	for _, a := range this.inConnRecord.InConnectingAddrs {
		if strings.Compare(a, addr) != 0 {
			addrs = append(addrs, a)
		}
	}
	this.log.Debugf("[p2p] remove in record  %s", addr)
	this.inConnRecord.InConnectingAddrs = addrs
}

//GetInConnRecordLen return length of inConnRecordList
func (this *NetServer) GetInConnRecordLen() int {
	this.inConnRecord.RLock()
	defer this.inConnRecord.RUnlock()
	return len(this.inConnRecord.InConnectingAddrs)
}

//GetIpCountInInConnRecord return count of in connections with single ip
func (this *NetServer) GetIpCountInInConnRecord(ip string) uint {
	this.inConnRecord.RLock()
	defer this.inConnRecord.RUnlock()
	var count uint
	var ipRecord string
	for _, addr := range this.inConnRecord.InConnectingAddrs {
		ipRecord, _ = common.ParseIPAddr(addr)
		if 0 == strings.Compare(ipRecord, ip) {
			count++
		}
	}
	return count
}

//AddOutConnRecord add out connection to outConnRecord
func (this *NetServer) AddOutConnRecord(addr string) {
	this.outConnRecord.Lock()
	defer this.outConnRecord.Unlock()
	for _, a := range this.outConnRecord.OutConnectingAddrs {
		if strings.Compare(a, addr) == 0 {
			return
		}
	}
	this.outConnRecord.OutConnectingAddrs = append(this.outConnRecord.OutConnectingAddrs, addr)
	this.log.Debugf("[p2p] add out record  %s", addr)
}

//IsAddrInOutConnRecord return result whether addr is in outConnRecord
func (this *NetServer) IsAddrInOutConnRecord(addr string) bool {
	this.outConnRecord.RLock()
	defer this.outConnRecord.RUnlock()
	for _, a := range this.outConnRecord.OutConnectingAddrs {
		if strings.Compare(a, addr) == 0 {
			return true
		}
	}
	return false
}

//RemoveOutConnRecord remove out connection from outConnRecord
func (this *NetServer) RemoveFromOutConnRecord(addr string) {
	this.outConnRecord.Lock()
	defer this.outConnRecord.Unlock()
	addrs := []string{}
	for _, a := range this.outConnRecord.OutConnectingAddrs {
		if strings.Compare(a, addr) != 0 {
			addrs = append(addrs, a)
		}
	}
	this.log.Debugf("[p2p] remove out record  %s", addr)
	this.outConnRecord.OutConnectingAddrs = addrs
}

//GetOutConnRecordLen return length of outConnRecord
func (this *NetServer) GetOutConnRecordLen() int {
	this.outConnRecord.RLock()
	defer this.outConnRecord.RUnlock()
	return len(this.outConnRecord.OutConnectingAddrs)
}

//AddrValid whether the addr could be connect or accept
func (this *NetServer) AddrValid(addr string) bool {
	if this.ctx.Config().P2P.ReservedPeersOnly && len( this.ctx.Config().P2P.ReservedCfg.ReservedPeers ) > 0 {
		for _, ip := range this.ctx.Config().P2P.ReservedCfg.ReservedPeers {
			if strings.HasPrefix(addr, ip) {
				this.log.Info("[p2p] found reserved peer :", addr)
				return true
			}
		}
		return false
	}
	return true
}

//check own network address
func (this *NetServer) IsOwnAddress(addr string) bool {
	if addr == this.OwnAddress {
		return true
	}
	return false
}

//Set own network address
func (this *NetServer) SetOwnAddress(addr string) {
	if addr != this.OwnAddress {
		this.log.Infof("[p2p] set own address %s", addr)
		this.OwnAddress = addr
	}

}

func (this *NetServer) GetService(str string) (interface{}, error) {
	s, err := this.ctx.Service(str)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (this *NetServer) GetContex() *node.ServiceContext{
	return this.ctx
}

func (this *NetServer) GetLog() *logrus.Logger{
	return this.log
}

func (this *NetServer) GetMagic() uint32 {
	return this.NetworkMagic
}

func (this *NetServer) RememberMsg(hash [common.HashSize]byte) bool {
	return this.msgCache.PutIfNotFound(hash)
}

//reqNbrList ask the peer for its neighbor list
func (this *NetServer) ReqNbrList(p *peer.Peer, randomSignal bool) {
	// open random signal and random select not pass
	if randomSignal && !common.RandomSelect(common.OneOfThree) {
		return
	}

	now := time.Now().Unix()
	if p.ReqNbrList.LastAskTime == 0 ||
		( p.ReqNbrList.LastAskTime != 0 && (now - p.ReqNbrList.LastAskTime > common.KEEPALIVE_TIMEOUT) ) {
		rand.Seed(time.Now().UnixNano())
		authNumber := rand.Uint64()

		// set require neighbours state
		p.ReqNbrList.Lock()
		p.ReqNbrList.LastAskTime = now
		p.ReqNbrList.AuthNumber = authNumber
		p.ReqNbrList.Unlock()

		msg := msgpack.NewAddrReq(authNumber)
		go this.Send(p, msg, false)
	}
}
