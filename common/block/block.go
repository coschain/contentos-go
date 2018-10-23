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
	cnt++
	return []byte("hello" + strconv.Itoa(cnt))
}

func (psb *PhonySignedBlock) Unmarshall(b []byte) error {
	psb.payload = b
	return nil
}
