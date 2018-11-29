package msg

import (
	"fmt"
	"testing"

	"github.com/coschain/contentos-go/prototype"
	"github.com/gogo/protobuf/proto"
)

func Test_Serialize(t *testing.T) {
	// transaction
	trx := &prototype.Transaction{
		RefBlockNum:    1,
		RefBlockPrefix: 2,
	}

	sigtrx := new(prototype.SignedTransaction)
	sigtrx.Trx = trx
	msg := new(BroadcastSigTrx)
	msg.SigTrx = sigtrx

	fmt.Printf("before Marshal sig_trx:    +%v\n", msg)

	trxdata, err := proto.Marshal(msg)
	if err != nil {
		t.Error("sig_trx Marshal failed")
	}

	var obj BroadcastSigTrx
	err = proto.Unmarshal(trxdata, &obj)
	if err != nil {
		t.Error("sig_trx Marshal failed")
	}

	fmt.Printf("after Unmarshal sig_trx:     +%v\n", &obj)

	sigBlk := new(prototype.SignedBlock)
	sigBlkHdr := new(prototype.SignedBlockHeader)
	sigBlkHdr.Header = new(prototype.BlockHeader)
	sigBlkHdr.Header.Witness = "hanyunlong"
	sigBlk.SignedHeader = sigBlkHdr

	msg2 := new(BroadcastSigBlk)
	msg2.SigBlk = sigBlk

	fmt.Printf("before Marshal sig_blk:    +%v\n", msg2)

	blkdata, err := proto.Marshal(msg2)
	if err != nil {
		t.Error("sig_trx Marshal failed")
	}

	var obj2 BroadcastSigBlk
	err = proto.Unmarshal(blkdata, &obj2)
	if err != nil {
		t.Error("sig_trx Marshal failed")
	}

	fmt.Printf("after Unmarshal sig_trx:     +%v\n", &obj2)
}
