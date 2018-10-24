package common

// Marshaller ...
type Marshaller interface {
	Marshall() []byte
	Unmarshall([]byte) error
}

// BlockID is a sha256 byte array, the first 2 byte is
// replaced by the block number
type BlockID struct {
	data [32]byte
}

// BlockHeader ...
type BlockHeader interface {
	Previous() BlockID
}

// SignedBlockHeader ...
type SignedBlockHeader interface {
	Id() BlockID
}

// SignedBlock ...
type SignedBlock interface {
	BlockHeader
	SignedBlockHeader
	Marshaller
}
