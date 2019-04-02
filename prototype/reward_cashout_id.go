package prototype

import "github.com/coschain/contentos-go/common/encoding/kope"

func (m *RewardCashoutId) OpeEncode() ([]byte, error) {
	return kope.Encode(m.Account, m.BlockHeight)
}