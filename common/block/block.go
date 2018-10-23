package block

import (
	"contentos-go/common/marshall"
)

type SignedBlock interface {
	marshall.Marshaller
}

type PhonySignedBlock struct {
}

func (psb *PhonySignedBlock) Marshall() []byte {
	return []byte("hello")
}

func (psb *PhonySignedBlock) Unmarshall() error {
	return nil
}
