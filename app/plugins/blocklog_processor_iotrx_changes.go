package plugins

import (
	"encoding/json"
	"errors"
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/common"
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
		before := common.JsonNumberUint64(change.Change.Before.(json.Number))
		after := common.JsonNumberUint64(change.Change.After.(json.Number))
		owner := op.GetOwner().GetValue()
		contract := op.GetContract()
		contractName := owner + "@" + contract
		userName := change.Change.Id.(string)
		ioTrx := baseRecord.(iservices.IOTrxRecord)
		ioTrx.From = contractName
		ioTrx.To = userName
		ioTrx.Action = "contract_transfer_to_user"
		ioTrx.Amount = after - before
		return []interface{}{ioTrx}, nil
	}
	return nil, nil
}

func ProcessContractTransferToUserVestChangeProcessor(opType string, operation prototype.BaseOperation, change *blocklog.StateChange, baseRecord interface{}) ([]interface{}, error) {
	if opType != "contract_apply" || change.What != "Account.Vest" {
		return nil, nil
	}
	op, ok := operation.(*prototype.ContractApplyOperation)
	if !ok {
		return nil, errors.New("failed conversion to ContractApplyOperation")
	}
	if change.Cause == "contract_apply.vm_native.transfer_to_user_vest"{
		before := common.JsonNumberUint64(change.Change.Before.(json.Number))
		after := common.JsonNumberUint64(change.Change.After.(json.Number))
		owner := op.GetOwner().GetValue()
		contract := op.GetContract()
		contractName := owner + "@" + contract
		userName := change.Change.Id.(string)
		ioTrx := baseRecord.(iservices.IOTrxRecord)
		ioTrx.From = contractName
		ioTrx.To = userName
		ioTrx.Action = "contract_transfer_vest_to_user"
		ioTrx.Amount = after - before
		return []interface{}{ioTrx}, nil
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
		ioTrx := baseRecord.(iservices.IOTrxRecord)
		ioTrx.From = userName
		ioTrx.To = contractName
		ioTrx.Action = "user_transfer_to_contract"
		ioTrx.Amount = op.GetAmount().GetValue()
		return []interface{}{ioTrx}, nil
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
		ioTrx := baseRecord.(iservices.IOTrxRecord)
		if fromContractName != toContractName {
			before := common.JsonNumberUint64(change.Change.Before.(json.Number))
			after := common.JsonNumberUint64(change.Change.After.(json.Number))
			ioTrx.From = fromContractName
			ioTrx.To = toContractName
			ioTrx.Action = "contract_transfer_to_contract"
			ioTrx.Amount = after - before
			return []interface{}{ioTrx}, nil
		}
		return nil, nil
	}
	return nil, nil
}

func ProcessUnDelegateVestChangeProcessor(opType string, operation prototype.BaseOperation, change *blocklog.StateChange, baseRecord interface{}) ([]interface{}, error) {
	if opType != "un_delegate_vest" || change.What != "Account.BorrowedVest" {
		return nil, nil
	}
	op, ok := operation.(*prototype.UnDelegateVestOperation)
	if !ok {
		return nil, errors.New("failed conversion to UnDelegateVestOperation")
	}
	borrower := change.Change.Id.(string)
	before := common.JsonNumberUint64(change.Change.Before.(json.Number))
	after := common.JsonNumberUint64(change.Change.After.(json.Number))
	ioTrx := baseRecord.(iservices.IOTrxRecord)
	ioTrx.From = op.GetAccount().GetValue()
	ioTrx.To = borrower
	ioTrx.Amount = before - after
	return []interface{}{ioTrx}, nil
}
