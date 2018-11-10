package main

import (
	"fmt"
	"time"
//	"github.com/coschain/contentos-go/p2p/message/msg_pack"

	"github.com/coschain/contentos-go/p2p/depend/common/log"
	conn "github.com/coschain/contentos-go/p2p/depend/common"
	"github.com/coschain/contentos-go/p2p/common"
//	"github.com/coschain/contentos-go/p2p/depend/core/types"
	myp2p "github.com/coschain/contentos-go/p2p"
)

var ch chan int

func init() {
	log.InitLog(log.DebugLog)
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

	time.Sleep(28 * time.Second)

	for i:=0;i<10;i++ {
		var tmp conn.Uint256
		tmp[0] = 8
		tmp[1] = 88
		tmp[2] = 188
		p2p.Xmit(tmp)
//		var trx types.Transaction
//		trx.GasLimit = uint64(2 * i + 6)
//		trx.GasPrice = uint64(3 * i + 8)
//		msg := msgpack.NewTxn(&trx)
//		p2p.Xmit(msg)
//		log.Info("Successfully broadcast a transaction, trx.GasLimit: ", trx.GasLimit, " trx.GasPrice: ", trx.GasPrice)
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
