package msg

//import (
//	"github.com/coschain/contentos-go/p2p/common"
//	comm "github.com/coschain/contentos-go/p2p/depend/common"
//	"github.com/gogo/protobuf/proto"
//)
//
////Serialize message payload
//func (this *IdMsg) Serialization(sink *comm.ZeroCopySink) error {
//	data, _ := proto.Marshal(this)
//	sink.WriteBytes(data)
//	return nil
//}
//
//func (this *IdMsg) CmdType() string {
//	return common.ID_TYPE
//}
//
////Deserialize message payload
//func (this *IdMsg) Deserialization(source *comm.ZeroCopySource) error {
//	var tmp IdMsg
//	err := proto.Unmarshal(source.Data(), &tmp)
//	if err != nil {
//		return err
//	}
//	*this = tmp
//	return nil
//}