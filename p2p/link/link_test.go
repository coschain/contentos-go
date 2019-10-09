package link

import (
	"bytes"
	"github.com/coschain/contentos-go/p2p/common"
	"github.com/coschain/contentos-go/prototype"
	"github.com/sirupsen/logrus"
	"testing"
	"time"

	mt "github.com/coschain/contentos-go/p2p/message/types"
	"github.com/stretchr/testify/assert"
)

var (
	cliLink    *Link
	serverLink *Link
	cliChan    chan *mt.MsgPayload
	serverChan chan *mt.MsgPayload
)

func init() {
	lg := logrus.New()
	cliLink = NewLink(lg)
	serverLink = NewLink(lg)

	cliLink.id = 0x12345
	serverLink.id = 0x6789

	cliLink.port = 20338
	serverLink.port = 20339

	cliChan = make(chan *mt.MsgPayload, 100)
	serverChan = make(chan *mt.MsgPayload, 100)
}

func TestNewLink(t *testing.T) {

	id := 0x74936295
	port := 40339

	if cliLink.GetID() != 0x12345 {
		t.Fatal("link GetID failed")
	}

	cliLink.SetID(uint64(id))
	if cliLink.GetID() != uint64(id) {
		t.Fatal("link SetID failed")
	}

	if cliLink.GetPort() != 20338 {
		t.Fatal("link GetPort failed")
	}

	cliLink.SetPort(uint32(port))
	if cliLink.GetPort() != uint32(port) {
		t.Fatal("link SetPort failed")
	}

	cliLink.SetChan(cliChan)
	serverLink.SetChan(serverChan)

	cliLink.UpdateRXTime(time.Now())

	msgdata := new(mt.TransferMsg)
	msgdata.Msg = &mt.TransferMsg_Msg7{Msg7:&mt.Disconnected{}}

	msg := &mt.MsgPayload{
		Id:      cliLink.id,
		Addr:    cliLink.addr,
		Payload: msgdata,
	}
	go func() {
		time.Sleep(5000000)
		cliChan <- msg
	}()

	timeout := time.NewTimer(time.Second)
	select {
	case <-cliLink.recvChan:
		t.Log("read data from channel")
	case <-timeout.C:
		timeout.Stop()
		t.Fatal("can`t read data from link channel")
	}

}

func TestUnpackBufNode(t *testing.T) {
	cliLink.SetChan(cliChan)

	msgType := "block"

	var msg mt.Message

	switch msgType {
	case "sig_trx":
		msgdata := new(mt.TransferMsg)
		trx := &prototype.Transaction{
			RefBlockNum:    1,
			RefBlockPrefix: 2,
		}

		sigtrx := new(prototype.SignedTransaction)
		sigtrx.Trx = trx
		data := new(mt.BroadcastSigTrx)
		data.SigTrx = sigtrx

		msgdata.Msg = &mt.TransferMsg_Msg1{Msg1:data}

		msg = msgdata
	case "block":
		msgdata := new(mt.TransferMsg)
		sigBlk := new(prototype.SignedBlock)
		sigBlkHdr := new(prototype.SignedBlockHeader)
		sigBlkHdr.Header = new(prototype.BlockHeader)
		sigBlkHdr.Header.BlockProducer = new(prototype.AccountName)
		sigBlkHdr.Header.BlockProducer.Value = "alice"
		sigBlk.SignedHeader = sigBlkHdr

		data := new(mt.SigBlkMsg)
		data.SigBlk = sigBlk

		msgdata.Msg = &mt.TransferMsg_Msg3{Msg3:data}

		msg = msgdata
	}

	sink := common.NewZeroCopySink(nil)
	err := mt.WriteMessage(sink, msg, 0x12345)
	assert.Nil(t, err)

	buf := bytes.NewBuffer(sink.Bytes())
	demsg, _, err := mt.ReadMessage(buf, 0x12345)
	assert.NotNil(t, demsg)
	assert.Nil(t, err)

	serverLink.disconnectNotify()
}
