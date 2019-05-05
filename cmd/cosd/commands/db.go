package commands

import (
	"database/sql"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/config"
	"github.com/coschain/contentos-go/node"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"time"
)

var DbCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
	}

	initCmd := &cobra.Command{
		Use: "init",
		Short: "initialize external db",
		Run: initDb,
	}

	cleanCmd := &cobra.Command{
		Use: "clean",
		Short: "clean external db",
		Run: cleanDb,
	}

	cmd.AddCommand(initCmd)
	cmd.AddCommand(cleanCmd)
	return cmd
}

func readConfig() *node.Config {
	var cfg node.Config
	if cfgName == "" {
		cfg.Name = ClientIdentifier
	} else {
		cfg.Name = cfgName
	}
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	confdir := filepath.Join(config.DefaultDataDir(), cfg.Name)
	viper.AddConfigPath(confdir)
	err := viper.ReadInConfig()
	if err == nil {
		_ = viper.Unmarshal(&cfg)
	} else {
		fmt.Printf("fatal: not be initialized (do `init` first)\n")
		os.Exit(1)
	}
	return &cfg
}

func initDb(cmd *cobra.Command, args []string) {
	cfg := readConfig()
	dbConfig := cfg.Database
	dsn := fmt.Sprintf("%s:%s@/%s", dbConfig.User, dbConfig.Password, dbConfig.Db)
	db, err := sql.Open(dbConfig.Driver, dsn)
	if err != nil {
		fmt.Printf("fatal: init database failed, dsn:%s\n", dsn)
		os.Exit(1)
	}
	createTrxInfo := `create table trxinfo
(
	trx_id varchar(64) not null,
	block_height int unsigned not null,
	block_time int unsigned not null,
	invoice json null,
	operations json null,
	block_id varchar(64) not null,
	creator varchar(64) not null,
	INDEX trxinfo_block_height_index (block_height),
	INDEX trxinfo_block_time_index (block_time),
	INDEX trxinfo_block_id (block_id),
	INDEX trxinfo_block_creator (creator),
	constraint trxinfo_trx_id_uindex
		unique (trx_id)
);`
	createLibInfo := `create table libinfo
(
	lib int unsigned not null,
	last_check_time int unsigned not null
);`
	createCreateAccountInfo := `create table createaccountinfo
(
	trx_id varchar(64) not null,
	create_time int unsigned not null,
	creator varchar(64) not null,
	pubkey varchar(64) not null,
	account varchar(64) not null,
	INDEX createaccount_create_time (create_time),
	INDEX createaccount_creator (creator),
	INDEX creatoraccount_account (account),
  constraint createaccount_trx_id_uindex unique (trx_id)
);`
	createTransferInfo := `create table transferinfo
(
	trx_id varchar(64) not null,
	create_time int unsigned not null,
	sender varchar(64) not null,
	receiver varchar(64) not null,
	amount int unsigned default 0,
	memo TEXT ,
	INDEX transfer_create_time (create_time),
	INDEX transfer_sender (sender),
	INDEX transfer_receiver (receiver),
  constraint transferinfo_trx_id_uindex unique (trx_id)
);`
	createDAUStat := `create table daustat (
  date varchar(64) not null ,
  dapp varchar(64) not null ,
  count int unsigned not null default 0,
  INDEX daustat_dapp (dapp),
  constraint daustat_date_dapp_uindex
  unique (date, dapp)
);`

	createDNUStat := `create table dnustat (
  date varchar(64) not null ,
  dapp varchar(64) not null ,
  count int unsigned not null default 0,
  INDEX dnustat_dapp (dapp),
  constraint dnustat_date_dapp_uindex
  unique (date, dapp)
);`

	createDailyStatInfo := `create table dailystatinfo
(
  lib int unsigned not null,
  date varchar(64) not null,
  last_check_time int unsigned not null
);`

	createDailyStatDapp := `create table dailystatdapp (
  dapp varchar(64) not null,
  prefix varchar(64) not null,
  status smallint default 1
);`

	dropTables := []string{"trxinfo", "libinfo", "createaccountinfo", "transferinfo", "daustat", "dnustat",
		"dailystatinfo", "dailystatdapp"}
	for _, table := range dropTables {
		dropSql := fmt.Sprintf("DROP TABLE IF EXISTS `%s`", table)
		if _, err = db.Exec(dropSql); err != nil {
			fmt.Println(err)
		}
	}
	createTables := []string{createTrxInfo, createLibInfo, createCreateAccountInfo,
		createTransferInfo, createDAUStat, createDNUStat, createDailyStatInfo, createDailyStatDapp}
	for _, table := range createTables {
		if _, err = db.Exec(table); err != nil {
			fmt.Println(err)
		}
	}
	_, _ = db.Exec("INSERT INTO `libinfo` (lib, last_check_time) VALUES (?, ?)", 0, time.Now().UTC().Unix())
	_, _ = db.Exec("INSERT INTO `dailystatinfo` (lib, date, last_check_time) VALUES (?, ?, ?)", 0, "", 0)
	_, _ = db.Exec("INSERT INTO `dailystatdapp` (dapp, prefix) VALUES (?, ?), (?, ?), (?, ?), (?, ?)",
		"photogrid", "pg", "contentos", "ct", "game 2048", "g2", "walk coin", "ec")
}

func cleanDb(cmd *cobra.Command, args []string) {
	cfg := readConfig()
	dbConfig := cfg.Database
	dsn := fmt.Sprintf("%s:%s@/%s", dbConfig.User, dbConfig.Password, dbConfig.Db)
	db, err := sql.Open(dbConfig.Driver, dsn)
	if err != nil {
		fmt.Printf("fatal: init database failed, dsn:%s\n", dsn)
		os.Exit(1)
	}
	dropTables := []string{"trxinfo", "libinfo", "createaccountinfo", "transferinfo", "daustat", "dnustat",
		"dailystatinfo", "dailystatdapp"}
	for _, table := range dropTables {
		dropSql := fmt.Sprintf("DROP TABLE IF EXISTS `%s`", table)
		if _, err = db.Exec(dropSql); err != nil {
			fmt.Println(err)
		}
	}
}