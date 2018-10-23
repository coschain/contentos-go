package commoninterface

type admin_type int

const (
	comment_delete admin_type = iota
	commercial
)

type account_admin_pair struct {
	Name      prototype.Namex
	AdminType admin_type
}

type base_operation interface {
	/*	get_required_authorities(*[]prototype.Authority)
		get_required_active_authorities(*map[prototype.Namex]bool)
		get_required_posting_authorities(*map[prototype.Namex]bool)
		get_required_owner_authorities(*map[prototype.Namex]bool)
		get_required_admin(*[]account_admin_pair)
		is_virtual()
	*/
	validate()
}
