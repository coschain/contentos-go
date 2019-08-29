package commands

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/coschain/cobra"
	ctrl "github.com/coschain/contentos-go/app"
	"github.com/coschain/contentos-go/cmd/multinodetester/commands/test"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/config"
	"github.com/coschain/contentos-go/consensus"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/mylog"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/p2p"
	"github.com/coschain/gobft/message"
	"github.com/spf13/viper"
)

var VERSION = "defaultVersion"
var latency int
var shut bool
var testSync bool

var TestCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test count",
		Short: "start cosd nodes",
		Run:   startNodes,
	}
	cmd.Flags().IntVarP(&latency, "latency", "l", 1500, "test count -l 1500 (in ms)")
	cmd.Flags().BoolVarP(&shut, "random_shutdown", "s", false, "")
	cmd.Flags().BoolVarP(&testSync, "test_sync", "e", false, "")
	return cmd
}

func makeNode(name string) (*node.Node, node.Config) {
	var cfg node.Config
	cfg.Name = name
	confdir := filepath.Join(config.DefaultDataDir(), cfg.Name)
	viper.SetConfigFile(confdir + "/config.toml")
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

type emptyWriter struct{}

func (ew emptyWriter) Write(p []byte) (int, error) {
	return 0, nil
}
func startNodes(cmd *cobra.Command, args []string) {
	// _ is cfg as below process has't used

	_, _ = cmd, args
	cnt, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Println(err)
		return
	}
	startNodes2(cnt, 0)
}
func startNodes2(cnt int, runSecond int32) {

	clear(nil, nil)
	initConfByCount(cnt)

	nodes := make([]*node.Node, 0, cnt)
	sks := make([]string, 0, cnt)
	names := make([]string, cnt)

	for i := 0; i < cnt; i++ {
		name := fmt.Sprintf("%s_%d", TesterClientIdentifier, i)
		app, cfg := makeNode(name)
		app.Log = mylog.Init(cfg.ResolvePath("logs"), cfg.LogLevel, 3600*24*7)
		app.Log.Out = &emptyWriter{}
		app.Log.Info("Cosd running version: ", VERSION)
		RegisterService(app, cfg)
		if err := app.Start(); err != nil {
			common.Fatalf("start node failed, err: %v\n", err)
		}
		nodes = append(nodes, app)
		sks = append(sks, cfg.Consensus.LocalBpPrivateKey)
	}

	for i := 1; i < cnt; i++ {
		names[i] = fmt.Sprintf("initminer%d", i)
	}

	stopCh := make(chan struct{})
	go func() {
		if runSecond != 0 {
			time.Sleep(time.Duration(uint64(runSecond) * uint64(time.Second)))
			close(stopCh)
			return
		}

		SIGSTOP := syscall.Signal(0x13) //for windows compile
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT, SIGSTOP, syscall.SIGUSR1, syscall.SIGUSR2)
		for {
			s := <-sigc
			switch s {
			case syscall.SIGQUIT, syscall.SIGTERM, SIGSTOP, syscall.SIGINT:
				for i := range nodes {
					nodes[i].Log.Infoln("Got interrupt, shutting down...")
				}
				close(stopCh)
				return
			default:
				return
			}
		}
	}()

	c, err := nodes[0].Service(iservices.ConsensusServerName)
	if err != nil {
		panic(err)
	}
	css := c.(iservices.IConsensus)

	time.Sleep(2 * time.Second)
	for i := 1; i < cnt; i++ {
		if err = test.CreateAcc(names[i], sks[i], sks[0], css); err != nil {
			panic(err)
		}
	}
	fmt.Printf("created %d accounts\n", cnt-1)

	time.Sleep(2 * time.Second)
	for i := 1; i < cnt-1; i++ {
		if err = test.RegisterBP(names[i], sks[i], css); err != nil {
			panic(err)
		}
	}
	fmt.Printf("registered %d accounts\n", cnt-1)

	comp := make([]*test.Components, len(nodes))
	for i := 0; i < len(nodes); i++ {
		comp[i] = &test.Components{}
		resetSvc(nodes[i], comp[i])
	}
	monitor := test.NewMonitor(comp)
	go monitor.Run()

	time.Sleep(2 * time.Second)
	go monitor.Shuffle(names[1:cnt-1], sks[1:cnt-1], css, stopCh)
	if shut {
		go randomlyShutdownNodes(nodes, comp, stopCh)
	}
	if testSync {
		go eraseNodeDataAndRestart(nodes[cnt-1], comp[cnt-1], cnt-1, stopCh)
	}

	<-stopCh
	for i := range nodes {
		nodes[i].Log.Info("start complete")
		//nodes[i].Wait()
		nodes[i].Stop()
		nodes[i].Log.Info("app exit success")
	}
}

func resetSvc(node *node.Node, comp *test.Components) {
	c, err := node.Service(iservices.ConsensusServerName)
	if err != nil {
		panic(err)
	}
	css := c.(iservices.IConsensus)

	p, err := node.Service(iservices.P2PServerName)
	if err != nil {
		panic(err)
	}
	p2p := p.(iservices.IP2P)
	p2p.SetMockLatency(latency)
	css.EnableMockSignal()
	comp.ConsensusSvc = css
	comp.P2pSvc = p2p
	comp.State = test.Syncing
	go func() {
		for {
			time.Sleep(time.Second)
			if css.CheckSyncFinished() {
				comp.State = test.OnLine
				break
			}
		}
	}()
}

func readyToShutDown(node *node.Node) bool {
	c, err := node.Service(iservices.ConsensusServerName)
	if err != nil {
		panic(err)
	}
	css := c.(iservices.IConsensus)
	lastCommit := css.GetLastBFTCommit()
	return time.Since(lastCommit.(*message.Commit).CommitTime) < 10*time.Second
}

func eraseNodeDataAndRestart(node *node.Node, comp *test.Components, idx int, ch chan struct{}) {
	c, err := node.Service(iservices.ConsensusServerName)
	if err != nil {
		panic(err)
	}
	css := c.(iservices.IConsensus)

	ticker := time.NewTicker(10 * time.Second).C
	for {
		select {
		case <-ch:
			return
		case <-ticker:
			if css.CheckSyncFinished() {
				if err := node.Stop(); err != nil {
					panic(err)
				}

				name := fmt.Sprintf("%s_%d", TesterClientIdentifier, idx)
				confdir := filepath.Join(config.DefaultDataDir(), name)
				cmdLine := fmt.Sprintf("ls %s | grep -v %q", confdir, "config")
				cmd := exec.Command("/bin/bash", "-c", cmdLine)
				out, err := cmd.CombinedOutput()
				if err != nil {
					panic(err)
				}

				files := strings.Split(string(out), "\n")
				for i := range files {
					if files[i] == "" {
						continue
					}
					cmdLine = fmt.Sprintf("rm -rf %s", filepath.Join(confdir, files[i]))
					_, err = exec.Command("/bin/bash", "-c", cmdLine).CombinedOutput()
					if err != nil {
						panic(err)
					}
				}

				if err := node.Start(); err != nil {
					panic(err)
				}
				resetSvc(node, comp)
			}
		}
	}
}

func randomlyShutdownNodes(nodes []*node.Node, c []*test.Components, ch chan struct{}) {
	ticker := time.NewTicker(10 * time.Second).C
	for {
		select {
		case <-ch:
			return
		case <-ticker:
			if !readyToShutDown(nodes[1]) {
				continue
			}

			idx := rand.Int() % len(nodes)
			if idx <= 1 {
				idx = 2
			}
			if idx == len(nodes)-1 {
				idx--
			}
			totalOffline := 0
			for i := range c {
				if c[i].State == test.OffLine {
					totalOffline++
				}
			}
			if c[idx].State == test.OffLine || totalOffline > len(nodes)/2 {
				continue
			}
			if err := nodes[idx].Stop(); err != nil {
				panic(err)
			}
			c[idx].State = test.OffLine
			go func() {
				time.Sleep(10 * time.Second)
				if err := nodes[idx].Start(); err != nil {
					panic(err)
				}
				resetSvc(nodes[idx], c[idx])
			}()
		}
	}
}

func RegisterService(app *node.Node, cfg node.Config) {
	_ = app.Register(iservices.DbServerName, func(ctx *node.ServiceContext) (node.Service, error) {
		return storage.NewGuardedDatabaseService(ctx, "./db/")
	})

	_ = app.Register(iservices.TxPoolServerName, func(ctx *node.ServiceContext) (node.Service, error) {
		return ctrl.NewController(ctx, app.Log)
	})

	_ = app.Register(iservices.ConsensusServerName, func(ctx *node.ServiceContext) (node.Service, error) {
		return consensus.NewSABFT(ctx, app.Log), nil
	})

	_ = app.Register(iservices.P2PServerName, func(ctx *node.ServiceContext) (node.Service, error) {
		return p2p.NewServer(ctx, app.Log)
	})
}
