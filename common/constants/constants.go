package constants

const (
	COSChainName     = "contentos"
	COSTokenDecimals = 1000000
	COSInitSupply    = 10000000000 * COSTokenDecimals
	COSConsensusName = "iBFT"
	COSInitMiner     = "initminer"

	CoinSymbol = "COS"
	VestSymbol = "VEST"

	BlockInterval       = 1  // 1000 ms for one block produce
	BlockProdRepetition = 10 // each producer produces 10 blocks in a row

	NoticeOpPre        = "oppre"
	NoticeOpPost       = "oppost"
	NoticeTrxPre       = "trxpre"
	NoticeTrxPost      = "trxpost"
	NoticeTrxPending   = "trxpending"
	NoticeTrxApplied   = "trxapplyresult"
	NoticeBlockApplied = "blockapply"
	NoticeAddTrx     = "addTrx"

	GenesisTime = 0

	MaxTransactionSize = 1024 * 256

	MaxBlockSize = 1024 * 1024 * 2
	MinBlockSize = 115

	InitminerPubKey  = "COS5JVLLcTPhq4Unr194JzWPDNSYGoMcam8yxnsjgRVo3Nb7ioyFW"
	InitminerPrivKey = "4DjYx2KAGh1NP3dai7MZTLUBMMhMBPmwouKE8jhVSESywccpVZ"

	RpcPageSizeLimit = 100

	MaxWitnessCount = 21

	PostInvalidId        = 0
	PostMaxDepth         = 8
	PostCashOutDelayTime = 60 * 60 * 24 * 7
	MaxBpVoteCount       = 30

	BlocksPerDay = 24 * 60 * 60 / BlockInterval

	MaxUndoHistory = 10000

	MinVoteInterval = 20
	MinPostInterval = 0 // 5 * 60 TODO for unit test

	PERCENT = 10000

	VoteRegenerateTime = (60 * 60 * 24) * 3

	VoteLimitDuringRegenerate = 30

	VpDecayTime = (60 * 60 * 24) * 15

	TrxMaxExpirationTime = 30 * 60

	PerBlockCurrent = 10

	RewardRateAuthor = 7000
	RewardRateReply  = 2000
	RewardRateBP     = 1000

	// 10 ** 18 ?
	BaseRate = 1

	// resource parameter
	LimitPrecision = 1000 * 1000
	NetConsumePointNum = 10
	NetConsumePointDen = 1
	CpuConsumePointNum = 1
	CpuConsumePointDen = 100
	MaxGasPerCall = 20000 * CpuConsumePointDen
	MaxStaminaPerBlock = 1000000
	WindowSize = 60 * 60 * 24
	FreeStamina = 100000
	OneDayStamina = MaxStaminaPerBlock * WindowSize
	CommonOpGas = 100
	StakeFreezeTime = 60 * 60 * 24 * 3
)

var GlobalId int32 = 1