// Package p2p provides an network interface
package p2p

import (
	"github.com/coschain/contentos-go/p2p/common"
	"github.com/coschain/contentos-go/p2p/message/types"
	"github.com/coschain/contentos-go/p2p/peer"
	 comn "github.com/coschain/contentos-go/common"
)

//P2P represent the net interface of p2p package
type P2P interface {
	Start()
	Halt()
	Connect(addr string, isConsensus bool) error
	GetID() uint64
	GetVersion() uint32
	GetSyncPort() uint16
	GetConsPort() uint16
	GetHttpInfoPort() uint16
	GetRelay() bool
	GetHeight() uint64
	GetTime() int64
	GetServices() uint64
	GetNeighbors() []*peer.Peer
	GetNeighborAddrs() []common.PeerAddr
	GetConnectionCnt() uint32
	GetNp() *peer.NbrPeers
	GetPeer(uint64) *peer.Peer
	SetHeight(uint64)
	IsPeerEstablished(p *peer.Peer) bool
	GetMsgChan(isConsensus bool) chan *types.MsgPayload
	GetPeerFromAddr(addr string) *peer.Peer
	AddOutConnectingList(addr string) (added bool)
	GetOutConnRecordLen() int
	RemoveFromConnectingList(addr string)
	RemoveFromOutConnRecord(addr string)
	RemoveFromInConnRecord(addr string)
	AddPeerSyncAddress(addr string, p *peer.Peer)
	AddPeerConsAddress(addr string, p *peer.Peer)
	GetOutConnectingListLen() (count uint)
	RemovePeerSyncAddress(addr string)
	RemovePeerConsAddress(addr string)
	AddNbrNode(*peer.Peer)
	DelNbrNode(id uint64) (*peer.Peer, bool)
	NodeEstablished(uint64) bool
	SetOwnAddress(addr string)
	IsAddrFromConnecting(addr string) bool

	// Broadcast sigTrx or sigBlk msg
	Broadcast(message interface{})

	// trigger sync request remote peer the block hashes we do not have
	Triggeer_sync(p *peer.Peer, current_head_blk_id comn.BlockID)

	// Send msg to specific peer
	Send(p *peer.Peer, msg types.Message, isConsensus bool) error
}
