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
)

type BlockLogHeap []*iservices.BlockLog

func (logHeap BlockLogHeap) Len() int {return len(logHeap)}

func (logHeap BlockLogHeap) Less(i, j int) bool {
	return logHeap[i].BlockHeight < logHeap[j].BlockHeight
}

func (logHeap BlockLogHeap) Swap(i, j int) {
	logHeap[i], logHeap[j] = logHeap[j], logHeap[i]
}

func (logHeap *BlockLogHeap) Push(x interface{}) {
	*logHeap = append(*logHeap, x.(*iservices.BlockLog))
}

func (logHeap *BlockLogHeap) Pop() interface{} {
	old := *logHeap
	n := len(old)
	item := old[n-1]
	*logHeap = old[0 : n-1]
	return item
}

var StateLogServiceName = "statelogservice"

type StateLogService struct {
	sync.Mutex								// lock for block logs
	node.Service
	ctx *node.ServiceContext				// not used
	config *service_configs.DatabaseConfig	// for sql dsn config
	ev  EventBus.Bus						// for events (un)subscription
	logger *logrus.Logger					// logger
	blockLogs BlockLogHeap					// block logs
	db *sql.DB								// the sql database
}

// service constructor
func NewStateLogService(ctx *node.ServiceContext, config *service_configs.DatabaseConfig, log *logrus.Logger) (*StateLogService, error) {
	return &StateLogService{ctx:ctx, config:config, logger:log}, nil
}

func (s *StateLogService) Start(node *node.Node) error {
	// dns: data source name
	dsn := fmt.Sprintf("%s:%s@/%s", s.config.User, s.config.Password, s.config.Db)
	db, err := sql.Open(s.config.Driver, dsn)
	if err != nil {
		s.logger.Errorf("start statelog service failed. Database can't be connected")
		return err
	}
	s.db = db
	s.ev = node.EvBus
	heap.Init(&s.blockLogs)

	s.hookEvent()
	return nil
}

func (s *StateLogService) pushLog(blockLog *iservices.BlockLog, isPicked bool) {
	blockLogJson, _ := json.Marshal(blockLog.TrxLogs)
	_, err := s.db.Exec("INSERT IGNORE INTO `statelog` (`block_id`, `block_height`, `block_time`, `pick`, `block_log`) " +
		"VALUES (?, ?, ?, ?, ?)", blockLog.BlockId, blockLog.BlockHeight, blockLog.BlockTime, isPicked, blockLogJson)
	if err != nil {
		s.logger.Error(err)
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
	if len(blocks) == 0 {
		return
	}
	// get ids & numbers of committed blocks
	commitBlockIds := make(map[string]bool)
	commitBlock := uint64(0)
	for _ , block := range blocks {
		blkId := (block.(*prototype.SignedBlock)).Id()
		blkNum := blkId.BlockNum()
		commitBlockIds[hex.EncodeToString(blkId.Data[:])] = true
		if commitBlock < blkNum {
			commitBlock = blkNum
		}
	}
	// get block logs that can be dumped to database
	s.Lock()
	logs := make([]*iservices.BlockLog, 0, s.blockLogs.Len())
	for s.blockLogs.Len() > 0 {
		blockLog := s.blockLogs.Pop().(*iservices.BlockLog)
		if blockLog.BlockHeight > commitBlock {
			s.blockLogs.Push(blockLog)
			break
		}
		logs = append(logs, blockLog)
	}
	s.Unlock()
	// dump blocks to database
	for _, blockLog := range logs {
		s.pushLog(blockLog, commitBlockIds[blockLog.BlockId])
	}
}

func (s *StateLogService) onStateLogOperation(blockLog *iservices.BlockLog) {
	s.Lock()
	s.blockLogs.Push(blockLog)
	s.Unlock()
}


func (s *StateLogService) Stop() error {
	s.unhookEvent()
	return nil
}
