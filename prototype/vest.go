package prototype

import "github.com/coschain/contentos-go/common/encoding"

func (m *Vest) OpeEncode() ([]byte, error) {
	return encoding.Encode(m.Value)
}

func (m *Vest) Add( o *Vest) error {

	if m.Value < o.Value + m.Value {
		return ErrVestOverflow
	}
	m.Value += o.Value
	return nil
}

func (m *Vest) Sub( o *Vest) error {
	if m.Value < o.Value {
		return ErrVestOverflow
	}
	m.Value -= o.Value
	return nil
}

func NewVest(value uint64) *Vest {
	return &Vest{Value:value}
}