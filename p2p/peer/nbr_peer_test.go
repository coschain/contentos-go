package peer

import (
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

var nm *NbrPeers

func creatPeers(cnt uint32) []*Peer {
	np := []*Peer{}
	var syncport uint32
	var consport uint32
	var id uint64
	var height uint64
	for i := uint32(0); i < cnt; i++ {
		syncport = 20000 + i
		consport = 30000 + i
		id = 0x12345 + uint64(i)
		height = 10086 + uint64(i)
		p = NewPeer()
		p.UpdateInfo(time.Now(), 2, 3, syncport, consport, id, 0, height, "abc")
		p.SetConsState(2)
		p.SetSyncState(3)
		p.SyncLink.SetAddr("127.0.0.1:10338")
		np = append(np, p)
	}
	return np

}

func init() {
	nm = &NbrPeers{}
	nm.Init()
	np := creatPeers(5)
	for _, v := range np {
		nm.List[v.GetID()] = v
	}
}

func TestNodeExisted(t *testing.T) {
	if nm.NodeExisted(0x12345) == false {
		t.Fatal("0x12345 should in nbr peers")
	}
	if nm.NodeExisted(0x123456) == true {
		t.Fatal("0x5533345 should not in nbr peers")
	}
}

func TestGetPeer(t *testing.T) {
	p := nm.GetPeer(0x12345)
	if p == nil {
		t.Fatal("TestGetPeer error")
	}
}

func TestAddNbrNode(t *testing.T) {
	p := NewPeer()
	p.UpdateInfo(time.Now(), 2, 3, 10335, 10336, 0x7123456, 0, 100, "abc")
	p.SetConsState(2)
	p.SetSyncState(3)
	p.SyncLink.SetAddr("127.0.0.1")
	nm.AddNbrNode(p)
	if nm.NodeExisted(0x7123456) == false {
		t.Fatal("0x7123456 should be added in nbr peer")
	}
	if len(nm.List) != 6 {
		t.Fatal("0x7123456 should be added in nbr peer")
	}
}

func TestDelNbrNode(t *testing.T) {
	cnt := len(nm.List)
	p := nm.GetPeer(0x12345)
	p, ret := nm.DelNbrNode(p)
	if p == nil || ret != true {
		t.Fatal("TestDelNbrNode err")
	}
	if len(nm.List) != cnt-1 {
		t.Fatal("TestDelNbrNode not work")
	}
	log := logrus.New()
	p.DumpInfo(log)
}

func TestNodeEstablished(t *testing.T) {
	p := nm.GetPeer(0x12346)
	if p == nil {
		t.Fatal("TestNodeEstablished:get peer error")
	}
	p.SetSyncState(4)
	if nm.NodeEstablished(0x12346) == false {
		t.Fatal("TestNodeEstablished error")
	}
}

func TestGetNeighborAddrs(t *testing.T) {
	p := nm.GetPeer(0x12346)
	if p == nil {
		t.Fatal("TestGetNeighborAddrs:get peer error")
	}
	p.SetSyncState(4)

	p = nm.GetPeer(0x12347)
	if p == nil {
		t.Fatal("TestGetNeighborAddrs:get peer error")
	}
	p.SetSyncState(4)

	pList := nm.GetNeighborAddrs()
	for i := 0; i < int(len(pList)); i++ {
		fmt.Printf("peer id = %x \n", pList[i].ID)
	}
	if len(pList) != 2 {
		t.Fatal("TestGetNeighborAddrs error")
	}
}

func TestGetNeighborHeights(t *testing.T) {
	p := nm.GetPeer(0x12346)
	if p == nil {
		t.Fatal("TestGetNeighborHeights:get peer error")
	}
	p.SetSyncState(4)

	p = nm.GetPeer(0x12347)
	if p == nil {
		t.Fatal("TestGetNeighborHeights:get peer error")
	}
	p.SetSyncState(4)

	pMap := nm.GetNeighborHeights()
	for k, v := range pMap {
		fmt.Printf("peer id = %x height = %d \n", k, v)
	}
}

func TestGetNeighbors(t *testing.T) {
	p := nm.GetPeer(0x12346)
	if p == nil {
		t.Fatal("TestGetNeighbors:get peer error")
	}
	p.SetSyncState(4)

	p = nm.GetPeer(0x12347)
	if p == nil {
		t.Fatal("TestGetNeighbors:get peer error")
	}
	p.SetSyncState(4)

	pList := nm.GetNeighbors()
	for _, v := range pList {
		log := logrus.New()
		v.DumpInfo(log)
	}
}

func TestGetNbrNodeCnt(t *testing.T) {
	p := nm.GetPeer(0x12346)
	if p == nil {
		t.Fatal("TestGetNbrNodeCnt:get peer error")
	}
	p.SetSyncState(4)

	p = nm.GetPeer(0x12347)
	if p == nil {
		t.Fatal("TestGetNbrNodeCnt:get peer error")
	}
	p.SetSyncState(4)

	if nm.GetNbrNodeCnt() != 2 {
		t.Fatal("TestGetNbrNodeCnt error")
	}
}
