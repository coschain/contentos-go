package vminjector

type Injector interface {
	Error(code uint32, msg string)
	Log(msg string)
	RequireAuth(name string) error
	DeductGasFee(caller string, spent uint64)
	// only panic, no error return
	ContractTransfer(contract, owner, to string, amount uint64)
	UserTransfer(from, contract, owner string, amount uint64)
}
