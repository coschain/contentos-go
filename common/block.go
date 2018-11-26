package common

import (
	"encoding/binary"
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
	GetSignee() (interface{}, error)
}

// SignedBlock ...
type ISignedBlock interface {
	ISignedBlockHeader
	Marshaller
}

type ITransaction interface {
	Validate() bool
}

type ISignedTransaction interface {
	ITransaction
}
