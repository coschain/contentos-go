package p2p

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"reflect"
	"strconv"
	"sync"
	"time"

	coomn "github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/p2p/common"
	"github.com/coschain/contentos-go/p2p/message/msg_pack"
	msgtypes "github.com/coschain/contentos-go/p2p/message/types"
	"github.com/coschain/contentos-go/p2p/message/utils"
	"github.com/coschain/contentos-go/p2p/net/netserver"
	"github.com/coschain/contentos-go/p2p/net/protocol"
	"github.com/coschain/contentos-go/p2p/peer"
	"github.com/coschain/contentos-go/prototype"
	consmsg "github.com/coschain/gobft/message"
	"github.com/sirupsen/logrus"
)

//P2PServer control all network activities
type P2PServer struct {
	iservices.IP2P
	Network   p2p.P2P
	msgRouter *utils.MessageRouter
	ReconnectAddrs
	quitOnline     chan bool
	quitHeartBeat  chan bool
	mockLatency  int

	ctx *node.ServiceContext
	log *logrus.Logger
}

//ReconnectAddrs contain addr need to reconnect
type ReconnectAddrs struct {
	sync.RWMutex
	RetryAddrs map[string]int
}

//NewServer return a new p2pserver according to the pubkey
func NewServer(ctx *node.ServiceContext, lg *logrus.Logger) (*P2PServer, error) {
	if lg == nil {
		lg = logrus.New()
		lg.SetOutput(ioutil.Discard)
	}
	n := netserver.NewNetServer(ctx, lg)

	p := &P2PServer{
		Network: n,
	}

	p.log = lg
	p.ctx = ctx
	p.msgRouter = utils.NewMsgRouter(p.Network)
	p.quitOnline = make(chan bool)
	p.quitHeartBeat = make(chan bool)
	return p, nil
}

//GetConnectionCnt return the established connect count
//func (this *P2PServer) GetConnectionCnt() uint32 {
//	return this.Network.GetConnectionCnt()
//}

//Start create all services
func (this *P2PServer) Start(node *node.Node) error {
	if this.Network != nil {
		this.Network.Start()
	} else {
		return errors.New("[p2p]network invalid")
	}
	if this.msgRouter != nil {
		this.msgRouter.Start()
	} else {
		return errors.New("[p2p]msg router invalid")
	}
	go this.connectSeedService()
	go this.keepOnlineService()
	go this.heartBeatService()
	return nil
}

//Stop halt all service by send signal to channels
func (this *P2PServer) Stop() error {
	this.Network.Halt()
	this.quitOnline <- true
	this.quitHeartBeat <- true
	this.msgRouter.Stop()
	return nil
}


//GetPort return two network port
//func (this *P2PServer) GetPort() (uint32, uint32) {
//	return this.Network.GetSyncPort(), this.Network.GetConsPort()
//}

//GetVersion return self version
//func (this *P2PServer) GetVersion() uint32 {
//	return this.Network.GetVersion()
//}

//GetNeighborAddrs return all nbr`s address
//func (this *P2PServer) GetNeighborAddrs() []*msgtypes.PeerAddr {
//	return this.Network.GetNeighborAddrs()
//}

//Send tranfer buffer to peer
func (this *P2PServer) Send(p *peer.Peer, msg msgtypes.Message,
	isConsensus bool) error {
	if this.Network.IsPeerEstablished(p) {
		return this.Network.Send(p, msg, isConsensus)
	}
	this.log.Warnf("[p2p] send to a not ESTABLISH peer %d",
		p.GetID())
	return errors.New("[p2p] send to a not ESTABLISH peer")
}

// GetID returns local node id
//func (this *P2PServer) GetID() uint64 {
//	return this.Network.GetID()
//}

//GetTime return lastet contact time
//func (this *P2PServer) GetTime() int64 {
//	return this.Network.GetTime()
//}

//connectSeeds connect the seeds in seedlist and call for nbr list
func (this *P2PServer) connectSeeds() {
	seedNodes := make([]string, 0)
	pList := make([]*peer.Peer, 0)
	for _, n := range this.ctx.Config().P2P.Genesis.SeedList {
		ip, err := common.ParseIPAddr(n)
		if err != nil {
			this.log.Warnf("[p2p] seed peer %s address format is wrong", n)
			continue
		}
		ns, err := net.LookupHost(ip)
		if err != nil {
			this.log.Warnf("[p2p] resolve err: %s", err.Error())
			continue
		}
		port, err := common.ParseIPPort(n)
		if err != nil {
			this.log.Warnf("[p2p] seed peer %s address format is wrong", n)
			continue
		}
		seedNodes = append(seedNodes, ns[0]+port)
	}

	for _, nodeAddr := range seedNodes {
		var ip net.IP
		np := this.Network.GetNp()
		np.Lock()
		for _, tn := range np.List {
			ipAddr, err := tn.GetAddr16()
			if err != nil {
				this.log.Error("parse ip error ", err)
				return
			}
			ip = ipAddr[:]
			addrString := ip.To16().String() + ":" +
				strconv.Itoa(int(tn.GetSyncPort()))
			if nodeAddr == addrString && tn.GetSyncState() == common.ESTABLISH {
				pList = append(pList, tn)
			}
			if tn.GetSyncState() == common.ESTABLISH {
				this.reqNbrList(tn)
			}
		}
		np.Unlock()
	}
	if len(pList) > 1 {
		for _, p := range pList {
			this.reqNbrList(p)
		}
		//rand.Seed(time.Now().UnixNano())
		//index := rand.Intn(len(pList))
		//this.reqNbrList(pList[index])
	} else { //not found
		for _, nodeAddr := range seedNodes {
			go this.Network.Connect(nodeAddr, false)
		}
	}
}

//getNode returns the peer with the id
//func (this *P2PServer) getNode(id uint64) *peer.Peer {
//	return this.Network.GetPeer(id)
//}

//retryInactivePeer try to connect peer in INACTIVITY state
func (this *P2PServer) retryInactivePeer() {
	np := this.Network.GetNp()
	np.Lock()
	var ip net.IP
	neighborPeers := make(map[uint64]*peer.Peer)
	for _, p := range np.List {
		addr, _ := p.GetAddr16()
		ip = addr[:]
		nodeAddr := ip.To16().String() + ":" +
			strconv.Itoa(int(p.GetSyncPort()))
		if p.GetSyncState() == common.INACTIVITY {
			this.log.Debugf("[p2p] try reconnect %s", nodeAddr)
			//add addr to retry list
			this.addToRetryList(nodeAddr)
			this.Network.RemoveFromInConnRecord(p.GetAddr())
			this.Network.RemoveFromOutConnRecord(p.GetAddr())
			this.Network.RemoveFromConnectingList(p.GetAddr())
			p.CloseSync()
			p.CloseCons()
		} else {
			//add others to tmp node map
			this.removeFromRetryList(nodeAddr)
			neighborPeers[p.GetID()] = p
		}
	}

	np.List = neighborPeers
	np.Unlock()

	connCount := uint(this.Network.GetOutConnRecordLen())
	if connCount >= this.ctx.Config().P2P.MaxConnOutBound {
		this.log.Warnf("[p2p] Connect: out connections(%d) reach the max limit(%d)", connCount,
			this.ctx.Config().P2P.MaxConnOutBound)
		return
	}

	//try connect
	if len(this.RetryAddrs) > 0 {
		this.ReconnectAddrs.Lock()

		list := make(map[string]int)
		addrs := make([]string, 0, len(this.RetryAddrs))
		for addr, v := range this.RetryAddrs {
			v += 1
			addrs = append(addrs, addr)
			if v < common.MAX_RETRY_COUNT {
				list[addr] = v
			}
			if v >= common.MAX_RETRY_COUNT {
				this.Network.RemoveFromInConnRecord(addr)
				this.Network.RemoveFromOutConnRecord(addr)
				this.Network.RemoveFromConnectingList(addr)
				remotePeer := this.Network.GetPeerFromAddr(addr)
				if remotePeer != nil {
					if remotePeer.SyncLink.GetAddr() == addr {
						this.Network.RemovePeerSyncAddress(addr)
						this.Network.RemovePeerConsAddress(addr)
					}
					if remotePeer.ConsLink.GetAddr() == addr {
						this.Network.RemovePeerConsAddress(addr)
					}
					this.Network.DelNbrNode(remotePeer)
				}
			}
		}

		this.RetryAddrs = list
		this.ReconnectAddrs.Unlock()
		for _, addr := range addrs {
			rand.Seed(time.Now().UnixNano())
			this.log.Debug("[p2p] Try to reconnect peer, peer addr is ", addr)
			<-time.After(time.Duration(rand.Intn(common.CONN_MAX_BACK)) * time.Millisecond)
			this.log.Debug("[p2p] Back off time`s up, start connect node")
			this.Network.Connect(addr, false)
		}

	}
}

//connectSeedService make sure seed peer be connected
func (this *P2PServer) connectSeedService() {
	t := time.NewTimer(time.Second * common.CONN_MONITOR)
	for {
		select {
		case <-t.C:
			this.connectSeeds()
			t.Stop()
			t.Reset(time.Second * common.CONN_MONITOR)
		case <-this.quitOnline:
			t.Stop()
			break
		}
	}
}

//keepOnline try connect lost peer
func (this *P2PServer) keepOnlineService() {
	t := time.NewTimer(time.Second * common.CONN_MONITOR)
	for {
		select {
		case <-t.C:
			this.retryInactivePeer()
			t.Stop()
			t.Reset(time.Second * common.CONN_MONITOR)
		case <-this.quitOnline:
			t.Stop()
			break
		}
	}
}

//reqNbrList ask the peer for its neighbor list
func (this *P2PServer) reqNbrList(p *peer.Peer) {
	msg := msgpack.NewAddrReq()
	go this.Send(p, msg, false)
}

//heartBeat send ping to nbr peers and check the timeout
func (this *P2PServer) heartBeatService() {
	var periodTime uint
	periodTime = constants.BlockInterval
	t := time.NewTicker(time.Second * (time.Duration(periodTime)))

	for {
		select {
		case <-t.C:
			this.ping()
			this.timeout()
		case <-this.quitHeartBeat:
			t.Stop()
			break
		}
	}
}

//ping send pkg to get pong msg from others
func (this *P2PServer) ping() {
	peers := this.Network.GetNeighbors()
	this.pingTo(peers)
}

//pings send pkgs to get pong msg from others
func (this *P2PServer) pingTo(peers []*peer.Peer) {
	//service, err := this.Network.GetService(iservices.ConsensusServerName)
	//if err != nil {
	//	this.log.Error("[p2p] can't get other service, service name: ", iservices.ConsensusServerName)
	//	return
	//}
	//ctrl := service.(iservices.IConsensus)
	for _, p := range peers {
		if p.GetSyncState() == common.ESTABLISH {

			//height := ctrl.GetHeadBlockId().BlockNum()
			var height uint64 = 0
			ping := msgpack.NewPingMsg(height)
			go this.Send(p, ping, false)
		}
	}
}

//timeout trace whether some peer be long time no response
func (this *P2PServer) timeout() {
	peers := this.Network.GetNeighbors()
	var periodTime uint
	periodTime = constants.BlockInterval
	for _, p := range peers {
		if p.GetSyncState() == common.ESTABLISH {
			t := p.GetContactTime()
			if t.Before(time.Now().Add(-1 * time.Second *
				time.Duration(periodTime) * common.KEEPALIVE_TIMEOUT)) {
				this.log.Warnf("[p2p] keep alive timeout!!!lost remote peer %d - %s from %s", p.GetID(), p.SyncLink.GetAddr(), t.String())
				this.Network.RemoveFromInConnRecord(p.GetAddr())
				this.Network.RemoveFromOutConnRecord(p.GetAddr())
				this.Network.RemoveFromConnectingList(p.GetAddr())
				p.CloseSync()
				p.CloseCons()
			}
		}
	}
}

//addToRetryList add retry address to ReconnectAddrs
func (this *P2PServer) addToRetryList(addr string) {
	this.ReconnectAddrs.Lock()
	defer this.ReconnectAddrs.Unlock()
	if this.RetryAddrs == nil {
		this.RetryAddrs = make(map[string]int)
	}
	if _, ok := this.RetryAddrs[addr]; ok {
		delete(this.RetryAddrs, addr)
	}
	//alway set retry to 0
	this.RetryAddrs[addr] = 0
}

//removeFromRetryList remove connected address from ReconnectAddrs
func (this *P2PServer) removeFromRetryList(addr string) {
	this.ReconnectAddrs.Lock()
	defer this.ReconnectAddrs.Unlock()
	if len(this.RetryAddrs) > 0 {
		if _, ok := this.RetryAddrs[addr]; ok {
			delete(this.RetryAddrs, addr)
		}
	}
}

func (this *P2PServer) Broadcast(message interface{}) {
	if !this.Network.CheckStartUpFinished() {
		return
	}
	var msg msgtypes.Message
	isConsensus := false
	switch message.(type) {
	case *prototype.SignedTransaction:
		sigtrx := message.(*prototype.SignedTransaction)
		msg = msgpack.NewTxn(sigtrx)
	case *prototype.SignedBlock:
		block := message.(*prototype.SignedBlock)
		msg = msgpack.NewSigBlkIdMsg(block)
	case consmsg.ConsensusMessage:
		cmsg := message.(consmsg.ConsensusMessage)
		msg = msgpack.NewConsMsg(cmsg, true)
	default:
		this.log.Warnf("[p2p] Unknown Xmit message %v , type %v", message,
			reflect.TypeOf(message))
		return
	}

	if this.mockLatency > 0 {
		go func() {
			time.Sleep(time.Duration(rand.Int()%this.mockLatency) * time.Millisecond)
			this.Network.Broadcast(msg, isConsensus)
		}()
	} else {
		this.Network.Broadcast(msg, isConsensus)
	}
}

func (this *P2PServer) TriggerSync(current_head_blk_id coomn.BlockID) {
	if !this.Network.CheckStartUpFinished() {
		return
	}
	reqmsg := new(msgtypes.TransferMsg)
	reqdata := new(msgtypes.ReqIdMsg)
	reqdata.HeadBlockId = current_head_blk_id.Data[:]
	reqmsg.Msg = &msgtypes.TransferMsg_Msg4{Msg4:reqdata}
	currentHeadNum := current_head_blk_id.BlockNum()
	//this.log.Info("enter TriggerSync func")
	np := this.Network.GetNp()
	np.RLock()
	defer np.RUnlock()

	for _, p := range np.List {
		//this.log.Info("[p2p] cons call TriggerSync func, head id :  ", reqmsg.HeadBlockId)
		num := p.GetLastSeenBlkNum()
		if currentHeadNum < num {
			go p.Send(reqmsg, false, this.Network.GetMagic())
			return
		}
	}

	for _, p := range np.List {
		go p.Send(reqmsg, false, this.Network.GetMagic())
		return
	}
}

func (this *P2PServer) FetchUnlinkedBlock(prevId coomn.BlockID) {
	if !this.Network.CheckStartUpFinished() {
		return
	}
	var reqmsg msgtypes.TransferMsg
	reqdata := new(msgtypes.IdMsg)
	reqdata.Msgtype = msgtypes.IdMsg_request_sigblk_by_id

	reqdata.Value = append(reqdata.Value, prevId.Data[:])
	reqmsg.Msg = &msgtypes.TransferMsg_Msg2{Msg2:reqdata}

	currentHeadNum := prevId.BlockNum()
	//this.log.Info("enter TriggerSync func")
	np := this.Network.GetNp()
	np.RLock()
	defer np.RUnlock()

	for _, p := range np.List {
		//this.log.Info("[p2p] cons call TriggerSync func, head id :  ", reqmsg.HeadBlockId)
		num := p.GetLastSeenBlkNum()
		if currentHeadNum < num {
			go p.Send(&reqmsg, false, this.Network.GetMagic())
			return
		}
	}

	for _, p := range np.List {
		go p.Send(&reqmsg, false, this.Network.GetMagic())
		return
	}
}

func (this *P2PServer) RequestCheckpoint(startNum, endNum uint64) {
	if !this.Network.CheckStartUpFinished() {
		return
	}
	this.log.Infof("RequestCheckpoint from %d to %d", startNum, endNum)
	reqmsg := msgpack.NewCheckpointBatchMsg(startNum, endNum)

	np := this.Network.GetNp()
	np.RLock()
	defer np.RUnlock()

	for _, p := range np.List {
		//this.log.Info("[p2p] cons call RequestCheckpoint func, start number: ",  startNum, " end number: ", endNum)
		num := p.GetLastSeenBlkNum()
		if endNum < num {
			go p.Send(reqmsg, false, this.Network.GetMagic())
			return
		}
	}

	for _, p := range np.List {
		go p.Send(reqmsg, false, this.Network.GetMagic())
		return
	}
}

func (this *P2PServer) FetchOutOfRange(localHeadID, targetID coomn.BlockID) {
	if !this.Network.CheckStartUpFinished() {
		return
	}
	if localHeadID.BlockNum() >= targetID.BlockNum() {
		this.log.Warn("local head number less than target number.local number: ", localHeadID.BlockNum(), " target number: ", targetID.BlockNum())
		return
	}
	reqmsg := msgpack.NewRequestOutOfRangeIds(localHeadID.Data[:], targetID.Data[:])

	np := this.Network.GetNp()
	np.RLock()
	defer np.RUnlock()

	for _, p := range np.List {
		num := p.GetLastSeenBlkNum()
		p.OutOfRangeState.Lock()
		if len(p.OutOfRangeState.KeyPointIDList) == 0 && targetID.BlockNum() <= num {
			p.OutOfRangeState.KeyPointIDList = append(p.OutOfRangeState.KeyPointIDList, targetID.Data[:])
			this.log.Infof("FetchOutOfRange from %d to %d", localHeadID.BlockNum(), targetID.BlockNum() )
			go p.Send(reqmsg, false, this.Network.GetMagic())
			p.OutOfRangeState.Unlock()
			return
		}
		p.OutOfRangeState.Unlock()
	}
	this.log.Info("all peers are busy, should wait idle peer")
}

func (this *P2PServer) SendToPeer(p *peer.Peer, message interface{}) {
	if !this.Network.CheckStartUpFinished() {
		return
	}
	if p == nil {
		this.log.Error("send message to a nil peer")
		return
	}
	if this.Network.IsPeerEstablished(p) {
		cmsg := message.(consmsg.ConsensusMessage)
		msg := msgpack.NewConsMsg(cmsg, false)
		this.log.Info("send message to a specific peer ", p.GetAddr())
		go p.Send(msg, false, this.Network.GetMagic())
		return
	}
	this.log.Errorf("[p2p] send to a not ESTABLISH peer in SendToPeer %s",
		p.GetAddr())
}

func (this *P2PServer) RandomSend(message interface{}) {
	if !this.Network.CheckStartUpFinished() {
		return
	}
	this.log.Info("send message to a random peer")

	cmsg := message.(consmsg.ConsensusMessage)
	msg := msgpack.NewConsMsg(cmsg, false)
	hash := msg.(*msgtypes.ConsMsg).Hash()

	np := this.Network.GetNp()
	np.RLock()
	defer np.RUnlock()

	for _, p := range np.List {
		state := p.GetSyncState()
		if state == common.ESTABLISH && !p.HasConsensusMsg(hash) {
			p.RecordConsensusMsg(hash)
			this.log.Info("send message to a random peer ", p.GetAddr(), " msg hash: ", hash)
			go p.Send(msg, false, this.Network.GetMagic())
			return
		}
	}
}

func (this *P2PServer) GetNodeNeighbours() string {
	if !this.Network.CheckStartUpFinished() {
		return ""
	}
	var peerList string
	peers := this.Network.GetNeighbors()
	for _, p := range peers {
		if p.GetSyncState() == common.ESTABLISH {
			ip := p.GetAddr()
			pStr := fmt.Sprintf("%s, ", ip)
			peerList += pStr
		}
	}
	return peerList
}

func (this *P2PServer) SetMockLatency(t int) {
	this.mockLatency = t
}

func (this *P2PServer) GetMockLatency() int {
	return this.mockLatency
}
