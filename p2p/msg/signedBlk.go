package msg

import (
	"github.com/coschain/contentos-go/p2p/common"
	comm "github.com/coschain/contentos-go/p2p/depend/common"
	"github.com/gogo/protobuf/proto"
)

//Serialize message payload
func (this *BroadcastSigBlk) Serialization(sink *comm.ZeroCopySink) error {
	data, _ := proto.Marshal(this)
	sink.WriteBytes(data)
	return nil
}

func (this *BroadcastSigBlk) CmdType() string {
	return common.BLOCK_TYPE
}

//Deserialize message payload
func (this *BroadcastSigBlk) Deserialization(source *comm.ZeroCopySource) error {
	var tmp BroadcastSigBlk
	err := proto.Unmarshal(source.Data(), &tmp)
	if err != nil {
		return err
	}
	*this = tmp
	return nil
}