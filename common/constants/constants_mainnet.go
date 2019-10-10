// +build !testnet,!devnet,!tests

package constants

const (
	PostCashOutDelayBlock 	= 60 * 60 * 24 * 7
	VoteCashOutDelayBlock = PostCashOutDelayBlock
	VoteRegenerateTime 		= 60 * 60 * 24
	PowerDownBlockInterval 	= (60 * 60 * 24) * 7
	MinEpochDuration 		= 60 * 60 * 24

	StakeFreezeTime      	= 60 * 60 * 24 * 3
	WindowSize           = 60 * 60 * 24

	PerTicketPrice = 10000000
	PerTicketPriceStr = "10000000.000000"
	PerTicketWeight = uint64(1)

	ClientName              = "Cos-go-mainnet"
)

// hard forks
const (
	Original uint64 = 0
	HardFork1 uint64 = 1900000
)
