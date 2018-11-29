package types

import (
	"io"

	comm "github.com/coschain/contentos-go/p2p/common"
	"github.com/coschain/contentos-go/p2p/depend/common"
)

type NotFound struct {
	Hash common.Uint256
}

//Serialize message payload
func (this NotFound) Serialization(sink *common.ZeroCopySink) error {
	sink.WriteHash(this.Hash)
	return nil
}

func (this NotFound) CmdType() string {
	return comm.NOT_FOUND_TYPE
}

//Deserialize message payload
func (this *NotFound) Deserialization(source *common.ZeroCopySource) error {
	var eof bool
	this.Hash, eof = source.NextHash()
	if eof {
		return io.ErrUnexpectedEOF
	}

	return nil
}
