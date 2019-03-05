package prototype

import (
	"fmt"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/common/encoding/kope"
	"strconv"
	"strings"
)

func (m *Coin) OpeEncode() ([]byte, error) {
	return kope.Encode(m.Value)
}

func (m *Coin) NonZero() bool {
	return m.Value != 0
}

func (m *Coin) Add(o *Coin) error {

	if m.Value > o.Value+m.Value {
		return ErrCoinOverflow
	}
	m.Value += o.Value
	return nil
}

func (m *Coin) Sub(o *Coin) error {
	if m.Value < o.Value {
		return ErrCoinOverflow
	}
	m.Value -= o.Value
	return nil
}

func (m *Coin) ToString() string {
	var result float64

	result = float64(m.Value * 1.0) / float64(constants.COSTokenDecimals * 1.0)
	return fmt.Sprintf("%.6f %s",
		result,
		constants.CoinSymbol)
}

func (m *Coin) MarshalJSON() ([]byte, error) {
	val := fmt.Sprintf("\"%s\"", m.ToString())
	return []byte(val), nil
}


func CoinFromString(buf string) (*Coin, error) {

	if len(buf) < 7 + 4 {
		return nil, ErrCoinFormatErr
	}

	if !strings.HasSuffix( buf, " " + constants.CoinSymbol ){
		return nil, ErrCoinFormatErr
	}

	str := string( buf[0:len(buf)-4] )

	res := strings.Split(str, ".")
	if len(res) != 2 {
		return nil, ErrCoinFormatErr
	}

	if len(res[1]) != 6 {
		return nil, ErrCoinFormatErr
	}

	high, err := strconv.Atoi(res[0])
	if err != nil {
		return nil, ErrCoinFormatErr
	}

	low, err := strconv.Atoi(res[0])
	if err != nil {
		return nil, ErrCoinFormatErr
	}

	return &Coin{ Value:uint64(high*constants.COSTokenDecimals+low) }, nil
}


func (m *Coin) UnmarshalJSON(input []byte) error {

	buffer, err := stripJsonQuota(input)
	if err != nil {
		return err
	}

	res, err := CoinFromString(string(buffer))
	if err != nil {
		return err
	}
	m.Value = res.Value
	return nil
}



func (m *Coin) ToVest() *Vest {
	return NewVest(m.Value)
}

func NewCoin(value uint64) *Coin {
	return &Coin{Value: value}
}
