package dandelion

import (
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/p2p/message/types"
	"github.com/coschain/contentos-go/p2p/peer"
)

type DandelionP2P struct {
}

func NewDandelionP2P() *DandelionP2P {
	return &DandelionP2P{}
}

func (d *DandelionP2P) Broadcast(message interface{}) {

}

func (d *DandelionP2P) TriggerSync(HeadId common.BlockID) {

}

func (d *DandelionP2P) Send(p *peer.Peer, msg types.Message, isConsensus bool) error {
	return nil
}
