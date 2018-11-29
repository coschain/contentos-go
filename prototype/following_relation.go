package prototype

import "github.com/coschain/contentos-go/common/encoding/kope"

func (m *FollowingRelation) OpeEncode() ([]byte, error) {
	return kope.Encode(m.Account.Value, m.Following.Value)
}

func (m *FollowingCreatedOrder) OpeEncode() ([]byte, error) {
	return kope.Encode(m.Account.Value, m.CreatedTime.UtcSeconds, m.Following.Value)
}
