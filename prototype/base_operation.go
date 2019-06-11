package prototype

import "errors"

type BaseOperation interface {
	GetSigner(*map[string]bool)
	Validate() error

	// GetAffectedProps sets affected properties into given map.
	// e.g. TransferOperation should set 2 properties, names of sender and receiver.
	// "*" is the wildcard, meaning all properties.
	GetAffectedProps(props *map[string]bool)
}

type unknownOp struct {}

func (u unknownOp) GetSigner(auth *map[string]bool) {

}

func (u unknownOp) Validate() error {
	return errors.New("try to validate an unknown operation")
}

func (u unknownOp) GetAffectedProps(props *map[string]bool) {

}

var UnknownOperation unknownOp
