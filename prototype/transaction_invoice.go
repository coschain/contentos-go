package prototype


func (m *TransactionInvoice) Validate() error {


	return nil
}

func (m *TransactionInvoice) IsSuccess() bool {
	return m.Status == 200
}
