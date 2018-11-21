package prototype

import "github.com/coschain/contentos-go/common/encoding"

func (m *Coin) OpeEncode() ([]byte, error) {
	return encoding.Encode(m.Value)
}

func (m *Coin) NonZero() bool {
	return m.Value != 0
}

func (m *Coin) Add( o *Coin) error {

	if m.Value > o.Value + m.Value {
		return ErrCoinOverflow
	}
	m.Value += o.Value
	return nil
}

func (m *Coin) Sub( o *Coin) error {
	if m.Value < o.Value {
		return ErrCoinOverflow
	}
	m.Value -= o.Value
	return nil
}

func (m *Coin) ToVest() *Vest {
	return NewVest(m.Value)
}

func NewCoin(value uint64) *Coin {
	return &Coin{Value:value}
}