package prototype

func (m *BpUpdateOperation) GetSigner(auths *map[string]bool) {
	(*auths)[m.Owner.Value] = true
}

func (m *BpUpdateOperation) Validate() error {
	// TODO
	return nil
}

func (m *BpUpdateOperation) GetAffectedProps(props *map[string]bool) {
	(*props)["*"] = true
}

func init() {
	registerOperation("bp_update", (*Operation_Op19)(nil), (*BpUpdateOperation)(nil));
}
