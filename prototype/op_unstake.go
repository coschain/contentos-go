package prototype
func (m *UnStakeOperation) GetSigner(auths *map[string]bool) {
	(*auths)[m.Creditor.Value] = true
}


func (m *UnStakeOperation) Validate() error {
	// TODO
	return nil
}

func (m *UnStakeOperation) GetAffectedProps(props *map[string]bool) {
	(*props)["*"] = true
}

func init() {
	registerOperation("un_stake", (*Operation_Op18)(nil), (*UnStakeOperation)(nil));
}
