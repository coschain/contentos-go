package types

import "github.com/coschain/contentos-go/p2p/depend/common"

type SmartCodeEvent struct {
	TxHash common.Uint256
	Action string
	Result interface{}
	Error  int64
}
