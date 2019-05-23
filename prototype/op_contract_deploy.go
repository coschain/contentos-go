package prototype

import "github.com/pkg/errors"

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

	if len(m.Contract) <= 0 || len(m.Contract) > 16 {
		return errors.New("contract Name length must cant be 1-16")
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
