package consensus

import (
	"errors"
)

var (
	ErrInvalidProducer = errors.New("invalid producer")
	ErrInvalidBlockNum = errors.New("invalid block number")
	ErrInternal = errors.New("internal error")
	ErrBlockNotExist = errors.New("block doesn't exist")
	ErrEmptyForkDB = errors.New("ForkDB is empty")
	ErrForkDBChanged = errors.New("ForkDB changed, please try again")
)
