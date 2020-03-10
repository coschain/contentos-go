package peer

import (
	"errors"
	"github.com/coschain/contentos-go/common/constants"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coschain/contentos-go/p2p/common"
	conn "github.com/coschain/contentos-go/p2p/link"
	"github.com/coschain/contentos-go/p2p/message/types"
	"github.com/sirupsen/logrus"
	"github.com/willf/bloom"
)

// PeerCom provides the basic information of a peer
type PeerCom struct {
	id       uint64
	version  uint32
	services uint64
	relay    bool
	syncPort uint32
	consPort uint32
	height   uint64
}

// SetID sets a peer's id
func (this *PeerCom) SetID(id uint64) {
	this.id = id
}

// GetID returns a peer's id
func (this *PeerCom) GetID() uint64 {
	return this.id
}

// SetVersion sets a peer's version
func (this *PeerCom) SetVersion(version uint32) {
	this.version = version
}

// GetVersion returns a peer's version
func (this *PeerCom) GetVersion() uint32 {
	return this.version
}

// SetServices sets a peer's services
func (this *PeerCom) SetServices(services uint64) {
	this.services = services
}

// GetServices returns a peer's services
func (this *PeerCom) GetServices() uint64 {
	return this.services
}

// SerRelay sets a peer's relay
func (this *PeerCom) SetRelay(relay bool) {
	this.relay = relay
}

// GetRelay returns a peer's relay
func (this *PeerCom) GetRelay() bool {
	return this.relay
}

// SetSyncPort sets a peer's sync port
func (this *PeerCom) SetSyncPort(port uint32) {
	this.syncPort = port
}

// GetSyncPort returns a peer's sync port
func (this *PeerCom) GetSyncPort() uint32 {
	return this.syncPort
}

// SetConsPort sets a peer's consensus port
func (this *PeerCom) SetConsPort(port uint32) {
	this.consPort = port
}

// GetConsPort returns a peer's consensus port
func (this *PeerCom) GetConsPort() uint32 {
	return this.consPort
}

// SetHeight sets a peer's height
func (this *PeerCom) SetHeight(height uint64) {
	this.height = height
}

// GetHeight returns a peer's height
func (this *PeerCom) GetHeight() uint64 {
	return this.height
}

type trxCache struct {
	trxCount         int
	bloomFilter1     *bloom.BloomFilter
	bloomFilter2     *bloom.BloomFilter
	useFilter2       bool
	sync.Mutex
}

type FetchOutOfRangeState struct {
	KeyPointIDList      [][]byte
	sync.Mutex
}

type RequireNbrListState struct {
	LastAskTime int64
	AuthNumber  uint64
	sync.Mutex
}

//Peer represent the node in p2p
type Peer struct {
	log                *logrus.Logger
	base               PeerCom
	cap                [32]byte
	SyncLink           *conn.Link
	ConsLink           *conn.Link
	syncState          uint32
	consState          uint32
	runningCosdVersion string

	TrxCache           trxCache

	OutOfRangeState    FetchOutOfRangeState

	ConsensusCache     *common.HashCache

	lastSeenBlkNum     uint64

	connLock           sync.RWMutex
	busy			   int32
	busyFetchingCP     int32

	ReqNbrList         RequireNbrListState

	BlockQueryLimiter      *common.RateLimiter
	CheckpointQueryLimiter *common.RateLimiter
	BlobSizeLimiter        *common.RateLimiter
}

//NewPeer return new peer without publickey initial
func NewPeer(lg *logrus.Logger) *Peer {
	p := &Peer{
		log:       lg,
		syncState: common.INIT,
		consState: common.INIT,
		busy: 0,
		busyFetchingCP: 0,
	}

	p.TrxCache.bloomFilter1 = bloom.New(common.BloomFilterOfRecvTrxArgM, common.BloomFilterOfRecvTrxArgK)
	p.TrxCache.bloomFilter2 = bloom.New(common.BloomFilterOfRecvTrxArgM, common.BloomFilterOfRecvTrxArgK)
	p.TrxCache.trxCount = 0
	p.TrxCache.useFilter2 = false

	p.ConsensusCache = common.NewHashCache(common.DefaultHashCacheMaxCount)

	maxBytesPerSecond := uint64(common.CheckPointSize * common.MaxCheckPointQueriesPerSecond) + uint64(constants.MaxBlockSize * 5)
	if maxBytesPerSecond > common.MaxBlobSizePerSecond {
		maxBytesPerSecond = common.MaxBlobSizePerSecond
	}
	p.BlockQueryLimiter        = common.NewRateLimiter(common.MaxBlockQueriesPerSecond)
	p.CheckpointQueryLimiter   = common.NewRateLimiter(common.MaxCheckPointQueriesPerSecond)
	p.BlobSizeLimiter          = common.NewRateLimiter(uint32(maxBytesPerSecond))

	p.SyncLink = conn.NewLink(p.log)
	p.ConsLink = conn.NewLink(p.log)
	runtime.SetFinalizer(p, rmPeer)
	return p
}

//rmPeer print a debug log when peer be finalized by system
func rmPeer(p *Peer) {
	//logging.CLog().Debugf("[p2p] Remove unused peer: %d", p.GetID())
}

func (this *Peer) LockBusy() bool {
	return atomic.CompareAndSwapInt32( &this.busy, 0, 1 )
}

func (this *Peer) UnlockBusy()  {
	if ! atomic.CompareAndSwapInt32( &this.busy, 1, 0 ) {
		panic("cant unlock, should lock success first")
	}
}

func (this *Peer) LockBusyFetchingCP() bool {
	return atomic.CompareAndSwapInt32( &this.busyFetchingCP, 0, 1 )
}

func (this *Peer) UnlockBusyFetchingCP()  {
	if ! atomic.CompareAndSwapInt32( &this.busyFetchingCP, 1, 0 ) {
		panic("cant unlock, should lock busyFetchingCP success first")
	}
}

func (this *Peer) HasTrx(hash []byte) bool {
	this.TrxCache.Lock()
	defer this.TrxCache.Unlock()

	if this.TrxCache.useFilter2 == true {
		return this.TrxCache.bloomFilter2.Test(hash)
	}

	return this.TrxCache.bloomFilter1.Test(hash)
}

func (this *Peer) RecordTrxCache(hash []byte) {
	this.TrxCache.Lock()
	defer this.TrxCache.Unlock()

	this.TrxCache.trxCount++

	if this.TrxCache.trxCount <= common.MaxTrxCountInBloomFiler / 2 {
		this.TrxCache.bloomFilter1.Add(hash)
	} else {
		this.TrxCache.bloomFilter1.Add(hash)
		this.TrxCache.bloomFilter2.Add(hash)

		if this.TrxCache.trxCount == common.MaxTrxCountInBloomFiler {
			if this.TrxCache.useFilter2 == true {
				this.TrxCache.useFilter2 = false
				this.TrxCache.bloomFilter2 = bloom.New(common.BloomFilterOfRecvTrxArgM, common.BloomFilterOfRecvTrxArgK)
			} else {
				this.TrxCache.useFilter2 = true
				this.TrxCache.bloomFilter1 = bloom.New(common.BloomFilterOfRecvTrxArgM, common.BloomFilterOfRecvTrxArgK)
			}
			this.TrxCache.trxCount = common.MaxTrxCountInBloomFiler / 2
		}
	}
}

func (this *Peer) HasConsensusMsg(hash [common.HashSize]byte) bool {
	return this.ConsensusCache.Has(hash)
}

func (this *Peer) RecordConsensusMsg(hash [common.HashSize]byte) {
	this.ConsensusCache.Put(hash)
}

//DumpInfo print all information of peer
func (this *Peer) DumpInfo(log *logrus.Logger) {
	log.Debug("[p2p] Node info:")
	log.Debug("[p2p] \t syncState = ", this.syncState)
	log.Debug("[p2p] \t consState = ", this.consState)
	log.Debug("[p2p] \t id = ", this.GetID())
	log.Debug("[p2p] \t addr = ", this.SyncLink.GetAddr())
	log.Debug("[p2p] \t cap = ", this.cap)
	log.Debug("[p2p] \t version = ", this.GetVersion())
	log.Debug("[p2p] \t services = ", this.GetServices())
	log.Debug("[p2p] \t syncPort = ", this.GetSyncPort())
	log.Debug("[p2p] \t consPort = ", this.GetConsPort())
	log.Debug("[p2p] \t relay = ", this.GetRelay())
	log.Debug("[p2p] \t height = ", this.GetHeight())
	log.Debug("[p2p] \t runningCosdVersion = ", this.runningCosdVersion)
}

//GetVersion return peer`s version
func (this *Peer) GetVersion() uint32 {
	return this.base.GetVersion()
}

//GetHeight return peer`s block height
func (this *Peer) GetHeight() uint64 {
	return this.base.GetHeight()
}

//SetHeight set height to peer
func (this *Peer) SetHeight(height uint64) {
	this.base.SetHeight(height)
}

//GetConsConn return consensus link
func (this *Peer) GetConsConn() *conn.Link {
	return this.ConsLink
}

//SetConsConn set consensue link to peer
func (this *Peer) SetConsConn(consLink *conn.Link) {
	this.ConsLink = consLink
}

//GetSyncState return sync state
func (this *Peer) GetSyncState() uint32 {
	return this.syncState
}

//SetSyncState set sync state to peer
func (this *Peer) SetSyncState(state uint32) {
	atomic.StoreUint32(&(this.syncState), state)
}

//GetConsState return peer`s consensus state
func (this *Peer) GetConsState() uint32 {
	return this.consState
}

//SetConsState set consensus state to peer
func (this *Peer) SetConsState(state uint32) {
	atomic.StoreUint32(&(this.consState), state)
}

//GetSyncPort return peer`s sync port
func (this *Peer) GetSyncPort() uint32 {
	return this.SyncLink.GetPort()
}

func (this *Peer) Port() uint16 {
	return uint16(this.SyncLink.GetPort())
}

func (this *Peer) IPv4() string {
	return this.SyncLink.GetAddr()
}

//GetConsPort return peer`s consensus port
func (this *Peer) GetConsPort() uint32 {
	return this.ConsLink.GetPort()
}

//SetConsPort set peer`s consensus port
func (this *Peer) SetConsPort(port uint32) {
	this.ConsLink.SetPort(port)
}

//SendToSync call sync link to send buffer
func (this *Peer) SendToSync(msg types.Message, magic uint32) error {
	if this.SyncLink != nil && this.SyncLink.Valid() {
		return this.SyncLink.SendMessage(msg)
	}
	return errors.New("[p2p]sync link invalid")
}

//SendToCons call consensus link to send buffer
func (this *Peer) SendToCons(msg types.Message, magic uint32) error {
	if this.ConsLink != nil && this.ConsLink.Valid() {
		return this.ConsLink.SendMessage(msg)
	}
	return errors.New("[p2p]cons link invalid")
}

//CloseSync halt sync connection
func (this *Peer) CloseSync() {
	this.SetSyncState(common.INACTIVITY)

	this.connLock.Lock()
	defer this.connLock.Unlock()

	this.SyncLink.StopRecvMessage()
	this.SyncLink.StopSendMessage()
	this.SyncLink.CloseConn()
}

//CloseCons halt consensus connection
func (this *Peer) CloseCons() {
	this.SetConsState(common.INACTIVITY)

	this.connLock.Lock()
	defer this.connLock.Unlock()

	this.ConsLink.StopRecvMessage()
	this.ConsLink.StopSendMessage()
	this.ConsLink.CloseConn()
}

//GetID return peer`s id
func (this *Peer) GetID() uint64 {
	return this.base.GetID()
}

//GetRelay return peer`s relay state
func (this *Peer) GetRelay() bool {
	return this.base.GetRelay()
}

//GetServices return peer`s service state
func (this *Peer) GetServices() uint64 {
	return this.base.GetServices()
}

//GetTimeStamp return peer`s latest contact time in ticks
func (this *Peer) GetTimeStamp() int64 {
	return this.SyncLink.GetRXTime().UnixNano()
}

//GetContactTime return peer`s latest contact time in Time struct
func (this *Peer) GetContactTime() time.Time {
	return this.SyncLink.GetRXTime()
}

//GetAddr return peer`s sync link address
func (this *Peer) GetAddr() string {
	return this.SyncLink.GetAddr()
}

//GetAddr16 return peer`s sync link address in []byte
func (this *Peer) GetAddr16() ([16]byte, error) {
	var result [16]byte
	addrIp, err := common.ParseIPAddr(this.GetAddr())
	if err != nil {
		return result, err
	}
	ip := net.ParseIP(addrIp).To16()
	if ip == nil {
		return result, errors.New("[p2p]parse ip address error")
	}

	copy(result[:], ip[:16])
	return result, nil
}

//AttachSyncChan set msg chan to sync link
func (this *Peer) AttachSyncChan(msgchan chan *types.MsgPayload) {
	this.SyncLink.SetChan(msgchan)
}

//AttachConsChan set msg chan to consensus link
func (this *Peer) AttachConsChan(msgchan chan *types.MsgPayload) {
	this.ConsLink.SetChan(msgchan)
}

//Send transfer buffer by sync or cons link
func (this *Peer) Send(msg types.Message, isConsensus bool, magic uint32) error {
	if isConsensus && this.ConsLink.Valid() {
		return this.SendToCons(msg, magic)
	}
	return this.SendToSync(msg, magic)
}

//UpdateInfo update peer`s information
func (this *Peer) UpdateInfo(t time.Time, version uint32, services uint64,
	syncPort uint32, consPort uint32, nonce uint64, relay uint32, height uint64, runningVersion string) {

	this.SyncLink.UpdateRXTime(t)
	this.base.SetID(nonce)
	this.base.SetVersion(version)
	this.base.SetServices(services)
	this.base.SetSyncPort(syncPort)
	this.base.SetConsPort(consPort)
	this.SyncLink.SetPort(syncPort)
	this.ConsLink.SetPort(consPort)
	this.runningCosdVersion = runningVersion
	if relay == 0 {
		this.base.SetRelay(false)
	} else {
		this.base.SetRelay(true)
	}
	this.SetHeight(uint64(height))
}

func (this *Peer) SetLastSeenBlkNum(num uint64) {
	this.connLock.Lock()
	if this.lastSeenBlkNum < num {
		this.lastSeenBlkNum = num
	}
	this.connLock.Unlock()
}

func (this *Peer) GetLastSeenBlkNum() uint64 {
	this.connLock.RLock()
	defer this.connLock.RUnlock()

	return this.lastSeenBlkNum
}
