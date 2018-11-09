package p2p

import (
	"fmt"
	"testing"

	"github.com/coschain/contentos-go/p2p/depend/common/log"
	"github.com/coschain/contentos-go/p2p/common"
)

var ch chan int

func init() {
	log.InitLog(log.InfoLog)
	fmt.Println("Start test the netserver...")

}
func TestNewP2PServer(t *testing.T) {
	log.Init(log.Stdout)
	fmt.Println("Start test new p2pserver...")

	p2p := NewServer()

	err := p2p.Start()
	if err != nil {
		fmt.Println("Start p2p error: ", err)
	}

	if p2p.GetVersion() != common.PROTOCOL_VERSION {
		t.Error("TestNewP2PServer p2p version error", p2p.GetVersion())
	}

	if p2p.GetVersion() != common.PROTOCOL_VERSION {
		t.Error("TestNewP2PServer p2p version error")
	}
	sync, cons := p2p.GetPort()
	if sync != 20338 {
		t.Error("TestNewP2PServer sync port error")
	}

	if cons != 20339 {
		t.Error("TestNewP2PServer consensus port error")
	}

	<- ch
}
