package prototype

import "github.com/coschain/contentos-go/common/encoding/kope"

func (m *UserTrxCreateOrder)OpeEncode() ([]byte, error) {
	return kope.Encode(m.Creator,m.CreateTime)
}