package prototype

import (
	"crypto/sha256"
	"errors"
	cmn "github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/p2p/depend/common"
	"github.com/coschain/contentos-go/common/crypto"
	"github.com/coschain/contentos-go/common/crypto/secp256k1"
	"github.com/gogo/protobuf/proto"
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
	for _,sig := range p.Signatures{
		if err := sig.Validate(); err != nil{
			return err
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

func (p *SignedTransaction) VerifyAuthority(cid ChainId,max_recursion_depth uint32,posting AuthorityGetter,active AuthorityGetter,owner AuthorityGetter) {
	pubs,err := p.ExportPubKeys(cid)
	if err != nil {
		panic(err)
	}
	verifyAuthority(p.Trx.Operations,pubs,max_recursion_depth,posting,active,owner)
}

func verifyAuthority(ops []*Operation, trxPubs []*PublicKeyType, max_recursion_depth uint32,posting AuthorityGetter,active AuthorityGetter,owner AuthorityGetter) {
	required_active := map[string]bool{}
	required_posting := map[string]bool{}
	required_owner := map[string]bool{}
	other := []Authority{}

	for _,op := range ops {
		baseOp := getBaseOp(op)

		baseOp.GetAuthorities(&other)
		baseOp.GetRequiredPosting(&required_posting)
		baseOp.GetRequiredActive(&required_active)
		baseOp.GetRequiredOwner(&required_owner)
	}

	if len(required_posting) > 0 {
		if len(required_active) > 0 || len(required_owner) > 0 || len(other) > 0 {
			panic("can not combinme posing authority with others")
		}
		s := SignState{}
		s.Init(trxPubs,max_recursion_depth,posting,active,owner)
		for k,_ := range required_posting {
			if !s.CheckAuthorityByName(k,0,Posting) &&
				!s.CheckAuthorityByName(k,0,Active) &&
				!s.CheckAuthorityByName(k,0,Owner) {
				panic("check posting authority failed")
			}
		}
		return
	}

	s := SignState{}
	s.Init(trxPubs,max_recursion_depth,posting,active,owner)
	for _,auth := range other {
		if !s.CheckAuthority(&auth,0,Active) {
			panic("missing authority")
		}
	}

	for k,_ := range required_active {
		if !s.CheckAuthorityByName(k,0,Active) &&
			!s.CheckAuthorityByName(k,0,Owner) {
			panic("check active authority failed")
		}
	}

	for k,_ := range required_owner {
		if !s.CheckAuthorityByName(k,0,Owner) {
			panic("check active authority failed")
		}
	}
}

func getBaseOp(op *Operation) BaseOperation {
	switch t := op.Op.(type) {
	case *Operation_Op1:
		return BaseOperation(t.Op1)
	case *Operation_Op2:
		return BaseOperation(t.Op2)
	default:
		return nil
	}
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