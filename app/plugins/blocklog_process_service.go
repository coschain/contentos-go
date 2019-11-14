package plugins

import (
	"fmt"
	"github.com/coschain/contentos-go/common/constants"
	service_configs "github.com/coschain/contentos-go/iservices/service-configs"
	"github.com/coschain/contentos-go/node"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type Progress struct {
	ID 				uint64	`gorm:"primary_key;auto_increment"`
	Processor       string  `gorm:"index"`
	BlockHeight 	uint64
	FastForward     bool
	FinishAt 		time.Time
}

type BlockLogProcessService struct {
	sync.Mutex
	config *service_configs.DatabaseConfig
	logger *logrus.Logger
	db *gorm.DB
	ctx *node.ServiceContext
	fastForwardService node.Service
	syncForwardService node.Service
	processors map[string]IBlockLogProcessor
}

func NewBlockLogProcessService(ctx *node.ServiceContext, config *service_configs.DatabaseConfig, log *logrus.Logger) (*BlockLogProcessService, error) {
	return &BlockLogProcessService{ctx: ctx, logger: log, config: config}, nil
}

func (s *BlockLogProcessService) Start(node *node.Node) error {
	if err := s.initDatabase(); err != nil {
		return fmt.Errorf("invalid database: %s", err)
	}
	s.register("blocklog", NewBlockLogProcessor())
	s.register("iotrx", NewIOTrxProcessor())
	s.fastForwardService = NewFastForwardManagerService(s.logger, s.db, s.processors)
	s.syncForwardService = NewSyncForwardManagerService(s.logger, s.db, s.processors)
	go s.fastForwardService.Start(node)
	go s.syncForwardService.Start(node)
	return nil
}

func (s *BlockLogProcessService) Stop() error {
	_ = s.fastForwardService.Stop()
	_ = s.syncForwardService.Stop()
	if s.db != nil {
		_ = s.db.Close()
	}
	return nil
}

func (s *BlockLogProcessService) register(name string, processor IBlockLogProcessor) {
	s.processors[name] = processor
}

func (s *BlockLogProcessService) initDatabase() error {
	connStr := fmt.Sprintf("%s:%s@/%s?charset=utf8mb4&parseTime=True&loc=Local", s.config.User, s.config.Password, s.config.Db)
	if db, err := gorm.Open(s.config.Driver, connStr); err != nil {
		return err
	} else {
		s.db = db
	}
	if !s.db.HasTable(&Progress{}) {
		if err := s.db.CreateTable(&Progress{}).Error; err != nil {
			_ = s.db.Close()
			return err
		}
	}
	for k := range s.processors {
		progress := &Progress{}
		if s.db.Where(&Progress{Processor: k}).RecordNotFound() {
			progress.Processor = k
			progress.BlockHeight = 0
			progress.FastForward = true
			progress.FinishAt = time.Unix(constants.GenesisTime, 0)
			if err := s.db.Create(progress).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func init() {
	RegisterSQLTableNamePattern("progress")
}