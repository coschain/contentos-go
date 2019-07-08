package plugins

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/iservices/service-configs"
	"github.com/coschain/contentos-go/node"
	"github.com/sirupsen/logrus"
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

type ContractData struct {
	Contract string
	ContractOwner string
	Record string
}

type TokenData struct {
	TokenOwner string
	Amount uint64
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
	markedTokens := make(map[string]bool)
	rows, _ := s.db.Query("select symbol, owner from markedtoken")
	for rows.Next() {
		var symbol string
		var owner string
		if err := rows.Scan(&symbol, &owner); err != nil {
			s.log.Error(err)
			continue
		}
		key := fmt.Sprintf("%s#%s", symbol, owner)
		markedTokens[key] = true
	}
	if len(markedTokens) == 0 {
		return nil
	}
	rows, _ = s.db.Query("SELECT block_height, block_log from statelog where block_height > ? limit 1000", lib)
	for rows.Next() {
		var blockHeight uint64
		var log interface{}
		var blockLog iservices.BlockLog
		if err := rows.Scan(&blockHeight, &log); err != nil {
			s.log.Error(err)
			continue
		}
		data := log.([]byte)
		if err := json.Unmarshal(data, &blockLog); err != nil {
			s.log.Error(err)
			continue
		}
		s.handleBlockLog(markedTokens, blockLog)
		utcTimestamp := time.Now().UTC().Unix()
		_, _ = s.db.Exec("UPDATE tokenlibinfo SET lib=?, last_check_time=?", blockHeight, utcTimestamp)
	}
	return nil
}

func (s *TokenInfoService) handleBlockLog(tokens map[string]bool, blockLog iservices.BlockLog) {
	blockId := blockLog.BlockId
	trxLogs := blockLog.TrxLogs
	for _, trxLog := range trxLogs {
		trxId := trxLog.TrxId
		opLogs := trxLog.OpLogs
		for _, opLog := range opLogs {
			action := opLog.Action
			property := opLog.Property
			target := opLog.Target
			result := opLog.Result
			switch property {
			case "contract":
				s.handleTokenInfo(tokens, blockId, trxId, action, target, result)
			}
		}
	}
}

func (s *TokenInfoService) handleTokenInfo(tokens map[string]bool, blockId string, trxId string, action int, target string, result interface{}) {
	contractData := result.(ContractData)
	contract := contractData.Contract
	owner := contractData.ContractOwner
	key := fmt.Sprintf("%s#%s", contract, owner)
	if _, ok := tokens[key]; ok {
		record := contractData.Record
		data := []byte(record)
		var tokenData TokenData
		if err := json.Unmarshal(data, &tokenData); err != nil {
			s.log.Error(err)
			return
		}
		switch action {
		case iservices.Insert:
			_, _ = s.db.Exec("INSERT INTO tokenbalance (symbol, owner, account, balance) VALUES (?, ?, ?, ?)",
				contract, owner, tokenData.TokenOwner, tokenData.Amount)
		case iservices.Update:
			_, _ = s.db.Exec("update tokenbalance set balance=? where symbol=? and owner=? and account=?",
				tokenData.Amount, contract, owner, tokenData.TokenOwner)
		}
	}
}

func (s *TokenInfoService) stop() {
	s.ticker.Stop()
}


func (t *TokenInfoService) Stop() error {
	t.quit <- true
	close(t.quit)
	return nil
}

