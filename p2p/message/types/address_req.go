package types

import (
	comm "github.com/coschain/contentos-go/p2p/common"
	"github.com/coschain/contentos-go/p2p/depend/common"
)

type AddrReq struct{}

//Serialize message payload
func (this AddrReq) Serialization(sink *common.ZeroCopySink) error {
	return nil
}

func (this *AddrReq) CmdType() string {
	return comm.GetADDR_TYPE
}

//Deserialize message payload
func (this *AddrReq) Deserialization(source *common.ZeroCopySource) error {
	return nil
}
