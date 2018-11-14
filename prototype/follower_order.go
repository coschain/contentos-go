package prototype

import "github.com/coschain/contentos-go/common/encoding"

func (m *FollowerOrder) OpeEncode() ([]byte, error) {
	return encoding.Encode(m.Follower)
}
