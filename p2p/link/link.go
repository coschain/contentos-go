package link

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/coschain/contentos-go/p2p/common"
	"github.com/coschain/contentos-go/p2p/message/types"
)

//Link used to establish
type Link struct {
	id        uint64
	addr      string                 // The address of the node
	conn      net.Conn               // Connect socket with the peer node
	port      uint32                 // The server port of the node
	time      time.Time              // The latest time the node activity
	recvChan  chan *types.MsgPayload //msgpayload channel
	reqIdRecord int64                //Map RequestId to Timestamp, using for rejecting too fast REQ_ID request in specific time

	lock sync.RWMutex
}

func NewLink() *Link {
	link := &Link{
		reqIdRecord: 0,
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

func (this *Link) Rx(magic uint32) {
	conn := this.conn
	if conn == nil {
		return
	}

	reader := bufio.NewReaderSize(conn, common.MAX_BUF_LEN)

	for {
		msg, payloadSize, err := types.ReadMessage(reader, magic)
		if err != nil {
			fmt.Println("read msg error: ", err)
			break
		}

		t := time.Now()
		this.UpdateRXTime(t)

		//if !this.needSendMsg(msg) {
		//	continue
		//}

		this.recvChan <- &types.MsgPayload{
			Id:          this.id,
			Addr:        this.addr,
			PayloadSize: payloadSize,
			Payload:     msg,
		}

	}

	this.disconnectNotify()
}

//disconnectNotify push disconnect msg to channel
func (this *Link) disconnectNotify() {
	this.CloseConn()

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

func (this *Link) Tx(msg types.Message, magic uint32) error {
	conn := this.conn
	if conn == nil {
		return errors.New("[p2p]tx link invalid")
	}

	// TODO just for test,should be deleted when test is done
	// **********************************
	// sleepRandomTime()
	// **********************************

	sink := common.NewZeroCopySink(nil)
	err := types.WriteMessage(sink, msg, magic)
	if err != nil {
		return errors.New( fmt.Sprintf("[p2p] error serialize messge: %v", err) )
	}

	payload := sink.Bytes()
	nByteCnt := len(payload)

	nCount := nByteCnt / common.PER_SEND_LEN
	if nCount == 0 {
		nCount = 1
	}

	this.sleepForSpeedLimit(nByteCnt)

	conn.SetWriteDeadline(time.Now().Add(time.Duration(nCount*common.WRITE_DEADLINE) * time.Second))
	_, err = conn.Write(payload)
	if err != nil {
		errStr := fmt.Sprintf("[p2p] socket buffer write too much time, error sending messge to %s :%s", this.GetAddr(), err.Error())
		fmt.Println(errStr, " ,timestamp: ", time.Now())
		this.disconnectNotify()
		return errors.New( fmt.Sprintf("[p2p] socket buffer write too much time, error sending messge to %s :%s", this.GetAddr(), err.Error()) )
	}

	return nil
}

func (this *Link) sleepForSpeedLimit( count int) {

	this.lock.Lock()
	defer this.lock.Unlock()
	time.Sleep( time.Duration(count) * time.Second / time.Duration(common.SpeedLimit) )
}


func sleepRandomTime() {
	//r := rand.New(rand.NewSource(time.Now().UnixNano()))
	//delay := 500 + r.Intn(501)
	time.Sleep( 150 * time.Millisecond )
}

//needSendMsg check whether the msg is needed to push to channel
func (this *Link) needSendMsg(msg types.Message) bool {
	if msg.CmdType() != common.REQ_ID_TYPE {
		return true
	}
	now := time.Now().Unix()

	if now - this.reqIdRecord < common.REQ_INTERVAL {
		return false
	}

	this.reqIdRecord = now
	return true
}
