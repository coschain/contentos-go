package prototype

import "errors"

func (m *ContractApplyOperation) GetSigner(auths *map[string]bool) {
	(*auths)[m.Caller.Value] = true
}

func (m *ContractApplyOperation) Validate() error {
	if err := m.Owner.Validate(); err != nil{
		return err
	}
	if err := m.Caller.Validate(); err != nil{
		return err
	}

	if len(m.Contract) == 0 {
		return errors.New("invalid contract name")
	}

	if len(m.Method) == 0 {
		return errors.New("invalid method name")
	}

	return nil
}

func (m *ContractApplyOperation) GetAffectedProps(props *map[string]bool) {
	(*props)["*"] = true
}

func init() {
	registerOperation("contract_apply", (*Operation_Op14)(nil), (*ContractApplyOperation)(nil));
}
