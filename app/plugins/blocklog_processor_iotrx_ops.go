package plugins

import (
	"errors"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
)


func ProcessAccountCreateOperation(baseOp prototype.BaseOperation, baseRecord interface{}) ([]interface{}, error) {
	op, ok := baseOp.(*prototype.AccountCreateOperation)
	if !ok {
		return nil, errors.New("failed conversion to AccountCreateOperation")
	}
	ioTrx := baseRecord.(iservices.IOTrxRecord)
	ioTrx.From = op.GetCreator().GetValue()
	ioTrx.To = op.GetNewAccountName().GetValue()
	ioTrx.Amount = op.GetFee().GetValue()
	return []interface{}{ioTrx}, nil
}

func ProcessTransferOperation(baseOp prototype.BaseOperation, baseRecord interface{}) ([]interface{}, error) {
	op, ok := baseOp.(*prototype.TransferOperation)
	if !ok {
		return nil, errors.New("failed conversion to TransferOperation")
	}
	ioTrx := baseRecord.(iservices.IOTrxRecord)
	ioTrx.From= op.GetFrom().GetValue()
	ioTrx.To = op.GetTo().GetValue()
	ioTrx.Amount = op.GetAmount().GetValue()
	return []interface{}{ioTrx}, nil
}

func ProcessTransferVestOperation(baseOp prototype.BaseOperation, baseRecord interface{}) ([]interface{}, error) {
	op, ok := baseOp.(*prototype.TransferToVestOperation)
	if !ok {
		return nil, errors.New("failed conversion to TransferToVestOperation")
	}
	ioTrx := baseRecord.(iservices.IOTrxRecord)
	ioTrx.From = op.GetFrom().GetValue()
	ioTrx.To = op.GetTo().GetValue()
	ioTrx.Amount = op.GetAmount().GetValue()
	return []interface{}{ioTrx}, nil
}

func ProcessStakeOperation(baseOp prototype.BaseOperation, baseRecord interface{}) ([]interface{}, error) {
	op, ok := baseOp.(*prototype.StakeOperation)
	if !ok {
		return nil, errors.New("failed conversion to StakeOperation")
	}
	fromUser := op.GetFrom().GetValue()
	toUser := op.GetTo().GetValue()
	ioTrx := baseRecord.(iservices.IOTrxRecord)
	ioTrx.From = fromUser
	ioTrx.To = toUser
	ioTrx.Amount = op.GetAmount().GetValue()
	return []interface{}{ioTrx}, nil
}

func ProcessUnStakeOperation(baseOp prototype.BaseOperation, baseRecord interface{}) ([]interface{}, error) {
	op, ok := baseOp.(*prototype.UnStakeOperation)
	if !ok {
		return nil, errors.New("failed conversion to UnStakeOperation")
	}
	creditor := op.GetCreditor().GetValue()
	debtor := op.GetDebtor().GetValue()
	ioTrx := baseRecord.(iservices.IOTrxRecord)
	ioTrx.From = creditor
	ioTrx.To = debtor
	ioTrx.Amount = op.GetAmount().GetValue()
	return []interface{}{ioTrx}, nil
}

func ProcessAccountUpdateOperation(baseOp prototype.BaseOperation, baseRecord interface{}) ([]interface{}, error) {
	op, ok := baseOp.(*prototype.AccountUpdateOperation)
	if !ok {
		return nil, errors.New("failed conversion to AccountUpdateOperation")
	}
	ioTrx := baseRecord.(iservices.IOTrxRecord)
	ioTrx.From = op.GetOwner().GetValue()
	return []interface{}{ioTrx}, nil
}

func ProcessVoteOperation(baseOp prototype.BaseOperation, baseRecord interface{}) ([]interface{}, error) {
	op, ok := baseOp.(*prototype.VoteOperation)
	if !ok {
		return nil, errors.New("failed conversion to VoteOperation")
	}
	ioTrx := baseRecord.(iservices.IOTrxRecord)
	ioTrx.From = op.GetVoter().GetValue()
	return []interface{}{ioTrx}, nil
}

func ProcessBpRegisterOperation(baseOp prototype.BaseOperation, baseRecord interface{}) ([]interface{}, error) {
	op, ok := baseOp.(*prototype.BpRegisterOperation)
	if !ok {
		return nil, errors.New("failed conversion to BpRegisterOperation")
	}
	ioTrx := baseRecord.(iservices.IOTrxRecord)
	ioTrx.From = op.GetOwner().GetValue()
	return []interface{}{ioTrx}, nil
}

func ProcessBpUpdateOperation(baseOp prototype.BaseOperation, baseRecord interface{}) ([]interface{}, error) {
	op, ok := baseOp.(*prototype.BpUpdateOperation)
	if !ok {
		return nil, errors.New("failed conversion to BpUpdateOperation")
	}
	ioTrx := baseRecord.(iservices.IOTrxRecord)
	ioTrx.From = op.GetOwner().GetValue()
	return []interface{}{ioTrx}, nil
}

func ProcessBpEnableOperation(baseOp prototype.BaseOperation, baseRecord interface{}) ([]interface{}, error) {
	op, ok := baseOp.(*prototype.BpEnableOperation)
	if !ok {
		return nil, errors.New("failed conversion to BpEnableOperation")
	}
	ioTrx := baseRecord.(iservices.IOTrxRecord)
	ioTrx.From = op.GetOwner().GetValue()
	return []interface{}{ioTrx}, nil
}

func ProcessBpVoteOperation(baseOp prototype.BaseOperation, baseRecord interface{}) ([]interface{}, error) {
	op, ok := baseOp.(*prototype.BpVoteOperation)
	if !ok {
		return nil, errors.New("failed conversion to BpVoteOperation")
	}
	ioTrx := baseRecord.(iservices.IOTrxRecord)
	ioTrx.From = op.GetVoter().GetValue()
	return []interface{}{ioTrx}, nil
}

func ProcessContractDeployOperation(baseOp prototype.BaseOperation, baseRecord interface{}) ([]interface{}, error) {
	op, ok := baseOp.(*prototype.ContractDeployOperation)
	if !ok {
		return nil, errors.New("failed conversion to ContractDeployOperation")
	}
	ioTrx := baseRecord.(iservices.IOTrxRecord)
	ioTrx.From = op.GetOwner().GetValue()
	ioTrx.To = op.GetOwner().GetValue() + "@" + op.GetContract()
	return []interface{}{ioTrx}, nil
}

func ProcessContractApplyOperation(baseOp prototype.BaseOperation, baseRecord interface{}) ([]interface{}, error) {
	op, ok := baseOp.(*prototype.ContractApplyOperation)
	if !ok {
		return nil, errors.New("failed conversion to ContractApplyOperation")
	}
	ioTrx := baseRecord.(iservices.IOTrxRecord)
	ioTrx.From = op.GetCaller().GetValue()
	ioTrx.To = op.GetOwner().GetValue() + "@" + op.GetContract()
	return []interface{}{ioTrx}, nil
}

func ProcessPostOperation(baseOp prototype.BaseOperation, baseRecord interface{}) ([]interface{}, error) {
	op, ok := baseOp.(*prototype.PostOperation)
	if !ok {
		return nil, errors.New("failed conversion to PostOperation")
	}
	ioTrx := baseRecord.(iservices.IOTrxRecord)
	ioTrx.From = op.GetOwner().GetValue()
	return []interface{}{ioTrx}, nil
}

func ProcessReplyOperation(baseOp prototype.BaseOperation, baseRecord interface{}) ([]interface{}, error) {
	op, ok := baseOp.(*prototype.ReplyOperation)
	if !ok {
		return nil, errors.New("failed conversion to ReplyOperation")
	}
	ioTrx := baseRecord.(iservices.IOTrxRecord)
	ioTrx.From = op.GetOwner().GetValue()
	return []interface{}{ioTrx}, nil
}

func ProcessConvertVestOperation(baseOp prototype.BaseOperation, baseRecord interface{}) ([]interface{}, error) {
	op, ok := baseOp.(*prototype.ConvertVestOperation)
	if !ok {
		return nil, errors.New("failed conversion to ConvertVestOperation")
	}
	ioTrx := baseRecord.(iservices.IOTrxRecord)
	ioTrx.From = op.GetFrom().GetValue()
	return []interface{}{ioTrx}, nil
}

func ProcessAcquireTicketOperation(baseOp prototype.BaseOperation, baseRecord interface{}) ([]interface{}, error) {
	op, ok := baseOp.(*prototype.AcquireTicketOperation)
	if !ok {
		return nil, errors.New("failed conversion to AcquireTicketOperation")
	}
	ioTrx := baseRecord.(iservices.IOTrxRecord)
	ioTrx.From = op.GetAccount().GetValue()
	return []interface{}{ioTrx}, nil
}

func ProcessVoteByTicketOperation(baseOp prototype.BaseOperation, baseRecord interface{}) ([]interface{}, error) {
	op, ok := baseOp.(*prototype.VoteByTicketOperation)
	if !ok {
		return nil, errors.New("failed conversion to VoteByTicketOperation")
	}
	ioTrx := baseRecord.(iservices.IOTrxRecord)
	ioTrx.From = op.GetAccount().GetValue()
	return []interface{}{ioTrx}, nil
}

