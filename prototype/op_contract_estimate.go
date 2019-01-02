package prototype

func (m *ContractEstimateApplyOperation) GetAuthorities(auths *[]Authority) {

}

func (m *ContractEstimateApplyOperation) GetRequiredOwner(auths *map[string]bool) {
	(*auths)[m.Caller.Value] = true
}
func (m *ContractEstimateApplyOperation) GetAdmin(*[]AccountAdminPair) {

}
func (m *ContractEstimateApplyOperation) IsVirtual() {

}

func (m *ContractEstimateApplyOperation) Validate() error {
	// TODO
	return nil
}
