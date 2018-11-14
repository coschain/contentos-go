package prototype

import "github.com/coschain/contentos-go/common/encoding"

func (m *FollowingOrder) OpeEncode() ([]byte, error) {
	return encoding.Encode(m.Following)
}
