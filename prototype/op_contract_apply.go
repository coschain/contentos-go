package prototype

func (m *ContractApplyOperation) GetSigner(auths *map[string]bool) {
	(*auths)[m.Caller.Value] = true
}

func (m *ContractApplyOperation) Validate() error {
	// TODO
	return nil
}

func (m *ContractApplyOperation) GetAffectedProps(props *map[string]bool) {
	(*props)["*"] = true
}

func init() {
	registerOperation("contract_apply", (*Operation_Op14)(nil), (*ContractApplyOperation)(nil));
}
