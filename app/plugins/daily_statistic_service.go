package plugins

import (
	"database/sql"
	"fmt"
	"github.com/coschain/contentos-go/iservices/itype"
	"github.com/coschain/contentos-go/iservices/service-configs"
	"github.com/coschain/contentos-go/node"
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
	dsn := fmt.Sprintf("%s:%s@/%s", s.config.User, s.config.Password, s.config.Db)
	outDb, err := sql.Open(s.config.Driver, dsn)

	if err != nil {
		return err
	}
	s.outDb = outDb

	s.ticker = time.NewTicker(time.Second * INTERVAL)
	go func() {
		for {
			select {
			case <- s.ticker.C:
				end := time.Now().UTC().Unix()
				start := end - INTERVAL
				if err := s.statisticDAU(start, end); err != nil {
					s.log.Error(err)
				}
				if err = s.statisticDNU(start, end); err != nil {
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

func (s *DailyStatisticService) statisticDAU(start int64, end int64) error {
	selectStmt, _ := s.outDb.Prepare("SELECT creator FROM trxinfo WHERE block_time >= ? and block_time < ?")
	defer selectStmt.Close()
	rows, err := selectStmt.Query(start, end)
	defer rows.Close()
	if err != nil {
		s.log.Error(err)
	}
	//  PG/CT/G2/EC refers to photoGrid, official website, game 2048, walkcoin
	counter := []uint32{0, 0, 0, 0}
	prefixes := []string{"PG", "CT", "G2", "EC"}
	for rows.Next() {
		var (
			creator string
		)
		if err := rows.Scan(&creator); err != nil {
			s.log.Error(err)
		}
		for index, prefix := range prefixes {
			if strings.HasPrefix(creator, prefix) {
				counter[index] ++
			}
		}
	}
	datetime := time.Unix(end, 0)
	date := fmt.Sprintf("%d-%02d-%02d", datetime.Year(), datetime.Month(), datetime.Day())
	hour := datetime.Hour()
	insertStmt, _ := s.outDb.Prepare("INSERT INTO dailydau (date, hour, pg, ct, g2, ec) values (?, ?, ?, ?, ?, ?)")
	defer insertStmt.Close()
	_, _ = insertStmt.Exec(date, hour, counter[0], counter[1], counter[2], counter[3])
	return nil
}

func (s *DailyStatisticService) statisticDNU(start int64, end int64) error {
	selectStmt, _ := s.outDb.Prepare("SELECT creator FROM createaccountinfo WHERE create_time >= ? and create_time < ?")
	defer selectStmt.Close()
	rows, err := selectStmt.Query(start, end)
	defer rows.Close()
	if err != nil {
		s.log.Error(err)
	}
	//  PG/CT/G2/EC refers to photoGrid, official website, game 2048, walkcoin
	counter := []uint32{0, 0, 0, 0}
	prefixes := []string{"PG", "CT", "G2", "EC"}
	for rows.Next() {
		var (
			creator string
		)
		if err := rows.Scan(&creator); err != nil {
			s.log.Error(err)
		}
		for index, prefix := range prefixes {
			if strings.HasPrefix(creator, prefix) {
				counter[index] ++
			}
		}
	}
	datetime := time.Unix(end, 0)
	date := fmt.Sprintf("%d-%d-%d", datetime.Year(), datetime.Month(), datetime.Day())
	hour := datetime.Hour()
	insertStmt, _ := s.outDb.Prepare("INSERT INTO dailydnu (date, hour, pg, ct, g2, ec) values (?, ?, ?, ?, ?, ?)")
	defer insertStmt.Close()
	_, _ = insertStmt.Exec(date, hour, counter[0], counter[1], counter[2], counter[3])
	return nil
}

func (s *DailyStatisticService) DAUStatsOn(date string) *itype.Row {
	var pg, ct, g2, ec int
	stmt, _ := s.outDb.Prepare("select sum(pg) as pg, sum(ct) as ct, sum(g2) as g2, sum(ec) as ec from dailydau where date=?")
	defer stmt.Close()
	_ = stmt.QueryRow(date).Scan(&pg, &ct, &g2, &ec)
	return &itype.Row{Date: date, Pg: pg, Ct: ct, G2: g2, Ec: ec}
}

func (s *DailyStatisticService) DAUStatsSince(days int) []*itype.Row {
	now := time.Now().UTC()
	d, _ := time.ParseDuration("-1d")
	then := now.Add(d * time.Duration(days))
	date := fmt.Sprintf("%d-%02d-%02d", then.Year(), then.Month(), then.Day())
	stmt, _ := s.outDb.Prepare("select date, sum(pg) as pg, sum(ct) as ct, sum(g2) as g2, sum(ec) as ec from dailydau where date >= ? group by date order by date")
	defer stmt.Close()
	//dauRows := make(map[string]Row)
	var dauRows []*itype.Row
	rows, err := stmt.Query(date)
	if err != nil {
		return dauRows
	}
	for rows.Next() {
		var d string
		var pg, ct, g2, ec int
		_ = rows.Scan(&d, &pg, &ct, &g2, &ec)
		r := &itype.Row{Date: d, Pg: pg, Ct: ct, G2: g2, Ec: ec}
		dauRows = append(dauRows, r)
	}
	return dauRows
}

func (s *DailyStatisticService) DNUStatsOn(date string) *itype.Row {
	var pg, ct, g2, ec int
	stmt, _ := s.outDb.Prepare("select sum(pg) as pg, sum(ct) as ct, sum(g2) as g2, sum(ec) as ec from dailydnu where date=?")
	defer stmt.Close()
	_ = stmt.QueryRow(date).Scan(&pg, &ct, &g2, &ec)
	return &itype.Row{Date: date, Pg: pg, Ct: ct, G2: g2, Ec: ec}
}

func (s *DailyStatisticService) DNUStatsSince(days int) []*itype.Row {
	now := time.Now().UTC()
	d, _ := time.ParseDuration("-1d")
	then := now.Add(d * time.Duration(days))
	date := fmt.Sprintf("%d-%02d-%02d", then.Year(), then.Month(), then.Day())
	stmt, _ := s.outDb.Prepare("select date, sum(pg) as pg, sum(ct) as ct, sum(g2) as g2, sum(ec) as ec from dailydnu where date >= ? group by date")
	defer stmt.Close()
	var dauRows []*itype.Row
	rows, err := stmt.Query(date)
	if err != nil {
		return dauRows
	}
	for rows.Next() {
		var d string
		var pg, ct, g2, ec int
		_ = rows.Scan(&d, &pg, &ct, &g2, &ec)
		r := &itype.Row{Date: d, Pg: pg, Ct: ct, G2: g2, Ec: ec}
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