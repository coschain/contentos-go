package vminjector

type Injector interface {
	Error(code uint32, msg string)
	Log(msg string)
	RequireAuth(name string) error
	DeductGasFee(caller string, spent uint64)
	// only panic, no error return
	TransferFromContractToUser(contract, owner, to string, amount uint64)
	TransferFromUserToContract(from, contract, owner string, amount uint64)
	TransferFromContractToContract(fromContract, fromOwner, toContract, toOwner string, amount uint64)
	ContractCall(caller, owner, contract, method string, params []byte, coins, maxGas uint64)
}
