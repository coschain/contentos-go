package prototype

import "github.com/coschain/contentos-go/common/encoding"

func (m *Sha256) OpeEncode() ([]byte, error) {
	return encoding.Encode(m.Hash)
}
