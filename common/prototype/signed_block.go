package prototype

import (
	"crypto/sha256"
	"encoding/binary"
	"github.com/coschain/contentos-go/common"
	"github.com/gogo/protobuf/proto"
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

func (sb *SignedBlock) Hash() (hash [Size]byte) {
	data, _ := proto.Marshal(sb)
	hash = sha256.Sum256(data)
	return
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