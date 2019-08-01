package prototype

import "github.com/pkg/errors"

func (m *BpVoteOperation) GetSigner(auths *map[string]bool) {
	(*auths)[m.Voter.Value] = true
}

func (m *BpVoteOperation) Validate() error {
	if m == nil {
		return ErrNpe
	}

	if err := m.Voter.Validate(); err != nil {
		return errors.WithMessage(err, "Voter error")
	}
	if err := m.BlockProducer.Validate(); err != nil {
		return errors.WithMessage(err, "BlockProducer error")
	}

	return nil
}

func (m *BpVoteOperation) GetAffectedProps(props *map[string]bool) {
	(*props)["*"] = true
}

func init() {
	registerOperation("bp_vote", (*Operation_Op5)(nil), (*BpVoteOperation)(nil));
}
