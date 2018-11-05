package prototype

func (a *AccountCreateOperation) GetAuthorities(auths *[]Authority) {

}
func (a *AccountCreateOperation) GetRequiredPosting(auths *map[string]bool) {

}
func (a *AccountCreateOperation) GetRequiredActive(auths *map[string]bool) {
	(*auths)[a.Creator.Value] = true
}
func (a *AccountCreateOperation) GetRequiredOwner(auths *map[string]bool) {

}
func (a *AccountCreateOperation) GetAdmin(*[]AccountAdminPair) {

}
func (a *AccountCreateOperation) IsVirtual() {

}
func (a *AccountCreateOperation) Validate() {

}
