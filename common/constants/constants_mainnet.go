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

	ClientName              = "Cos-go-mainnet"
)

