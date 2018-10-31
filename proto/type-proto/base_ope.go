package prototype

import "github.com/coschain/contentos-go/common/encoding"

func (m *AccountName) OpeEncode() ([]byte, error) {
	return encoding.Encode(m.Value)
}

func (m *PublicKeyType) OpeEncode() ([]byte, error) {
	return encoding.Encode(m.Data)
}

func (m *TimePointSec) OpeEncode() ([]byte, error) {
	return encoding.Encode(m.UtcSeconds)
}

func (m *Safe64) OpeEncode() ([]byte, error) {
	return encoding.Encode(m.Value)
}

func (m *Coin) OpeEncode() ([]byte, error) {
	return m.Amount.OpeEncode()
}
