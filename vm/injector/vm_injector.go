package vminjector

import (
	"github.com/coschain/contentos-go/app/blocklog"
	vmcache "github.com/coschain/contentos-go/vm/cache"
	"github.com/go-interpreter/wagon/exec"
)

type Injector interface {
	Error(code uint32, msg string)
	Log(msg string)
	RequireAuth(name string) error
	RecordStaminaFee(caller string, spent uint64)
	GetVmRemainCpuStamina(name string) uint64
	// only panic, no error return
	TransferFromContractToUser(contract, owner, to string, amount uint64)
	TransferFromContractToUserVest(contract, owner, to string, amount uint64)
	TransferFromUserToContract(from, contract, owner string, amount uint64)
	TransferFromContractToContract(fromContract, fromOwner, toContract, toOwner string, amount uint64)
	ContractCall(caller, fromOwner, fromContract, fromMethod, toOwner, toContract, toMethod string, params []byte, coins, remainGas uint64,preVm *exec.VM)
	ContractABI(owner, contract string) string
	GetBlockProducers() []string
	DiscardAccountCache(name string)
	VmCache() *vmcache.VmCache
	StateChangeContext() *blocklog.StateChangeContext
	NewRecordID() uint64
	CurrentRecordID() uint64
	HardFork() uint64
}
