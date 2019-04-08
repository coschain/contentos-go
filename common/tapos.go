package common

import "encoding/binary"

const (
	TaposMaxBlockCount = 0x800
)

func TaposRefBlockNum(blockNum uint64) uint32 {
	return uint32(blockNum % TaposMaxBlockCount)
}

func TaposRefBlockPrefix(blockId []byte) uint32 {
	return binary.BigEndian.Uint32(blockId[8:12])
}
