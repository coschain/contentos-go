package prototype

import "github.com/coschain/contentos-go/common/encoding"

func (m *BpVoterId) OpeEncode() ([]byte, error) {
	return encoding.Encode(m.Voter.Value + "|" + m.Witness.Value)
}
