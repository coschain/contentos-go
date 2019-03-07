package prototype

import (
	"fmt"
	"github.com/coschain/contentos-go/common/encoding/kope"
	"time"
)

var timeFormat = "2006-01-01 00:00:00"

func (m *TimePointSec) OpeEncode() ([]byte, error) {
	return kope.Encode(m.UtcSeconds)
}

func (m TimePointSec) Add(value uint32) TimePointSec {
	m.UtcSeconds += value
	return m
}

func (m *TimePointSec) ToString() string {
	return time.Unix( int64(m.UtcSeconds), 0).Format(timeFormat)
}



func (m *TimePointSec) MarshalJSON() ([]byte, error) {
	val := fmt.Sprintf("\"%s\"", m.ToString())
	return []byte(val), nil
}

func stripJsonQuota(input []byte)([]byte, error)  {
	if len(input) < 2 {
		return nil,ErrJSONFormatErr
	}
	if input[0] != '"' {
		return nil,ErrJSONFormatErr
	}
	return input[1 : len(input)-1], nil
}

func TimePointSecFromString(str string) (*TimePointSec, error) {
	value, err := time.Parse(timeFormat, str)
	if err != nil {
		return nil, err
	}
	return &TimePointSec{ UtcSeconds:uint32(value.Second()) }, nil
}


func (m *TimePointSec) UnmarshalJSON(input []byte) error {

	strBuffer, err := stripJsonQuota(input)
	if err != nil {
		return err
	}

	res, err := TimePointSecFromString(string(strBuffer))
	if err != nil {
		return err
	}
	m.UtcSeconds = res.UtcSeconds
	return nil
}



func NewTimePointSec(value uint32) *TimePointSec {
	return &TimePointSec{UtcSeconds: value}
}
