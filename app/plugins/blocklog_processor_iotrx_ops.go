package plugins

import (
	"errors"
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/prototype"
	"github.com/jinzhu/gorm"
	"time"
)

func ProcessAccountCreateOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	trxLog := blockLog.Transactions[trxIdx]
	opLog := trxLog.Operations[opIdx]
	if opLog.Type != "account_create" {
		return nil
	}
	op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.AccountCreateOperation)
	if !ok {
		return errors.New("failed conversion to AccountCreateOperation")
	}
	return db.Create(&IOTrxRecord{
		TrxHash:     trxLog.TrxId,
		BlockHeight: blockLog.BlockNum,
		BlockTime:   time.Unix(int64(blockLog.BlockTime), 0),
		From:        op.GetCreator().GetValue(),
		To:          op.GetNewAccountName().GetValue(),
		Action:      opLog.Type,
	}).Error
}

func ProcessTransferOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	trxLog := blockLog.Transactions[trxIdx]
	opLog := trxLog.Operations[opIdx]
	if opLog.Type != "transfer" {
		return nil
	}
	op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.TransferOperation)
	if !ok {
		return errors.New("failed conversion to TransferOperation")
	}
	return db.Create(&IOTrxRecord{
		TrxHash: trxLog.TrxId,
		BlockHeight: blockLog.BlockNum,
		BlockTime: time.Unix(int64(blockLog.BlockTime), 0),
		From: op.GetFrom().GetValue(),
		To: op.GetTo().GetValue(),
		Action: opLog.Type,
	}).Error
}

func ProcessTransferVestOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	trxLog := blockLog.Transactions[trxIdx]
	opLog := trxLog.Operations[opIdx]
	if opLog.Type != "transfer_to_vest" {
		return nil
	}
	op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.TransferToVestOperation)
	if !ok {
		return errors.New("failed conversion to TransferToVestOperation")
	}
	return db.Create(&IOTrxRecord{
		TrxHash:     trxLog.TrxId,
		BlockHeight: blockLog.BlockNum,
		BlockTime:   time.Unix(int64(blockLog.BlockTime), 0),
		From:        op.GetFrom().GetValue(),
		To:          op.GetTo().GetValue(),
		Action:      opLog.Type,
	}).Error
}

func ProcessStakeOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	trxLog := blockLog.Transactions[trxIdx]
	opLog := trxLog.Operations[opIdx]
	if opLog.Type != "stake" {
		return nil
	}
	op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.StakeOperation)
	if !ok {
		return errors.New("failed conversion to StakeOperation")
	}
	return db.Create(&IOTrxRecord{
		TrxHash:     trxLog.TrxId,
		BlockHeight: blockLog.BlockNum,
		BlockTime:   time.Unix(int64(blockLog.BlockTime), 0),
		From:        op.GetFrom().GetValue(),
		To:          op.GetTo().GetValue(),
		Action:      opLog.Type,
	}).Error
}

func ProcessUnStakeOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	trxLog := blockLog.Transactions[trxIdx]
	opLog := trxLog.Operations[opIdx]
	if opLog.Type != "un_stake" {
		return nil
	}
	op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.UnStakeOperation)
	if !ok {
		return errors.New("failed conversion to UnStakeOperation")
	}
	return db.Create(&IOTrxRecord{
		TrxHash:     trxLog.TrxId,
		BlockHeight: blockLog.BlockNum,
		BlockTime:   time.Unix(int64(blockLog.BlockTime), 0),
		From:        op.GetCreditor().GetValue(),
		To:          op.GetDebtor().GetValue(),
		Action:      opLog.Type,
	}).Error
}
