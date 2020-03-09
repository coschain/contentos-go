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
	BorrowedVest uint64			`gorm:"index"`
	LentVest uint64				`gorm:"index"`
	DeliveringVest uint64		`gorm:"index"`
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
		} else {
			p.tableReady = true
		}
	}
	return
}

func (p *HolderProcessor) ProcessChange(db *gorm.DB, change *blocklog.StateChange, blockLog *blocklog.BlockLog, changeIdx, opIdx, trxIdx int) (err error) {
	if change.What != "Account.Balance" && change.What != "Account.Vest" && change.What != "Account.StakeVestFromMe" && change.What != "Contract.Balance" &&
		change.What != "Account.BorrowedVest" && change.What != "Account.LentVest" && change.What != "Account.DeliveringVest" {
		return nil
	}
	rec := new(Holder)
	update := true
	name := change.Change.Id.(string)
	value := common.JsonNumberUint64(change.Change.After.(json.Number))
	if db.Where(Holder{Name: name}).First(rec).RecordNotFound() {
		rec.Name = name
		update = false
	}
	rec.IsContract = change.What == "Contract.Balance"
	switch change.What {
	case "Account.Balance":
		rec.Balance = value
	case "Account.Vest":
		rec.Vest = value
	case "Account.BorrowedVest":
		rec.BorrowedVest = value
	case "Account.LentVest":
		rec.LentVest = value
	case "Account.DeliveringVest":
		rec.DeliveringVest = value
	case "Account.StakeVestFromMe":
		rec.StakeVestFromMe = value
	case "Contract.Balance":
		rec.Balance = value
	}
	if update {
		err = db.Save(rec).Error
	} else {
		err = db.Create(rec).Error
	}
	return
}

func (p *HolderProcessor) ProcessOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	return nil
}

func (p *HolderProcessor) Finalize(db *gorm.DB, blockLog *blocklog.BlockLog) error {
	return nil
}


func init() {
	RegisterSQLTableNamePattern("holders")
}
