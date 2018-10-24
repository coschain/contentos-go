package block

import (
	"contentos-go/common/marshall"
)

// SignedBlock ...
type SignedBlock interface {
	marshall.Marshaller
}
