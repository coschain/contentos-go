package common

type PhonySignedBlock struct {
	payload []byte
}

var cnt int

func (psb *PhonySignedBlock) Marshall() []byte {
	return psb.payload
}

func (psb *PhonySignedBlock) Unmarshall(b []byte) error {
	psb.payload = b
	return nil
}

func (psb *PhonySignedBlock) Set(data string) {
	psb.payload = []byte(data)
}

func (psb *PhonySignedBlock) Data() string {
	return string(psb.payload)
}
