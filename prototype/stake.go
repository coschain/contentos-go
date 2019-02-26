package prototype
func (m *StakeOperation) GetRequiredOwner(auths *map[string]bool) {
	(*auths)[m.Account.Value] = true
}


func (m *StakeOperation) Validate() error {
	// TODO
	return nil
}