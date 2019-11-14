package plugins

import (
	"github.com/coschain/contentos-go/iservices"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

type FastForwardManagerService struct {
	*ForwardManagerService
}

func NewFastForwardManagerService(logger *logrus.Logger, db *gorm.DB, processors map[string]IBlockLogProcessor) *FastForwardManagerService {
	return &FastForwardManagerService{
		&ForwardManagerService{
			logger:logger,
			db:db,
			mainProcessors:processors,
			point: &FastForwardManagerCheckpoint{db: db},
		},
	}
}

type FastForwardManagerCheckpoint struct {
	db *gorm.DB
}

func (cp FastForwardManagerCheckpoint) HasNeedSyncProcessors() bool {
	return cp.db.Where(&Progress{FastForward:true}).RecordNotFound()
}

func (cp FastForwardManagerCheckpoint) ProgressesOfNeedSyncProcessors() []*Progress {
	var progresses []*Progress
	cp.db.Where(&Progress{FastForward:true}).Find(&progresses)
	return progresses
}

func (cp FastForwardManagerCheckpoint) TryToTransferProcessorManager(progress *Progress) error {
	blogLog := &iservices.BlockLogRecord{}
	cp.db.Last(blogLog)
	if progress.FastForward == true {
		if blogLog.BlockHeight - progress.BlockHeight < 1000 {
			progress.FastForward = false
			tx := cp.db.Begin()
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

