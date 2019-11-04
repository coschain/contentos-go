package plugins

import (
	"errors"
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/prototype"
	"github.com/jinzhu/gorm"
	"time"
)

func makeIOTrx(trxHash string, blockHeight uint64, blockTime time.Time, account string, action string) *IOTrxRecord {
	return &IOTrxRecord{
		TrxHash:     trxHash,
		BlockHeight: blockHeight,
		BlockTime:  blockTime,
		Account:     account,
		Action:      account,
	}
}

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
	ioTrxFrom := makeIOTrx(trxLog.TrxId, blockLog.BlockNum, time.Unix(int64(blockLog.BlockTime), 0),
		op.GetCreator().GetValue(), opLog.Type)
	ioTrxTo := makeIOTrx(trxLog.TrxId, blockLog.BlockNum, time.Unix(int64(blockLog.BlockTime), 0),
		op.GetNewAccountName().GetValue(), opLog.Type)
	if err := db.Create(ioTrxFrom).Error; err != nil {
		return err
	}
	if err := db.Create(ioTrxTo).Error; err != nil {
		return err
	}
	return nil
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
	ioTrxFrom := makeIOTrx(trxLog.TrxId, blockLog.BlockNum, time.Unix(int64(blockLog.BlockTime), 0),
		op.GetFrom().GetValue(), opLog.Type)
	ioTrxTo := makeIOTrx(trxLog.TrxId, blockLog.BlockNum, time.Unix(int64(blockLog.BlockTime), 0),
		op.GetTo().GetValue(), opLog.Type)
	if err := db.Create(ioTrxFrom).Error; err != nil {
		return err
	}
	if err := db.Create(ioTrxTo).Error; err != nil {
		return err
	}
	return nil
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
	ioTrxFrom := makeIOTrx(trxLog.TrxId, blockLog.BlockNum, time.Unix(int64(blockLog.BlockTime), 0),
		op.GetFrom().GetValue(), opLog.Type)
	ioTrxTo := makeIOTrx(trxLog.TrxId, blockLog.BlockNum, time.Unix(int64(blockLog.BlockTime), 0),
		op.GetTo().GetValue(), opLog.Type)
	if err := db.Create(ioTrxFrom).Error; err != nil {
		return err
	}
	if err := db.Create(ioTrxTo).Error; err != nil {
		return err
	}
	return nil
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
	ioTrxFrom := makeIOTrx(trxLog.TrxId, blockLog.BlockNum, time.Unix(int64(blockLog.BlockTime), 0),
		op.GetFrom().GetValue(), opLog.Type)
	ioTrxTo := makeIOTrx(trxLog.TrxId, blockLog.BlockNum, time.Unix(int64(blockLog.BlockTime), 0),
		op.GetTo().GetValue(), opLog.Type)
	if err := db.Create(ioTrxFrom).Error; err != nil {
		return err
	}
	if err := db.Create(ioTrxTo).Error; err != nil {
		return err
	}
	return nil
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
	ioTrxFrom := makeIOTrx(trxLog.TrxId, blockLog.BlockNum, time.Unix(int64(blockLog.BlockTime), 0),
		op.GetCreditor().GetValue(), opLog.Type)
	ioTrxTo := makeIOTrx(trxLog.TrxId, blockLog.BlockNum, time.Unix(int64(blockLog.BlockTime), 0),
		op.GetDebtor().GetValue(), opLog.Type)
	if err := db.Create(ioTrxFrom).Error; err != nil {
		return err
	}
	if err := db.Create(ioTrxTo).Error; err != nil {
		return err
	}
	return nil
}
