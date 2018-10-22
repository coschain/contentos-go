package common

import (
	"errors"
)

// Name is a uint128 representation of name
type Name struct {
	lo uint64
	hi uint64
}

// StringToName converts a string representation to Name
func StringToName(s string) (*Name, error) {
	len := len(s)
	if len > 16 {
		return nil, errors.New("Name too long")
	}

	var l, h uint64
	b := []byte(s)
	for i := 0; i < len && i < 8; i++ {
		h |= uint64(b[i]) << uint64((7-i)*8)
	}

	for i := 8; i < len && i < 16; i++ {
		l |= uint64(b[i]) << uint64((15-i)*8)
	}
	return &Name{
		lo: l,
		hi: h,
	}, nil
}

// Set populates Name with another name
func (n *Name) Set(other Name) {
	n.lo = other.lo
	n.hi = other.hi
}

// FromString populates Name with a string
func (n *Name) FromString(s string) error {
	tmp, err := StringToName(s)
	if err != nil {
		return err
	}
	n.hi = tmp.hi
	n.lo = tmp.lo
	return nil
}

// ToString converts a Name to string
func (n *Name) ToString() string {
	ret := ""
	for _, n64 := range []uint64{n.hi, n.lo} {
		if n64 != 0 {
			for i := 0; i < 8; i++ {
				tmp := byte((n64 >> (byte(7-i) * 8)) & 0xff)
				if tmp == 0 {
					return ret
				}
				ret += string([]byte{tmp})
			}
		}
	}

	return ret
}
