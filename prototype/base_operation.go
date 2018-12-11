package prototype

type admin_type int

const (
	comment_delete admin_type = iota
	commercial
)

type AccountAdminPair struct {
	Name      AccountName
	AdminType admin_type
}

type BaseOperation interface {
	GetAuthorities(*[]Authority)
	GetRequiredOwner(*map[string]bool)
	GetAdmin(*[]AccountAdminPair)
	IsVirtual()

	Validate() error
}
