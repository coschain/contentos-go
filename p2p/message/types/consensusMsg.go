package types

import (
	"github.com/coschain/contentos-go/p2p/common"
	"github.com/coschain/gobft/message"
)

type ConsMsg struct {
	MsgData message.ConsensusMessage
}

//Serialize message payload
func (this *ConsMsg) Serialization(sink *common.ZeroCopySink) error {
	data := this.MsgData.Bytes()
	sink.WriteBytes(data)
	return nil
}

func (this *ConsMsg) CmdType() string {
	return common.CONSENSUS_TYPE
}

//Deserialize message payload
func (this *ConsMsg) Deserialization(source *common.ZeroCopySource) error {
	msgdata, err := message.DecodeConsensusMsg(source.Data())
	if err != nil {
		return err
	}
	this.MsgData = msgdata
	return nil
}