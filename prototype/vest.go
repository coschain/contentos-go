package prototype


func (m *Vest) OpeEncode() ([]byte, error) {
	return m.Amount.OpeEncode()
}

func MakeVest(value int64) *Vest {
	return &Vest{Amount: MakeSafe64(value)}
}