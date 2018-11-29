package constants

const (
	COS_CHAIN_NAME     = "contentos"
	COS_INIT_SUPPLY    = 10000000000
	COS_CONSENSUS_NAME = "iBFT"
	COS_INIT_MINER     = "initminer"

	COIN_SYMBOL = "COS"
	VEST_SYMBOL = "VEST"

	BLOCK_INTERVAL = 3 // 3000 ms for one block produce

	NOTICE_OP_PRE      = "oppre"
	NOTICE_OP_POST     = "oppost"
	NOTICE_TRX_PRE     = "trxpre"
	NOTICE_TRX_POST    = "trxpost"
	NOTICE_TRX_PENDING = "trxpending"
	NOTICE_BLOCK_APPLY = "blockapply"

	NOTICE_HANDLE_P2P_SIGTRX = "p2p_get_sigtrx" // handle function need one parameter *prototype.SignedTransaction
	NOTICE_HANDLE_P2P_SIGBLK = "p2p_get_sigblk" // handle function need two parameters 1.*peer.Peer 2.*prototype.SignedBlock

	ProducerNum = 21
	GenesisTime = 11111

	INIT_MINER_NAME      = "initminer"
	MAX_TRANSACTION_SIZE = 1024 * 256

	MAX_BLOCK_SIZE = MAX_TRANSACTION_SIZE * BLOCK_INTERVAL * 2000
	MIN_BLOCK_SIZE = 115

	GENESIS_TIME = 0
	INIT_SUPPLY  = 10000000000

	INITMINER_PUBKEY = "COS6oKUcS7jNfPk48SEHENfeHbkWWjH7QAJt6C5tzGyL46yTWWBBv"
	INITMINER_PRIKEY = "27Pah3aJ8XbaQxgU1jxmYdUzWaBbBbbxLbZ9whSH9Zc8GbPMhw"

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
)
