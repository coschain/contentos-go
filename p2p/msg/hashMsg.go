package msg

import (
	"github.com/coschain/contentos-go/p2p/common"
	comm "github.com/coschain/contentos-go/p2p/depend/common"
	"github.com/gogo/protobuf/proto"
)

//Serialize message payload
func (this *HashMsg) Serialization(sink *comm.ZeroCopySink) error {
	data, _ := proto.Marshal(this)
	sink.WriteBytes(data)
	return nil
}

func (this *HashMsg) CmdType() string {
	return common.HASH_TYPE
}

//Deserialize message payload
func (this *HashMsg) Deserialization(source *comm.ZeroCopySource) error {
	var tmp HashMsg
	err := proto.Unmarshal(source.Data(), &tmp)
	if err != nil {
		return err
	}
	*this = tmp
	return nil
}