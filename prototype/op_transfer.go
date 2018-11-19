package prototype

import "errors"

func (t *TransferOperation) GetAuthorities(auths *[]Authority) {

}
func (t *TransferOperation) GetRequiredPosting(auths *map[string]bool) {

}

func (t *TransferOperation) GetAdmin(*[]AccountAdminPair) {

}
func (t *TransferOperation) IsVirtual() {

}


func (t *TransferOperation) GetRequiredActive(auths *map[string]bool) {
	(*auths)[t.From.Value] = true
}
func (t *TransferOperation) GetRequiredOwner(auths *map[string]bool) {
	(*auths)[t.From.Value] = true
}

func (t *TransferOperation) Validate() error {
	if t == nil {
		return ErrNpe
	}
	if err := t.From.Validate(); err != nil{
		return err
	}
	if err := t.To.Validate(); err != nil{
		return err
	}
	if t.Amount == nil || !t.Amount.NonZero() {
		return errors.New("transfer op must has amount value")
	}
	return nil
}
