package prototype

import "github.com/pkg/errors"

func (op *InternalContractApplyOperation) GetSigner(auths *map[string]bool) {
	(*auths)[op.FromCaller.Value] = true
}

func (op *InternalContractApplyOperation) Validate() error {
	if op == nil {
		return ErrNpe
	}
	if err := op.FromCaller.Validate(); err != nil {
		return errors.WithMessage(err, "fromCaller error")
	}
	if err := op.FromOwner.Validate(); err != nil {
		return errors.WithMessage(err, "fromOwner error")
	}
	if err := ValidContractName(op.FromContract); err != nil {
		return errors.WithMessage(err, "fromContract error")
	}
	if err := ValidContractMethodName(op.FromMethod); err != nil {
		return errors.WithMessage(err, "fromMethod error")
	}
	if err := op.ToOwner.Validate(); err != nil {
		return errors.WithMessage(err, "toOwner error")
	}
	if err := ValidContractName(op.ToContract); err != nil {
		return errors.WithMessage(err, "toContract error")
	}
	if err := ValidContractMethodName(op.ToMethod); err != nil {
		return errors.WithMessage(err, "toMethod error")
	}
	return nil
}

func (m *InternalContractApplyOperation) GetAffectedProps(props *map[string]bool) {
	(*props)["*"] = true
}
