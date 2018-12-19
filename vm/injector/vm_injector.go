package vminjector

type RequireAuth func(name string)
type Transfer func(from, to string, amount uint64, memo string)

type Injector interface {
	RequireAuth(name string) error
	ContractTransfer(contract, owner, to string, amount uint64) error
}
