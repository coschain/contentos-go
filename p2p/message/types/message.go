package types

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/coschain/contentos-go/p2p/common"
)

type Message interface {
	Serialization(sink *common.ZeroCopySink) error
	Deserialization(source *common.ZeroCopySource) error
	CmdType() string
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

func writeMessageHeaderInto(sink *common.ZeroCopySink, msgh messageHeader) {
	sink.WriteUint32(msgh.Magic)
	sink.WriteBytes(msgh.CMD[:])
	sink.WriteUint32(msgh.Length)
	sink.WriteBytes(msgh.Checksum[:])
}

func writeMessageHeader(writer io.Writer, msgh messageHeader) error {
	return binary.Write(writer, binary.LittleEndian, msgh)
}

func newMessageHeader(cmd string, length uint32, checksum [common.CHECKSUM_LEN]byte, magic uint32) messageHeader {
	msgh := messageHeader{}
	msgh.Magic = magic
	copy(msgh.CMD[:], cmd)
	msgh.Checksum = checksum
	msgh.Length = length
	return msgh
}

func WriteMessage(sink *common.ZeroCopySink, msg Message, magic uint32) error {
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
	hdr := newMessageHeader(msg.CmdType(), uint32(payLen), checksum, magic)

	sink.BackUp(total)
	writeMessageHeaderInto(sink, hdr)
	sink.NextBytes(payLen)

	return err
}

func ReadMessage(reader io.Reader, magic uint32) (Message, uint32, error) {
	hdr, err := readMessageHeader(reader)
	if err != nil {
		//return nil, 0, err
		return nil, 0, fmt.Errorf("read msg header error, %s", err)
	}

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
		//return nil, 0, err
		return nil, 0, fmt.Errorf("io.ReadFull error, %s", err)
	}

	checksum := common.Checksum(buf)
	if checksum != hdr.Checksum {
		return nil, 0, fmt.Errorf("message checksum mismatch: %x != %x ", hdr.Checksum, checksum)
	}

	cmdType := string(bytes.TrimRight(hdr.CMD[:], string(0)))
	msg, err := MakeEmptyMessage(cmdType)
	if err != nil {
		//return nil, 0, err
		return nil, 0, fmt.Errorf("make empty message error, %s", err)
	}

	// the buf is referenced by msg to avoid reallocation, so can not reused
	source := common.NewZeroCopySource(buf)
	err = msg.Deserialization(source)
	if err != nil {
		//return nil, 0, err
		return nil, 0, fmt.Errorf("p2p msg Deserialization error, %s", err)
	}

	return msg, hdr.Length, nil
}

func MakeEmptyMessage(cmdType string) (Message, error) {
	switch cmdType {
	case common.PING_TYPE:
		return &TransferMsg{}, nil
	case common.VERSION_TYPE:
		return &TransferMsg{}, nil
	case common.VERACK_TYPE:
		return &TransferMsg{}, nil
	case common.ADDR_TYPE:
		return &TransferMsg{}, nil
	case common.GetADDR_TYPE:
		return &TransferMsg{}, nil
	case common.PONG_TYPE:
		return &TransferMsg{}, nil
	case common.ID_TYPE:
		return &TransferMsg{}, nil
	case common.REQ_ID_TYPE:
		return &TransferMsg{}, nil
	case common.BLOCK_TYPE:
		return &TransferMsg{}, nil
	case common.TX_TYPE:
		return &TransferMsg{}, nil
	case common.DISCONNECT_TYPE:
		return &TransferMsg{}, nil
	case common.CHECKPOINT_TYPE:
		return &TransferMsg{}, nil
	case common.REQUEST_OUT_OF_RANGE_IDS_TYPE:
		return &TransferMsg{}, nil
	case common.REQUEST_BLOCK_BATCH_TYPE:
		return &TransferMsg{}, nil
	case common.DETECT_FORMER_IDS_TYPE:
		return &TransferMsg{}, nil
	case common.CLEAR_OUT_OF_RABGE_STATE:
		return &TransferMsg{}, nil
	case common.CONSENSUS_TYPE:
		return &ConsMsg{Extra:&ConsensusExtraData{},}, nil
	default:
		return nil, errors.New("unsupported cmd type:" + cmdType)
	}
}
