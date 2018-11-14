package prototype


func (m *Coin) OpeEncode() ([]byte, error) {
	return m.Amount.OpeEncode()
}

func MakeCoin(value int64) *Coin {
	return &Coin{Amount: MakeSafe64(value)}
}