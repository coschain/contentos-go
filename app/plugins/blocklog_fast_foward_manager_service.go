package plugins

import (
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"sync/atomic"
	"time"
)

type FastForwardManagerService struct {
	*ForwardManagerService
}

func NewFastForwardManagerService(logger *logrus.Logger, db *gorm.DB, processors map[string]IBlockLogProcessor) *FastForwardManagerService {
	 return &FastForwardManagerService{
		NewForwardManagerService(logger, db, processors),
	}
}

func (s *FastForwardManagerService) Start(node *node.Node) error  {
	s.scheduleNextJob()
	return nil
}

func (s *FastForwardManagerService) Stop() error  {
	s.waitWorkDone()
	if s.db != nil {
		_ = s.db.Close()
	}
	s.db, s.stop, s.working = nil, 0, 0
	return nil
}

func (s *FastForwardManagerService) scheduleNextJob() {
	s.jobTimer = time.AfterFunc(1*time.Second, s.work)
}

func (s *FastForwardManagerService) work() {
	const maxJobSize = 1000
	var (
		userBreak = false
		err       error
	)
	atomic.StoreInt32(&s.working, 1)

	progresses := s.progressesOfNeedFastProcessors()

	for _, progress := range progresses {
		if atomic.LoadInt32(&s.stop) != 0 {
			userBreak = true
			break
		}
		processor, ok := s.mainProcessors[progress.Processor]
		if !ok {
			continue
		}
		minBlockNum, maxBlockNum := progress.BlockHeight+1, progress.BlockHeight+maxJobSize
		for blockNum := minBlockNum; blockNum <= maxBlockNum; blockNum++ {
			if atomic.LoadInt32(&s.stop) != 0 {
				userBreak = true
				break
			}
			blockLogRec := &iservices.BlockLogRecord{BlockHeight: blockNum}
			if s.db.Where(&iservices.BlockLogRecord{BlockHeight: blockNum, Final: true}).First(blockLogRec).RecordNotFound() {
				break
			}
			blockLog := new(blocklog.BlockLog)
			if err = blockLog.FromJsonString(blockLogRec.JsonLog); err != nil {
				break
			}
			tx := s.db.Begin()
			userBreak, err = s.processLog(tx, blockLog, processor)
			if !userBreak && err == nil {
				progress.BlockHeight = blockNum
				progress.FinishAt = time.Now()
				if err = tx.Save(progress).Error; err == nil {
					tx.Commit()
				} else {
					s.logger.Errorf("save service progress failed and rolled back, error: %v", err)
					tx.Rollback()
					break
				}
			} else {
				s.logger.Errorf("process log failed and rolled back, error: %v", err)
				tx.Rollback()
				break
			}
		}
		if err := s.tryToTransferProcessorManager(progress); err != nil {
			s.logger.Errorf("try to transfer processor manager error: %v", err)
		}
	}
	s.Lock()
	atomic.StoreInt32(&s.working, 0)
	if !userBreak {
		if s.hasNeedSyncProcessors() {
			s.scheduleNextJob()
		}
	}
	s.workStop.Signal()
	s.Unlock()
}

func (s *FastForwardManagerService) hasNeedSyncProcessors() bool {
	progress := &iservices.Progress{}
	syncStatus := iservices.FastForwardStatus
	notFound := s.db.Where(&iservices.Progress{SyncStatus:&syncStatus}).First(progress).RecordNotFound()
	return !notFound
}

func (s *FastForwardManagerService) progressesOfNeedFastProcessors() []*iservices.Progress {
	var progresses []*iservices.Progress
	syncStatus := iservices.FastForwardStatus
	s.db.Where(&iservices.Progress{SyncStatus:&syncStatus}).Find(&progresses)
	return progresses
}

func (s *FastForwardManagerService) findLastBlockLog() *iservices.BlockLogRecord {
	tableIndex := uint64(0)
	for {
		blockHeight := tableIndex * iservices.BlockLogSplitSizeInMillion * 1000000
		if s.db.HasTable(&iservices.BlockLogRecord{BlockHeight: blockHeight}) {
			tableIndex ++
		} else {
			tableIndex --
			break
		}
	}
	last := &iservices.BlockLogRecord{}
	blockLogRecord := &iservices.BlockLogRecord{BlockHeight: tableIndex * iservices.BlockLogSplitSizeInMillion * 1000000}
	tableName := blockLogRecord.TableName()
	s.db.Table(tableName).Last(last)
	return last
}

func (s *FastForwardManagerService) tryToTransferProcessorManager(progress *iservices.Progress) error {
	if atomic.LoadInt32(&s.stop) != 0 {
		return nil
	}
	lastBlockLog := s.findLastBlockLog()
	if *progress.SyncStatus == iservices.FastForwardStatus {
		if lastBlockLog.BlockHeight - progress.BlockHeight < uint64(iservices.ThresholdForFastConvertToSync) {
			syncStatus := iservices.MiddleStatus
			progress.SyncStatus = &syncStatus
			tx := s.db.Begin()
			if err := tx.Save(progress).Error; err == nil {
				tx.Commit()
			} else {
				tx.Rollback()
				return err
			}
		}
	}
	return nil
}

