package plugins

import (
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

type SyncForwardManagerService struct {
	*ForwardManagerService
}

func NewSyncForwardManagerService(logger *logrus.Logger, db *gorm.DB, processors map[string]IBlockLogProcessor) *SyncForwardManagerService {
	return &SyncForwardManagerService{
		&ForwardManagerService{
			logger:logger,
			db:db,
			mainProcessors:processors,
			point: &SyncForwardMangerCheckpoint{db:db},
		},
	}
}

type SyncForwardMangerCheckpoint struct {
	db *gorm.DB
}

func (cp SyncForwardMangerCheckpoint) HasNeedSyncProcessors() bool {
	return true
}

func (cp SyncForwardMangerCheckpoint) ProgressesOfNeedSyncProcessors() []*Progress {
	var progresses []*Progress
	cp.db.Where(&Progress{FastForward:false}).Find(&progresses)
	return progresses
}

func (cp SyncForwardMangerCheckpoint) TryToTransferProcessorManager(progress *Progress) error {
	return nil
}
