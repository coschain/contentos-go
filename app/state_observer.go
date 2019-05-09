package app

import (
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/sirupsen/logrus"
)

type StateObserver struct {
	blockNum uint64
	blockId string
	trxLogs []iservices.TrxLog
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
		s.log.Debugf("[statelog] trxlog: observer: blockNum %d, %v\n", s.blockNum, s.trxLogs)
		s.noticer.Publish(constants.NoticeState, &iservices.BlockLog{BlockHeight: s.blockNum, BlockId: blockId, TrxLogs: s.trxLogs})
		s.trxLogs = nil
	}
}

// should a reference counter be introduced ?
func (s *StateObserver) Notify(log *iservices.TrxLog) {
	s.trxLogs = append(s.trxLogs, *log)
}

type TrxLogger struct {
	observer *StateObserver
	trxId string
	opLogs []iservices.OpLog
}

func (t *TrxLogger) BeginTrx(trxId string) {
	t.trxId = trxId
}

func (t *TrxLogger) AddOpState(action int, property string, target string, result interface{}) {
	opLog := iservices.OpLog{Action: action, Property: property, Target: target, Result: result}
	t.opLogs = append(t.opLogs, opLog)
	//s.log.Debugf("[statelog] trxlog: observer: AddOpState, %v", )
	t.observer.log.Debugf("[statelog] trxlog: observer: AddOpState, %v", opLog)
}

func (t *TrxLogger) EndTrx(keep bool) {
	if keep {
		trxLog := &iservices.TrxLog{TrxId: t.trxId, OpLogs: t.opLogs}
		t.observer.Notify(trxLog)
	}
}