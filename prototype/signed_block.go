package prototype

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"github.com/coschain/contentos-go/common/crypto/secp256k1"

	"github.com/coschain/contentos-go/common"
	"github.com/gogo/protobuf/proto"
	"github.com/coschain/contentos-go/common/crypto"
)

const Size = 32

func (sb *SignedBlock) Previous() common.BlockID {
	var ret common.BlockID
	copy(ret.Data[:], sb.SignedHeader.Header.Previous.Hash[:32])
	return ret
}

func (sb *SignedBlock) Id() common.BlockID {
	var ret, prev common.BlockID
	copy(prev.Data[:], sb.SignedHeader.Header.Previous.Hash[:32])
	copy(ret.Data[:], sb.SignedHeader.Header.TransactionMerkleRoot.Hash[:32])
	binary.LittleEndian.PutUint64(ret.Data[:8], prev.BlockNum()+1)
	return ret
}

func (sb *SignedBlock) CalculateMerkleRoot() *common.BlockID {
	if len(sb.Transactions) == 0 {
		return &common.BlockID{}
	}

	var ids = make([]*Sha256, len(sb.Transactions))

	for i := 0; i < len(sb.Transactions); i++ {
		ids[i], _ = sb.Transactions[i].SigTrx.MerkleDigest()
	}

	currentHashes := uint32(len(ids))

	for currentHashes > 1 {
		var iMax uint32 = uint32(currentHashes - (currentHashes & 1))
		var k uint32 = 0

		for i := uint32(0); i < iMax; i += 2 {
			ids[k], _ = calculatePairHash(ids[i], ids[i+1])
			k++
		}

		if currentHashes&1 == 1 {
			ids[k] = ids[iMax]
			k++
		}
		currentHashes = k
	}
	root := &common.BlockID{}
	copy(root.Data[:], ids[0].Hash)
	return root
}

func (sb *SignedBlock) Hash() (hash [Size]byte) {
	data, _ := proto.Marshal(sb)
	hash = sha256.Sum256(data)
	return
}

func (sb *SignedBlock) GetSignee() (*PublicKeyType,error) {
	// TODO: get pub key
	hash := sb.SignedHeader.Header.Hash()
	buf,err := secp256k1.RecoverPubkey(hash[:],sb.SignedHeader.WitnessSignature.Sig)
	if err != nil {
		return nil,errors.New("RecoverPubkey error")
	}
	ecPubKey, err := crypto.UnmarshalPubkey(buf)
	if err != nil {
		return nil,errors.New("UnmarshalPubkey error")
	}
	pub := PublicKeyFromBytes(secp256k1.CompressPubkey(ecPubKey.X, ecPubKey.Y))
	return pub,nil
}

func (bh *BlockHeader) Hash() (hash [Size]byte) {
	data, _ := proto.Marshal(bh)
	hash = sha256.Sum256(data)
	return
}

func (sbh *SignedBlockHeader) ValidateSig(key *PublicKeyType) bool {
	return false
}

func (sbh *SignedBlockHeader) Sign(secKey *PrivateKeyType) error {
	hash := sbh.Header.Hash()
	res ,err := secp256k1.Sign(hash[:],secKey.Data)
	if err != nil {
		errors.New("secp256k1 sign error")
	}
	sbh.WitnessSignature.Sig = append(sbh.WitnessSignature.Sig,res...)
	return nil
}

func (sbh *SignedBlockHeader) Hash() (hash [Size]byte) {
	data, _ := proto.Marshal(sbh)
	hash = sha256.Sum256(data)
	return
}

func (sbh *SignedBlockHeader) Number() uint64 {
	var ret, prev common.BlockID
	copy(prev.Data[:], sbh.Header.Previous.Hash[:32])
	copy(ret.Data[:], sbh.WitnessSignature.Sig[:32])
	binary.LittleEndian.PutUint64(ret.Data[:8], prev.BlockNum()+1)
	return binary.LittleEndian.Uint64(ret.Data[:8])
}

func calculatePairHash(a *Sha256, b *Sha256) (*Sha256, error) {
	size := len(a.Hash) + len(b.Hash)
	buf := make([]byte, 0, size)
	buf = append(buf, a.Hash...)
	buf = append(buf, b.Hash...)

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
