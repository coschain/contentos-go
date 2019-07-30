package prototype

const StatusSuccess  uint32 = 200
const StatusDeductStamina  uint32 = 201
const StatusError  uint32 = 500

func (m *TransactionReceiptWithInfo) Validate() error {
	return nil
}

func (m *TransactionReceiptWithInfo) IsSuccess() bool {
	return m.Status == StatusSuccess || m.Status == StatusDeductStamina
}