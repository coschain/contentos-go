package prototype
func (m *StakeOperation) GetSigner(auths *map[string]bool) {
	(*auths)[m.Account.Value] = true
}


func (m *StakeOperation) Validate() error {
	// TODO
	return nil
}

func (m *StakeOperation) GetAffectedProps(props *map[string]bool) {
	(*props)["*"] = true
}