package prototype

import "github.com/coschain/contentos-go/common/encoding"

func (m *FollowerRelation) OpeEncode() ([]byte, error) {
	return encoding.Encode(m.Account.Value)
}
