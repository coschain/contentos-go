package prototype

import (
	"fmt"
)

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
	if err := ValidContractName(m.Contract); err != nil{
		return fmt.Errorf("invalid contract name: %s", err.Error())
	}
	if err := ValidContractMethodName(m.Method); err != nil{
		return fmt.Errorf("invalid contract method name: %s", err.Error())
	}
	return nil
}

func (m *ContractApplyOperation) GetAffectedProps(props *map[string]bool) {
	(*props)["*"] = true
}

func init() {
	registerOperation("contract_apply", (*Operation_Op14)(nil), (*ContractApplyOperation)(nil));
}
