package plugins

import (
	"container/heap"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/iservices/service-configs"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/prototype"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

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
	Heap *BlockLogHeap
}

func (proxy *HeapProxy) Push(blockLog interface{}) {
	proxy.Mu.Lock()
	defer proxy.Mu.Unlock()
	heap.Push(proxy.Heap, blockLog)
}

func (proxy *HeapProxy) Pop() interface{} {
	proxy.Mu.Lock()
	defer proxy.Mu.Unlock()
	return heap.Pop(proxy.Heap)
}

func (proxy *HeapProxy) Len() int {
	proxy.Mu.RLock()
	defer proxy.Mu.RUnlock()
	return proxy.Heap.Len()
}

var StateLogServiceName = "statelogservice"

type StateLogService struct {
	node.Service
	config *service_configs.DatabaseConfig
	ev  EventBus.Bus
	ctx *node.ServiceContext
	quit chan bool
	log *logrus.Logger
	// for block log
	logHeapProxy *HeapProxy
	// for receiving last irreversible blocks when onLibChange
	libHeapProxy *HeapProxy
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
	dsn := fmt.Sprintf("%s:%s@/%s", s.config.User, s.config.Password, s.config.Db)
	db, err := sql.Open(s.config.Driver, dsn)
	if err != nil {
		s.log.Errorf("start statelog service failed. Database can't be connected")
		return err
	}
	s.db = db

	s.ev = node.EvBus

	logHeap := BlockLogHeap{}
	heap.Init(&logHeap)
	s.logHeapProxy = &HeapProxy{Heap: &logHeap}

	libHeap := BlockLogHeap{}
	heap.Init(&libHeap)
	s.libHeapProxy = &HeapProxy{Heap: &libHeap}

	s.hookEvent()
	s.ticker = time.NewTicker(time.Second)
	go func() {
		for {
			select {
			case <- s.ticker.C:
				if err := s.pollLIBHeap(); err != nil {
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

func (s *StateLogService) pollLIBHeap() error {
	for s.libHeapProxy.Len() > 0 {
		log := s.libHeapProxy.Pop()
		libLog := log.(*iservices.BlockLog)
		blockHeight := libLog.BlockHeight
		blockId := libLog.BlockId
		s.handleLib(blockHeight , blockId)
	}
	return nil
}

func (s *StateLogService) handleLib(lib uint64, blockId string) {
	s.log.Debugf("[statelog] heap length: %d\n", s.logHeapProxy.Len())
	for s.logHeapProxy.Len() > 0 {
		log := s.logHeapProxy.Pop()
		blockLog := log.(*iservices.BlockLog)
		s.log.Debugf("[statelog] blocklog: blockHeight:%d, blockId: %s, lib blockHeight:%d, lib blockId:%s \n",
			blockLog.BlockHeight, blockLog.BlockId, lib, blockId)
		// if the block log from heap > lib, re-push it else pop it
		if blockLog.BlockHeight > lib {
			//heap.Push(&s.logHeap, blockLog)
			s.logHeapProxy.Push(blockLog)
			break
		}
		// really ??
		if blockLog.BlockHeight < lib {
			s.pushLog(blockLog, false)
		}
		if blockLog.BlockHeight == lib {
			if blockLog.BlockId == blockId {
				s.pushLog(blockLog, true)
			} else {
				s.pushLog(blockLog, false)
			}
		}
	}
}

func (s *StateLogService) pushLog(blockLog *iservices.BlockLog, isPicked bool) {
	blockLogJson, _ := json.Marshal(blockLog.TrxLogs)
	_, err := s.db.Exec("INSERT IGNORE INTO `statelog` (`block_id`, `block_height`, `block_time`, `pick`, `block_log`) " +
		"VALUES (?, ?, ?, ?, ?)", blockLog.BlockId, blockLog.BlockHeight, blockLog.BlockTime, isPicked, blockLogJson)
	if err != nil {
		s.log.Error(err)
	}
}

func (s *StateLogService) hookEvent() {
	_ = s.ev.Subscribe(constants.NoticeState, s.onStateLogOperation)
	_ = s.ev.Subscribe(constants.NoticeLibChange, s.onLibChange)
}

func (s *StateLogService) unhookEvent() {
	_ = s.ev.Unsubscribe(constants.NoticeState, s.onStateLogOperation)
	_ = s.ev.Unsubscribe(constants.NoticeLibChange, s.onLibChange)
}

func (s *StateLogService) onLibChange(blocks []common.ISignedBlock) {
	for _ , block := range blocks {
		blk := block.(*prototype.SignedBlock)
		data := blk.Id().Data
		blockId := hex.EncodeToString(data[:])
		blockHeight := blk.GetSignedHeader().Number()
		libLog := &iservices.BlockLog{BlockHeight:blockHeight, BlockId:blockId}
		s.libHeapProxy.Push(libLog)
	}
}

func (s *StateLogService) onStateLogOperation(blockLog *iservices.BlockLog) {
	s.logHeapProxy.Push(blockLog)
}

func (s *StateLogService) stop() {
	s.ticker.Stop()
}

func (s *StateLogService) Stop() error {
	s.unhookEvent()
	return nil
}
