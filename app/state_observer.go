package app

import (
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/common/constants"
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
}

type StateObserver struct {
	blockNum uint64
	blockId string
	trxLogs []*TrxLog
	noticer EventBus.Bus
	log *logrus.Logger
}

func NewStateObserver(noticer EventBus.Bus, log *logrus.Logger) *StateObserver {
	return &StateObserver{noticer: noticer, log: log}
}

func (s *StateObserver) BeginBlock(blockNum uint64) {
	s.blockNum = blockNum
}

func (s *StateObserver) NewTrxObserver() *TrxLogger{
	return &TrxLogger{observer: s}
}

func (s *StateObserver) EndBlock(blockId string) {
	if len(blockId) > 0 {
		s.noticer.Publish(constants.NoticeState, &BlockLog{BlockHeight: s.blockNum, BlockId: blockId, TrxLogs: s.trxLogs})
		s.trxLogs = nil
	}
}

// should a reference counter be introduced ?
func (s *StateObserver) Notify(log *TrxLog) {
	s.trxLogs = append(s.trxLogs, log)
}

type TrxLogger struct {
	observer *StateObserver
	trxId string
	opLogs []*OpLog
}

func (t *TrxLogger) BeginTrx(trxId string) {
	t.trxId = trxId
}

func (t *TrxLogger) AddOpState(action int, property string, target string, result interface{}) {
	opLog := &OpLog{Action: action, Property: property, Target: target, Result: result}
	t.opLogs = append(t.opLogs, opLog)
}

func (t *TrxLogger) EndTrx(keep bool) {
	if keep {
		trxLog := &TrxLog{TrxId: t.trxId, OpLogs: t.opLogs}
		t.observer.Notify(trxLog)
	}
}