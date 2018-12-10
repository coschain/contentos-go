package msg

import (
	"github.com/coschain/contentos-go/p2p/common"
	comm "github.com/coschain/contentos-go/p2p/depend/common"
	"github.com/gogo/protobuf/proto"
)

//Serialize message payload
func (this *TransferMsg) Serialization(sink *comm.ZeroCopySink) error {
	data, _ := proto.Marshal(this)
	sink.WriteBytes(data)
	return nil
}

func (this *TransferMsg) CmdType() ( res string) {
	switch this.Msg.(type) {
	case *TransferMsg_Msg1:
		res = common.TX_TYPE
	case *TransferMsg_Msg2:
		res = common.ID_TYPE
	case *TransferMsg_Msg3:
		res = common.BLOCK_TYPE
	case *TransferMsg_Msg4:
		res = common.REQ_ID_TYPE
	case *TransferMsg_Msg5:
		res = common.ADDR_TYPE
	case *TransferMsg_Msg6:
		res = common.GetADDR_TYPE
	case *TransferMsg_Msg7:
		res = common.DISCONNECT_TYPE
	case *TransferMsg_Msg8:
		res = common.PING_TYPE
	case *TransferMsg_Msg9:
		res = common.PONG_TYPE
	case *TransferMsg_Msg10:
		res = common.VERACK_TYPE
	case *TransferMsg_Msg11:
		res = common.VERSION_TYPE
	default:
		res = "unknow msg"
	}
	return res
}

//Deserialize message payload
func (this *TransferMsg) Deserialization(source *comm.ZeroCopySource) error {
	var tmp TransferMsg
	err := proto.Unmarshal(source.Data(), &tmp)
	if err != nil {
		return err
	}
	*this = tmp
	return nil
}