package plugins

import (
	"fmt"
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/iservices/service-configs"
	"github.com/coschain/contentos-go/node"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/sirupsen/logrus"
	"time"
)

var BlockLogTable = &iservices.BlockLogRecord{}

type BlockLogService struct {
	ctx *node.ServiceContext
	config *service_configs.DatabaseConfig
	logger *logrus.Logger
	bus EventBus.Bus
	db *gorm.DB
}

func NewBlockLogService(ctx *node.ServiceContext, config *service_configs.DatabaseConfig, logger *logrus.Logger) (*BlockLogService, error) {
	return &BlockLogService{ctx:ctx, config:config, logger:logger}, nil
}

func (s *BlockLogService) Start(node *node.Node) error  {
	if err := s.initDatabase(); err != nil {
		return err
	}
	s.bus = node.EvBus
	_ = s.bus.Subscribe(constants.NoticeBlockLog, s.onBlockLog)
	_ = s.bus.Subscribe(constants.NoticeLibChange, s.onLibChange)
	return nil
}

func (s *BlockLogService) Stop() error {
	_ = s.bus.Unsubscribe(constants.NoticeState, s.onBlockLog)
	_ = s.bus.Unsubscribe(constants.NoticeLibChange, s.onLibChange)
	_ = s.db.Close()
	return nil
}

func (s *BlockLogService) initDatabase() error {
	connStr := fmt.Sprintf("%s:%s@/%s?charset=utf8&parseTime=True&loc=Local", s.config.User, s.config.Password, s.config.Db)
	if db, err := gorm.Open("mysql", connStr); err != nil {
		return err
	} else {
		s.db = db
	}
	if !s.db.HasTable(BlockLogTable) {
		s.db.CreateTable(BlockLogTable)
	}
	return nil
}

func (s *BlockLogService) onBlockLog(blockLog *blocklog.BlockLog) {
	s.db.Create(&iservices.BlockLogRecord{
		BlockId:     blockLog.BlockId,
		BlockHeight: blockLog.BlockNum,
		BlockTime:   time.Unix(int64(blockLog.BlockTime), 0),
		Final:       false,
		JsonLog:     blockLog.ToJsonString(),
	})
}

func (s *BlockLogService) onLibChange(blocks []common.ISignedBlock) {
	if count := len(blocks); count > 0 {
		blockIds := make([]string, count)
		for i , block := range blocks {
			blockIds[i] = fmt.Sprintf("%x", block.Id().Data)
		}
		s.db.Table(iservices.BlockLogDBTableName).Where("block_id IN (?)", blockIds).Update("final", true)
	}
}
