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
)

