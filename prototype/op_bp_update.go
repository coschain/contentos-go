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