package prototype

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/common/encoding/kope"
	"github.com/itchyny/base58-go"
	"math/big"
	"strings"
)

func PublicKeyFromBytes(buffer []byte) *PublicKeyType {
	result := new(PublicKeyType)
	result.Data = buffer
	return result
}

func PublicKeyFromWIF(encoded string) (*PublicKeyType, error) {
	if encoded == "" {
		return nil, errors.New("invalid address 1")
	}

	if len(encoded) < len(constants.COIN_SYMBOL) {
		return nil, errors.New("invalid address 2")
	}

	if !strings.HasPrefix(encoded, constants.COIN_SYMBOL) {
		return nil, errors.New("invalid address 3")
	}

	buffer := ([]byte(encoded))[3:]
	decoded, err := base58.BitcoinEncoding.Decode(buffer)
	if err != nil {
		return nil, err
	}

	x, ok := new(big.Int).SetString(string(decoded), 10)
	if !ok {
		return nil, errors.New("invalid address 4")
	}

	buf := x.Bytes()
	if len(buf) <= 4 {
		return nil, errors.New("invalid address 5")
	}

	temp := sha256.Sum256(buf[:len(buf)-4])
	temps := sha256.Sum256(temp[:])

	if !bytes.Equal(temps[0:4], buf[len(buf)-4:]) {
		return nil, errors.New("invalid address 6")
	}

	return PublicKeyFromBytes(buf[:len(buf)-4]), nil
}

func (m *PublicKeyType) Equal(other *PublicKeyType) bool {
	return bytes.Equal(m.Data, other.Data)
}

func (m *PublicKeyType) ToWIF() string {
	return fmt.Sprintf("%s%s", constants.COIN_SYMBOL, m.ToBase58())
}

// ToBase58 returns base58 encoded address string
func (m *PublicKeyType) ToBase58() string {
	data := m.Data
	temp := sha256.Sum256(data)
	temps := sha256.Sum256(temp[:])
	data = append(data, temps[0:4]...)

	bi := new(big.Int).SetBytes(data).String()
	encoded, _ := base58.BitcoinEncoding.Encode([]byte(bi))
	return string(encoded)
}

func (m *PublicKeyType) MarshalJSON() ([]byte, error) {
	val := fmt.Sprintf("\"%s\"", m.ToWIF())
	return []byte(val), nil
}

func (m *PublicKeyType) UnmarshalJSON(input []byte) error {

	if len(input) < 2 {
		return errors.New("public key length error")
	}
	if input[0] != '"' {
		return errors.New("public key error")
	}
	if input[len(input)-1] != '"' {
		return errors.New("public key error")
	}

	res, err := PublicKeyFromWIF(string(input[1 : len(input)-1]))
	if err != nil {
		return err
	}
	m.Data = res.Data
	return nil
}

func (m *PublicKeyType) Validate() error {
	if m == nil {
		return ErrNpe
	}
	if len(m.Data) != 33 {
		return ErrKeyLength
	}
	return nil
}

func (m *PublicKeyType) OpeEncode() ([]byte, error) {
	return kope.Encode(m.Data)
}
