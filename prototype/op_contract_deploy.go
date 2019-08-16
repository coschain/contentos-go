package prototype

import (
	"fmt"
	"github.com/pkg/errors"
)

func (m *ContractDeployOperation) GetSigner(auths *map[string]bool) {
	(*auths)[m.Owner.Value] = true
}

func (m *ContractDeployOperation) Validate() error {

	if len(m.Code) <= 0 {
		return errors.New("code size must cant be 0")
	}
	if len(m.Abi) <= 0 {
		return errors.New("abi size must cant be 0")
	}

	if err := ValidContractName(m.Contract); err != nil {
		return fmt.Errorf("invalid contract name: %s", err.Error())
	}
	if err := AtMost1KChars(m.Url); err != nil {
		return fmt.Errorf("invalid contract url: %s", err.Error())
	}
	if err := AtMost4KChars(m.Describe); err != nil {
		return fmt.Errorf("invalid contract description: %s", err.Error())
	}
	if err := m.Owner.Validate(); err != nil{
		return err
	}

	return nil
}

func (m *ContractDeployOperation) GetAffectedProps(props *map[string]bool) {
	(*props)["*"] = true
}

func init() {
	registerOperation("contract_deploy", (*Operation_Op13)(nil), (*ContractDeployOperation)(nil));
}
