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

type SyncForwardManagerService struct {
	*ForwardManagerService
}

func NewSyncForwardManagerService(logger *logrus.Logger, db *gorm.DB, processors map[string]IBlockLogProcessor) *SyncForwardManagerService {
	return &SyncForwardManagerService{
		NewForwardManagerService(logger, db, processors),
	}
}

func (s *SyncForwardManagerService) Start(node *node.Node) error  {
	s.scheduleNextJob()
	return nil
}

func (s *SyncForwardManagerService) Stop() error  {
	s.waitWorkDone()
	if s.db != nil {
		_ = s.db.Close()
	}
	s.db, s.stop, s.working = nil, 0, 0
	return nil
}

func (s *SyncForwardManagerService) scheduleNextJob() {
	s.jobTimer = time.AfterFunc(1*time.Second, s.work)
}

func (s *SyncForwardManagerService) work() {
	const maxJobSize = 1000
	var (
		userBreak = false
		interrupt = false
		err       error
		progresses []*iservices.Progress
		tx *gorm.DB
	)
	atomic.StoreInt32(&s.working, 1)

	var currentSyncProgress iservices.Progress

	// for sync forward, it must be every processor in a same progress.
	syncStatus := iservices.SyncForwardStatus
	s.db.Where(&iservices.Progress{SyncStatus: &syncStatus}).First(&currentSyncProgress)

	progresses = s.progressesOfNeedCatchUpSyncProgress()

	// first, pull middle progresses into as same as sync-progress
	tx = s.db.Begin()
	for _, progress := range progresses {
		// when sync progress is empty, using the first middle
		// just set it to sync
		if currentSyncProgress.BlockHeight == 0 {
			currentSyncProgress.BlockHeight = progress.BlockHeight
			currentSyncProgressStatus := iservices.SyncForwardStatus
			progress.SyncStatus = &currentSyncProgressStatus
			err = tx.Save(progress).Error
		}
		// different processor maybe have different progress
		if progress.BlockHeight + 1 < currentSyncProgress.BlockHeight {
			minBlockNum, maxBlockNum := progress.BlockHeight+1, currentSyncProgress.BlockHeight
			for blockNum := minBlockNum; blockNum <= maxBlockNum; blockNum++ {
				userBreak, _, err = s.processProcessors(tx, []*iservices.Progress{progress}, blockNum)
				if userBreak || err != nil {
					break
				}
			}
			if !userBreak && err == nil {
				progressSyncStatus := iservices.SyncForwardStatus
				progress.SyncStatus = &progressSyncStatus
				err = tx.Save(progress).Error
			} else {
				break
			}
		}
	}
	if !userBreak && err == nil {
		tx.Commit()
	} else {
		tx.Rollback()
	}

	progresses = s.progressesOfNeedSyncProcessors()

	tx = s.db.Begin()
	minBlockNum, maxBlockNum := currentSyncProgress.BlockHeight+1, currentSyncProgress.BlockHeight+maxJobSize
	for blockNum := minBlockNum; blockNum <= maxBlockNum; blockNum++ {
		userBreak, interrupt, err = s.processProcessors(tx, progresses, blockNum)
		if userBreak || err != nil || interrupt {
			break
		}
	}
	if !userBreak && err == nil {
		tx.Commit()
	} else {
		tx.Rollback()
	}
	s.Lock()
	atomic.StoreInt32(&s.working, 0)
	if !userBreak {
		s.scheduleNextJob()
	}
	s.workStop.Signal()
	s.Unlock()
}

// userBreak: user stop the program, drop any changes in the trx
// interrupt: a processor meet the end edge of block log record, it should be the first processor.
// prevent later processors to process newer block log record to keep global consistent. It isn't an error.
func (s *SyncForwardManagerService) processProcessors(tx *gorm.DB, progresses []*iservices.Progress, blockNum uint64) (userBreak bool, interrupt bool, err error){
	if atomic.LoadInt32(&s.stop) != 0 {
		userBreak = true
		return
	}
	for _, progress := range progresses {
		processor, exist := s.mainProcessors[progress.Processor];
		if !exist {
			continue
		}
		if atomic.LoadInt32(&s.stop) != 0 {
			userBreak = true
			break
		}
		blockLogRec := &iservices.BlockLogRecord{BlockHeight: blockNum}
		if tx.Where(&iservices.BlockLogRecord{BlockHeight: blockNum, Final: true}).First(blockLogRec).RecordNotFound() {
			interrupt = true
			break
		}
		blockLog := new(blocklog.BlockLog)
		if err = blockLog.FromJsonString(blockLogRec.JsonLog); err != nil {
			break
		}
		userBreak, err = s.processLog(tx, blockLog, processor)
		if !userBreak && err == nil {
			progress.BlockHeight = blockNum
			progress.FinishAt = time.Now()
			if err = tx.Save(progress).Error; err != nil {
				s.logger.Errorf("save service progress failed and rolled back, error: %v", err)
				break
			}
		} else {
			s.logger.Errorf("process log failed and rolled back, error: %v", err)
			break
		}
		// only for compatible, it's hard code
		if progress.Processor == "blocklog" {
			deprecatedProgress := &iservices.DeprecatedBlockLogProgress{}
			if tx.HasTable(deprecatedProgress) {
				if !tx.First(deprecatedProgress).RecordNotFound() {
					deprecatedProgress.BlockHeight = blockNum
					deprecatedProgress.FinishAt = time.Now()
					if err = tx.Save(deprecatedProgress).Error; err != nil {
						s.logger.Errorf("save blocklog progress failed and rolled back, error: %v", err)
						break
					}
				}
			}
		}
	}
	return
}

func (s *SyncForwardManagerService) hasNeedSyncProcessors() bool {
	progress := &iservices.Progress{}
	syncStatus := iservices.SyncForwardStatus
	notFound := s.db.Where(&iservices.Progress{SyncStatus:&syncStatus}).First(progress).RecordNotFound()
	return !notFound
}

func (s *SyncForwardManagerService) progressesOfNeedSyncProcessors() []*iservices.Progress {
	var progresses []*iservices.Progress
	syncStatus := iservices.SyncForwardStatus
	s.db.Where(&iservices.Progress{SyncStatus:&syncStatus}).Find(&progresses)
	return progresses
}

func (s *SyncForwardManagerService) progressesOfNeedCatchUpSyncProgress() []*iservices.Progress {
	var progresses []*iservices.Progress
	middleStatus := iservices.MiddleStatus
	s.db.Where(&iservices.Progress{SyncStatus: &middleStatus}).Find(&progresses)
	return progresses
}
