package plugins

import (
	"container/heap"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/iservices/service-configs"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/prototype"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)


type BlockLogHeap struct {
	Mu sync.RWMutex
	Logs []*iservices.BlockLog
}

func (logHeap BlockLogHeap) Len() int {return len(logHeap.Logs)}

func (logHeap BlockLogHeap) Less(i, j int) bool {
	return logHeap.Logs[i].BlockHeight < logHeap.Logs[j].BlockHeight
}

func (logHeap BlockLogHeap) Swap(i, j int) {
	logHeap.Logs[i], logHeap.Logs[j] = logHeap.Logs[j], logHeap.Logs[j]
	logHeap.Logs[i].Index = i
	logHeap.Logs[j].Index = j
}

func (logHeap *BlockLogHeap) Push(x interface{}) {
	logHeap.Mu.RLock()
	defer logHeap.Mu.RUnlock()
	n := len(logHeap.Logs)
	item := x.(*iservices.BlockLog)
	item.Index = n
	logHeap.Logs = append(logHeap.Logs, item)
}

func (logHeap *BlockLogHeap) Pop() interface{} {
	logHeap.Mu.RLock()
	defer logHeap.Mu.RUnlock()
	old := logHeap.Logs
	n := len(old)
	item := old[n-1]
	item.Index = -1 // for safety
	logHeap.Logs = old[0 : n-1]
	return item
}

var StateLogServiceName = "statelogsrv"

type StateLogService struct {
	node.Service
	config *service_configs.DatabaseConfig
	consensus iservices.IConsensus
	ev  EventBus.Bus
	ctx *node.ServiceContext
	quit chan bool
	log *logrus.Logger
	logHeap BlockLogHeap
	db *sql.DB
	ticker *time.Ticker
}

// service constructor
func NewStateLogService(ctx *node.ServiceContext, config *service_configs.DatabaseConfig, log *logrus.Logger) (*StateLogService, error) {
	return &StateLogService{ctx:ctx, config:config, log:log}, nil
}

func (s *StateLogService) Start(node *node.Node) error {
	s.quit = make(chan bool)
	// dns: data source name
	consensus, err := s.ctx.Service(iservices.ConsensusServerName)
	if err != nil {
		return err
	}
	s.consensus = consensus.(iservices.IConsensus)
	dsn := fmt.Sprintf("%s:%s@/%s", s.config.User, s.config.Password, s.config.Db)
	db, err := sql.Open(s.config.Driver, dsn)
	s.db = db

	s.ev = node.EvBus
	heap.Init(&s.logHeap)
	s.hookEvent()
	s.ticker = time.NewTicker(time.Second)
	go func() {
		for {
			select {
			case <- s.ticker.C:
				if err := s.pollLIB(); err != nil {
					s.log.Error(err)
				}
			case <- s.quit:
				s.stop()
				return
			}
		}
	}()
	return nil
}

func (s *StateLogService) pollLIB() error {
	lib := s.consensus.GetLIB().BlockNum()
	var lastLib uint64 = 0
	_ = s.db.QueryRow("SELECT lib from stateloglibinfo limit 1").Scan(&lastLib)
	var waitingSyncLib []uint64
	var count = 0
	for lastLib < lib {
		if count > 1000 {
			break
		}
		waitingSyncLib = append(waitingSyncLib, lastLib)
		lastLib ++
		count ++
	}
	for _, block := range waitingSyncLib {
		s.handleLibNotification(block)
		utcTimestamp := time.Now().UTC().Unix()
		_, _ = s.db.Exec("UPDATE stateloglibinfo SET lib=?, last_check_time=?", block, utcTimestamp)
	}
	return nil
}

func (s *StateLogService) handleLibNotification(lib uint64) {
	blks , err := s.consensus.FetchBlocks(lib, lib)
	if err != nil {
		s.log.Error(err)
		return
	}
	if len(blks) == 0 {
		return
	}
	blk := blks[0].(*prototype.SignedBlock)
	data := blk.Id().Data
	blockId := hex.EncodeToString(data[:])
	for s.logHeap.Len() > 0 {
		log := s.logHeap.Pop()
		blockLog := log.(*iservices.BlockLog)
		// if the block log from heap > lib, re-push it else pop it
		if blockLog.BlockHeight > lib {
			s.logHeap.Push(blockLog)
			break
		}
		if blockLog.BlockHeight == lib && blockLog.BlockId == blockId {
			s.pushIntoDb(blockLog)
		}
	}
}

func (s *StateLogService) pushIntoDb(blockLog *iservices.BlockLog) {
	blockId := blockLog.BlockId
	blockHeight := blockLog.BlockHeight
	trxLogs := blockLog.TrxLogs
	for _, trxLog := range trxLogs {
		trxId := trxLog.TrxId
		opLogs := trxLog.OpLogs
		for _, opLog := range opLogs {
			action := opLog.Action
			property := opLog.Property
			target := opLog.Target
			result := opLog.Result
			data := make(map[string]interface{})
			data[target] = result
			jsonData, _ := json.Marshal(data)
			_, _ = s.db.Exec("INSERT INTO `statelog` (`block_id`, `block_height`, `trx_id`, `action`, `property`, `state`) " +
				"values (?, ?, ?, ?, ?, ?)", blockId, blockHeight, trxId, action, property, jsonData)
		}
	}
}

func (s *StateLogService) hookEvent() {
	_ = s.ev.Subscribe(constants.NoticeState, s.onStateLogOperation)
}
func (s *StateLogService) unhookEvent() {
	_ = s.ev.Unsubscribe(constants.NoticeState, s.onStateLogOperation)
}

func (s *StateLogService) onStateLogOperation(blockLog *iservices.BlockLog) {
	//s.log.Debug("statelog", blockLog.BlockHeight, blockLog.BlockId, blockLog.Index, len(blockLog.TrxLogs))
	heap.Push(&s.logHeap, blockLog)
}

func (s *StateLogService) stop() {
	s.ticker.Stop()
}

func (s *StateLogService) Stop() error {
	s.unhookEvent()
	return nil
}
