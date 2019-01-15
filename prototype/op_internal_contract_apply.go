package prototype

func (op *InternalContractApplyOperation) GetAuthorities(auths *[]Authority) {

}
func (op *InternalContractApplyOperation) GetRequiredOwner(auths *map[string]bool) {
	(*auths)[op.FromCaller.Value] = true
}
func (op *InternalContractApplyOperation) GetAdmin(*[]AccountAdminPair) {

}
func (op *InternalContractApplyOperation) IsVirtual() {

}
func (op *InternalContractApplyOperation) Validate() error {
	// TODO
	return nil
}
