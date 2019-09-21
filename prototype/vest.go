package prototype

import (
	"fmt"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/common/encoding/kope"
	"math"
	"strconv"
	"strings"
)

func (m *Vest) OpeEncode() ([]byte, error) {
	return kope.Encode(m.Value)
}

func (m *Vest) Add(o *Vest) *Vest {

	if m.Value > o.Value+m.Value {
		panic(fmt.Sprintf("Vest Add Overflow: %v, %v", m.Value, o.Value))
	}
	m.Value += o.Value
	return m
}

func (m *Vest) Sub(o *Vest) *Vest {
	if m.Value < o.Value {
		panic(fmt.Sprintf("Vest Sub Overflow: %v, %v", m.Value, o.Value))
	}
	m.Value -= o.Value
	return m
}

func (m *Vest) Mul(c uint64) *Vest {
	if m.Value == 0 {
		return m
	}
	if math.MaxUint64 / m.Value < c {
		panic(fmt.Sprintf("Vest Mul Overflow: %v, %v", m.Value, c))
	}
	m.Value *= c
	return m
}

func (m *Vest) ToCoin() *Coin {
	return NewCoin(m.Value)
}

func (m *Vest) ToString() string {
	var result = m.Value

	var high = result / constants.COSTokenDecimals
	var low = result % constants.COSTokenDecimals

	return fmt.Sprintf("%d.%06d %s",
		high,low,
		constants.VestSymbol)
}

func (m *Vest) MarshalJSON() ([]byte, error) {
	val := fmt.Sprintf("\"%s\"", m.ToString())
	return []byte(val), nil
}


func VestFromString(buf string) (*Coin, error) {

	if len(buf) < 7 + 5 {
		return nil, ErrVestFormatErr
	}

	if !strings.HasSuffix( buf, " " + constants.VestSymbol ){
		return nil, ErrVestFormatErr
	}

	str := string( buf[0:len(buf)-5] )

	res := strings.Split(str, ".")
	if len(res) != 2 {
		return nil, ErrVestFormatErr
	}

	if len(res[1]) != 6 {
		return nil, ErrVestFormatErr
	}

	high, err := strconv.Atoi(res[0])
	if err != nil {
		return nil, ErrVestFormatErr
	}

	low, err := strconv.Atoi(res[0])
	if err != nil {
		return nil, ErrVestFormatErr
	}

	return &Coin{ Value:uint64(high*constants.COSTokenDecimals+low) }, nil
}


func (m *Vest) UnmarshalJSON(input []byte) error {

	buffer, err := stripJsonQuota(input)
	if err != nil {
		return err
	}

	res, err := VestFromString(string(buffer))
	if err != nil {
		return err
	}
	m.Value = res.Value
	return nil
}

func NewVest(value uint64) *Vest {
	return &Vest{Value: value}
}
