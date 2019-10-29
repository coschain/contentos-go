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
		fromContractName := owner + "@" + contract
		toUserName := change.Change.Id.(string)
		return db.Create(&IOTrxRecord{
			TrxHash:     trxLog.TrxId,
			BlockHeight: blockLog.BlockNum,
			BlockTime:   time.Unix(int64(blockLog.BlockTime), 0),
			From:        fromContractName,
			To:          toUserName,
			Action:      "contract transfer",
		}).Error
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
			return db.Create(&IOTrxRecord{
				TrxHash:     trxLog.TrxId,
				BlockHeight: blockLog.BlockNum,
				BlockTime:   time.Unix(int64(blockLog.BlockTime), 0),
				From:        fromContractName,
				To:          toContractName,
				Action:      "contract transfer",
			}).Error
		}
		return nil
	}
	return nil
}
