package blocklog

const (
	AccountBalance = "account_balance"
)

type CoinChange struct {
	Name string			`json:"name"`
	Before uint64		`json:"before"`
	After uint64		`json:"after"`
}
