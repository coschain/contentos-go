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
func (t *TransferOperation) Validate() bool {
	if t == nil {
		return false
	}
	if !t.From.Validate(){
		return false
	}
	if !t.To.Validate(){
		return false
	}
	if t.Amount == nil {
		return false
	}
	if !t.Amount.NonZero(){
		return false
	}
	return true
}
