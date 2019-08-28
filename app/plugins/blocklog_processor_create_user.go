package plugins

import (
	"errors"
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/prototype"
	"github.com/jinzhu/gorm"
	"time"
)

type CreateUserRecord struct {
	ID uint64					`gorm:"primary_key;auto_increment"`
	BlockHeight uint64			`gorm:"index"`
	BlockTime time.Time
	Creator string				`gorm:"index"`
	NewAccount string			`gorm:"index"`
	Fee uint64
}

type CreateUserProcessor struct {
	tableReady bool
}

func NewCreateUserProcessor() *CreateUserProcessor {
	return &CreateUserProcessor{}
}

func (p *CreateUserProcessor) Prepare(db *gorm.DB, blockLog *blocklog.BlockLog) (err error) {
	if !p.tableReady {
		if !db.HasTable(&CreateUserRecord{}) {
			if err = db.CreateTable(&CreateUserRecord{}).Error; err == nil {
				p.tableReady = true
			}
		} else {
			p.tableReady = true
		}
	}
	return
}

func (p *CreateUserProcessor) ProcessChange(db *gorm.DB, change *blocklog.StateChange, blockLog *blocklog.BlockLog, changeIdx, opIdx, trxIdx int) error {
	return nil
}

func (p *CreateUserProcessor) ProcessOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	trxLog := blockLog.Transactions[trxIdx]

	if trxLog.Receipt.Status != prototype.StatusSuccess {
		return nil
	}
	opLog := trxLog.Operations[opIdx]
	if opLog.Type != "account_create" {
		return nil
	}

	op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.AccountCreateOperation)
	if !ok {
		return errors.New("failed conversion to AccountCreateOperation")
	}
	return db.Create(&CreateUserRecord{
		BlockHeight: blockLog.BlockNum,
		BlockTime: time.Unix(int64(blockLog.BlockTime), 0),
		Creator: op.GetCreator().GetValue(),
		NewAccount: op.GetNewAccountName().GetValue(),
		Fee: op.GetFee().GetValue(),
	}).Error
}

func (p *CreateUserProcessor) Finalize(db *gorm.DB, blockLog *blocklog.BlockLog) error {
	return nil
}

func init() {
	RegisterSQLTableNamePattern("create_user_records")
}
