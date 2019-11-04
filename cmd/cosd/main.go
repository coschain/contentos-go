package main

import (
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/cosd/commands"
	"os"
	"syscall"
)

// cosd is the main entry point into the system if no special subcommand is pointed
// It creates a default node based on the command line arguments and runs it
// in blocking mode, waiting for it to be shut down.
var rootCmd = &cobra.Command{
	Use:   "cosd",
	Short: "Cosd is a fast blockchain designed for content",
}

var globalFile *os.File

func addCommands() {
	rootCmd.AddCommand(commands.InitCmd())
	rootCmd.AddCommand(commands.StartCmd())
	rootCmd.AddCommand(commands.DbCmd())
}

func main() {
	addCommands()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	err := InitPanicFile()
	if err != nil {
		panic(fmt.Sprintf("init panic file failed, error:%v",err))
	}
}

func InitPanicFile() error {
	file, err := os.OpenFile("crash.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	globalFile = file
	if err != nil {
		return err
	}
	if err = syscall.Dup2(int(globalFile.Fd()), int(os.Stderr.Fd())); err != nil {
		return err
	}
	return nil
}