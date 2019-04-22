package constants

const (
	COS_CHAIN_NAME     = "contentos"
	COS_INIT_SUPPLY    = 10000000000
	COS_CONSENSUS_NAME = "iBFT"
	COS_INIT_MINER     = "initminer"

	COIN_SYMBOL = "COS"
	VEST_SYMBOL = "VEST"

	BLOCK_INTERVAL        = 1  // 1000 ms for one block produce
	BLOCK_PROD_REPETITION = 10 // each producer produces 10 blocks in a row

	NOTICE_OP_PRE           = "oppre"
	NOTICE_OP_POST          = "oppost"
	NOTICE_TRX_PRE          = "trxpre"
	NOTICE_TRX_POST         = "trxpost"
	NOTICE_TRX_PENDING      = "trxpending"
	NOTICE_TRX_APLLY_RESULT = "trxapplyresult"
	NOTICE_BLOCK_APPLY      = "blockapply"

	ProducerNum = 21
	GenesisTime = 11111

	INIT_MINER_NAME      = "initminer"
	MAX_TRANSACTION_SIZE = 1024 * 256

	MaxBlockSize = 1024 * 1024 * 2
	MinBlockSize = 115

	GENESIS_TIME = 0

	INITMINER_PUBKEY = "COS5JVLLcTPhq4Unr194JzWPDNSYGoMcam8yxnsjgRVo3Nb7ioyFW"
	INITMINER_PRIKEY = "4DjYx2KAGh1NP3dai7MZTLUBMMhMBPmwouKE8jhVSESywccpVZ"

	RPC_PAGE_SIZE_LIMIT = 100

	MAX_WITNESSES = 21

	POST_INVALID_ID         = 0
	POST_MAX_DEPTH          = 8
	POST_CASHPUT_DELAY_TIME = 60 * 60 * 24 * 7
	MAX_BP_VOTE_COUNT       = 30

	BLOCKS_PER_DAY = 24 * 60 * 60 / BLOCK_INTERVAL

	MAX_UNDO_HISTORY = 10000

	MIN_VOTE_INTERVAL = 20
	MIN_POST_INTERVAL = 0 // 5 * 60 TODO for unit test

	PERCENT = 10000

	VOTE_REGENERATE_TIME = (60 * 60 * 24) * 3

	VOTE_LIMITE_DURING_REGENERATE = 30

	VP_DECAY_TIME = (60 * 60 * 24) * 15

	TRX_MAX_EXPIRATION_TIME = 30 * 60

	PER_BLOCK_CURRENT = 10

	AUTHOR_REWARD = 7000
	REPLY_REWARD  = 2000
	BP_REWARD     = 1000

	// 10 ** 18 ?
	BaseRate = 1

	// resource parameter
	LimitPrecision     = 1000 * 1000
	NetConsumePointNum = 10
	NetConsumePointDen = 1
	CpuConsumePointNum = 1
	CpuConsumePointDen = 100
	MaxGasPerCall      = 20000 * CpuConsumePointDen
	MaxStaminaPerBlock = 1000000
	WindowSize         = 60 * 60 * 24
	FreeStamina        = 100000
	OneDayStamina      = MaxStaminaPerBlock * WindowSize
	CommonOpGas        = 100
	StakeFreezeTime    = 60 * 60 * 24 * 3
)

var GlobalId int32 = 1
