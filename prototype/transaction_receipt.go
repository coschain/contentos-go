package prototype

func (m *TransactionReceipt) Validate() error {
	return nil
}

func (m *TransactionReceipt) IsSuccess() bool {
	return m.Status == StatusSuccess || m.Status == StatusDeductGas
}