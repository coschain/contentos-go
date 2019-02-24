package prototype


func (op *InternalContractApplyOperation) GetRequiredOwner(auths *map[string]bool) {
	(*auths)[op.FromCaller.Value] = true
}

func (op *InternalContractApplyOperation) Validate() error {
	// TODO
	return nil
}
