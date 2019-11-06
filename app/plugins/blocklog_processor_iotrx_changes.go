package plugins

import (
	"errors"
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/prototype"
	"github.com/jinzhu/gorm"
	"time"
)

func ProcessContractTransferToUserChangeProcessor(db *gorm.DB, change *blocklog.StateChange, blockLog *blocklog.BlockLog, changeIdx, opIdx, trxIdx int) error {
	if change.What != "Account.Balance" {
		return nil
	}
	trxLog := blockLog.Transactions[trxIdx]
	opLog := trxLog.Operations[opIdx]
	if opLog.Type != "contract_apply" {
		return nil
	}
	op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.ContractApplyOperation)
	if !ok {
		return errors.New("failed conversion to ContractApplyOperation")
	}
	if change.Cause == "contract_apply.vm_native.transfer_to_user"{
		owner := op.GetOwner().GetValue()
		contract := op.GetContract()
		contractName := owner + "@" + contract
		userName := change.Change.Id.(string)
		ioTrxRecordContract := makeIOTrx(trxLog.TrxId, blockLog.BlockNum, time.Unix(int64(blockLog.BlockTime), 0),
			contractName, "contract_transfer_to_user")
		ioTrxRecordUser := makeIOTrx(trxLog.TrxId, blockLog.BlockNum, time.Unix(int64(blockLog.BlockTime), 0),
			userName, "contract_transfer_to_user")
		if err := db.Create(ioTrxRecordContract).Error; err != nil {
			return err
		}
		if err := db.Create(ioTrxRecordUser).Error; err != nil {
			return err
		}
		return nil
	}
	return nil
}

func ProcessUserToContractChangeProcessor(db *gorm.DB, change *blocklog.StateChange, blockLog *blocklog.BlockLog, changeIdx, opIdx, trxIdx int) error {
	if change.What != "Account.Balance" {
		return nil
	}
	trxLog := blockLog.Transactions[trxIdx]
	opLog := trxLog.Operations[opIdx]
	if opLog.Type != "contract_apply" {
		return nil
	}
	op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.ContractApplyOperation)
	if !ok {
		return errors.New("failed conversion to ContractApplyOperation")
	}
	if change.Cause == "contract_apply.u2c"{
		owner := op.GetOwner().GetValue()
		contract := op.GetContract()
		contractName := owner + "@" + contract
		userName := change.Change.Id.(string)
		ioTrxRecordContract := makeIOTrx(trxLog.TrxId, blockLog.BlockNum, time.Unix(int64(blockLog.BlockTime), 0),
			contractName, "user_transfer_to_contract")
		ioTrxRecordUser := makeIOTrx(trxLog.TrxId, blockLog.BlockNum, time.Unix(int64(blockLog.BlockTime), 0),
			userName, "user_transfer_to_contract")
		if err := db.Create(ioTrxRecordContract).Error; err != nil {
			return err
		}
		if err := db.Create(ioTrxRecordUser).Error; err != nil {
			return err
		}
		return nil
	}
	return nil
}

func ProcessContractTransferToContractChangeProcessor(db *gorm.DB, change *blocklog.StateChange, blockLog *blocklog.BlockLog, changeIdx, opIdx, trxIdx int) error {
	if change.What != "Contract.Balance" {
		return nil
	}
	trxLog := blockLog.Transactions[trxIdx]
	opLog := trxLog.Operations[opIdx]
	if opLog.Type != "contract_apply" {
		return nil
	}
	if change.Cause == "contract_apply.vm_native.transfer_to_contract"{
		op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.ContractApplyOperation)
		if !ok {
			return errors.New("failed conversion to ContractApplyOperation")
		}
		owner := op.GetOwner().GetValue()
		contract := op.GetContract()
		fromContractName := owner + "@" + contract
		toContractName := change.Change.Id.(string)

		if fromContractName != toContractName {
			ioTrxRecordContractFrom := makeIOTrx(trxLog.TrxId, blockLog.BlockNum, time.Unix(int64(blockLog.BlockTime), 0),
				fromContractName, "contract_transfer_to_contract")
			ioTrxRecordUserTo := makeIOTrx(trxLog.TrxId, blockLog.BlockNum, time.Unix(int64(blockLog.BlockTime), 0),
				toContractName, "contract_transfer_to_contract")
			if err := db.Create(ioTrxRecordContractFrom).Error; err != nil {
				return err
			}
			if err := db.Create(ioTrxRecordUserTo).Error; err != nil {
				return err
			}
			return nil
		}
		return nil
	}
	return nil
}
