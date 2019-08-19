package prototype

const StatusSuccess  uint32 = 200
const StatusFailDeductStamina uint32 = 201
const StatusError  uint32 = 500

func (m *TransactionReceiptWithInfo) Validate() error {
	return nil
}

func (m *TransactionReceiptWithInfo) IsSuccess() bool {
	return m.Status == StatusSuccess
}

func (m *TransactionReceiptWithInfo) IsExecuted() bool {
	return m.Status == StatusSuccess || m.Status == StatusFailDeductStamina
}

func (m *TransactionReceiptWithInfo) IsFailDeductStamina() bool {
	return m.Status == StatusFailDeductStamina
}