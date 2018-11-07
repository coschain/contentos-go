// Package cos implements the Contentos protocol.
package cos

import (
	//"errors"
	"fmt"
	//"math/big"
	//"runtime"
	"sync"
	//"sync/atomic"

	//"github.com/ethereum/go-ethereum/accounts"
	"github.com/coschain/contentos-go/p2p/depend/common"

	//"github.com/ethereum/go-ethereum/core/vm"
	"github.com/coschain/contentos-go/cos/downloader"
	//"github.com/ethereum/go-ethereum/ethdb"
	"github.com/coschain/contentos-go/p2p/depend/event"
	//log "github.com/inconshreveable/log15"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/p2p"
	//"github.com/ethereum/go-ethereum/params"
	//"github.com/coschain/contentos-go/p2p/depend/rlp"
)

/*
type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	Protocols() []p2p.Protocol
	SetBloomBitsIndexer(bbIndexer *core.ChainIndexer)
}
*/

// Contentos implements the Contentos full node service.
type Contentos struct {
	config      *Config
	//chainConfig *params.ChainConfig

	// Channel for shutting down the service
	shutdownChan chan bool // Channel for shutting down the Contentos

	// Handlers
	//txPool          *core.TxPool
	//blockchain      *core.BlockChain
	protocolManager *ProtocolManager
	//lesServer       LesServer

	// DB interfaces
	//chainDb ethdb.Database // Block chain database

	eventMux       *event.TypeMux
	//engine         consensus.Engine

	etherbase common.Address

	networkID     uint64

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and etherbase)
}

/*
func (s *Contentos) AddLesServer(ls LesServer) {
	s.lesServer = ls
	ls.SetBloomBitsIndexer(s.bloomIndexer)
}
*/

// New creates a new Contentos object (including the
// initialisation of the common Contentos object)
func New(ctx *node.ServiceContext) (*Contentos, error) {
	//if config.SyncMode == downloader.LightSync {
	//	return nil, errors.New("can't run eth.Contentos in light sync mode, use les.LightEthereum")
	//}
	//if !config.SyncMode.IsValid() {
	//	return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	//}

	/*
	chainDb, err := CreateDB(ctx, config, "chaindata")
	if err != nil {
		return nil, err
	}
	*/

	//chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	/*
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)
	*/

	eth := &Contentos{
		//config:         config,
		//chainDb:        chainDb,
		//chainConfig:    chainConfig,
		//eventMux:       ctx.EventMux,
		//engine:         CreateConsensusEngine(ctx, chainConfig, &config.Ethash, config.MinerNotify, chainDb),
		shutdownChan:   make(chan bool),
		//networkID:      config.NetworkId,
		//etherbase:      config.Etherbase,
	}

	//log.Info("Initialising Contentos protocol", "versions", ProtocolVersions, "network", config.NetworkId)

	/*
	if !config.SkipBcVersionCheck {
		bcVersion := rawdb.ReadDatabaseVersion(chainDb)
		if bcVersion != core.BlockChainVersion && bcVersion != 0 {
			return nil, fmt.Errorf("Blockchain DB version mismatch (%d / %d). Run geth upgradedb.\n", bcVersion, core.BlockChainVersion)
		}
		rawdb.WriteDatabaseVersion(chainDb, core.BlockChainVersion)
	}
	*/

	/*
	var (
		vmConfig    = vm.Config{EnablePreimageRecording: config.EnablePreimageRecording}
		cacheConfig = &core.CacheConfig{Disabled: config.NoPruning, TrieNodeLimit: config.TrieCache, TrieTimeLimit: config.TrieTimeout}
	)
	eth.blockchain, err = core.NewBlockChain(chainDb, cacheConfig, eth.chainConfig, eth.engine, vmConfig)

	if err != nil {
		return nil, err
	}
	*/

	// Rewind the chain in case of an incompatible config upgrade.
	/*
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		eth.blockchain.SetHead(compat.RewindTo)
		rawdb.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}
	eth.bloomIndexer.Start(eth.blockchain)
	*/

	//if config.TxPool.Journal != "" {
	//	config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	//}
	//eth.txPool = core.NewTxPool(config.TxPool, eth.chainConfig, eth.blockchain)

	//if eth.protocolManager, err = NewProtocolManager(eth.chainConfig, config.SyncMode, config.NetworkId, eth.eventMux, eth.txPool, eth.engine, eth.blockchain, chainDb); err != nil {
	//	return nil, err
	//}

	/*
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	*/

	return eth, nil
}

/*
func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"geth",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		log.Warn("Miner extra data exceed limit", "extra", hexutil.Bytes(extra), "limit", params.MaximumExtraDataSize)
		extra = nil
	}
	return extra
}
*/

// CreateDB creates the chain database.
/*
func CreateDB(ctx *node.ServiceContext, config *Config, name string) (ethdb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := db.(*ethdb.LDBDatabase); ok {
		db.Meter("eth/db/chaindata/")
	}
	return db, nil
}
*/

//func (s *Contentos) ResetWithGenesisBlock(gb *prototype.SignedBlock) {
//	s.blockchain.ResetWithGenesisBlock(gb)
//}

func (s *Contentos) Etherbase() (eb common.Address, err error) {
	s.lock.RLock()
	etherbase := s.etherbase
	s.lock.RUnlock()

	if etherbase != (common.Address{}) {
		return etherbase, nil
	}
	/*
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			etherbase := accounts[0].Address

			s.lock.Lock()
			s.etherbase = etherbase
			s.lock.Unlock()

			log.Info("Etherbase automatically configured", "address", etherbase)
			return etherbase, nil
		}
	}
	*/
	return common.Address{}, fmt.Errorf("etherbase must be explicitly specified")
}


func (s *Contentos) EventMux() *event.TypeMux           { return s.eventMux }
//func (s *Contentos) ChainDb() ethdb.Database            { return s.chainDb }
func (s *Contentos) IsListening() bool                  { return true } // Always listening
func (s *Contentos) EthVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *Contentos) NetVersion() uint64                 { return s.networkID }
func (s *Contentos) Downloader() *downloader.Downloader { return s.protocolManager.downloader }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *Contentos) Protocols() []p2p.Protocol {
	//if s.lesServer == nil {
		return s.protocolManager.SubProtocols
	//}
	//return append(s.protocolManager.SubProtocols, s.lesServer.Protocols()...)
}

// Start implements node.Service, starting all internal goroutines needed by the
// Contentos protocol implementation.
func (s *Contentos) Start(srvr *p2p.Server) error {

	// Figure out a max peers count based on the server limits
	maxPeers := srvr.MaxPeers
	if s.config.LightServ > 0 {
		if s.config.LightPeers >= srvr.MaxPeers {
			return fmt.Errorf("invalid peer config: light peer count (%d) >= total peer count (%d)", s.config.LightPeers, srvr.MaxPeers)
		}
		maxPeers -= s.config.LightPeers
	}
	// Start the networking layer and the light server if requested
	s.protocolManager.Start(maxPeers)

	/*
	if s.lesServer != nil {
		s.lesServer.Start(srvr)
	}
	*/

	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// Contentos protocol.
func (s *Contentos) Stop() error {
	s.protocolManager.Stop()

	/*
	if s.lesServer != nil {
		s.lesServer.Stop()
	}
	*/

	//s.txPool.Stop()
	s.eventMux.Stop()

	//s.chainDb.Close()
	close(s.shutdownChan)
	return nil
}
