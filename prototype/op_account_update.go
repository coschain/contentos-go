package prototype

func (m *AccountUpdateOperation) GetSigner(auths *map[string]bool) {
	(*auths)[m.Owner.Value] = true
}

func (m *AccountUpdateOperation) Validate() error {
	if err := m.Pubkey.Validate(); err != nil{
		return err
	}
	if err := m.Owner.Validate(); err != nil{
		return err
	}

	return nil
}

func (m *AccountUpdateOperation) GetAffectedProps(props *map[string]bool) {
	(*props)["*"] = true
}

func init() {
	registerOperation("account_update", (*Operation_Op20)(nil), (*AccountUpdateOperation)(nil));
}
