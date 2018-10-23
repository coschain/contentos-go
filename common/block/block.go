package block

import (
	"strconv"

	"contentos-go/common/marshall"
)

type SignedBlock interface {
	marshall.Marshaller
}

type PhonySignedBlock struct {
	payload []byte
}

var cnt int

func (psb *PhonySignedBlock) Marshall() []byte {
	psb.payload = []byte("hello" + strconv.Itoa(cnt))
	cnt++
	return psb.payload
}

func (psb *PhonySignedBlock) Unmarshall(b []byte) error {
	psb.payload = b
	return nil
}

func (psb *PhonySignedBlock) Data() string {
	return string(psb.payload)
}
