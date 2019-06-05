package prototype

func (m *AccountUpdateOperation) GetSigner(auths *map[string]bool) {
	(*auths)[m.Owner.Value] = true
}

func (m *AccountUpdateOperation) Validate() error {
	// TODO
	return nil
}

func (m *AccountUpdateOperation) GetAffectedProps(props *map[string]bool) {
	(*props)["*"] = true
}

func init() {
	registerOperation("account_update", (*Operation_Op20)(nil), (*AccountUpdateOperation)(nil));
}
