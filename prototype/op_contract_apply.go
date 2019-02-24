package prototype

func (m *ContractApplyOperation) GetRequiredOwner(auths *map[string]bool) {
	(*auths)[m.Caller.Value] = true
}

func (m *ContractApplyOperation) Validate() error {
	// TODO
	return nil
}
