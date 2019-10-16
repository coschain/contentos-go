// +build tests

package constants

const (
	PostCashOutDelayBlock 	= 60*5
	VoteCashOutDelayBlock = PostCashOutDelayBlock
	VoteRegenerateTime 		= 10000
	PowerDownBlockInterval 	= 100
	MinEpochDuration 		= 60*5
	StakeFreezeTime      	= 60*5
	WindowSize              = 7000

	PerTicketPrice = 1
	PerTicketPriceStr = "1.000000"
	PerTicketWeight = uint64(1e7)

	ClientName              = "Cos-go-tests"
)

// hard forks
const (
	Original uint64 = 0
	HardFork1 uint64 = 1000
)
