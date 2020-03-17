package commands

import (
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/AWSHealthCheck"
	ctrl "github.com/coschain/contentos-go/app"
	"github.com/coschain/contentos-go/app/plugins"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/common/pprof"
	"github.com/coschain/contentos-go/config"
	"github.com/coschain/contentos-go/consensus"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/mylog"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/p2p"
	"github.com/coschain/contentos-go/rpc"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"errors"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

var pluginList []string
var globalFile *os.File
var optReplayReuseSQL bool

var StartCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "start",
		Short:     "start cosd node",
		Long:      "start cosd node,if has arg 'replay',will sync the lost block to db",
		ValidArgs: []string{"replay"},
		Run:       startNode,
	}
	cmd.Flags().StringVarP(&cfgName, "name", "n", "", "node name (default is cosd)")
	cmd.Flags().StringArrayVarP(&pluginList, "plugin", "", []string{}, "--plugin=[trxsqlservice, dailystatservice, statelogservice, tokeninfoservice]")
	cmd.Flags().BoolVarP(&optReplayReuseSQL, "reuse-sql", "r", false, "reuse current sql data")
	return cmd
}

var NodeName string
const (
	ClientTag  = "v1.0.7"

	spacePrecision = 1024 * 1024 * 1024  // 1 GB in Bytes
)


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
	NodeName = config.MakeName(constants.ClientName, ClientTag)
	cfg.P2P.RunningCodeVersion = NodeName
	app, err := node.New(&cfg)
	if err != nil {
		fmt.Println("Fatal: ", err)
		os.Exit(1)
	}
	return app, cfg

}

func InitCrashFile(fName string) error {
	file, err := os.OpenFile(fName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	globalFile = file
	if err != nil {
		return err
	}
	if err = syscall.Dup2(int(globalFile.Fd()), int(os.Stderr.Fd())); err != nil {
		return err
	}
	return nil
}

// NO OTHER CONFIGS HERE EXCEPT NODE CONFIG
func startNode(cmd *cobra.Command, args []string) {
	// _ is cfg as below process has't used

	pprof.StartPprof()

	_, _ = cmd, args
	app, cfg := makeNode()
	app.Log = mylog.Init(cfg.ResolvePath("logs"), cfg.LogLevel, 3600 * 24 * 7)
	app.Log.Info("Cosd running version: ", NodeName)

	err := checkNTPTime(app.Log)
	if err != nil {
		app.Log.Error(err)
		return
	}


	tStr := time.Unix(time.Now().Unix(),0).Format("2006-01-02-15-04-05")
	crashFileName := cfg.ResolvePath("logs") + "/" + "crash-log-" + tStr
	fmt.Println("crash log:",crashFileName)
	err = InitCrashFile(crashFileName)
	if err != nil {
		panic(fmt.Errorf("init crash file failed, error:%v",err))
	}

	all, err := CheckDiskSpace(filepath.Join(cfg.DataDir, cfg.Name), app.Log)
	if err != nil {
		app.Log.Errorf("check disk space failed, err: %v\n", err)
		return
	}
	if all <= uint64(cfg.MinDiskSpaceInGB) {
		app.Log.Errorf("disk space too small need more than %d GB, you have %d GB", cfg.MinDiskSpaceInGB, all)
		return
	}

	replaying := len(args) > 0 && args[0] == "replay"
	if replaying {
		//If replay, remove level db first then  sync blocks from block log and snapshot to db
		err := os.RemoveAll(cfg.ResolvePath("db"))
		if err != nil {
			panic("remove db fail when node replay")
		}
		// we won't remove sql tables if we want to reuse them
		if len(pluginList) > 0 && !optReplayReuseSQL {
			if err := plugins.RemoveSQLTables(cfg.Database); err != nil {
				panic(fmt.Sprintf("remove sql tables fail when node replay: %s", err.Error()))
			}
		}
	}

	//pprof.StartPprof()

	RegisterService(app, cfg)

	startArgs := make(map[string]interface{})
	startArgs["reuse_sql"] = optReplayReuseSQL

	if err := app.StartWithArgs(startArgs); err != nil {
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
	pluginMgr := plugins.NewPluginMgt(pluginList)

	_ = app.Register(iservices.DbServerName, func(ctx *node.ServiceContext) (node.Service, error) {
		return storage.NewGuardedDatabaseService(ctx, "./db/")
	})

	pluginMgr.RegisterSQLServices(app, &cfg)

	_ = app.Register(iservices.TxPoolServerName, func(ctx *node.ServiceContext) (node.Service, error) {
		return ctrl.NewController(ctx, app.Log)
	})

	pluginMgr.RegisterTrxPoolDependents(app, &cfg)

	_ = app.Register(iservices.ConsensusServerName, func(ctx *node.ServiceContext) (node.Service, error) {
		return consensus.NewSABFT(ctx, app.Log), nil
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

func CheckDiskSpace(path string, log *logrus.Logger) (all uint64, err error) {
	freeDiskInGB, err := getDiskFreeSpace(path)
	if err != nil {
		return 0, err
	}

	nodeUsedSpace, err := getNodeUsedSpace(path)
	if err != nil {
		return 0, err
	}

	log.Infof("Disk free space %d GB, node used space %d GB", freeDiskInGB, nodeUsedSpace)
	return nodeUsedSpace + freeDiskInGB, nil
}

func getDiskFreeSpace(path string) (free uint64, err error) {
	fs := syscall.Statfs_t{}
	err = syscall.Statfs(path, &fs)
	if err != nil {
		return 0, err
	}
	//allInBytes := fs.Blocks * uint64(fs.Bsize)
	freeInBytes := fs.Bfree * uint64(fs.Bsize)
	//usedInBytes := allInBytes - freeInBytes

	//allInGB := allInBytes / spacePrecision
	freeInGB := freeInBytes / spacePrecision
	//usedInGB := usedInBytes / spacePrecision

	return freeInGB, nil
}

func getNodeUsedSpace(path string) (used uint64, err error) {
	var size int64
	err = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	if err != nil {
		return 0, errors.New(fmt.Sprintf("Get node used space error %s", err))
	}
	used = uint64(size) / spacePrecision
	return used, err
}
