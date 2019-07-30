package prototype
func (m *StakeOperation) GetSigner(auths *map[string]bool) {
	(*auths)[m.From.Value] = true
}


func (m *StakeOperation) Validate() error {
	if err := m.From.Validate(); err != nil{
		return err
	}
	if err := m.To.Validate(); err != nil{
		return err
	}

	if m.Amount.Value == 0 {
		return ErrCoinZero
	}

	return nil
}

func (m *StakeOperation) GetAffectedProps(props *map[string]bool) {
	(*props)["*"] = true
}

func init() {
	registerOperation("stake", (*Operation_Op17)(nil), (*StakeOperation)(nil));
}
