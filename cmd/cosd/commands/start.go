package commands

import (
	"fmt"
	"github.com/coschain/cobra"
	ctrl "github.com/coschain/contentos-go/app"
	"github.com/coschain/contentos-go/app/plugins"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/config"
	"github.com/coschain/contentos-go/consensus"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/AWSHealthCheck"
	"github.com/coschain/contentos-go/mylog"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/p2p"
	"github.com/coschain/contentos-go/rpc"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

var trxSqlFlag bool
var dailyStatFlag bool
var stateLogFlag bool

var StartCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "start",
		Short:     "start cosd node",
		Long:      "start cosd node,if has arg 'replay',will sync the lost block to db",
		ValidArgs: []string{"replay"},
		Run:       startNode,
	}
	cmd.Flags().StringVarP(&cfgName, "name", "n", "", "node name (default is cosd)")
	cmd.Flags().BoolVarP(&trxSqlFlag, "trxsqlservice", "", false, "--trxsqlservice=true")
	cmd.Flags().Lookup("trxsqlservice").NoOptDefVal = "true"
	cmd.Flags().BoolVarP(&dailyStatFlag, "dailystatservice", "", false, "--dailystatservice=true")
	cmd.Flags().Lookup("dailystatservice").NoOptDefVal = "true"
	cmd.Flags().BoolVarP(&stateLogFlag, "statelogservice", "", false, "--statelogservice=true")
	cmd.Flags().Lookup("statelogservice").NoOptDefVal = "true"
	return cmd
}

var VERSION string = "defaultVersion"


func makeNode() (*node.Node, node.Config) {
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
	if cfg.DataDir != "" {
		dir, err := filepath.Abs(cfg.DataDir)
		if err != nil {
			common.Fatalf("DataDir in cfg cannot be converted to absolute path")
		}
		cfg.DataDir = dir
	}
	cfg.P2P.RunningCodeVersion = VERSION
	app, err := node.New(&cfg)
	if err != nil {
		fmt.Println("Fatal: ", err)
		os.Exit(1)
	}
	return app, cfg

}

// NO OTHER CONFIGS HERE EXCEPT NODE CONFIG
func startNode(cmd *cobra.Command, args []string) {
	// _ is cfg as below process has't used

	_, _ = cmd, args
	if len(args) > 0 && args[0] == "replay" {
		//If replay, remove level db first then  sync blocks from block log and snapshot to db
		err := os.RemoveAll(filepath.Join(config.DefaultDataDir(), ClientIdentifier, "db"))
		if err != nil {
			panic("remove db fail when node replay")
		}
	}
	app, cfg := makeNode()
	app.Log = mylog.Init(cfg.ResolvePath("logs"), cfg.LogLevel, 0)
	app.Log.Info("Cosd running version: ", VERSION)

	//pprof.StartPprof()

	RegisterService(app, cfg)

	if err := app.Start(); err != nil {
		common.Fatalf("start node failed, err: %v\n", err)
	}

	go func() {
		SIGSTOP := syscall.Signal(0x13) //for windows compile
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT, SIGSTOP, syscall.SIGUSR1, syscall.SIGUSR2)
		for {
			s := <-sigc
			app.Log.Infof("get a signal %s", s.String())
			switch s {
			case syscall.SIGQUIT, syscall.SIGTERM, SIGSTOP, syscall.SIGINT:
				app.Log.Infoln("Got interrupt, shutting down...")
				app.MainLoop.Stop()
				return
			case syscall.SIGHUP:
				app.Log.Info("syscall.SIGHUP custom operation")
			case syscall.SIGUSR1:
				app.Log.Info("syscall.SIGUSR1 custom operation")
			case syscall.SIGUSR2:
				app.Log.Info("syscall.SIGUSR2 custom operation")
			default:
				return
			}
		}
	}()
	app.Log.Info("start complete")
	app.Wait()
	app.Stop()
	app.Log.Info("app exit success")
}


func RegisterService(app *node.Node, cfg node.Config) {
	_ = app.Register(iservices.DbServerName, func(ctx *node.ServiceContext) (node.Service, error) {
		return storage.NewGuardedDatabaseService(ctx, "./db/")
	})

	_ = app.Register(iservices.TxPoolServerName, func(ctx *node.ServiceContext) (node.Service, error) {
		return ctrl.NewController(ctx, app.Log)
	})

	_ = app.Register(plugins.FollowServiceName, func(ctx *node.ServiceContext) (node.Service, error) {
		return plugins.NewFollowService(ctx, app.Log)
	})
	_ = app.Register(plugins.PostServiceName, func(ctx *node.ServiceContext) (node.Service, error) {
		return plugins.NewPostService(ctx)
	})
	_ = app.Register(plugins.TrxServiceName, func(ctx *node.ServiceContext) (node.Service, error) {
		return plugins.NewTrxSerVice(ctx, app.Log)
	})

	if trxSqlFlag {
		_ = app.Register(plugins.TrxMysqlServiceName, func(ctx *node.ServiceContext) (service node.Service, e error) {
			return plugins.NewTrxMysqlSerVice(ctx, cfg.Database, app.Log)
		})
	}

	if stateLogFlag {
		_ = app.Register(plugins.StateLogServiceName, func(ctx *node.ServiceContext) (service node.Service, e error) {
			return plugins.NewStateLogService(ctx, cfg.Database, app.Log)
		})
	}
	
	if dailyStatFlag {
		_ = app.Register(iservices.DailyStatisticServiceName, func(ctx *node.ServiceContext) (node.Service, error) {
			return plugins.NewDailyStatisticService(ctx, cfg.Database, app.Log)
		})
	}

	_ = app.Register(iservices.ConsensusServerName, func(ctx *node.ServiceContext) (node.Service, error) {
		var s node.Service
		switch ctx.Config().Consensus.Type {
		case "DPoS":
			s = consensus.NewDPoS(ctx, app.Log)
		case "SABFT":
			s = consensus.NewSABFT(ctx, app.Log)
		default:
			s = consensus.NewDPoS(ctx, app.Log)
		}
		return s, nil
	})


	_ = app.Register(plugins.RewardServiceName, func(ctx *node.ServiceContext) (service node.Service, e error) {
		return plugins.NewRewardService(ctx)
	})

	_ = app.Register(iservices.RpcServerName, func(ctx *node.ServiceContext) (node.Service, error) {
		return rpc.NewGRPCServer(ctx, ctx.Config().GRPC, app.Log)
	})

	_ = app.Register(iservices.P2PServerName, func(ctx *node.ServiceContext) (node.Service, error) {
		return p2p.NewServer(ctx, app.Log)
	})

	_ = app.Register(AWSHealthCheck.HealthCheckName, func(ctx *node.ServiceContext) (node.Service, error) {
		return AWSHealthCheck.NewAWSHealthCheck(ctx, app.Log)
	})


}
