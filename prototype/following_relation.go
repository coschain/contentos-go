package prototype

import "github.com/coschain/contentos-go/common/encoding/kope"

func (m *FollowingRelation) OpeEncode() ([]byte, error) {
	return kope.Encode(m.Account.Value)
}
