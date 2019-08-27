package plugins

import (
	"encoding/json"
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/common"
	"github.com/jinzhu/gorm"
)

type Account struct {
	Name string					`gorm:"primary_key"`
	Balance uint64				`gorm:"index"`
	Vest uint64					`gorm:"index"`
	StakeVestFromMe uint64		`gorm:"index"`
}

type AccountProcessor struct {
	tableReady bool
}

func NewAccountProcessor() *AccountProcessor {
	return &AccountProcessor{}
}

func (p *AccountProcessor) Prepare(db *gorm.DB, blockLog *blocklog.BlockLog) (err error) {
	if !p.tableReady {
		if !db.HasTable(&Account{}) {
			if err = db.CreateTable(&Account{}).Error; err == nil {
				p.tableReady = true
			}
		}
	}
	return
}

func (p *AccountProcessor) ProcessChange(db *gorm.DB, change *blocklog.StateChange, blockLog *blocklog.BlockLog, changeIdx, opIdx, trxIdx int) error {
	rec := new(Account)
	switch change.What {
	case "Account.Balance":
		return db.Where(Account{Name: change.Change.Id.(string)}).
			Assign(Account{Balance: uint64(common.JsonNumber(change.Change.After.(json.Number)))}).
			FirstOrCreate(rec).Error
	case "Account.Vest":
		return db.Where(Account{Name: change.Change.Id.(string)}).
			Assign(Account{Vest: uint64(common.JsonNumber(change.Change.After.(json.Number)))}).
			FirstOrCreate(rec).Error
	case "Account.StakeVestFromMe":
		return db.Where(Account{Name: change.Change.Id.(string)}).
			Assign(Account{StakeVestFromMe: uint64(common.JsonNumber(change.Change.After.(json.Number)))}).
			FirstOrCreate(rec).Error
	}
	return nil
}

func (p *AccountProcessor) ProcessOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	return nil
}

func (p *AccountProcessor) Finalize(db *gorm.DB, blockLog *blocklog.BlockLog) error {
	return nil
}
