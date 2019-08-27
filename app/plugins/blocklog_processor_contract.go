package plugins

import (
	"encoding/json"
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/common"
	"github.com/jinzhu/gorm"
)

type Contract struct {
	Name string					`gorm:"primary_key"`
	Balance uint64				`gorm:"index"`
}

type ContractProcessor struct {
	tableReady bool
}

func NewContractProcessor() *ContractProcessor {
	return &ContractProcessor{}
}

func (p *ContractProcessor) Prepare(db *gorm.DB, blockLog *blocklog.BlockLog) (err error) {
	if !p.tableReady {
		if !db.HasTable(&Contract{}) {
			if err = db.CreateTable(&Contract{}).Error; err == nil {
				p.tableReady = true
			}
		}
	}
	return
}

func (p *ContractProcessor) ProcessChange(db *gorm.DB, change *blocklog.StateChange, blockLog *blocklog.BlockLog, changeIdx, opIdx, trxIdx int) error {
	rec := new(Contract)
	switch change.What {
	case "Contract.Balance":
		return db.Where(Contract{Name: change.Change.Id.(string)}).
			Assign(Contract{Balance: uint64(common.JsonNumber(change.Change.After.(json.Number)))}).
			FirstOrCreate(rec).Error
	}
	return nil
}

func (p *ContractProcessor) ProcessOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	return nil
}

func (p *ContractProcessor) Finalize(db *gorm.DB, blockLog *blocklog.BlockLog) error {
	return nil
}
