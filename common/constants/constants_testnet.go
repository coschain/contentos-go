// +build testnet

package constants

const (
	PostCashOutDelayBlock 	= 60*60*24
	VoteCashOutDelayBlock = PostCashOutDelayBlock
	VoteRegenerateTime 		= 60*60*24
	PowerDownBlockInterval 	= 60*60*24
	MinEpochDuration 		= 60*60*24

	StakeFreezeTime      	= 60*60*24
	WindowSize           = 60 * 60 * 24

	PerTicketPrice = 1
	PerTicketPriceStr = "1.000000"
	PerTicketWeight = uint64(1e7)

	ClientName              = "Cos-go-testnet"

	// vest delegation
	VestDelegationDeliveryInBlocks = 3 * 60 / BlockInterval	// 3 minutes
)

// hard forks
const (
	Original uint64 = 0
	HardFork1 uint64 = 1375000
	HardFork2 uint64 = 9734951
	HardFork3 uint64 = 14834555
	HardFork4 uint64 = 14835555
)
