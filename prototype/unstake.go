package prototype
func (m *UnStakeOperation) GetSigner(auths *map[string]bool) {
	(*auths)[m.Account.Value] = true
}


func (m *UnStakeOperation) Validate() error {
	// TODO
	return nil
}