package consensus

import (
	"errors"
)

var (
	ErrInvalidProducer         = errors.New("invalid producer")
	ErrInvalidBlockNum         = errors.New("invalid block number")
	ErrBlockOutOfScope         = errors.New("block number out of scope")
	ErrConsensusNotReady       = errors.New("consensus not ready")
	ErrInternal                = errors.New("internal error")
	ErrBlockNotExist           = errors.New("block doesn't exist")
	ErrDupBlock                = errors.New("duplicated block")
	ErrInvalidBlock            = errors.New("invalid block")
	ErrEmptyForkDB             = errors.New("ForkDB is empty")
	ErrForkDBChanged           = errors.New("ForkDB changed, please try again")
	ErrCommittingNonExistBlock = errors.New("committing a non-existed block")
	ErrCommittingBlockOnFork   = errors.New("committing a block on fork")
	ErrCommitted               = errors.New("committed already")
	ErrSwitchFork              = errors.New("switch fork error")

	ErrInvalidCheckPoint    = errors.New("invalid checkpoint")
	ErrCheckPointOutOfRange = errors.New("checkpoint out of range")
	ErrCheckPointExists     = errors.New("checkpoint exists")
)
