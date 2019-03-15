package prototype


func (m *ContractEstimateApplyOperation) GetSigner(auths *map[string]bool) {
	(*auths)[m.Caller.Value] = true
}


func (m *ContractEstimateApplyOperation) Validate() error {
	// TODO
	return nil
}
