package plugins

import (
	"database/sql"
	"fmt"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/iservices/itype"
	"github.com/coschain/contentos-go/iservices/service-configs"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/prototype"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

type TokenInfoService struct {
	node.Service
	config *service_configs.DatabaseConfig
	db *sql.DB
	log *logrus.Logger
	ctx *node.ServiceContext
	ticker *time.Ticker
	quit chan bool
}

func NewTokenInfoService(ctx *node.ServiceContext, config *service_configs.DatabaseConfig, log *logrus.Logger) (*TokenInfoService, error) {
	return &TokenInfoService{ctx: ctx, log: log, config: config}, nil
}

func (s *TokenInfoService) Start(node *node.Node) error {
	s.quit = make(chan bool)
	// dns: data source name
	dsn := fmt.Sprintf("%s:%s@/%s", s.config.User, s.config.Password, s.config.Db)
	db, err := sql.Open(s.config.Driver, dsn)

	if err != nil {
		return err
	}
	s.db = db

	s.ticker = time.NewTicker(time.Second)
	go func() {
		for {
			select {
			case <- s.ticker.C:
				if err := s.pollStateLog(); err != nil {
					s.log.Error(err)
				}
			case <- s.quit:
				s.stop()
				return
			}
		}
	}()
	return nil
}

func (s *TokenInfoService) pollStateLog() error  {
	var lib uint64
	_ = s.db.QueryRow("select lib from tokenlibinfo limit 1").Scan(&lib)
	//lib := s.consensus.GetLIB().BlockNum()
	//s.log.Debugf("[trx db] sync lib: %d \n", lib)
	var lastLib uint64 = 0
	var lastDate string
	_ = s.outDb.QueryRow("SELECT lib, date from dailystatinfo limit 1").Scan(&lastLib, &lastDate)
	var waitingSyncLib []uint64
	var count = 0
	for lastLib < lib {
		if count > 1000 {
			break
		}
		waitingSyncLib = append(waitingSyncLib, lastLib)
		lastLib ++
		count ++
	}
	for _, block := range waitingSyncLib {
		blks , err := s.consensus.FetchBlocks(block, block)
		if err != nil {
			s.log.Error(err)
			continue
		}
		if len(blks) == 0 {
			if block != 0 {
				s.log.Errorf("cannot fetch block %d in consensus", block)
			}
			continue
		}
		blk := blks[0].(*prototype.SignedBlock)
		blockTime := blk.Timestamp()
		datetime := time.Unix(int64(blockTime), 0)
		date := fmt.Sprintf("%d-%02d-%02d", datetime.Year(), datetime.Month(), datetime.Day())
		// trigger
		if lastDate != date {
			s.handleDailyStatistic(blk, lastDate)
			//s.handleDNUStatistic(blk, lastDate)
			s.log.Debugf("[daily stat] trigger handle, timestamp: %d, datetime: %s", blockTime, date)
			lastDate = date
		}
		utcTimestamp := time.Now().UTC().Unix()
		_, _ = s.outDb.Exec("UPDATE dailystatinfo SET lib = ?, date = ?, last_check_time = ?", block, date, utcTimestamp)
	}
	return nil
}

func (s *TokenInfoService) stop() {
	s.ticker.Stop()
}


func (t *TokenInfoService) Stop() error {
	t.quit <- true
	close(t.quit)
	return nil
}

