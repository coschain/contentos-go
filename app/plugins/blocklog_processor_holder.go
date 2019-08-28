package plugins

import (
	"encoding/json"
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/common"
	"github.com/jinzhu/gorm"
)

type Holder struct {
	Name string					`gorm:"primary_key"`
	IsContract bool				`gorm:"index"`
	Balance uint64				`gorm:"index"`
	Vest uint64					`gorm:"index"`
	StakeVestFromMe uint64		`gorm:"index"`
}

type HolderProcessor struct {
	tableReady bool
}

func NewHolderProcessor() *HolderProcessor {
	return &HolderProcessor{}
}

func (p *HolderProcessor) Prepare(db *gorm.DB, blockLog *blocklog.BlockLog) (err error) {
	if !p.tableReady {
		if !db.HasTable(&Holder{}) {
			if err = db.CreateTable(&Holder{}).Error; err == nil {
				p.tableReady = true
			}
		}
	}
	return
}

func (p *HolderProcessor) ProcessChange(db *gorm.DB, change *blocklog.StateChange, blockLog *blocklog.BlockLog, changeIdx, opIdx, trxIdx int) error {
	rec := new(Holder)
	switch change.What {
	case "Account.Balance":
		return db.Where(Holder{Name: change.Change.Id.(string)}).
			Assign(Holder{Balance: uint64(common.JsonNumber(change.Change.After.(json.Number))), IsContract:false}).
			FirstOrCreate(rec).Error
	case "Account.Vest":
		return db.Where(Holder{Name: change.Change.Id.(string)}).
			Assign(Holder{Vest: uint64(common.JsonNumber(change.Change.After.(json.Number))), IsContract:false}).
			FirstOrCreate(rec).Error
	case "Account.StakeVestFromMe":
		return db.Where(Holder{Name: change.Change.Id.(string)}).
			Assign(Holder{StakeVestFromMe: uint64(common.JsonNumber(change.Change.After.(json.Number))), IsContract:false}).
			FirstOrCreate(rec).Error
	case "Contract.Balance":
		return db.Where(Holder{Name: change.Change.Id.(string)}).
			Assign(Holder{Balance: uint64(common.JsonNumber(change.Change.After.(json.Number))), IsContract:true}).
			FirstOrCreate(rec).Error
	}
	return nil
}

func (p *HolderProcessor) ProcessOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	return nil
}

func (p *HolderProcessor) Finalize(db *gorm.DB, blockLog *blocklog.BlockLog) error {
	return nil
}
