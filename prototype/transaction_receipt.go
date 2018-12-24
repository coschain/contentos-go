package prototype

const StatusSuccess  = 200
const StatusError  = 500

func (m *TransactionReceipt) Validate() error {
	return nil
}

func (m *TransactionReceipt) IsSuccess() bool {
	return m.Status == StatusSuccess
}


func (m *TransactionReceiptWithInfo) Validate() error {
	return nil
}

func (m *TransactionReceiptWithInfo) ToReceipt() *TransactionReceipt {
	res := &TransactionReceipt{ Status:m.Status }

	for _ , v := range m.OpResults{
		rpt := &OperationReceipt{ Status: v.Status }
		res.OpResults = append(res.OpResults, rpt)
	}
	return res
}

func (m *TransactionReceiptWithInfo) IsSuccess() bool {
	return m.Status == StatusSuccess
}

func (m *EstimateTrxResult) ToTrxWrapper() *TransactionWrapper {
	return &TransactionWrapper{ SigTrx:m.SigTrx, Invoice:m.Receipt.ToReceipt() }
}