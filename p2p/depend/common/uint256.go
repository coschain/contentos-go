package common

import (
	"errors"
	"fmt"
	"io"
)

const UINT256_SIZE = 32

type Uint256 [UINT256_SIZE]byte

func (u *Uint256) ToHexString() string {
	return fmt.Sprintf("%x", ToArrayReverse(u[:]))
}

func (u *Uint256) Serialize(w io.Writer) error {
	_, err := w.Write(u[:])
	return err
}

func (u *Uint256) Deserialize(r io.Reader) error {
	_, err := io.ReadFull(r, u[:])
	if err != nil {
		return errors.New("deserialize Uint256 error")
	}
	return nil
}
