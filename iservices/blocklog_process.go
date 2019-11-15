package iservices

import (
	"time"
)

const BlockLogProcessServiceName = "block_log_proc_svc"

const ThresholdForFastConvertToSync = 1000

type Progress struct {
	ID 				uint64	`gorm:"primary_key;auto_increment"`
	Processor       string  `gorm:"index"`
	BlockHeight 	uint64
	// restrict from gorm
	FastForward     *bool
	FinishAt 		time.Time
}

