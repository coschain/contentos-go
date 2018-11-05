package prototype

func (t *TransferOperation) GetAuthorities(auths *[]Authority) {

}
func (t *TransferOperation) GetRequiredPosting(auths *map[string]bool) {

}
func (t *TransferOperation) GetRequiredActive(auths *map[string]bool) {
	(*auths)[t.From.Value] = true
}
func (t *TransferOperation) GetRequiredOwner(auths *map[string]bool) {
	(*auths)[t.From.Value] = true
}
func (t *TransferOperation) GetAdmin(*[]AccountAdminPair) {

}
func (t *TransferOperation) IsVirtual() {

}
func (t *TransferOperation) Validate() {

}
