package constants

const (
	COS_CHAIN_NAME     = "contentos"
	COS_INIT_SUPPLY    = 10000000000
	COS_CONSENSUS_NAME = "iBFT"
	COS_INIT_MINER     = "initminer"

	COIN_SYMBOL = "COS"
	VEST_SYMBOL = "VEST"

	BLOCK_INTERNAL = 3 // 3000 ms for one block produce

	NOTICE_OP_PRE      = "oppre"
	NOTICE_OP_POST     = "oppost"
	NOTICE_TRX_PRE     = "trxpre"
	NOTICE_TRX_POST    = "trxpost"
	NOTICE_TRX_PENDING = "trxpending"
	NOTICE_BLOCK_APPLY = "blockapply"

 	ProducerNum = 21
	GenesisTime = 11111

	INIT_MINER_NAME = "initminer"
	MAX_TRANSACTION_SIZE = 1024 * 256
	MAX_BLOCK_SIZE = MAX_TRANSACTION_SIZE * BLOCK_INTERNAL * 2000
	GENESIS_TIME = 0
	INIT_SUPPLY = 10000000000
)
