package iservices

import (
	comn "github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/p2p/peer"
)

var P2PServerName = "p2p"

//IP2P represent the net interface of p2p package which can be called by other service
type IP2P interface {
	// Broadcast sigTrx or sigBlk msg
	Broadcast(message interface{})

	// trigger sync request remote peer the block hashes we do not have
	TriggerSync(HeadId comn.BlockID)

	// when got one unlinked block, to fetch its previous block
	FetchUnlinkedBlock(prevId comn.BlockID)

	// (localHeadID, targetID]
	FetchMissingBlock(from, to comn.BlockID)

	// Send message to a specific peer
	SendToPeer(p *peer.Peer, message interface{})

	// Send message to a random peer
	RandomSend(message interface{})

	// Request checkpoint batch [startNum, endNum)
	RequestCheckpoint(startNum, endNum uint64)

	GetNodeNeighbours() string

	// for test only
	SetMockLatency(t int)
	GetMockLatency() int
}
