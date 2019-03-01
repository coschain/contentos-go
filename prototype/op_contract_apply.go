package prototype

func (m *ContractApplyOperation) GetSigner(auths *map[string]bool) {
	(*auths)[m.Caller.Value] = true
}

func (m *ContractApplyOperation) Validate() error {
	// TODO
	return nil
}
