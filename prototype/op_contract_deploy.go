package prototype

func (m *ContractDeployOperation) GetAuthorities(auths *[]Authority) {

}

func (m *ContractDeployOperation) GetRequiredOwner(auths *map[string]bool) {
	(*auths)[m.Owner.Value] = true
}
func (m *ContractDeployOperation) GetAdmin(*[]AccountAdminPair) {

}
func (m *ContractDeployOperation) IsVirtual() {

}

func (m *ContractDeployOperation) Validate() error {
	// TODO
	return nil
}
