package plugins

import (
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/prototype"
	"github.com/jinzhu/gorm"
	"time"
)

type TransferRecord struct {
	ID uint64					`gorm:"primary_key;auto_increment"`
	BlockHeight uint64			`gorm:"index"`
	OperationId string			`gorm:"not null;unique_index"`
	BlockTime time.Time
	From string					`gorm:"index"`
	To string					`gorm:"index"`
	Amount uint64				`gorm:"index"`
	Memo string					`gorm:"type:varchar(756);index"`
}

type TransferProcessor struct {
	tableReady bool
}

func NewTransferProcessor() *TransferProcessor {
	return &TransferProcessor{}
}

func (p *TransferProcessor) Prepare(db *gorm.DB, blockLog *blocklog.BlockLog) (err error) {
	if !p.tableReady {
		if !db.HasTable(&TransferRecord{}) {
			if err = db.CreateTable(&TransferRecord{}).Error; err == nil {
				p.tableReady = true
			}
		} else {
			p.tableReady = true
		}
	}
	return
}

func (p *TransferProcessor) ProcessChange(db *gorm.DB, change *blocklog.StateChange, blockLog *blocklog.BlockLog, changeIdx, opIdx, trxIdx int) error {
	return nil
}

func (p *TransferProcessor) ProcessOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	trxLog := blockLog.Transactions[trxIdx]

	if trxLog.Receipt.Status != prototype.StatusSuccess {
		return nil
	}
	opLog := trxLog.Operations[opIdx]
	if opLog.Type != "transfer" {
		return nil
	}

	operationId := fmt.Sprintf("%s_%d", trxLog.TrxId, opIdx)
	op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.TransferOperation)
	if !ok {
		return errors.New("failed conversion to TransferOperation")
	}

	const MemoCharsLimit = 700
	memo := op.Memo
	memoRunes := []rune(memo)
	if len(memoRunes) > MemoCharsLimit {
		memo = string(memoRunes[:MemoCharsLimit])
	}

	return db.Create(&TransferRecord{
		BlockHeight: blockLog.BlockNum,
		BlockTime: time.Unix(int64(blockLog.BlockTime), 0),
		From: op.GetFrom().GetValue(),
		To: op.GetTo().GetValue(),
		OperationId: operationId,
		Amount: op.GetAmount().GetValue(),
		Memo: memo,
	}).Error
}

func (p *TransferProcessor) Finalize(db *gorm.DB, blockLog *blocklog.BlockLog) error {
	return nil
}


func init() {
	RegisterSQLTableNamePattern("transfer_records")
}
