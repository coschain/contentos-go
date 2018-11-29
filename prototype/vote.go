package prototype

import "github.com/coschain/contentos-go/common/encoding/kope"

func (m *VoterId) OpeEncode() ([]byte, error) {
	return kope.Encode(m.Voter.Value, m.PostId)
}
