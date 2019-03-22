package prototype


type BaseOperation interface {
	GetSigner(*map[string]bool)
	Validate() error

	// GetAffectedProps sets affected properties into given map.
	// e.g. TransferOperation should set 2 properties, names of sender and receiver.
	// "*" is the wildcard, meaning all properties.
	GetAffectedProps(props *map[string]bool)
}
