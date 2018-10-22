package marshall

type Marshaller interface {
	Marshall() []byte
	Unmarshall([]byte) error
}