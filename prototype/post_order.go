package prototype

import "github.com/coschain/contentos-go/common/encoding/kope"

func (m *PostCreatedOrder) OpeEncode() ([]byte, error) {
	return kope.Encode(m.Created, m.ParentId)
}

func (m *ReplyCreatedOrder) OpeEncode() ([]byte, error) {
	return kope.Encode(m.ParentId, m.Created)
}
