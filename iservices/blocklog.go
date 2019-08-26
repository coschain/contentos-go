package iservices

import (
	"time"
)

const BlockLogServiceName = "block_log_svc"
const BlockLogDBTableName = "blocklogs"

type BlockLogRecord struct {
	ID uint64						`gorm:"primary_key;auto_increment"`
	BlockId string					`gorm:"not null;unique_index"`
	BlockHeight uint64				`gorm:"not null;index"`
	BlockTime time.Time				`gorm:"not null"`
	Final bool						`gorm:"not null;index"`
	JsonLog string					`gorm:"type:longtext"`
}

func (BlockLogRecord) TableName() string {
	return BlockLogDBTableName
}
