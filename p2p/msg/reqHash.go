package msg

import (
	"github.com/coschain/contentos-go/p2p/common"
	comm "github.com/coschain/contentos-go/p2p/depend/common"
	"github.com/gogo/protobuf/proto"
)

//Serialize message payload
func (this *ReqHashMsg) Serialization(sink *comm.ZeroCopySink) error {
	data, _ := proto.Marshal(this)
	sink.WriteBytes(data)
	return nil
}

func (this *ReqHashMsg) CmdType() string {
	return common.REQ_HASH_TYPE
}

//Deserialize message payload
func (this *ReqHashMsg) Deserialization(source *comm.ZeroCopySource) error {
	var tmp ReqHashMsg
	err := proto.Unmarshal(source.Data(), &tmp)
	if err != nil {
		return err
	}
	*this = tmp
	return nil
}