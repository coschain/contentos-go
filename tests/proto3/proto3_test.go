package proto3

import (
	"crypto/sha256"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestProto3(t *testing.T) {
	a := assert.New(t)
	var (
		err error
		block1 *BlockV1
		block2 *BlockV2
		blockData1, blockData2 []byte
	)
	block1, block2 = produceBlocks()
	a.NotNil(block1)
	a.NotNil(block2)
	blockData1, err = proto.Marshal(block1)
	a.NoError(err)
	blockData2, err = proto.Marshal(block2)
	a.NoError(err)

	t.Run("v1_parse_v1", makeUnmarshalTest(blockData1, new(BlockV1)))
	t.Run("v2_parse_v2", makeUnmarshalTest(blockData2, new(BlockV2)))
	t.Run("v1_parse_v2", makeUnmarshalTest(blockData2, new(BlockV1)))
	t.Run("v2_parse_v1", makeUnmarshalTest(blockData1, new(BlockV2)))
}

func makeUnmarshalTest(data []byte, pb proto.Message) func(*testing.T) {
	return func(t *testing.T) {
		a := assert.New(t)
		dataHash := sha256.Sum256(data)
		a.NoError(proto.Unmarshal(data, pb))
		buf, err := proto.Marshal(pb)
		a.NoError(err)
		a.Equal(dataHash, sha256.Sum256(buf))
	}
}

func produceBlocks() (*BlockV1, *BlockV2) {
	block1 := &BlockV1{
		Header: &BlockHeaderV1{
			Magic: 0xCAFEBABE,
		},
		Records: []*BlockRecordV1{
			{
				Record: &BlockRecordV1_Person{
					Person: &PersonRecordV1{
						Name: "Alice",
						Gender: false,
						Age: 18,
					},
				},
			},
			{
				Record: &BlockRecordV1_Book{
					Book: &BookRecordV1{
						Isdn: "123-45678",
						Title: "Effective Go",
						Author: "John Smith",
					},
				},
			},
		},
	}
	block2 := &BlockV2{
		Header: &BlockHeaderV2{
			Magic: block1.Header.Magic,
			Timestamp: 12345678,
		},
		Records: []*BlockRecordV2{
			{
				Record: &BlockRecordV2_Person{
					Person: &PersonRecordV2{
						Name: "Alice",
						Gender: false,
						Age: 18,
						Address: "NewYork",
					},
				},
			},
			{
				Record: &BlockRecordV2_Book{
					Book: &BookRecordV2{
						Isdn: "123-45678",
						Title: "Effective Go",
						Author: "John Smith",
					},
				},
			},
			{
				Record: &BlockRecordV2_Car{
					Car: &CarRecordV2{
						Brand: "Porsche",
						Color: "Yellow",
						Power: 420,
					},
				},
			},
		},
		Signature: []byte(strings.Repeat("A", 8)),
	}
	return block1, block2
}
