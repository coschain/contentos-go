package types

import (
	"github.com/coschain/contentos-go/p2p/common"
	comm "github.com/coschain/contentos-go/p2p/depend/common"
)

type Disconnected struct{}

//Serialize message payload
func (this Disconnected) Serialization(sink *comm.ZeroCopySink) error {
	return nil
}

func (this Disconnected) CmdType() string {
	return common.DISCONNECT_TYPE
}

//Deserialize message payload
func (this *Disconnected) Deserialization(source *comm.ZeroCopySource) error {
	return nil
}
