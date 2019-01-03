package prototype

func (m *ClaimOperation) GetAuthorities(auths *[]Authority) {

}
func (m *ClaimOperation) GetRequiredOwner(auths *map[string]bool) {
	(*auths)[m.Account.Value] = true
}
func (m *ClaimOperation) GetAdmin(*[]AccountAdminPair) {

}
func (m *ClaimOperation) IsVirtual() {

}
func (m *ClaimOperation) Validate() error {
	return nil
}

func (m *ClaimAllOperation) GetAuthorities(auths *[]Authority) {

}
func (m *ClaimAllOperation) GetRequiredOwner(auths *map[string]bool) {
	(*auths)[m.Account.Value] = true
}
func (m *ClaimAllOperation) GetAdmin(*[]AccountAdminPair) {

}
func (m *ClaimAllOperation) IsVirtual() {

}
func (m *ClaimAllOperation) Validate() error {
	return nil
}
