package program

import (
	//"errors"
	"fmt"
	"github.com/ontio/ontology-crypto/keypair"
	"github.com/coschain/contentos-go/p2p/depend/common"
	//"github.com/coschain/contentos-go/p2p/depend/common/constants"
	//"io"
	//"math"
	//"math/big"
)

type ProgramBuilder struct {
	sink *common.ZeroCopySink
}

func (self *ProgramBuilder) PushPubKey(pubkey keypair.PublicKey) *ProgramBuilder {
	//buf := keypair.SerializePublicKey(pubkey)
	//return self.PushBytes(buf)
	return &ProgramBuilder{}
}

func (self *ProgramBuilder) Finish() []byte {
	return self.sink.Bytes()
}

func NewProgramBuilder() ProgramBuilder {
	return ProgramBuilder{sink: common.NewZeroCopySink(nil)}
}

func ProgramFromPubKey(pubkey keypair.PublicKey) []byte {
	sink := common.ZeroCopySink{}
	//EncodeSinglePubKeyProgramInto(&sink, pubkey)
	return sink.Bytes()
}

func ProgramFromMultiPubKey(pubkeys []keypair.PublicKey, m int) ([]byte, error) {
	sink := common.ZeroCopySink{}
	//err := EncodeMultiPubKeyProgramInto(&sink, pubkeys, m)
	//return sink.Bytes(), err
	return sink.Bytes(), nil
}

type ProgramInfo struct {
	PubKeys []keypair.PublicKey
	M       uint16
}

type programParser struct {
	source *common.ZeroCopySource
}

func newProgramParser(prog []byte) *programParser {
	return &programParser{source: common.NewZeroCopySource(prog)}
}

func (self *programParser) ExpectEOF() error {
	if self.source.Len() != 0 {
		return fmt.Errorf("expected eof, but remains %d bytes", self.source.Len())
	}
	return nil
}

func (self *programParser) IsEOF() bool {
	return self.source.Len() == 0
}


