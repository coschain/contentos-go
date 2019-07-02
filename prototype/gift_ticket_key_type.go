package prototype

import "github.com/coschain/contentos-go/common/encoding/kope"

func (m *GiftTicketKeyType) OpeEncode() ([]byte, error) {
	return kope.Encode(m.Type, m.From, m.To, m.CreateBlock)
}
