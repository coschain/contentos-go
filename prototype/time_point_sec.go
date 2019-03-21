package prototype

import (
	"fmt"
	"github.com/coschain/contentos-go/common/encoding/kope"
	"time"
)

var constTimeFormat = "2006/1/2 15:04:05"

func (m *TimePointSec) OpeEncode() ([]byte, error) {
	return kope.Encode(m.UtcSeconds)
}

func (m TimePointSec) Add(value uint32) TimePointSec {
	m.UtcSeconds += value
	return m
}

func (m *TimePointSec) ToString() string {
	return time.Unix( int64(m.UtcSeconds), 0).UTC().Format(constTimeFormat)
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
	value, err := time.Parse(constTimeFormat, str)
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
