package plugins

import (
	"container/heap"
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/iservices/service-configs"
	"github.com/coschain/contentos-go/node"
	"github.com/sirupsen/logrus"
)

type OpLog struct {
	Action int
	Property string
	Target string
	Result interface{}
}

type TrxLog struct {
	TrxId string
	OpLogs []*OpLog
}

type BlockLog struct {
	BlockHeight uint64
	BlockId string
	TrxLogs []*TrxLog
	Index int // the index of item in the heap
}

type BlockLogHeap []*BlockLog

func (logHeap BlockLogHeap) Len() int {return len(logHeap)}

func (logHeap BlockLogHeap) Less(i, j int) bool {
	return logHeap[i].BlockHeight < logHeap[j].BlockHeight
}

func (logHeap BlockLogHeap) Swap(i, j int) {
	logHeap[i], logHeap[j] = logHeap[j], logHeap[j]
	logHeap[i].Index = i
	logHeap[j].Index = j
}

func (logHeap *BlockLogHeap) Push(x interface{}) {
	n := len(*logHeap)
	item := x.(*BlockLog)
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

	s.ev = node.EvBus
	heap.Init(&s.logHeap)
	s.hookEvent()
	return nil
}

func (s *StateLogService) hookEvent() {
	err := s.ev.Subscribe(constants.NoticeState, s.onStateLogOperation)
	s.log.Error(err)
}
func (s *StateLogService) unhookEvent() {
	err := s.ev.Unsubscribe(constants.NoticeState, s.onStateLogOperation)
	s.log.Error(err)
}

func (s *StateLogService) onStateLogOperation(blockLog *BlockLog) {
	s.log.Debug("statelog", blockLog.BlockHeight, blockLog.BlockId, blockLog.Index, len(blockLog.TrxLogs))
	heap.Push(&s.logHeap, blockLog)
	s.log.Debug("statelog", "length", s.logHeap.Len())
}

func (s *StateLogService) Stop() error {
	s.unhookEvent()
	return nil
}
