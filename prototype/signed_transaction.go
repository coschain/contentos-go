package prototype

import (
	"crypto/sha256"
	"fmt"
	cmn "github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/crypto"
	"github.com/coschain/contentos-go/common/crypto/secp256k1"
	"github.com/coschain/contentos-go/p2p/common"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
)

func (p *SignedTransaction) ExportPubKeys(cid ChainId) ([]*PublicKeyType, error) {
	buf, err := p.GetTrxHash(cid)

	if err != nil {
		return nil, errors.New("sha256 error")
	}

	if len(p.Signatures) == 0 {
		return nil, errors.New("no signatures")
	}

	result := make([]*PublicKeyType, len(p.Signatures))

	for index, sig := range p.Signatures {
		buffer, err := secp256k1.RecoverPubkey(buf, sig.Sig)

		if err != nil {
			return nil, errors.New("recover error")
		}

		ecPubKey, err := crypto.UnmarshalPubkey(buffer)
		if err != nil {
			return nil, errors.New("recover error")
		}

		result[index] = PublicKeyFromBytes(secp256k1.CompressPubkey(ecPubKey.X, ecPubKey.Y))
	}

	return result, nil
}

func (p *SignedTransaction) Validate() error {
	if p == nil || p.Trx == nil || p.Signatures == nil {
		return ErrNpe
	}

	if err := p.Trx.Validate(); err != nil {
		return err
	}

	if len(p.Signatures) == 0 {
		return errors.New("no signatures")
	}
	for index, sig := range p.Signatures {
		if err := sig.Validate(); err != nil {
			return errors.WithMessage(err, fmt.Sprintf("Signatures error index: %d", index))
		}
	}
	return nil
}

func (p *SignedTransaction) VerifySig(pubKey *PublicKeyType, cid ChainId) bool {

	buf, err := p.GetTrxHash(cid)

	if err != nil {
		return false
	}

	for _, sig := range p.Signatures {
		//
		// sig.Sig[64] is the recovery id, which is useful only in case of public key recovery.
		// it's not needed by standard ECDSA verification.
		//
		if secp256k1.VerifySignature(pubKey.Data, buf, sig.Sig[0:64]) {
			return true
		}
	}

	return false
}

func (p *SignedTransaction) GetTrxHash(cid ChainId) ([]byte, error) {
	buf, err := proto.Marshal(p.Trx)

	if err != nil {
		return nil, err
	}

	h := sha256.New()

	cidBuf := cmn.Int2Bytes(cid.Value)
	h.Reset()
	h.Write(cidBuf)
	h.Write(buf)
	bs := h.Sum(nil)

	if bs == nil {
		return nil, errors.New("sha256 error")
	}

	return bs, nil
}

func (p *SignedTransaction) Sign(secKey *PrivateKeyType, cid ChainId) []byte {

	buf, err := p.GetTrxHash(cid)

	if err != nil {
		return nil
	}

	//
	// signatures produced by secp256k1.Sign() are immune to malleability attacks.
	//
	// secp256k1_ecdsa_sig_sign() ensures the "Low S values in signatures", which was specified in
	// https://github.com/bitcoin/bips/blob/master/bip-0062.mediawiki#low-s-values-in-signatures
	//
	res, err := secp256k1.Sign(buf, secKey.Data)

	if err != nil {
		return nil
	}

	return res
}

func (p *SignedTransaction) Id() (*Sha256, error) {
	buf, err := proto.Marshal(p.Trx)
	if err != nil {
		return nil, err
	}
	h := sha256.New()
	h.Reset()
	h.Write(buf)
	bs := h.Sum(nil)
	if bs == nil {
		return nil, errors.New("sha256 error")
	}
	id := &Sha256{Hash: bs}
	return id, nil
}

func (p *SignedTransaction) MerkleDigest() (*Sha256, error) {
	buf, err := proto.Marshal(p)
	if err != nil {
		return nil, err
	}
	h := sha256.New()
	h.Reset()
	h.Write(buf)
	bs := h.Sum(nil)
	if bs == nil {
		return nil, errors.New("sha256 error")
	}
	id := &Sha256{Hash: bs}
	return id, nil
}

func (p *SignedTransaction) Serialization(sink *common.ZeroCopySink) error {
	data, _ := proto.Marshal(p)
	sink.WriteBytes(data)
	return nil
}

func (tx *SignedTransaction) Deserialization(source *common.ZeroCopySource) error {
	tmp := &SignedTransaction{}
	err := proto.Unmarshal(source.Data(), tmp)
	if err != nil {
		return err
	}
	tx = tmp
	return nil
}
