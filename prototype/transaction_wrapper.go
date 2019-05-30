package prototype

func (w *TransactionWrapperWithInfo) ToWrapper() *TransactionWrapper {
	return &TransactionWrapper{
		SigTrx: w.SigTrx,
		Receipt: w.Receipt.ToReceipt(),
	}
}

func (r *TransactionReceiptWithInfo) ToReceipt() *TransactionReceipt {
	return &TransactionReceipt{
		Status: r.Status,
		NetUsage: r.NetUsage,
		CpuUsage: r.CpuUsage,
	}
}
