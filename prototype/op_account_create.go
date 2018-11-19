package prototype

import "errors"

func (a *AccountCreateOperation) GetAuthorities(auths *[]Authority) {

}
func (a *AccountCreateOperation) GetRequiredPosting(auths *map[string]bool) {

}

func (a *AccountCreateOperation) GetRequiredOwner(auths *map[string]bool) {

}
func (a *AccountCreateOperation) GetAdmin(*[]AccountAdminPair) {

}
func (a *AccountCreateOperation) IsVirtual() {

}


func (a *AccountCreateOperation) GetRequiredActive(auths *map[string]bool) {
	(*auths)[a.Creator.Value] = true
}


func (a *AccountCreateOperation) Validate() error {

	if a == nil{
		return ErrNpe
	}

	if err := a.Creator.Validate(); err != nil{
		return err
	}

	if err := a.NewAccountName.Validate();err != nil{
		return err
	}

	if a.MemoKey == nil {
		return errors.New("MemoKey cant be null")
	}
	if err := a.MemoKey.Validate(); err != nil {
		return err
	}

	if a.Posting == nil {
		return errors.New("Posting Key cant be empty")
	}

	if err := a.Posting.Validate(); err != nil {
		return err
	}

	if a.Active == nil {
		return errors.New("Posting Key cant be empty")
	}
	if err := a.Active.Validate(); err != nil {
		return err
	}
	if a.Owner == nil {
		return errors.New("Posting Key cant be empty")
	}
	if err := a.Owner.Validate(); err != nil {
		return err
	}

	if a.Fee == nil || a.Fee.Value == 0 {
		return errors.New("Account Create do not have Fee")
	}

	return nil
}
