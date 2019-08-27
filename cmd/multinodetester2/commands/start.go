package commands

import (
	"fmt"
	"github.com/coschain/cobra"
	ctrl "github.com/coschain/contentos-go/app"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/pprof"
	"github.com/coschain/contentos-go/config"
	"github.com/coschain/contentos-go/consensus"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/p2p"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"time"
)

var StartCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start cosd-path count(default 3)",
		Short: "start multi cosd node",
		Run:   AutoTest,
	}
	cmd.Flags().Int64VarP(&NodeCnt, "number", "n", 3, "number of cosd thread")
	return cmd
}

type GlobalObject struct {
	arr      []*node.Node
	cfgList  []node.Config
	dposList []iservices.IConsensus
	dbList   []iservices.IDatabaseService
}

var globalObj GlobalObject

func AutoTest(cmd *cobra.Command, args []string) {
	createAndTransfer()

	fmt.Println("test done")
}

func StartNode() {
	for i:=0;i<int(NodeCnt);i++{
		fmt.Println("i: ", i," NodeCnt: ", NodeCnt)
		app, cfg := makeNode(i)

		startNode(app, cfg)
	}
	time.Sleep(10 * time.Second) // this sleep wait the whole net to be constructed
}

func makeNode(index int) (*node.Node, node.Config) {
	var cfg node.Config
	confdir := filepath.Join(config.DefaultDataDir(), fmt.Sprintf("%s_%d", TesterClientIdentifier, index))
	fmt.Println("config dir: ", confdir)
	viper.Reset()
	viper.AddConfigPath(confdir)
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
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
	app, err := node.New(&cfg)
	if err != nil {
		fmt.Println("Fatal: ", err)
		os.Exit(1)
	}
	fmt.Println("Name: ", cfg.Name)
	fmt.Println("p2p node port: ", cfg.P2P.NodePort)
	fmt.Println("p2p consensus port: ", cfg.P2P.NodeConsensusPort)
	fmt.Println("consensus type: ", cfg.Consensus.Type)
	return app, cfg
}

func startNode(app *node.Node, cfg node.Config) {
	//app.Log = mylog.Init(cfg.ResolvePath("logs"), mylog.DebugLevel, 0)
	//app.Log.SetOutput(ioutil.Discard)

	pprof.StartPprof()

	RegisterService(app, cfg)

	if err := app.Start(); err != nil {
		common.Fatalf("start node failed, err: %v\n", err)
	}

	it, err := app.Service(iservices.ConsensusServerName)
	if err != nil {
		panic(err)
	}
	Icons := it.(iservices.IConsensus)
	Icons.ResetProdTimer( 86400 * time.Second )
	idb, err := app.Service(iservices.DbServerName)
	if err != nil {
		panic(err)
	}
	globalObj.arr = append(globalObj.arr, app)
	globalObj.cfgList = append(globalObj.cfgList, cfg)
	globalObj.dposList = append(globalObj.dposList, Icons)
	globalObj.dbList = append(globalObj.dbList, idb.(iservices.IDatabaseService))

	go app.Wait()
}

func RegisterService(app *node.Node, cfg node.Config) {
	_ = app.Register(iservices.DbServerName, func(ctx *node.ServiceContext) (node.Service, error) {
		return storage.NewGuardedDatabaseService(ctx, "./db/")
	})

	_ = app.Register(iservices.P2PServerName, func(ctx *node.ServiceContext) (node.Service, error) {
		return p2p.NewServer(ctx, nil)
	})

	_ = app.Register(iservices.TxPoolServerName, func(ctx *node.ServiceContext) (node.Service, error) {
		return ctrl.NewController(ctx, nil)
	})

	_ = app.Register(iservices.ConsensusServerName, func(ctx *node.ServiceContext) (node.Service, error) {
		var s node.Service
		switch ctx.Config().Consensus.Type {
		case "SABFT":
			s = consensus.NewSABFT(ctx, nil)
		default:
			s = consensus.NewSABFT(ctx, nil)
		}
		return s, nil
	})
}

func clearAll() {
	stopEachNode()
	home := os.Getenv("HOME")
	for i:=0;i<int(NodeCnt);i++ {
		clearPath(home, i)
	}
	globalObj.arr = globalObj.arr[:0]
	globalObj.cfgList = globalObj.cfgList[:0]
	globalObj.dposList = globalObj.dposList[:0]
	globalObj.dbList = globalObj.dbList[:0]
	time.Sleep(10 * time.Second)
}

func stopEachNode() {
	for i:=0;i<int(NodeCnt);i++ {
		stopNode(globalObj.arr[i])
	}
}

func stopNode(app *node.Node) {
	app.MainLoop.Stop()
	app.Stop()
}

func clearPath(home string, index int) {
	_ = os.RemoveAll( filepath.Join(home, fmt.Sprintf(".coschain/testcosd_%d/blog", index) ) )
	_ = os.RemoveAll( filepath.Join(home, fmt.Sprintf(".coschain/testcosd_%d/db", index) ) )
	_ = os.RemoveAll( filepath.Join(home, fmt.Sprintf(".coschain/testcosd_%d/forkdb_snapshot", index) ) )
	//_ = os.RemoveAll( filepath.Join(home, fmt.Sprintf(".coschain/testcosd_%d/logs", index) ) )
}