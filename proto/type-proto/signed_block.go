package prototype

import (
	"encoding/binary"

	"github.com/coschain/contentos-go/common"
)

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
