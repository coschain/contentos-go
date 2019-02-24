package prototype


type BaseOperation interface {
	GetRequiredOwner(*map[string]bool)
	Validate() error
}
