package commands

import (
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/config"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"
)

var ClearCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear all tester data",
		Run:   clear,
	}
	return cmd
}

func clear(cmd *cobra.Command, args []string) {

	cfg := config.DefaultNodeConfig
	cfg.Name = ClientIdentifier


	dir, err := ioutil.ReadDir(cfg.DataDir)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, fi := range dir {
		if strings.HasPrefix(strings.ToLower(fi.Name()), TesterClientIdentifier) ||
			fi.Name() == ClientIdentifier { //匹配文件
			subDirName := filepath.Join( cfg.DataDir, fi.Name() )

			fmt.Println("rm -rf ", subDirName )
			c   := exec.Command("rm", "-rf", subDirName )
			err := c.Run()
			if err != nil{
				fmt.Println(err)
				return
			}
		}
	}


}
