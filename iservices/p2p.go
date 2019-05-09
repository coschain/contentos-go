package iservices

import (
	comn "github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/p2p/message/types"
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

	// Send msg to specific peer
	Send(p *peer.Peer, msg types.Message, isConsensus bool) error

	// Request checkpoint batch [startNum, endNum)
	RequestCheckpoint(startNum, endNum uint64)

	// if receive a out-of-range signed block, call this method to fetch the gap signed blocks (localHeadID, targetID]
	FetchOutOfRange(localHeadID, targetID comn.BlockID)
}
