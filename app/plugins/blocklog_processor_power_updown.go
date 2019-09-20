package plugins

import (
	"fmt"
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/prototype"
	"github.com/jinzhu/gorm"
	"time"
)

type PowerUpDownRecord struct {
	ID uint64					`gorm:"primary_key;auto_increment"`
	Account string				`gorm:"index"`
	From string					`gorm:"index"`
	PowerUp bool 				`gorm:"index"`
	Amount uint64				`gorm:"index"`
	BlockHeight uint64			`gorm:"index"`
	BlockTime time.Time
	OperationId string			`gorm:"index"`
	Memo string
}

type PowerUpDownProcessor struct {
	tableReady bool
}

func NewPowerUpDownProcessor() *PowerUpDownProcessor {
	return &PowerUpDownProcessor{}
}

func (p *PowerUpDownProcessor) Prepare(db *gorm.DB, blockLog *blocklog.BlockLog) (err error) {
	if !p.tableReady {
		if !db.HasTable(&PowerUpDownRecord{}) {
			if err = db.CreateTable(&PowerUpDownRecord{}).Error; err == nil {
				p.tableReady = true
			}
		} else {
			p.tableReady = true
		}
	}
	return
}

func (p *PowerUpDownProcessor) ProcessChange(db *gorm.DB, change *blocklog.StateChange, blockLog *blocklog.BlockLog, changeIdx, opIdx, trxIdx int) error {
	return nil
}

func (p *PowerUpDownProcessor) ProcessOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	trxLog := blockLog.Transactions[trxIdx]

	if trxLog.Receipt.Status != prototype.StatusSuccess {
		return nil
	}
	opLog := trxLog.Operations[opIdx]

	if opLog.Type == "transfer_to_vest" {
		if op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.TransferToVestOperation); ok {
			r := &PowerUpDownRecord{
				Account:	 op.GetTo().GetValue(),
				From:        op.GetFrom().GetValue(),
				PowerUp:     true,
				Amount:      op.GetAmount().GetValue(),
				BlockHeight: blockLog.BlockNum,
				BlockTime:   time.Unix(int64(blockLog.BlockTime), 0),
				OperationId: fmt.Sprintf("%s_%d", trxLog.TrxId, opIdx),
				Memo:        op.GetMemo(),
			}
			return db.Create(r).Error
		}
	} else if opLog.Type == "convert_vest" {
		if op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.ConvertVestOperation); ok {
			r := &PowerUpDownRecord{
				Account:	 op.GetFrom().GetValue(),
				From:        op.GetFrom().GetValue(),
				PowerUp:     false,
				Amount:      op.GetAmount().GetValue(),
				BlockHeight: blockLog.BlockNum,
				BlockTime:   time.Unix(int64(blockLog.BlockTime), 0),
				OperationId: fmt.Sprintf("%s_%d", trxLog.TrxId, opIdx),
				Memo:        "",
			}
			return db.Create(r).Error
		}
	}
	return nil
}

func (p *PowerUpDownProcessor) Finalize(db *gorm.DB, blockLog *blocklog.BlockLog) error {
	return nil
}


func init() {
	RegisterSQLTableNamePattern("power_up_down_records")
}
