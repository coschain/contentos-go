package peer

import (
	"fmt"
	"sync"

	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/p2p/common"
	"github.com/coschain/contentos-go/p2p/message/types"
)

//NbrPeers: The neigbor list
type NbrPeers struct {
	sync.RWMutex
	List       map[uint64]*Peer
	SendTrxMap map[string][]byte  // the last trx hash send to a peer
	RecvTrxMap map[string][]byte  // the last trx hash receive from a peer
}

func ByteSliceEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	if (a == nil) != (b == nil) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}

//Broadcast tranfer msg buffer to all establish peer
func (this *NbrPeers) Broadcast(mesg types.Message, isConsensus bool, magic uint32) {
	this.RLock()
	defer this.RUnlock()
	for _, node := range this.List {
		id := new(prototype.Sha256)
		data := mesg.(*types.TransferMsg)
		if msgdata, ok := data.Msg.(*types.TransferMsg_Msg1); ok {
			id, _ = msgdata.Msg1.SigTrx.Id()
			sendCache := this.SendTrxMap[node.GetAddr()]
			recvCache := this.RecvTrxMap[node.GetAddr()]
			if ByteSliceEqual(sendCache, id.Hash) || ByteSliceEqual(recvCache, id.Hash) {
				continue
			}
		}
		if node.syncState == common.ESTABLISH && node.GetRelay() == true {
			this.SendTrxMap[node.GetAddr()] = id.Hash
			go node.Send(mesg, isConsensus, magic)
		}
	}
}

//NodeExisted return when peer in nbr list
func (this *NbrPeers) NodeExisted(uid uint64) bool {
	_, ok := this.List[uid]
	return ok
}

//GetPeer return peer according to id
func (this *NbrPeers) GetPeer(id uint64) *Peer {
	this.Lock()
	defer this.Unlock()
	n, ok := this.List[id]
	if ok == false {
		return nil
	}
	return n
}

//AddNbrNode add peer to nbr list
func (this *NbrPeers) AddNbrNode(p *Peer) {
	this.Lock()
	defer this.Unlock()

	if this.NodeExisted(p.GetID()) {
		fmt.Printf("[p2p]insert an existed node\n")
	} else {
		this.List[p.GetID()] = p
	}
}

//DelNbrNode delete peer from nbr list
func (this *NbrPeers) DelNbrNode(p *Peer) (*Peer, bool) {
	this.Lock()
	defer this.Unlock()

	n, ok := this.List[p.GetID()]
	if ok == false {
		return nil, false
	}

	delete(this.List, p.GetID())
	delete(this.SendTrxMap, p.GetAddr())
	delete(this.RecvTrxMap, p.GetAddr())

	return n, true
}

//initialize nbr list
func (this *NbrPeers) Init() {
	this.List = make(map[uint64]*Peer)
	this.SendTrxMap = make(map[string][]byte)
	this.RecvTrxMap = make(map[string][]byte)
}

//NodeEstablished whether peer established according to id
func (this *NbrPeers) NodeEstablished(id uint64) bool {
	this.RLock()
	defer this.RUnlock()

	n, ok := this.List[id]
	if ok == false {
		return false
	}

	if n.syncState != common.ESTABLISH {
		return false
	}

	return true
}

//GetNeighborAddrs return all establish peer address
func (this *NbrPeers) GetNeighborAddrs() []*types.PeerAddr {
	this.RLock()
	defer this.RUnlock()

	var addrs []*types.PeerAddr
	for _, p := range this.List {
		if p.GetSyncState() != common.ESTABLISH {
			continue
		}
		addr := &types.PeerAddr{}
		res, _ := p.GetAddr16()
		addr.IpAddr = res[:]
		addr.Time = p.GetTimeStamp()
		addr.Services = p.GetServices()
		addr.Port = uint32(p.GetSyncPort())
		addr.ID = p.GetID()
		addrs = append(addrs, addr)
	}

	return addrs
}

//GetNeighborHeights return the id-height map of nbr peers
func (this *NbrPeers) GetNeighborHeights() map[uint64]uint64 {
	this.RLock()
	defer this.RUnlock()

	hm := make(map[uint64]uint64)
	for _, n := range this.List {
		if n.GetSyncState() == common.ESTABLISH {
			hm[n.GetID()] = n.GetHeight()
		}
	}
	return hm
}

//GetNeighbors return all establish peers in nbr list
func (this *NbrPeers) GetNeighbors() []*Peer {
	this.RLock()
	defer this.RUnlock()
	peers := []*Peer{}
	for _, n := range this.List {
		if n.GetSyncState() == common.ESTABLISH {
			node := n
			peers = append(peers, node)
		}
	}
	return peers
}

//GetNbrNodeCnt return count of establish peers in nbrlist
func (this *NbrPeers) GetNbrNodeCnt() uint32 {
	this.RLock()
	defer this.RUnlock()
	var count uint32
	for _, n := range this.List {
		if n.GetSyncState() == common.ESTABLISH {
			count++
		}
	}
	return count
}
