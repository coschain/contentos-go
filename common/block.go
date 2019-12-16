package common

import (
	"encoding/binary"
	"github.com/coschain/contentos-go/common/constants"
)

var EmptyBlockID = BlockID{}

// Marshaller ...
type Marshaller interface {
	Marshall() ([]byte, error)
	Unmarshall([]byte) error
}

// BlockID is a sha256 byte array, the first 2 byte is
// replaced by the block number
type BlockID struct {
	Data [32]byte
}

// BlockNum returns the block num
func (bid BlockID) BlockNum() uint64 {
	return binary.LittleEndian.Uint64(bid.Data[:8])
}

// BlockHeader ...
type IBlockHeader interface {
	Previous() BlockID
	Timestamp() uint64
}

// SignedBlockHeader ...
type ISignedBlockHeader interface {
	IBlockHeader
	Id() BlockID
	Validate() bool
	GetSignee() (interface{}, error)
}

// SignedBlock ...
type ISignedBlock interface {
	ISignedBlockHeader
	Marshaller
}

type ITransaction interface {
	Validate() error
}

type ITransactionReceiptWithInfo interface {
}

type ISignedTransaction interface {
	ITransaction
}

func PackBlockApplyHash(dataChangeHash uint32) uint64 {
	hash := uint64(constants.BlockApplierVersion) << 32
	hash |= uint64(dataChangeHash)
	return hash
}

func UnpackBlockApplyHash(hash uint64) (version, dataChangeHash uint32) {
	return uint32(hash >> 32), uint32(hash & 0xffffffff)
}
