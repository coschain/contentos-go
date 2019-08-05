package prototype

import "github.com/coschain/contentos-go/common/encoding/kope"

func (m *BpVestId) OpeEncode() ([]byte, error) {
	return kope.Encode(m.Active, m.VoteVest.Value)
}