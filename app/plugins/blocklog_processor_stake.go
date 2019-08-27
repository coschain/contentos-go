package plugins

import (
	"errors"
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/prototype"
	"github.com/jinzhu/gorm"
	"time"
)

type Stake struct {
	ID uint64					`gorm:"primary_key;auto_increment"`
	BlockHeight uint64			`gorm:"index"`
	BlockTime time.Time
	StakeFrom string			`gorm:"index"`
	StakeTo string				`gorm:"index"`
	Amount uint64				`gorm:"index"`
}

type StakeProcessor struct {
	tableReady bool
}

func NewStakeProcessor() *StakeProcessor {
	return &StakeProcessor{}
}

func (p *StakeProcessor) Prepare(db *gorm.DB, blockLog *blocklog.BlockLog) (err error) {
	if !p.tableReady {
		if !db.HasTable(&Stake{}) {
			if err = db.CreateTable(&Stake{}).Error; err == nil {
				p.tableReady = true
			}
		}
	}
	return
}

func (p *StakeProcessor) ProcessChange(db *gorm.DB, change *blocklog.StateChange, blockLog *blocklog.BlockLog, changeIdx, opIdx, trxIdx int) error {
	return nil
}

func (p *StakeProcessor) ProcessOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	trxLog := blockLog.Transactions[trxIdx]
	if trxLog.Receipt.Status != prototype.StatusSuccess {
		return nil
	}
	opLog := trxLog.Operations[opIdx]
	if opLog.Type != "stake" {
		return nil
	}
	op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.StakeOperation)
	if !ok {
		return errors.New("failed conversion to StakeOperation")
	}
	return db.Create(&Stake{
		BlockHeight: blockLog.BlockNum,
		BlockTime: time.Unix(int64(blockLog.BlockTime), 0),
		StakeFrom: op.GetFrom().GetValue(),
		StakeTo: op.GetTo().GetValue(),
		Amount: op.GetAmount().GetValue(),
	}).Error
}

func (p *StakeProcessor) Finalize(db *gorm.DB, blockLog *blocklog.BlockLog) error {
	return nil
}
