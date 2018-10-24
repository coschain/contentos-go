package common

// Marshaller ...
type Marshaller interface {
	Marshall() []byte
	Unmarshall([]byte) error
}

// SignedBlock ...
type SignedBlock interface {
	Marshaller
}
