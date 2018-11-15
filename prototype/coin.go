package prototype


func (m *Coin) OpeEncode() ([]byte, error) {
	return m.Amount.OpeEncode()
}

func (m *Coin) NonZero() bool {
	return m.Amount.Value != 0
}

func MakeCoin(value int64) *Coin {
	return &Coin{Amount: MakeSafe64(value)}
}