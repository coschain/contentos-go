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

func ProcessAccountUpdateOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	trxLog := blockLog.Transactions[trxIdx]
	opLog := trxLog.Operations[opIdx]
	if opLog.Type != "account_update" {
		return nil
	}
	op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.AccountUpdateOperation)
	if !ok {
		return errors.New("failed conversion to AccountUpdateOperation")
	}
	return db.Create(&IOTrxRecord{
		TrxHash:     trxLog.TrxId,
		BlockHeight: blockLog.BlockNum,
		BlockTime:   time.Unix(int64(blockLog.BlockTime), 0),
		Account:        op.GetOwner().GetValue(),
		Action:      opLog.Type,
	}).Error
}

func ProcessVoteOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	trxLog := blockLog.Transactions[trxIdx]
	opLog := trxLog.Operations[opIdx]
	if opLog.Type != "vote" {
		return nil
	}
	op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.VoteOperation)
	if !ok {
		return errors.New("failed conversion to VoteOperation")
	}
	return db.Create(&IOTrxRecord{
		TrxHash:     trxLog.TrxId,
		BlockHeight: blockLog.BlockNum,
		BlockTime:   time.Unix(int64(blockLog.BlockTime), 0),
		Account:        op.GetVoter().GetValue(),
		Action:      opLog.Type,
	}).Error
}

func ProcessBpRegisterOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	trxLog := blockLog.Transactions[trxIdx]
	opLog := trxLog.Operations[opIdx]
	if opLog.Type != "bp_register" {
		return nil
	}
	op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.BpRegisterOperation)
	if !ok {
		return errors.New("failed conversion to BpRegisterOperation")
	}
	return db.Create(&IOTrxRecord{
		TrxHash:     trxLog.TrxId,
		BlockHeight: blockLog.BlockNum,
		BlockTime:   time.Unix(int64(blockLog.BlockTime), 0),
		Account:        op.GetOwner().GetValue(),
		Action:      opLog.Type,
	}).Error
}

func ProcessBpUpdateOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	trxLog := blockLog.Transactions[trxIdx]
	opLog := trxLog.Operations[opIdx]
	if opLog.Type != "bp_update" {
		return nil
	}
	op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.BpUpdateOperation)
	if !ok {
		return errors.New("failed conversion to BpUpdateOperation")
	}
	return db.Create(&IOTrxRecord{
		TrxHash:     trxLog.TrxId,
		BlockHeight: blockLog.BlockNum,
		BlockTime:   time.Unix(int64(blockLog.BlockTime), 0),
		Account:        op.GetOwner().GetValue(),
		Action:      opLog.Type,
	}).Error
}

func ProcessBpEnableOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	trxLog := blockLog.Transactions[trxIdx]
	opLog := trxLog.Operations[opIdx]
	if opLog.Type != "bp_enable" {
		return nil
	}
	op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.BpEnableOperation)
	if !ok {
		return errors.New("failed conversion to BpEnableOperation")
	}
	return db.Create(&IOTrxRecord{
		TrxHash:     trxLog.TrxId,
		BlockHeight: blockLog.BlockNum,
		BlockTime:   time.Unix(int64(blockLog.BlockTime), 0),
		Account:        op.GetOwner().GetValue(),
		Action:      opLog.Type,
	}).Error
}

func ProcessBpVoteOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	trxLog := blockLog.Transactions[trxIdx]
	opLog := trxLog.Operations[opIdx]
	if opLog.Type != "bp_vote" {
		return nil
	}
	op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.BpVoteOperation)
	if !ok {
		return errors.New("failed conversion to BpVoteOperation")
	}
	return db.Create(&IOTrxRecord{
		TrxHash:     trxLog.TrxId,
		BlockHeight: blockLog.BlockNum,
		BlockTime:   time.Unix(int64(blockLog.BlockTime), 0),
		Account:        op.GetVoter().GetValue(),
		Action:      opLog.Type,
	}).Error
}

func ProcessContractDeployOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	trxLog := blockLog.Transactions[trxIdx]
	opLog := trxLog.Operations[opIdx]
	if opLog.Type != "contract_deploy" {
		return nil
	}
	op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.ContractDeployOperation)
	if !ok {
		return errors.New("failed conversion to ContractDeployOperation")
	}
	return db.Create(&IOTrxRecord{
		TrxHash:     trxLog.TrxId,
		BlockHeight: blockLog.BlockNum,
		BlockTime:   time.Unix(int64(blockLog.BlockTime), 0),
		Account:        op.GetOwner().GetValue(),
		Action:      opLog.Type,
	}).Error
}

func ProcessContractApplyOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	trxLog := blockLog.Transactions[trxIdx]
	opLog := trxLog.Operations[opIdx]
	if opLog.Type != "contract_apply" {
		return nil
	}
	op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.ContractApplyOperation)
	if !ok {
		return errors.New("failed conversion to ContractApplyOperation")
	}
	return db.Create(&IOTrxRecord{
		TrxHash:     trxLog.TrxId,
		BlockHeight: blockLog.BlockNum,
		BlockTime:   time.Unix(int64(blockLog.BlockTime), 0),
		Account:        op.GetCaller().GetValue(),
		Action:      opLog.Type,
	}).Error
}

func ProcessPostOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	trxLog := blockLog.Transactions[trxIdx]
	opLog := trxLog.Operations[opIdx]
	if opLog.Type != "post" {
		return nil
	}
	op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.PostOperation)
	if !ok {
		return errors.New("failed conversion to PostOperation")
	}
	return db.Create(&IOTrxRecord{
		TrxHash:     trxLog.TrxId,
		BlockHeight: blockLog.BlockNum,
		BlockTime:   time.Unix(int64(blockLog.BlockTime), 0),
		Account:        op.GetOwner().GetValue(),
		Action:      opLog.Type,
	}).Error
}

func ProcessReplyOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	trxLog := blockLog.Transactions[trxIdx]
	opLog := trxLog.Operations[opIdx]
	if opLog.Type != "reply" {
		return nil
	}
	op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.ReplyOperation)
	if !ok {
		return errors.New("failed conversion to ReplyOperation")
	}
	return db.Create(&IOTrxRecord{
		TrxHash:     trxLog.TrxId,
		BlockHeight: blockLog.BlockNum,
		BlockTime:   time.Unix(int64(blockLog.BlockTime), 0),
		Account:        op.GetOwner().GetValue(),
		Action:      opLog.Type,
	}).Error
}

func ProcessConvertVestOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	trxLog := blockLog.Transactions[trxIdx]
	opLog := trxLog.Operations[opIdx]
	if opLog.Type != "convert_vest" {
		return nil
	}
	op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.ConvertVestOperation)
	if !ok {
		return errors.New("failed conversion to ConvertVestOperation")
	}
	return db.Create(&IOTrxRecord{
		TrxHash:     trxLog.TrxId,
		BlockHeight: blockLog.BlockNum,
		BlockTime:   time.Unix(int64(blockLog.BlockTime), 0),
		Account:        op.GetFrom().GetValue(),
		Action:      opLog.Type,
	}).Error
}

func ProcessAcquireTicketOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	trxLog := blockLog.Transactions[trxIdx]
	opLog := trxLog.Operations[opIdx]
	if opLog.Type != "acquire_ticket" {
		return nil
	}
	op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.AcquireTicketOperation)
	if !ok {
		return errors.New("failed conversion to AcquireTicketOperation")
	}
	return db.Create(&IOTrxRecord{
		TrxHash:     trxLog.TrxId,
		BlockHeight: blockLog.BlockNum,
		BlockTime:   time.Unix(int64(blockLog.BlockTime), 0),
		Account:        op.GetAccount().GetValue(),
		Action:      opLog.Type,
	}).Error
}

func ProcessVoteByTicketOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	trxLog := blockLog.Transactions[trxIdx]
	opLog := trxLog.Operations[opIdx]
	if opLog.Type != "vote_by_ticket" {
		return nil
	}
	op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.VoteByTicketOperation)
	if !ok {
		return errors.New("failed conversion to VoteByTicketOperation")
	}
	return db.Create(&IOTrxRecord{
		TrxHash:     trxLog.TrxId,
		BlockHeight: blockLog.BlockNum,
		BlockTime:   time.Unix(int64(blockLog.BlockTime), 0),
		Account:        op.GetAccount().GetValue(),
		Action:      opLog.Type,
	}).Error
}