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

//type Row map[string]int

const INTERVAL = 60 * 60
// for test easily
//const INTERVAL = 60 * 5

type DailyStatisticService struct {
	node.Service
	config *service_configs.DatabaseConfig
	consensus iservices.IConsensus
	outDb *sql.DB
	log *logrus.Logger
	ctx *node.ServiceContext
	ticker *time.Ticker
	quit chan bool
}

func NewDailyStatisticService(ctx *node.ServiceContext, config *service_configs.DatabaseConfig, log *logrus.Logger) (*DailyStatisticService, error) {
	return &DailyStatisticService{ctx: ctx, log: log, config: config}, nil
}

func (s *DailyStatisticService) Start(node *node.Node) error {
	s.quit = make(chan bool)
	// dns: data source name
	consensus, err := s.ctx.Service(iservices.ConsensusServerName)
	if err != nil {
		return err
	}
	s.consensus = consensus.(iservices.IConsensus)
	dsn := fmt.Sprintf("%s:%s@/%s", s.config.User, s.config.Password, s.config.Db)
	outDb, err := sql.Open(s.config.Driver, dsn)

	if err != nil {
		return err
	}
	s.outDb = outDb

	s.ticker = time.NewTicker(time.Second)
	go func() {
		for {
			select {
			case <- s.ticker.C:
				if err := s.pollLIB(); err != nil {
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

func (s *DailyStatisticService) pollLIB() error  {
	lib := s.consensus.GetLIB().BlockNum()
	s.log.Debugf("[trx db] sync lib: %d \n", lib)
	var lastLib uint64 = 0
	var lastDate string
	_ = s.outDb.QueryRow("SELECT lib, date from dailystatinfo limit 1").Scan(&lastLib, &lastDate)
	var waitingSyncLib []uint64
	var count = 0
	for lastLib < lib {
		if count > 100 {
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

func (s *DailyStatisticService) handleDailyStatistic(block *prototype.SignedBlock, lastDate string) {
	blockTime := block.Timestamp()
	//datetime := time.Unix(int64(blockTime), 0)
	//date := fmt.Sprintf("%d-%02d-%02d", datetime.Year(), datetime.Month(), datetime.Day())
	end := blockTime
	start := end - 86400
	dauRows, _ := s.outDb.Query("SELECT distinct creator FROM trxinfo WHERE block_time >= ? and block_time < ?", start, end)
	dapps := make(map[string]string)
	dappRows, _ := s.outDb.Query("select dapp, prefix from dailystatdapp where status=1")
	for dappRows.Next() {
		var dapp, prefix string
		_ = dappRows.Scan(&dapp, &prefix)
		dapps[prefix] = dapp
	}
	// dau
	dappsDauCounter := make(map[string]uint32)
	for dauRows.Next() {
		var creator string
		if err := dauRows.Scan(&creator); err != nil {
			s.log.Error(err)
			continue
		}
		for prefix, dapp := range dapps {
			if strings.HasPrefix(creator, prefix) {
				dappsDauCounter[dapp] += 1
			}
		}
	}
	dnuRows, _:= s.outDb.Query("SELECT account FROM createaccountinfo WHERE create_time >= ? and create_time < ?", start, end)
	dappsDnuCounter := make(map[string]uint32)
	// dnu
	for dnuRows.Next() {
		var account string
		if err := dnuRows.Scan(&account); err != nil {
			s.log.Error(err)
			continue
		}
		for prefix, dapp := range dapps {
			if strings.HasPrefix(account, prefix) {
				dappsDnuCounter[dapp] += 1
			}
		}
	}
	// trx count
	trxCountRows, _ := s.outDb.Query("select creator, count(creator) as count from trxinfo where block_time >= ? and block_time < ? group by creator", start, end)
	trxCountCounter := make(map[string]uint32)
	for trxCountRows.Next() {
		var creator string
		var count uint32
		if err := trxCountRows.Scan(&creator, &count); err != nil {
			s.log.Error(err)
			continue
		}
		for prefix, dapp := range dapps {
			if strings.HasPrefix(creator, prefix) {
				trxCountCounter[dapp] += count
			}
		}
	}
	// amount
	transferAmountRows, _ := s.outDb.Query("select sender, amount from transferinfo where create_time >= ? and create_time < ?", start, end)
	transferAmountCounter := make(map[string]uint64)
	for transferAmountRows.Next() {
		var sender string
		var amount uint64
		if err := transferAmountRows.Scan(&sender, &amount); err != nil {
			s.log.Error(err)
			continue
		}
		for prefix, dapp := range dapps {
			if strings.HasPrefix(sender, prefix) {
				transferAmountCounter[dapp] += amount
			}
		}
	}
	for _, dapp := range dapps {
		dau, _ := dappsDauCounter[dapp]
		dnu, _ := dappsDnuCounter[dapp]
		trxCount, _ := trxCountCounter[dapp]
		amount, _ := transferAmountCounter[dapp]
		if len(lastDate) > 0 {
			_, _ = s.outDb.Exec("insert ignore into dailystat (date, dapp, dau, dnu, trxs, amount) values (?, ?, ?, ?, ?, ?)", lastDate, dapp, dau, dnu, trxCount, amount)
		}
	}
}


func (s *DailyStatisticService) DailyStatsOn(date string, dapp string) *itype.Row {
	var dau, dnu, trxs uint32
	var amount uint64
	_ = s.outDb.QueryRow("select dau, dnu, trxs, amount from dailystat where date=? and dapp=?", date, dapp).Scan(&dau, &dnu, &trxs, &amount)
	return &itype.Row{Date: date, Dapp: dapp, Dau: dau, Dnu: dnu, TrxCount: trxs, Amount: amount}
}

func (s *DailyStatisticService) DailyStatsSince(days int, dapp string) []*itype.Row {
	now := time.Now().UTC()
	d, _ := time.ParseDuration("-24h")
	then := now.Add(d * time.Duration(days))
	start := fmt.Sprintf("%d-%02d-%02d", then.Year(), then.Month(), then.Day())
	var dauRows []*itype.Row
	rows, err := s.outDb.Query("select dau, dnu, trxs, amount, date from dailystat where date >= ? and dapp = ?  order by date", start, dapp)
	if err != nil {
		return dauRows
	}
	for rows.Next() {
		var dau, dnu, trxs uint32
		var amount uint64
		var date string
		_ = rows.Scan(&dau, &dnu, &trxs, &amount, &date)
		r := &itype.Row{Date: date, Dapp: dapp, Dau: dau, Dnu: dnu, TrxCount: trxs, Amount: amount}
		dauRows = append(dauRows, r)
	}
	return dauRows
}

func (s *DailyStatisticService) stop() {
	_ = s.outDb.Close()
	s.ticker.Stop()
}


func (t *DailyStatisticService) Stop() error {
	t.quit <- true
	close(t.quit)
	return nil
}

