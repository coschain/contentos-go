package netserver

import (
	"fmt"
	"github.com/coschain/contentos-go/config"
	"github.com/coschain/contentos-go/node"
	"github.com/sirupsen/logrus"
	"testing"
	"time"

	"github.com/coschain/contentos-go/p2p/common"
	"github.com/coschain/contentos-go/p2p/peer"
)

func init() {
	fmt.Println("Start test the netserver...")
}

func creatPeers(cnt uint32) []*peer.Peer {
	np := []*peer.Peer{}
	var syncport uint32
	var consport uint32
	var id uint64
	var height uint64
	for i := uint32(0); i < cnt; i++ {
		syncport = 20000 + i
		consport = 30000 + i
		id = 0x12345 + uint64(i)
		height = 10086 + uint64(i)
		p := peer.NewPeer(logrus.New())
		p.UpdateInfo(time.Now(), 2, 3, syncport, consport, id, 0, height, "abc")
		p.SetConsState(2)
		p.SetSyncState(4)
		p.SyncLink.SetAddr("127.0.0.1:10338")
		np = append(np, p)
	}
	return np

}
func TestNewNetServer(t *testing.T) {
	conf := &config.DefaultNodeConfig
	ctx := new(node.ServiceContext)
	ctx.ResetConfig(conf)
	log := logrus.New()
	server := NewNetServer(ctx, log)
	server.Start()
	defer server.Halt()

	server.SetHeight(1000)
	if server.GetHeight() != 1000 {
		t.Error("TestNewNetServer set server height error")
	}

	if server.GetRelay() != true {
		t.Error("TestNewNetServer server relay state error", server.GetRelay())
	}
	if server.GetServices() != 1 {
		t.Error("TestNewNetServer server service state error", server.GetServices())
	}
	if server.GetVersion() != common.PROTOCOL_VERSION {
		t.Error("TestNewNetServer server version error", server.GetVersion())
	}
	if server.GetSyncPort() != 20338 {
		t.Error("TestNewNetServer sync port error", server.GetSyncPort())
	}
	if server.GetConsPort() != 20339 {
		t.Error("TestNewNetServer sync port error", server.GetConsPort())
	}

	fmt.Printf("lastest server time is %s\n", time.Unix(server.GetTime()/1e9, 0).String())

}

func TestNetServerNbrPeer(t *testing.T) {
	conf := &config.DefaultNodeConfig
	ctx := new(node.ServiceContext)
	ctx.ResetConfig(conf)
	log := logrus.New()
	server := NewNetServer(ctx, log)
	server.Start()
	defer server.Halt()

	nm := &peer.NbrPeers{}
	nm.Init()
	np := creatPeers(5)
	for _, v := range np {
		server.AddNbrNode(v)
	}
	if server.GetConnectionCnt() != 5 {
		t.Error("TestNetServerNbrPeer GetConnectionCnt error", server.GetConnectionCnt())
	}
	addrs := server.GetNeighborAddrs()
	if len(addrs) != 5 {
		t.Error("TestNetServerNbrPeer GetNeighborAddrs error")
	}
	if server.NodeEstablished(0x12345) == false {
		t.Error("TestNetServerNbrPeer NodeEstablished error")
	}
	if server.GetPeer(0x12345) == nil {
		t.Error("TestNetServerNbrPeer GetPeer error")
	}
	p, ok := server.DelNbrNode(server.GetPeer(0x12345) )
	if ok != true || p == nil {
		t.Error("TestNetServerNbrPeer DelNbrNode error")
	}
	if len(server.GetNeighbors()) != 4 {
		t.Error("TestNetServerNbrPeer GetNeighbors error")
	}
	sp := &peer.Peer{}
	cp := &peer.Peer{}
	server.AddPeerSyncAddress("127.0.0.1:10338", sp)
	server.AddPeerConsAddress("127.0.0.1:20338", cp)
	if server.GetPeerFromAddr("127.0.0.1:10338") != sp {
		t.Error("TestNetServerNbrPeer Get/AddPeerConsAddress error")
	}
}
