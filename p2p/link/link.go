package link

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/coschain/contentos-go/p2p/common"
	"github.com/coschain/contentos-go/p2p/message/types"
)

//Link used to establish
type Link struct {
	log       *logrus.Logger
	id        uint64
	addr      string                 // The address of the node
	conn      net.Conn               // Connect socket with the peer node
	port      uint32                 // The server port of the node
	time      time.Time              // The latest time the node activity
	recvChan  chan *types.MsgPayload //msgpayload channel
	stopRecv  chan bool

	reqIdRecord int64                //Map RequestId to Timestamp, using for rejecting too fast REQ_ID request in specific time

	sendChan chan types.Message
	stopSend chan bool

	sync.RWMutex
}

func NewLink(lg *logrus.Logger) *Link {
	link := &Link{
		log:         lg,
		reqIdRecord: 0,
		sendChan:    make(chan types.Message, common.PEER_SEND_CHAN_LENGTH),
		stopRecv:    make(chan bool, 1),
		stopSend:    make(chan bool, 1),
	}
	return link
}

//SetID set peer id to link
func (this *Link) SetID(id uint64) {
	this.id = id
}

//GetID return if from peer
func (this *Link) GetID() uint64 {
	return this.id
}

//If there is connection return true
func (this *Link) Valid() bool {
	return this.conn != nil
}

//set message channel for link layer
func (this *Link) SetChan(msgchan chan *types.MsgPayload) {
	this.recvChan = msgchan
}

//get address
func (this *Link) GetAddr() string {
	return this.addr
}

//set address
func (this *Link) SetAddr(addr string) {
	this.addr = addr
}

//set port number
func (this *Link) SetPort(p uint32) {
	this.port = p
}

//get port number
func (this *Link) GetPort() uint32 {
	return this.port
}

//get connection
func (this *Link) GetConn() net.Conn {
	return this.conn
}

//set connection
func (this *Link) SetConn(conn net.Conn) {
	this.conn = conn
}

//record latest message time
func (this *Link) UpdateRXTime(t time.Time) {
	this.time = t
}

//GetRXTime return the latest message time
func (this *Link) GetRXTime() time.Time {
	return this.time
}

func (this *Link) SendMessage(msg types.Message) error {
	this.Lock()
	defer this.Unlock()

	if len(this.sendChan) == cap(this.sendChan) {
		this.log.Warn(errors.New(fmt.Sprintf("peer send buffer is full, discard this message. destination %s", this.addr)))
		return errors.New("peer send buffer is full, discard this message")
	}
	this.sendChan <- msg
	return nil
}

func (this *Link) StopSendMessage() {
	if len(this.stopSend) == cap(this.stopSend) {
		return
	}
	this.stopSend <- true
}

func (this *Link) StopRecvMessage() {
	if len(this.stopRecv) == cap(this.stopRecv) {
		return
	}
	this.stopRecv <- true
}

func (this *Link) Rx(magic uint32) {
	conn := this.conn
	if conn == nil {
		return
	}

	reader := bufio.NewReaderSize(conn, common.MAX_BUF_LEN)

	for {
		select {
		case <-this.stopRecv:
			this.log.Info("Stop receive message from peer: ", this.addr)
			return
		default:
			msg, payloadSize, err := types.ReadMessage(reader, magic)
			if err != nil {
				this.log.Infof("read from peer %s error: %v", this.addr, err)
				this.disconnectNotify()
				return
			}

			t := time.Now()
			this.UpdateRXTime(t)

			this.recvChan <- &types.MsgPayload{
				Id:          this.id,
				Addr:        this.addr,
				PayloadSize: payloadSize,
				Payload:     msg,
			}
		}
	}
}

//disconnectNotify push disconnect msg to channel
func (this *Link) disconnectNotify() {
	//this.CloseConn()

	reqmsg := NewDisconnected()

	discMsg := &types.MsgPayload{
		Id:      this.id,
		Addr:    this.addr,
		Payload: reqmsg,
	}
	this.recvChan <- discMsg
}

func NewDisconnected() *types.TransferMsg {
	var reqmsg types.TransferMsg
	data := new(types.Disconnected)

	reqmsg.Msg = &types.TransferMsg_Msg7{Msg7:data}
	return &reqmsg
}

//close connection
func (this *Link) CloseConn() {
	if this.conn != nil {
		this.conn.Close()
		this.conn = nil
	}
}

func (this *Link) Tx(magic uint32) {
	for {
		select {
		case <- this.stopSend:
			this.log.Info("Stop send message to peer ", this.GetAddr())
			return
		case msg := <-this.sendChan:
			conn := this.conn
			if conn == nil {
				this.log.Error("[p2p]tx link invalid")
				return
			}

			sink := common.NewZeroCopySink(nil)
			err := types.WriteMessage(sink, msg, magic)
			if err != nil {
				this.log.Error(errors.New( fmt.Sprintf("[p2p] error serialize messge: %v", err) ))
				return
			}

			payload := sink.Bytes()
			nByteCnt := len(payload)

			nCount := nByteCnt / common.PER_SEND_LEN
			if nCount == 0 {
				nCount = 1
			}
			conn.SetWriteDeadline(time.Now().Add(time.Duration(nCount*common.WRITE_DEADLINE) * time.Second))
			_, err = conn.Write(payload)
			if err != nil {
				this.log.Error(errors.New( fmt.Sprintf("[p2p] socket buffer write too much time, error sending messge to %s :%s", this.GetAddr(), err.Error()) ))
				this.disconnectNotify()
				return
			}
		}
	}
}
