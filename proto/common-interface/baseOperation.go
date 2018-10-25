package commoninterface

import "contentos-go/proto/type-proto"

type admin_type int

const (
	comment_delete admin_type = iota
	commercial
)

type account_admin_pair struct {
	Name      prototype.AccountName
	AdminType admin_type
}

type BaseOperation interface {
	/*	get_required_authorities(*[]prototype.Authority)
		get_required_active_authorities(*map[prototype.Namex]bool)
		get_required_posting_authorities(*map[prototype.Namex]bool)
		get_required_owner_authorities(*map[prototype.Namex]bool)
		get_required_admin(*[]account_admin_pair)
		is_virtual()
	*/
	Validate()
}
