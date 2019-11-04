package plugins

import (
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/iservices/itype"
	"github.com/coschain/contentos-go/iservices/service-configs"
	"github.com/coschain/contentos-go/node"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"sync"
	"sync/atomic"
	"time"
)

const LIMIT = 30

type dateRow map[uint64]*itype.Row

type DailyStatisticService struct {
	sync.Mutex
	config *service_configs.DatabaseConfig
	log *logrus.Logger
	db *gorm.DB
	jobTimer *time.Timer
	stop int32
	working int32
	workStop *sync.Cond
	dappsCache map[string]dateRow
	dappWithCreator map[string]string
}

func NewDailyStatisticService(ctx *node.ServiceContext, config *service_configs.DatabaseConfig, log *logrus.Logger) (*DailyStatisticService, error) {
	s := &DailyStatisticService{log: log, config: config}
	s.workStop = sync.NewCond(&s.Mutex)
	return s, nil
}

func (s *DailyStatisticService) Start(node *node.Node) error {
	connStr := fmt.Sprintf("%s:%s@/%s?charset=utf8mb4&parseTime=True&loc=Local", s.config.User, s.config.Password, s.config.Db)
	if db, err := gorm.Open(s.config.Driver, connStr); err != nil {
		return err
	} else {
		s.db = db
	}
	// hard code
	if !s.db.HasTable("trxinfo") {
		return errors.New("need init trxinfo table before start this plugin")
	}
	s.dappWithCreator = map[string]string{"contentos": "costvtycoon", "photogrid": "photogrid"}
	s.dappsCache = map[string]dateRow{}
	for dapp, _ := range s.dappWithCreator {
		s.dappsCache[dapp] = dateRow{}
	}
	s.scheduleNextJob()
	return nil
}

func (s *DailyStatisticService) Stop() error {
	s.waitWorkDone()
	if s.db != nil {
		_ = s.db.Close()
	}
	s.db, s.stop, s.working = nil, 0, 0
	return nil
}

func (s *DailyStatisticService) Reload(config *node.Config) error {
	return nil
}

func (s *DailyStatisticService) scheduleNextJob() {
	s.jobTimer = time.AfterFunc(24 * time.Hour, s.cron)
}

func (s *DailyStatisticService) waitWorkDone() {
	s.Lock()
	if s.jobTimer != nil {
		s.jobTimer.Stop()
	}
	atomic.StoreInt32(&s.stop, 1)
	for atomic.LoadInt32(&s.working) != 0 {
		s.workStop.Wait()
	}
	s.Unlock()
}

func (s *DailyStatisticService) cron() {
	atomic.StoreInt32(&s.working, 1)
	today := time.Now().UTC()
	yesterday := today.Add(-24 * time.Hour)
	yesterdayStr := yesterday.Format("2006-01-02")
	for dapp, _ := range s.dappWithCreator {
		if atomic.LoadInt32(&s.stop) != 0 {
			break
		}
		if row, err := s.make(dapp, yesterdayStr); err == nil {
			timestamp := row.Timestamp
			s.dappsCache[dapp][timestamp] = row
		}
	}
	s.Lock()
	atomic.StoreInt32(&s.working, 0)
	if atomic.LoadInt32(&s.stop) == 0 {
		s.scheduleNextJob()
	}
	s.workStop.Signal()
	s.Unlock()
}

func (s *DailyStatisticService) printErr(errs []error) {
	if len(errs) > 0 {
		s.log.Error("daily statistic:", errs)
	}
}

func (s *DailyStatisticService) make(dapp, datetime string) (*itype.Row, error) {
	s.Lock()
	defer s.Unlock()
	creator, ok := s.dappWithCreator[dapp]
	if !ok {
		return nil, errors.New(fmt.Sprintf("dapp %s is not exist", dapp))
	}
	startDate, err := time.Parse("2006-01-02", datetime)
	if err != nil {
		return nil, err
	}
	start := startDate.Unix()
	end := startDate.Add(24 * time.Hour).Unix()

	var dau uint32
	dauErr := s.db.Table("create_user_records").Where("create_user_records.creator = ?", creator).
		Joins("JOIN trxinfo on trxinfo.creator = create_user_records.new_account").
		Where("trxinfo.block_time > ? AND trxinfo.block_time < ?", start, end).
		Select("count(distinct(create_user_records.new_account))").
		Count(&dau).GetErrors()
	s.printErr(dauErr)

	var dnu uint32
	dnuErr := s.db.Model(&CreateUserRecord{}).Where("block_time > ? AND block_time < ? AND creator = ?", time.Unix(start, 0), time.Unix(end, 0), creator).Count(&dnu).GetErrors()
	s.printErr(dnuErr)

	var count uint32
	countErr := s.db.Table("create_user_records").Where("create_user_records.creator = ?", creator).
		Joins("JOIN trxinfo on trxinfo.creator = create_user_records.new_account").
		Where("trxinfo.block_time > ? AND trxinfo.block_time < ?", start, end).
		Count(&count).GetErrors()
	s.printErr(countErr)

	type AmountWrap struct {
		Amount uint64
	}
	var amount AmountWrap
	amountErr := s.db.Table("transfer_records").Select("sum(transfer_records.amount) as amount").
		Where("transfer_records.block_time > ? AND transfer_records.block_time < ?", time.Unix(start, 0), time.Unix(end, 0)).
		Joins("JOIN create_user_records on transfer_records.from = create_user_records.new_account").
		Where("create_user_records.creator = ?", creator).Scan(&amount).GetErrors()
	s.printErr(amountErr)

	var total uint32
	totalErr := s.db.Model(&CreateUserRecord{}).Where("creator = ? and block_time < ?", creator, time.Unix(end, 0)).Count(&total).GetErrors()
	s.printErr(totalErr)

	row := &itype.Row{Timestamp: uint64(start), Dapp: dapp, Dau: dau, Dnu: dnu, TrxCount: count, Amount:amount.Amount, TotalUserCount:total}
	return row, nil
}

func (s *DailyStatisticService) checkLimit(days int) int {
	if days > LIMIT {
		return LIMIT
	} else {
		return days
	}
}

func (s *DailyStatisticService) DailyStatsSince(days int, dapp string) []*itype.Row {
	var rows []*itype.Row
	days = s.checkLimit(days)
	if _, ok := s.dappWithCreator[dapp]; !ok {
		return rows
	}
	dappRows, _ := s.dappsCache[dapp]
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	d, _ := time.ParseDuration("-24h")
	start := today.Add(d * time.Duration(days))
	for start.Unix() < today.Unix() {
		timestamp := uint64(start.Unix())
		if row, ok := dappRows[timestamp]; ok {
			rows = append(rows, row)
		} else {
			tm := time.Unix(int64(timestamp), 0)
			date := tm.Format("2006-01-02")
			row, err := s.make(dapp, date)
			if err == nil {
				dappRows[timestamp] = row
				rows = append(rows, row)
			}
		}
		start = start.Add(24 * time.Hour)
	}
	return rows
}
