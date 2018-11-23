package consensus

import (
	"github.com/coschain/contentos-go/common"
	//"github.com/coschain/contentos-go/proto/type-proto"
)

type IProducer interface {
	Produce() (common.ISignedBlock, error)
}
