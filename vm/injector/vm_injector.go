package vminjector

type Log func(msg string)
type Error func(code uint32, msg string)
type RequireAuth func(name string)
type Transfer func(from, to string, amount uint64, memo string)

type Injector interface {
	Error(code uint32, msg string)
	Log(msg string)
	RequireAuth(name string) error
	ContractTransfer(contract, owner, to string, amount uint64) error
}
