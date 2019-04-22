package prototype


func (m *ContractEstimateApplyOperation) GetRequiredOwner(auths *map[string]bool) {
	(*auths)[m.Caller.Value] = true
}


func (m *ContractEstimateApplyOperation) Validate() error {
	// TODO
	return nil
}
