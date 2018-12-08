package prototype

func (m *ContractApplyOperation) GetAuthorities(auths *[]Authority) {

}
func (m *ContractApplyOperation) GetRequiredPosting(auths *map[string]bool) {

}

func (m *ContractApplyOperation) GetRequiredOwner(auths *map[string]bool) {

}
func (m *ContractApplyOperation) GetAdmin(*[]AccountAdminPair) {

}
func (m *ContractApplyOperation) IsVirtual() {

}

func (m *ContractApplyOperation) GetRequiredActive(auths *map[string]bool) {
	(*auths)[m.Owner.Value] = true
}

func (m *ContractApplyOperation) Validate() error {
	// TODO
	return nil
}
