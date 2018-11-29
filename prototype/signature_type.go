package prototype

func (m *SignatureType) Validate() error {
	if m == nil {
		return ErrNpe
	}
	if len(m.Sig) != 65 {
		return ErrSigLength
	}
	return nil
}
