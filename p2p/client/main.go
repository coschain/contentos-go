package main

import (
	"fmt"

	"github.com/coschain/contentos-go/p2p/depend/common/log"
	"github.com/coschain/contentos-go/p2p/common"
	myp2p "github.com/coschain/contentos-go/p2p"
)

var ch chan int

func init() {
	log.InitLog(log.InfoLog)
	fmt.Println("Start test the netserver...")

}
func main() {
	log.Init(log.Stdout)
	fmt.Println("Start test new p2pserver...")

	p2p := myp2p.NewServer()

	err := p2p.Start()
	if err != nil {
		fmt.Println("Start p2p error: ", err)
	}

	if p2p.GetVersion() != common.PROTOCOL_VERSION {
		log.Error("TestNewP2PServer p2p version error", p2p.GetVersion())
	}

	if p2p.GetVersion() != common.PROTOCOL_VERSION {
		log.Error("TestNewP2PServer p2p version error")
	}
	sync, cons := p2p.GetPort()
	if sync != 20338 {
		log.Error("TestNewP2PServer sync port error")
	}

	if cons != 20339 {
		log.Error("TestNewP2PServer consensus port error")
	}

	<- ch
}
