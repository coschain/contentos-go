package prototype

import "github.com/coschain/contentos-go/common/encoding/kope"

func (m *FollowerRelation) OpeEncode() ([]byte, error) {
	return kope.Encode(m.Account.Value, m.CreatedTime.UtcSeconds, m.Follower.Value)
}
