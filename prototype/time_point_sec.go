package prototype

import "github.com/coschain/contentos-go/common/encoding"

func (m *TimePointSec) OpeEncode() ([]byte, error) {
	return encoding.Encode(m.UtcSeconds)
}
