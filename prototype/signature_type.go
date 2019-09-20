package prototype

import "math/big"

var (
	sHalfN, _ = new(big.Int).SetString("7FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF5D576E7357A4501DDFE92F46681B20A0", 16)
)

func (m *SignatureType) Validate() error {
	if m == nil {
		return ErrNpe
	}
	if len(m.Sig) != 65 {
		return ErrSigLength
	}
	if sHalfN.Cmp(new(big.Int).SetBytes(m.Sig[32:64])) < 0 {
		return ErrSigInvalidS
	}
	return nil
}
