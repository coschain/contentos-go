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
		Short: "initialize all db",
		Run: initAllDb,
	}

	cmd.AddCommand(initCmd)
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

func initTrxDb(cmd *cobra.Command, args []string) {
	cfg := readConfig()
	dbConfig := cfg.Database
	dsn := fmt.Sprintf("%s:%s@/%s", dbConfig.User, dbConfig.Password, dbConfig.Db)
	db, err := sql.Open(dbConfig.Driver, dsn)
	defer db.Close()
	if err != nil {
		fmt.Printf("fatal: init database failed, dsn:%s\n", dsn)
		os.Exit(1)
	}
	createTrxInfo := `create table trxinfo
	(
        id bigint AUTO_INCREMENT PRIMARY KEY,
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

	dropTables := []string{"trxinfo", "libinfo"}
	for _, table := range dropTables {
		dropSql := fmt.Sprintf("DROP TABLE IF EXISTS `%s`", table)
		if _, err = db.Exec(dropSql); err != nil {
			fmt.Println(err)
		}
	}
	createTables := []string{createTrxInfo, createLibInfo }
	for _, table := range createTables {
		if _, err = db.Exec(table); err != nil {
			fmt.Println(err)
		}
	}
	_, _ = db.Exec("INSERT INTO `libinfo` (lib, last_check_time) VALUES (?, ?)", 0, time.Now().UTC().Unix())
}

func initAllDb(cmd *cobra.Command, args []string) {
	initTrxDb(cmd, args)
}
