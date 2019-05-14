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


//type BlockLogHeap struct {
//	Mu sync.RWMutex
//	Logs []*iservices.BlockLog
//}

type BlockLogHeap []*iservices.BlockLog

func (logHeap BlockLogHeap) Len() int {return len(logHeap)}

func (logHeap BlockLogHeap) Less(i, j int) bool {
	return logHeap[i].BlockHeight < logHeap[j].BlockHeight
}

func (logHeap BlockLogHeap) Swap(i, j int) {
	logHeap[i], logHeap[j] = logHeap[j], logHeap[i]
	logHeap[i].Index = i
	logHeap[j].Index = j
}

func (logHeap *BlockLogHeap) Push(x interface{}) {
	n := len(*logHeap)
	item := x.(*iservices.BlockLog)
	item.Index = n
	*logHeap = append(*logHeap, item)
}

func (logHeap *BlockLogHeap) Pop() interface{} {
	old := *logHeap
	n := len(old)
	item := old[n-1]
	item.Index = -1 // for safety
	*logHeap = old[0 : n-1]
	return item
}

type HeapProxy struct {
	Mu sync.RWMutex
	logHeap *BlockLogHeap
}

func (proxy *HeapProxy) Push(blockLog interface{}) {
	proxy.Mu.Lock()
	defer proxy.Mu.Unlock()
	heap.Push(proxy.logHeap, blockLog)
}

func (proxy *HeapProxy) Pop() interface{} {
	proxy.Mu.Lock()
	defer proxy.Mu.Unlock()
	return heap.Pop(proxy.logHeap)
}

func (proxy *HeapProxy) Len() int {
	proxy.Mu.RLock()
	defer proxy.Mu.RUnlock()
	return proxy.logHeap.Len()
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
	//logHeap BlockLogHeap
	proxy *HeapProxy
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

	var lastLib uint64 = 0

	logHeap := BlockLogHeap{}
	_ = s.db.QueryRow("select lib from stateloglibinfo limit 1").Scan(&lastLib)

	rows, _ := s.db.Query("SELECT block_log from statelog WHERE block_height >= ?", lastLib)
	for rows.Next() {
		var log interface{}
		var blockLog iservices.BlockLog
		if err := rows.Scan(&log); err != nil {
			s.log.Error(err)
			continue
		}
		data := log.([]byte)
		if err := json.Unmarshal(data, &blockLog); err != nil {
			s.log.Error(err)
			continue
		}
		//s.logHeap.Push(&blockLog)
		logHeap.Push(&blockLog)
	}
	s.log.Debugf("[statelog] heap db length: %d\n", logHeap.Len())

	// make logheap to heap
	heap.Init(&logHeap)
	proxy := &HeapProxy{logHeap: &logHeap}
	s.proxy = proxy

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
	s.log.Debugf("[statelog] heap length: %d\n", s.proxy.Len())
	for s.proxy.Len() > 0 {
		log := s.proxy.Pop()
		blockLog := log.(*iservices.BlockLog)
		s.log.Debugf("[statelog] blocklog: blockHeight:%d, blockId: %s, lib blockHeight:%d, lib blockId:%s \n",
			blockLog.BlockHeight, blockLog.BlockId, lib, blockId)
		// if the block log from heap > lib, re-push it else pop it
		if blockLog.BlockHeight > lib {
			//heap.Push(&s.logHeap, blockLog)
			s.proxy.Push(blockLog)
			break
		}
		if blockLog.BlockHeight == lib && blockLog.BlockId == blockId {
			s.handleLog(blockLog)
		}
	}
}

func (s *StateLogService) handleLog(blockLog *iservices.BlockLog) {
	blockId := blockLog.BlockId
	//blockHeight := blockLog.BlockHeight
	trxLogs := blockLog.TrxLogs
	for _, trxLog := range trxLogs {
		trxId := trxLog.TrxId
		opLogs := trxLog.OpLogs
		for _, opLog := range opLogs {
			action := opLog.Action
			property := opLog.Property
			target := opLog.Target
			result := opLog.Result
			switch property {
			case "balance":
				s.handleBalance(blockId, trxId, action, target, result)
			case "mint":
				s.handleMint(blockId, trxId, action, target, result)
			case "cashout":
				s.handleCashout(blockId, trxId, action, target, result)
			//default:
				//s.log.Errorf("Unknown property: %s\n", property)
			}
			//s.pushIntoDb(blockId, blockHeight, trxId, action, property, target, result)
			//data := make(map[string]interface{})
			//data[target] = result
			//jsonData, _ := json.Marshal(data)
			//_, err := s.db.Exec("INSERT INTO `statelog` (`block_id`, `block_height`, `trx_id`, `action`, `property`, `state`) " +
			//	"values (?, ?, ?, ?, ?, ?)", blockId, blockHeight, trxId, action, property, jsonData)
			//s.log.Debug("[statelog]", err)
		}
	}
}

func (s *StateLogService) handleBalance(blockId string, trxId string, action int, target string, result interface{}) {
	var resultValue uint64
	switch result.(type) {
	case uint64:
		resultValue = result.(uint64)
	case float64:
		resultValue = uint64(result.(float64))
	}
	switch action {
	case iservices.Replace:
		_, _ = s.db.Exec("REPLACE INTO stateaccount (account, balance) VALUES (?, ?)", target, resultValue)
	case iservices.Insert:
		_, _ = s.db.Exec("INSERT INTO stateaccount (account, balance) VALUES (?, ?)", target, resultValue)
	case iservices.Update:
		_, _ = s.db.Exec("UPDATE stateaccount set balance=? where account=?", resultValue, target)
	}
}

func (s *StateLogService) handleMint(blockId string, trxId string, action int, target string, result interface{}) {
	var resultValue uint64
	switch result.(type) {
	case uint64:
		resultValue = result.(uint64)
	case float64:
		resultValue = uint64(result.(float64))
	}
	switch action {
	case iservices.Add:
		var revenue uint64
		_ = s.db.QueryRow("SELECT revenue from statemint where bp=?", target).Scan(&revenue)
		_, _ = s.db.Exec("REPLACE INTO statemint (bp, revenue) VALUES (?, ?)", target, revenue + resultValue)
	}
}

func (s *StateLogService) handleCashout(blockId string, trxId string, action int, target string, result interface{}) {
	var resultValue uint64
	switch result.(type) {
	case uint64:
		resultValue = result.(uint64)
	case float64:
		resultValue = uint64(result.(float64))
	}
	switch action {
	case iservices.Add:
		var cashout uint64
		_ = s.db.QueryRow("select cashout from statecashout where account=?", target).Scan(&cashout)
		_, _ = s.db.Exec("REPLACE INTO statecashout (account, cashout) VALUES (?, ?)", target, cashout + resultValue)
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
	blockHeight := blockLog.BlockHeight
	blockLogJson, _ := json.Marshal(blockLog)
	_, _ = s.db.Exec("INSERT IGNORE INTO statelog (block_height, block_log) VALUES (?, ?)", blockHeight, blockLogJson)
	//heap.Push(&s.logHeap, blockLog)
	s.proxy.Push(blockLog)
}

func (s *StateLogService) stop() {
	s.ticker.Stop()
}

func (s *StateLogService) Stop() error {
	s.unhookEvent()
	return nil
}
