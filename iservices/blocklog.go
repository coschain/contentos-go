package iservices

import (
	"fmt"
	"time"
)

const BlockLogServiceName = "block_log_svc"
const BlockLogTable = "blocklogs"

// table splitting size in million records, set to 0 for no-splitting.
const BlockLogSplitSizeInMillion = 5

type BlockLogRecord struct {
	ID uint64						`gorm:"primary_key;auto_increment"`
	BlockId string					`gorm:"not null;unique_index"`
	BlockHeight uint64				`gorm:"not null;index"`
	BlockTime time.Time				`gorm:"not null"`
	Final bool						`gorm:"not null;index"`
	JsonLog string					`gorm:"type:longtext"`
}

func (log *BlockLogRecord) TableName() string {
	return BlockLogTableNameForBlockHeight(log.BlockHeight)
}

func BlockLogTableNameForBlockHeight(num uint64) string {
	if BlockLogSplitSizeInMillion <= 0 {
		return BlockLogTable
	}
	c := num / (BlockLogSplitSizeInMillion * 1000000)
	return fmt.Sprintf("%s_%04dm_%04dm", BlockLogTable, c * BlockLogSplitSizeInMillion, (c + 1) * BlockLogSplitSizeInMillion)
}
