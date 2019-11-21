package iservices

import (
	"time"
)

const BlockLogProcessServiceName = "block_log_proc_svc"

const ThresholdForFastConvertToSync = 10

const (
	FastForwardStatus = 0
	SyncForwardStatus = 1
	MiddleStatus = 2
)

type Progress struct {
	ID 				uint64	`gorm:"primary_key;auto_increment"`
	Processor       string  `gorm:"index"`
	BlockHeight 	uint64
	SyncStatus      *int
	FinishAt 		time.Time
}

type DeprecatedBlockLogProgress struct {
	ID 				uint64			`gorm:"primary_key;auto_increment"`
	BlockHeight 	uint64
	FinishAt 		time.Time
}

func (DeprecatedBlockLogProgress) TableName() string {
	return "blocklog_process"
}