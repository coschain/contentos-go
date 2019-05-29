package prototype

func (w *TransactionWrapperWithInfo) ToWrapper() *TransactionWrapper {
	return &TransactionWrapper{
		SigTrx: w.SigTrx,
		Receipt: w.Receipt.ToReceipt(),
	}
}

func (r *TransactionReceiptWithInfo) ToReceipt() *TransactionReceipt {
	opResults := make([]*OperationReceipt, len(r.OpResults))
	for i := range opResults {
		opResults[i] = r.OpResults[i].ToReceipt()
	}
	return &TransactionReceipt{
		Status: r.Status,
		NetUsage: r.NetUsage,
		CpuUsage: r.CpuUsage,
		OpResults: opResults,
	}
}

func (r *OperationReceiptWithInfo) ToReceipt() *OperationReceipt {
	return &OperationReceipt{
		Status: r.Status,
		GasUsage: r.GasUsage,
	}
}
