package prototype

import "github.com/coschain/contentos-go/common/encoding"

func (m *TimePointSec) OpeEncode() ([]byte, error) {
	return encoding.Encode(m.UtcSeconds)
}

func (m TimePointSec) Add( value uint32 ) TimePointSec {
	m.UtcSeconds += value
	return m
}

func NewTimePointSec(value uint32) *TimePointSec {
	return &TimePointSec{UtcSeconds:value}
}