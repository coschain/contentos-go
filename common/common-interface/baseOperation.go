package commoninterface

import (
	"github.com/coschain/contentos-go/common/prototype"
	"github.com/coschain/contentos-go/common/type-proto"
)

type admin_type int

const (
	comment_delete admin_type = iota
	commercial
)

type AccountAdminPair struct {
	Name      prototype.AccountName
	AdminType admin_type
}

type BaseOperation interface {
	GetAuthorities(*[]prototype.Authority)
	GetRequiredActive(*map[prototype.AccountName]bool)
	GetRequiredPosting(*map[prototype.AccountName]bool)
	GetRequiredOwner(*map[prototype.AccountName]bool)
	GetAdmin(*[]AccountAdminPair)
	IsVirtual()

	Validate()
}
