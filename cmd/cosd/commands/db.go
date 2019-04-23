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
		block_id varchar(64) null,
		INDEX trxinfo_block_height_index (block_height),
		INDEX trxinfo_block_time_index (block_time),
		INDEX trxinfo_block_id (block_id),
		constraint trxinfo_trx_id_uindex
	unique (trx_id)
	)`
	createLibInfo := `create table libinfo
(
	lib int unsigned not null,
	last_check_time int unsigned not null
);`
	if _, err = db.Exec("DROP TABLE IF EXISTS `trxinfo`"); err != nil {
		fmt.Println(err)
	}
	if _, err = db.Exec(createTrxInfo); err != nil {
		fmt.Println(err)
	}
	if _, err = db.Exec("DROP TABLE IF EXISTS `libinfo`"); err != nil {
		fmt.Println(err)
	}
	if _, err = db.Exec(createLibInfo); err != nil {
		fmt.Println(err)
	}
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
	if _, err = db.Exec("DROP TABLE IF EXISTS `trxinfo`"); err != nil {
		fmt.Println(err)
	}
	if _, err = db.Exec("DROP TABLE IF EXISTS `libinfo`"); err != nil {
		fmt.Println(err)
	}
}