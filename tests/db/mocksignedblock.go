package db

import (
	"github.com/coschain/contentos-go/common"
)

type MockSignedBlock struct {
	payload []byte
}

func (msb *MockSignedBlock) Marshall() []byte {
	return msb.payload
}

func (msb *MockSignedBlock) Unmarshall(b []byte) error {
	msb.payload = b
	return nil
}

func (msb *MockSignedBlock) Set(data string) {
	msb.payload = []byte(data)
}

func (msb *MockSignedBlock) Data() string {
	return string(msb.payload)
}

func (msb *MockSignedBlock) Id() common.BlockID {
	return common.BlockID{}
}

func (msb *MockSignedBlock) Previous() common.BlockID {
	return common.BlockID{}
}
