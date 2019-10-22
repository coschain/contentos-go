package main

import (
	"fmt"
	"github.com/coschain/contentos-go/cmd/autosetup/setup"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/config"
	"github.com/coschain/contentos-go/node"
	"github.com/spf13/viper"
	"os"
	"os/exec"
	"path/filepath"
)

func main(){
	autoSetUp := new(setup.SetupAdmin)
	autoSetUp.Init()

	autoSetUp.ReadAndProcess(setup.NodeName, fmt.Sprintf("Enter your node name (If you want to use default name, enter %s): ", setup.DefaultValueSignal))

	if setup.InitNewConfig {
		autoSetUp.ReadAndProcess(setup.ChainId,
			fmt.Sprintf("Which chain do you want to connect? (%s/%s/%s, connect default chain enter %s) ",
				common.ChainNameMainNet, common.ChainNameTestNet, common.ChainNameDevNet, setup.DefaultValueSignal))

		autoSetUp.ReadAndProcess(setup.IsBp, fmt.Sprintf("Do you want to start a bp node? (%s/%s) ", setup.Positive, setup.Negative))
		if setup.NodeIsBp {
			autoSetUp.ReadAndProcess(setup.BpName, "Enter your account name: ")
			autoSetUp.ReadAndProcess(setup.PriKey, "Enter your private key: ")
		}

		autoSetUp.ReadAndProcess(setup.SeedList, "Enter seed node list: (e.g. ip1:port1,ip2:port2) ")
		autoSetUp.ReadAndProcess(setup.LogLevel, fmt.Sprintf("Enter your log level, use default level enter %s: ", setup.DefaultValueSignal))
		autoSetUp.ReadAndProcess(setup.DataDir, fmt.Sprintf("Enter your data directory, use default directory enter %s: ", setup.DefaultValueSignal))

		err := autoSetUp.WriteConfig()
		if err != nil {
			fmt.Println("Create config file error: ", err)
			return
		}
	}

	autoSetUp.ReadAndProcess(setup.StartNode, fmt.Sprintf("Do you want to start the node right now? (%s/%s) ", setup.Positive, setup.Negative))
	if setup.StartNodeNow {
		var cmdStr string
		finalCfg := readConfig(autoSetUp.Cfg().Name)
		name := finalCfg.Name
		chainid := finalCfg.ChainId
		prefixCmd := fmt.Sprintf("./run.sh %s", name)

		switch chainid {
		case common.ChainNameMainNet:
			cmdStr = fmt.Sprintf("%s %s", prefixCmd, "mainnet")
		case common.ChainNameTestNet:
			cmdStr = fmt.Sprintf("%s %s", prefixCmd, "testnet")
		case common.ChainNameDevNet:
			cmdStr = fmt.Sprintf("%s %s", prefixCmd, "devnet")
		default:
			fmt.Println("Unknown chain id ", chainid)
		}

		cmd := exec.Command("/bin/bash", "-c", cmdStr)
		err := cmd.Start()
		if err != nil {
			fmt.Println("Start node error: ", err)
			return
		}
	}
}

func readConfig(nodeNae string) node.Config {
	var cfg node.Config

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	confdir := filepath.Join(config.DefaultDataDir(), nodeNae)
	viper.AddConfigPath(confdir)
	err := viper.ReadInConfig()
	if err == nil {
		_ = viper.Unmarshal(&cfg)
	} else {
		fmt.Printf("fatal: not be initialized (do `init` first)\n")
		os.Exit(1)
	}
	return cfg
}