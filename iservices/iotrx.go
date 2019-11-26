package iservices

import (
	"fmt"
	"time"
)

const IOTrxSplitSizeInMillion = 5
const IOTrxTable = "iotrx_record"

type IOTrxRecord struct {
	ID uint64			`gorm:"primary_key;auto_increment"`
	TrxHash string      `gorm:"index"`
	BlockHeight uint64
	BlockTime time.Time
	Account string			`gorm:"index"`
	Action string       `gorm:"index"`
}

func (log *IOTrxRecord) TableName() string {
	return IOTrxTableNameForBlockHeight(log.BlockHeight)
}

func IOTrxTableNameForBlockHeight(num uint64) string {
	c := num / (IOTrxSplitSizeInMillion * 1000000)
	return fmt.Sprintf("%s_%04dm_%04dm", IOTrxTable, c * IOTrxSplitSizeInMillion, (c + 1) * IOTrxSplitSizeInMillion)
}
