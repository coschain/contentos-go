package prototype


type BaseOperation interface {
	GetSigner(*map[string]bool)
	Validate() error
	GetAffectedProps(props *map[string]bool)
}
