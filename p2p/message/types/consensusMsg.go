package types

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"github.com/coschain/contentos-go/p2p/common"
	"github.com/coschain/gobft/message"
	"github.com/gogo/protobuf/proto"
)

type ConsMsg struct {
	Length  uint32
	MsgData message.ConsensusMessage
	Extra   *ConsensusExtraData
}

//Serialize message payload
func (this *ConsMsg) Serialization(sink *common.ZeroCopySink) error {
	data := this.MsgData.Bytes()
	msgLength := uint32(len(data))
	sink.WriteUint32(msgLength)
	sink.WriteBytes(data)

	extraData, err := proto.Marshal(this.Extra)
	if err != nil {
		return err
	}
	sink.WriteBytes(extraData)
	return nil
}

func (this *ConsMsg) CmdType() string {
	return common.CONSENSUS_TYPE
}

//Deserialize message payload
func (this *ConsMsg) Deserialization(source *common.ZeroCopySource) error {
	totalBuf := source.Data()
	msgLengthBuf := totalBuf[0:4]
	this.Length = binary.LittleEndian.Uint32(msgLengthBuf)

	consensusMsgLength := this.Length
	consensusBuf := totalBuf[4:4+consensusMsgLength]
	msgdata, err := message.DecodeConsensusMsg(consensusBuf)
	if err != nil {
		return err
	}
	this.MsgData = msgdata

	extraBuf := totalBuf[4+consensusMsgLength:]
	err = proto.Unmarshal(extraBuf, this.Extra)
	if err != nil {
		return err
	}
	return nil
}

func (this *ConsMsg) Hash() [common.HashSize]byte {
	buf := &bytes.Buffer{}
	binary.Write(buf, binary.BigEndian, this.MsgData.Bytes())
	binary.Write(buf, binary.BigEndian, this.Extra.Bcast)
	return sha256.Sum256(buf.Bytes())
}