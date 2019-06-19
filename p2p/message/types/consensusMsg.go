package types

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"github.com/coschain/contentos-go/p2p/common"
	"github.com/coschain/gobft/message"
)

type ConsMsg struct {
	MsgData message.ConsensusMessage
	Bcast   uint32
}

//Serialize message payload
func (this *ConsMsg) Serialization(sink *common.ZeroCopySink) error {
	data := this.MsgData.Bytes()
	sink.WriteUint32(this.Bcast)
	sink.WriteBytes(data)
	return nil
}

func (this *ConsMsg) CmdType() string {
	return common.CONSENSUS_TYPE
}

//Deserialize message payload
func (this *ConsMsg) Deserialization(source *common.ZeroCopySource) error {
	totalBuf := source.Data()
	bcastBuf := totalBuf[0:4]
	consensusBuf := totalBuf[4:]
	//msgdata, err := message.DecodeConsensusMsg(source.Data())
	msgdata, err := message.DecodeConsensusMsg(consensusBuf)
	if err != nil {
		return err
	}
	this.MsgData = msgdata
	this.Bcast = binary.LittleEndian.Uint32(bcastBuf)
	return nil
}

func (this *ConsMsg) Hash() [common.HASH_SIZE]byte {
	buf := &bytes.Buffer{}
	binary.Write(buf, binary.BigEndian, this.MsgData.Bytes())
	binary.Write(buf, binary.BigEndian, this.Bcast)
	return sha256.Sum256(buf.Bytes())
}