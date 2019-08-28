package plugins

import (
	"errors"
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/prototype"
	"github.com/jinzhu/gorm"
	"time"
)

type Stake struct {
	StakeFrom string			`gorm:"primary_key"`
	StakeTo string				`gorm:"primary_key"`
	BlockHeight uint64
	BlockTime time.Time
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
		} else {
			p.tableReady = true
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
	if opLog.Type == "stake" {
		op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.StakeOperation)
		if !ok {
			return errors.New("failed conversion to StakeOperation")
		}
		return p.processStake(db, op.From.GetValue(), op.To.GetValue(), op.Amount.GetValue(), true, blockLog)
	} else if opLog.Type == "un_stake" {
		op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.UnStakeOperation)
		if !ok {
			return errors.New("failed conversion to UnStakeOperation")
		}
		return p.processStake(db, op.Creditor.GetValue(), op.Debtor.GetValue(), op.Amount.GetValue(), false, blockLog)
	} else {
		return nil
	}
}

func (p *StakeProcessor) Finalize(db *gorm.DB, blockLog *blocklog.BlockLog) error {
	return nil
}

func (p *StakeProcessor) processStake(db *gorm.DB, from, to string, amount uint64, stakeOrUnstake bool, blockLog *blocklog.BlockLog) (err error) {
	stakeRec := new(Stake)
	update := true
	if db.Where(Stake{ StakeFrom:from, StakeTo:to }).First(stakeRec).RecordNotFound() {
		stakeRec.StakeFrom, stakeRec.StakeTo, stakeRec.Amount = from, to, 0
		update = false
	}
	if stakeOrUnstake {
		stakeRec.Amount += amount
	} else {
		stakeRec.Amount -= amount
	}
	stakeRec.BlockHeight = blockLog.BlockNum
	stakeRec.BlockTime = time.Unix(int64(blockLog.BlockTime), 0)
	if update {
		err = db.Save(stakeRec).Error
	} else {
		err = db.Create(stakeRec).Error
	}
	return
}
