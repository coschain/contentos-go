package setup

import (
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/cmd/cosd/commands"
	p2pCommon "github.com/coschain/contentos-go/p2p/common"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/config"
	"github.com/coschain/contentos-go/mylog"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/prototype"
	"net"
	"os"
	"path/filepath"
	"strings"
)

type SetupAdmin struct {
	cfg        node.Config
	readInfo   ReadInfoStruct
}

type ReadInfoStruct struct {
	readType   string
}

func (admin *SetupAdmin) Cfg() node.Config {
	return admin.cfg
}

func (admin *SetupAdmin) Init() {
	admin.cfg = config.DefaultNodeConfig
	admin.cfg.Consensus.BootStrap = false
	admin.cfg.Consensus.LocalBpName = ""
	admin.cfg.Consensus.LocalBpPrivateKey = ""
}

func (admin *SetupAdmin) ReadAndProcess(readType, displayInfo string) {
	readContent := admin.ReadAndValidate(readType, displayInfo)
	if readContent == EmptyLine {
		fmt.Println("auto setup internal error")
		os.Exit(1)
	}

	switch t := admin.readInfo.readType; t {
	case NodeName:
		if readContent == DefaultValueSignal {
			admin.cfg.Name = commands.ClientIdentifier
		} else {
			admin.cfg.Name = readContent
		}

		configDir := filepath.Join(config.DefaultDataDir(), admin.cfg.Name)
		exist := fileExist(configDir)
		if exist {
			var initNewConfSig string
			initNewConfSig = admin.ReadAndValidate(YesOrNo,
				fmt.Sprintf("Already has a config file, delete and init a new one? (%s/%s) ", Positive, Negative))

			if initNewConfSig == Negative {
				InitNewConfig = false
			}
		}
	case ChainId:
		if readContent == DefaultValueSignal {
			admin.cfg.ChainId = common.ChainNameMainNet
		} else {
			admin.cfg.ChainId = readContent
		}
	case BpName:
		admin.cfg.Consensus.LocalBpName = readContent
	case PriKey:
		admin.cfg.Consensus.LocalBpPrivateKey = readContent
	case SeedList:
		seedListStr := strings.Split(readContent, Separator)
		admin.cfg.P2P.Genesis.SeedList = seedListStr
	case LogLevel:
		if readContent == DefaultValueSignal {
			admin.cfg.LogLevel = mylog.DebugLevel
		} else {
			admin.cfg.LogLevel = readContent
		}
	case DataDir:
		if readContent != DefaultValueSignal {
			dir, _ := filepath.Abs(readContent)
			admin.cfg.DataDir = dir
		}
	case IsBp:
		if readContent == Positive {
			NodeIsBp = true
		}
	case StartNode:
		if readContent == Positive {
			StartNodeNow = true
		}
	}
}

func (admin *SetupAdmin) ReadAndValidate(readType, displayInfo string) (readContent string) {
	admin.readInfo.readType = readType
	for {
		readContent = ReadCmdLine(displayInfo)

		switch t := admin.readInfo.readType; t {
		case StartNode:
			fallthrough
		case IsBp:
			fallthrough
		case YesOrNo:
			if readContent != Positive && readContent != Negative {
				continue
			}
			return
		case ChainId:
			if readContent != common.ChainNameMainNet && readContent != common.ChainNameTestNet &&
				readContent != common.ChainNameDevNet && readContent != DefaultValueSignal {
				continue
			}
			return
		case BpName:
			bpName := &prototype.AccountName{Value:readContent}
			err := bpName.Validate()
			if err != nil {
				continue
			}
			return
		case LogLevel:
			if readContent != mylog.DebugLevel && readContent != mylog.InfoLevel && readContent != mylog.WarnLevel &&
				readContent != mylog.ErrorLevel && readContent != mylog.FatalLevel && readContent != mylog.PanicLevel &&
				readContent != DefaultValueSignal {
				continue
			}
			return
		case DataDir:
			if readContent == DefaultValueSignal {
				return
			}
			_, err := filepath.Abs(readContent)
			if err != nil {
				fmt.Println("DataDir cannot be converted to absolute path")
				continue
			}
			return
		case SeedList:
			err := validateSeedList(readContent)
			if err != nil {
				fmt.Println(err)
				continue
			}
			return
		default:
			return
		}
	}
}

func (admin *SetupAdmin) WriteConfig() error {
	confdir := filepath.Join(config.DefaultDataDir(), admin.cfg.Name)
	if _, err := os.Stat(confdir); os.IsNotExist(err) {
		if err = os.MkdirAll(confdir, 0700); err != nil {
			fmt.Println(err)
			return err
		}
	}

	err := config.WriteNodeConfigFile(confdir, "config.toml", admin.cfg, 0600)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func fileExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func ReadCmdLine(displayInfo string) string {
	var content string
	for {
		fmt.Printf(displayInfo)
		fmt.Scanln(&content)
		if content != EmptyLine {break}
	}
	return content
}

func validateSeedList(readContent string) error {
	seedList := strings.Split(readContent, Separator)
	for _, n := range seedList {
		ip, err := p2pCommon.ParseIPAddr(n)
		if err != nil {
			return errors.New(fmt.Sprintf("seed peer %s address format is wrong", n))
		}
		_, err = net.LookupHost(ip)
		if err != nil {
			return errors.New(fmt.Sprintf("resolve err: %s", err.Error()))
		}
		_, err = p2pCommon.ParseIPPort(n)
		if err != nil {
			return errors.New(fmt.Sprintf("seed peer %s address format is wrong", n))
		}
	}
	return nil
}