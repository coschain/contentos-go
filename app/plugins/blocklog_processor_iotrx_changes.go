package plugins

import (
	"errors"
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
)


func ProcessContractTransferToUserChangeProcessor(opType string, operation prototype.BaseOperation, change *blocklog.StateChange, baseRecord interface{}) ([]interface{}, error) {
	if opType != "contract_apply" || change.What != "Account.Balance" {
		return nil, nil
	}
	op, ok := operation.(*prototype.ContractApplyOperation)
	if !ok {
		return nil, errors.New("failed conversion to ContractApplyOperation")
	}
	if change.Cause == "contract_apply.vm_native.transfer_to_user"{
		owner := op.GetOwner().GetValue()
		contract := op.GetContract()
		contractName := owner + "@" + contract
		userName := change.Change.Id.(string)
		ioTrxRecordContract := baseRecord.(iservices.IOTrxRecord)
		ioTrxRecordContract.Account = contractName
		ioTrxRecordContract.Action = "contract_transfer_to_user"
		ioTrxRecordUser := baseRecord.(iservices.IOTrxRecord)
		ioTrxRecordUser.Account = userName
		ioTrxRecordUser.Account = "contract_transfer_to_user"
		return []interface{}{ioTrxRecordContract, ioTrxRecordUser}, nil
	}
	return nil, nil
}

func ProcessUserToContractChangeProcessor(opType string, operation prototype.BaseOperation, change *blocklog.StateChange, baseRecord interface{}) ([]interface{}, error) {
	if opType != "contract_apply" || change.What != "Account.Balance" {
		return nil, nil
	}
	op, ok := operation.(*prototype.ContractApplyOperation)
	if !ok {
		return nil, errors.New("failed conversion to ContractApplyOperation")
	}
	if change.Cause == "contract_apply.u2c"{
		owner := op.GetOwner().GetValue()
		contract := op.GetContract()
		contractName := owner + "@" + contract
		userName := change.Change.Id.(string)
		ioTrxRecordContract := baseRecord.(iservices.IOTrxRecord)
		ioTrxRecordContract.Account = contractName
		ioTrxRecordContract.Action = "user_transfer_to_contract"
		ioTrxRecordUser := baseRecord.(iservices.IOTrxRecord)
		ioTrxRecordUser.Account = userName
		ioTrxRecordUser.Account = "user_transfer_to_contract"
		return []interface{}{ioTrxRecordUser, ioTrxRecordContract}, nil
	}
	return nil, nil
}

func ProcessContractTransferToContractChangeProcessor(opType string, operation prototype.BaseOperation, change *blocklog.StateChange, baseRecord interface{}) ([]interface{}, error) {
	if opType != "contract_apply" || change.What != "Contract.Balance" {
		return nil, nil
	}
	op, ok := operation.(*prototype.ContractApplyOperation)
	if !ok {
		return nil, errors.New("failed conversion to ContractApplyOperation")
	}
	if change.Cause == "contract_apply.vm_native.transfer_to_contract"{
		owner := op.GetOwner().GetValue()
		contract := op.GetContract()
		fromContractName := owner + "@" + contract
		toContractName := change.Change.Id.(string)

		if fromContractName != toContractName {
			ioTrxRecordContractFrom := baseRecord.(iservices.IOTrxRecord)
			ioTrxRecordContractFrom.Account = fromContractName
			ioTrxRecordContractFrom.Action = "contract_transfer_to_contract"
			ioTrxRecordContractTo := baseRecord.(iservices.IOTrxRecord)
			ioTrxRecordContractTo.Account = toContractName
			ioTrxRecordContractTo.Action = "contract_transfer_to_contract"
			return []interface{}{ioTrxRecordContractFrom, ioTrxRecordContractTo}, nil
		}
		return nil, nil
	}
	return nil, nil
}
