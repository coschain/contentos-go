package prototype

import (
	"bytes"
	"crypto/sha256"
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

// fixme: ToBase58()/PublicKeyFromWIF() work well for now, but they are vulnerable.
//
// This pair of functions can't work with key data prefixed by 0x00 bytes, but fortunately
// ecc compressed public keys are always prefixed by 0x02 or 0x03.
// Review this code if we switched to a key scheme other than ecc in future.
// The same problem has already been fixed in private_key_type.go by git commit 1f7fe10.
func PublicKeyFromWIF(encoded string) (*PublicKeyType, error) {
	if encoded == "" {
		return nil, ErrKeyLength
	}

	if len(encoded) < len(constants.CoinSymbol) {
		return nil, ErrPubKeyFormatErr
	}

	if !strings.HasPrefix(encoded, constants.CoinSymbol) {
		return nil, ErrPubKeyFormatErr
	}

	buffer := ([]byte(encoded))[3:]
	decoded, err := base58.BitcoinEncoding.Decode(buffer)
	if err != nil {
		return nil, err
	}

	x, ok := new(big.Int).SetString(string(decoded), 10)
	if !ok {
		return nil, ErrPubKeyFormatErr
	}

	buf := x.Bytes()
	if len(buf) <= 4 {
		return nil, ErrPubKeyFormatErr
	}

	temp := sha256.Sum256(buf[:len(buf)-4])
	temps := sha256.Sum256(temp[:])

	if !bytes.Equal(temps[0:4], buf[len(buf)-4:]) {
		return nil, ErrPubKeyFormatErr
	}

	return PublicKeyFromBytes(buf[:len(buf)-4]), nil
}

func (m *PublicKeyType) Equal(other *PublicKeyType) bool {
	return bytes.Equal(m.Data, other.Data)
}

func (m *PublicKeyType) ToWIF() string {
	return fmt.Sprintf("%s%s", constants.CoinSymbol, m.ToBase58())
}

// ToBase58 returns base58 encoded address string
// fixme: ToBase58()/PublicKeyFromWIF() work well for now, but they are vulnerable.
//
// This pair of functions can't work with key data prefixed by 0x00 bytes, but fortunately
// ecc compressed public keys are always prefixed by 0x02 or 0x03.
// Review this code if we switched to a key scheme other than ecc in future.
// The same problem has already been fixed in private_key_type.go by git commit 1f7fe10.
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
		return ErrPubKeyFormatErr
	}
	if input[0] != '"' {
		return ErrPubKeyFormatErr
	}
	if input[len(input)-1] != '"' {
		return ErrPubKeyFormatErr
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
