package prototype


func (m *ClaimOperation) GetRequiredOwner(auths *map[string]bool) {
	(*auths)[m.Account.Value] = true
}

func (m *ClaimOperation) Validate() error {
	return nil
}


func (m *ClaimAllOperation) GetRequiredOwner(auths *map[string]bool) {
	(*auths)[m.Account.Value] = true
}

func (m *ClaimAllOperation) Validate() error {
	return nil
}
