package prototype
//
//func (m *ClaimOperation) GetSigner(auths *map[string]bool) {
//	(*auths)[m.Account.Value] = true
//}
//
//func (m *ClaimOperation) Validate() error {
//	if err := m.Account.Validate(); err != nil{
//		return err
//	}
//	return nil
//}
//
//func (m *ClaimOperation) GetAffectedProps(props *map[string]bool) {
//	(*props)["*"] = true
//}
//
//
//func (m *ClaimAllOperation) GetSigner(auths *map[string]bool) {
//	(*auths)[m.Account.Value] = true
//}
//
//func (m *ClaimAllOperation) Validate() error {
//	if err := m.Account.Validate(); err != nil{
//		return err
//	}
//	return nil
//}
//
//func (m *ClaimAllOperation) GetAffectedProps(props *map[string]bool) {
//	(*props)["*"] = true
//}
//
//func init() {
//	registerOperation("claim", (*Operation_Op11)(nil), (*ClaimOperation)(nil));
//	registerOperation("claim_all", (*Operation_Op12)(nil), (*ClaimAllOperation)(nil));
//}
