package commands

import (
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/config"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

var StartCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start cosd-path count(default 3)",
		Short: "start multi cosd node",
		Run:   startNode,
	}
	return cmd
}

// NO OTHER CONFIGS HERE EXCEPT NODE CONFIG
func startNode(cmd *cobra.Command, args []string) {
	var nodeCount int = 3
	if len(args) > 1 {
		cnt, err := strconv.Atoi(args[1])
		if err != nil {
			fmt.Println(err)
			return
		}
		nodeCount = cnt
	}

	cosdPath := args[0]

	for i := 0; i < nodeCount; i++ {
		cfg := config.DefaultNodeConfig
		subDir := fmt.Sprintf("%s_%d", TesterClientIdentifier, i)
		confDir := filepath.Join(cfg.DataDir, subDir)

		fmt.Println(confDir)
		if _, err := os.Stat(confDir); err == nil {
			fmt.Println(cosdPath, " start -n ", subDir)
			c := exec.Command(cosdPath, "start", "-n", subDir)

			//c.Stdout = os.Stdout
			c.Stderr = os.Stderr

			err := c.Start()
			if err != nil {
				fmt.Println(err)
				return
			}
		} else {
			fmt.Println(err)
			clear(nil, nil)
			os.Exit(-1)
		}
	}

	for {
		time.Sleep(1)
	}
}
