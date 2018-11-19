package prototype

func (m *BpRegisterOperation) GetAuthorities(auths *[]Authority) {

}
func (m *BpRegisterOperation) GetRequiredPosting(auths *map[string]bool) {

}

func (m *BpRegisterOperation) GetRequiredOwner(auths *map[string]bool) {

}
func (m *BpRegisterOperation) GetAdmin(*[]AccountAdminPair) {

}
func (m *BpRegisterOperation) IsVirtual() {

}

func (m *BpRegisterOperation) GetRequiredActive(auths *map[string]bool) {
	(*auths)[m.Owner.Value] = true
}

func (m *BpRegisterOperation)Validate() error  {
	if m == nil {
		return ErrNpe
	}

	if err := m.Owner.Validate(); err != nil{
		return err
	}
	if err := m.BlockSigningKey.Validate(); err != nil{
		return err
	}
	if m.Props == nil {
		return ErrNpe
	}

	// TODO chain property valid check
	return nil
}
