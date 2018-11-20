package main

import (
	"fmt"
	"github.com/coschain/contentos-go/prototype"
	"time"

	myp2p "github.com/coschain/contentos-go/p2p"
	"github.com/coschain/contentos-go/p2p/common"
	"github.com/coschain/contentos-go/p2p/depend/common/log"
)

var ch chan int

func init() {
	log.InitLog(log.DebugLog)
	fmt.Println("Start test the netserver...")
}

func main() {
	log.Init(log.Stdout)
	fmt.Println("Start test new p2pserver...")

	p2p, err := myp2p.NewServer(nil)

	err = p2p.Start(nil)
	if err != nil {
		fmt.Println("Start p2p error: ", err)
	}

	time.Sleep(28 * time.Second)

	for i:=0;i<1;i++ {
		// Broadcast signedTransaction
		trx := &prototype.Transaction{
			RefBlockNum:    1,
			RefBlockPrefix: 2,
		}

		sigtrx := new(prototype.SignedTransaction)
		sigtrx.Trx = trx
		p2p.Network.Broadcast(sigtrx)


		// Broadcast signedBlock
		sigBlk := new(prototype.SignedBlock)
		sigBlkHdr := new(prototype.SignedBlockHeader)
		sigBlkHdr.Header = new(prototype.BlockHeader)
		sigBlkHdr.Header.Witness = new(prototype.AccountName)
		sigBlkHdr.Header.Witness.Value = "alice"

		sigBlkHdr.Header.Previous = new(prototype.Sha256)
		sigBlkHdr.Header.TransactionMerkleRoot = new(prototype.Sha256)
		sigBlkHdr.Header.Previous.Hash = make([]byte, 32)
		sigBlkHdr.Header.TransactionMerkleRoot.Hash = make([]byte, 32)

		sigBlk.SignedHeader = sigBlkHdr
		p2p.Network.Broadcast(sigBlk)
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
