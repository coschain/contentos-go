package common

// Marshaller ...
type Marshaller interface {
	Marshall() []byte
	Unmarshall([]byte) error
}

// BlockID ...
type BlockID struct {
	data [32]byte
}

// SignedBlockHeader ...
type SignedBlockHeader interface {
	Id() BlockID
}

// SignedBlock ...
type SignedBlock interface {
	SignedBlockHeader
	Marshaller
}
