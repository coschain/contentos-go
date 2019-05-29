package prototype

const StatusSuccess  = 200
const StatusDeductStamina  = 201
const StatusError  = 500

func (m *TransactionReceiptWithInfo) Validate() error {
	return nil
}

func (m *TransactionReceiptWithInfo) IsSuccess() bool {
	return m.Status == StatusSuccess || m.Status == StatusDeductStamina
}