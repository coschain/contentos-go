package types

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/coschain/contentos-go/p2p/common"
	comm "github.com/coschain/contentos-go/p2p/depend/common"
	"github.com/coschain/contentos-go/p2p/depend/common/config"
	"github.com/coschain/contentos-go/p2p/msg"
)

type Message interface {
	Serialization(sink *comm.ZeroCopySink) error
	Deserialization(source *comm.ZeroCopySource) error
	CMDType() string
}

//MsgPayload in link channel
type MsgPayload struct {
	Id          uint64  //peer ID
	Addr        string  //link address
	PayloadSize uint32  //payload size
	Payload     Message //msg payload
}

type messageHeader struct {
	Magic    uint32
	CMD      [common.MSG_CMD_LEN]byte // The message type
	Length   uint32
	Checksum [common.CHECKSUM_LEN]byte
}

func readMessageHeader(reader io.Reader) (messageHeader, error) {
	msgh := messageHeader{}
	err := binary.Read(reader, binary.LittleEndian, &msgh)
	return msgh, err
}

func writeMessageHeaderInto(sink *comm.ZeroCopySink, msgh messageHeader) {
	sink.WriteUint32(msgh.Magic)
	sink.WriteBytes(msgh.CMD[:])
	sink.WriteUint32(msgh.Length)
	sink.WriteBytes(msgh.Checksum[:])
}

func writeMessageHeader(writer io.Writer, msgh messageHeader) error {
	return binary.Write(writer, binary.LittleEndian, msgh)
}

func newMessageHeader(cmd string, length uint32, checksum [common.CHECKSUM_LEN]byte) messageHeader {
	msgh := messageHeader{}
	msgh.Magic = config.DefConfig.P2PNode.NetworkMagic
	copy(msgh.CMD[:], cmd)
	msgh.Checksum = checksum
	msgh.Length = length
	return msgh
}

func WriteMessage(sink *comm.ZeroCopySink, msg Message) error {
	pstart := sink.Size()
	sink.NextBytes(common.MSG_HDR_LEN) // can not save the buf, since it may reallocate in sink
	err := msg.Serialization(sink)
	if err != nil {
		return err
	}
	pend := sink.Size()
	total := pend - pstart
	payLen := total - common.MSG_HDR_LEN

	sink.BackUp(total)
	buf := sink.NextBytes(total)
	checksum := common.Checksum(buf[common.MSG_HDR_LEN:])
	hdr := newMessageHeader(msg.CMDType(), uint32(payLen), checksum)

	sink.BackUp(total)
	writeMessageHeaderInto(sink, hdr)
	sink.NextBytes(payLen)

	return err
}

func ReadMessage(reader io.Reader) (Message, uint32, error) {
	hdr, err := readMessageHeader(reader)
	if err != nil {
		return nil, 0, err
	}

	magic := config.DefConfig.P2PNode.NetworkMagic
	if hdr.Magic != magic {
		return nil, 0, fmt.Errorf("unmatched magic number %d, expected %d", hdr.Magic, magic)
	}

	if hdr.Length > common.MAX_PAYLOAD_LEN {
		return nil, 0, fmt.Errorf("msg payload length:%d exceed max payload size: %d",
			hdr.Length, common.MAX_PAYLOAD_LEN)
	}

	buf := make([]byte, hdr.Length)
	_, err = io.ReadFull(reader, buf)
	if err != nil {
		return nil, 0, err
	}

	checksum := common.Checksum(buf)
	if checksum != hdr.Checksum {
		return nil, 0, fmt.Errorf("message checksum mismatch: %x != %x ", hdr.Checksum, checksum)
	}

	msgdata := &msg.TransferMsg{}

	// the buf is referenced by msg to avoid reallocation, so can not reused
	source := comm.NewZeroCopySource(buf)
	err = msgdata.Deserialization(source)
	if err != nil {
		return nil, 0, err
	}

	return msgdata, hdr.Length, nil
}
