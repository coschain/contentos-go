package msg

import (
	"github.com/coschain/contentos-go/p2p/common"
	comm "github.com/coschain/contentos-go/p2p/depend/common"
	"github.com/gogo/protobuf/proto"
)

//Serialize message payload
func (this *BroadcastSigTrx) Serialization(sink *comm.ZeroCopySink) error {
	data, _ := proto.Marshal(this)
	sink.WriteBytes(data)
	return nil
}

func (this *BroadcastSigTrx) CmdType() string {
	return common.TX_TYPE
}

//Deserialize message payload
func (this *BroadcastSigTrx) Deserialization(source *comm.ZeroCopySource) error {
	var tmp BroadcastSigTrx
	err := proto.Unmarshal(source.Data(), &tmp)
	if err != nil {
		return err
	}
	this = &tmp
	return nil
}