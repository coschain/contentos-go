package common

import (
	"errors"
	"strconv"
	"strings"

	"github.com/coschain/contentos-go/common/constants"
)

//peer capability
const (
	VERIFY_NODE  = 1 //peer involved in consensus
	SERVICE_NODE = 2 //peer only sync with consensus peer
)

//link and concurrent const
const (
	PER_SEND_LEN        = 1024 * 256 //byte len per conn write
	MAX_BUF_LEN         = 1024 * 256 //the maximum buffer to receive message
	WRITE_DEADLINE      = 5          //deadline of conn write
	REQ_INTERVAL        = 2          //single request max interval in second
	MAX_REQ_RECORD_SIZE = 1000       //the maximum request record size
	MAX_RESP_CACHE_SIZE = 50         //the maximum response cache
)

//msg cmd const
const (
	MSG_CMD_LEN      = 12               //msg type length in byte
	CHECKSUM_LEN     = 4                //checksum length in byte
	MSG_HDR_LEN      = 24               //msg hdr length in byte
	MAX_MSG_LEN      = 30 * 1024 * 1024 //the maximum message length
	MAX_PAYLOAD_LEN  = MAX_MSG_LEN - MSG_HDR_LEN
)

//msg type const
const (
	MAX_ADDR_NODE_CNT = 64 //the maximum peer address from msg
)

//info update const
const (
	PROTOCOL_VERSION      = 0     //protocol version
	KEEPALIVE_TIMEOUT     = 15    //contact timeout in sec
	DIAL_TIMEOUT          = 6     //connect timeout in sec
	CONN_MONITOR          = 6     //time to retry connect in sec
	CONN_MAX_BACK         = 4000  //max backoff time in micro sec
	MAX_RETRY_COUNT       = 3     //max reconnect time of remote peer
	CHAN_CAPABILITY       = 10000 //channel capability of recv link
)

// The peer state
const (
	INIT        = 0 //initial
	HAND        = 1 //send verion to peer
	HAND_SHAKE  = 2 //haven`t send verion to peer and receive peer`s version
	HAND_SHAKED = 3 //send verion to peer and receive peer`s version
	ESTABLISH   = 4 //receive peer`s verack
	INACTIVITY  = 5 //link broken
)

//const channel msg id and type
const (
	VERSION_TYPE    = "version"   //peer`s information
	VERACK_TYPE     = "verack"    //ack msg after version recv
	GetADDR_TYPE    = "getaddr"   //req nbr address from peer
	ADDR_TYPE       = "addr"      //nbr address
	PING_TYPE       = "ping"      //ping  sync height
	PONG_TYPE       = "pong"      //pong  recv nbr height
	//GET_DATA_TYPE   = "getdata"   //req data from peer
	BLOCK_TYPE      = "sig_block" //blk payload
	ID_TYPE         = "id"
	REQ_ID_TYPE     = "req_id"
	TX_TYPE         = "sig_trx"    //transaction
	DISCONNECT_TYPE = "disconnect" //peer disconnect info raise by link

	CONSENSUS_TYPE  = "consensus"
	CHECKPOINT_TYPE = "checkpoint"

	REQUEST_OUT_OF_RANGE_IDS_TYPE = "future_ids"
	REQUEST_BLOCK_BATCH_TYPE = "sigblk_batch"
	DETECT_FORMER_IDS_TYPE = "former_ids"
	CLEAR_OUT_OF_RABGE_STATE = "clear_state"
)

const (
	MaxTrxCountInBloomFiler       = 1000000    // max cache trx count
	BloomFilterOfRecvTrxArgM      = 14377588   // bloom filter bit size
	BloomFilterOfRecvTrxArgK      = 10         // bloom filter hash func num

	BATCH_LENGTH = 50   // length of id batch or block batch
	BLOCKS_SIZE_LIMIT = 2 * constants.MaxBlockSize
	MAX_BLOCK_COUNT = 50           // max block count

	HASH_SIZE = 32

	MaxConsensusMsgCount = 32768
)

//ParseIPAddr return ip address
func ParseIPAddr(s string) (string, error) {
	i := strings.Index(s, ":")
	if i < 0 {
		return "", errors.New("[p2p]split ip address error")
	}
	return s[:i], nil
}

//ParseIPPort return ip port
func ParseIPPort(s string) (string, error) {
	i := strings.Index(s, ":")
	if i < 0 {
		return "", errors.New("[p2p]split ip port error")
	}
	port, err := strconv.Atoi(s[i+1:])
	if err != nil {
		return "", errors.New("[p2p]parse port error")
	}
	if port <= 0 || port >= 65535 {
		return "", errors.New("[p2p]port out of bound")
	}
	return s[i:], nil
}
