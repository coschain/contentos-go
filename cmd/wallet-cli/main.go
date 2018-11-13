package main

import (
	"github.com/chzyer/readline"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/rpc"
	"github.com/coschain/contentos-go/rpc/pb"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

var rootCmd = &cobra.Command{
	Use:   "wallet-cli",
	Short: "wallet-cli is a key-pair storage",
}

func pcFromCommands(parent readline.PrefixCompleterInterface, c *cobra.Command) {
	pc := readline.PcItem(c.Use)
	parent.SetChildren(append(parent.GetChildren(), pc))
	for _, child := range c.Commands() {
		pcFromCommands(pc, child)
	}
}

func inheritContext(c *cobra.Command) {
	for _, child := range c.Commands() {
		child.Context = c.Context
		inheritContext(child)
	}
}

func runShell() {
	completer := readline.NewPrefixCompleter()
	for _, child := range rootCmd.Commands() {
		pcFromCommands(completer, child)
	}
	shell, err := readline.NewEx(&readline.Config{
		Prompt:       "> ",
		AutoComplete: completer,
		EOFPrompt:    "exit",
	})
	if err != nil {
		panic(err)
	}
	defer shell.Close()

shell_loop:
	for {
		l, err := shell.Readline()
		if err != nil {
			break shell_loop
		}
		cmd, flags, err := rootCmd.Find(strings.Fields(l))
		if err != nil {
			shell.Terminal.Write([]byte(err.Error()))
		}
		cmd.ParseFlags(flags)
		cmd.Run(cmd, flags)
	}

}

func DefaultDataDir() string {
	home := homeDir()
	if home != "" {
		return filepath.Join(home, ".coschain")
	}
	return ""
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}

func addCommands() {
	rootCmd.AddCommand(commands.CreateCmd())
	rootCmd.AddCommand(commands.LoadCmd())
	rootCmd.AddCommand(commands.UnlockCmd())
	rootCmd.AddCommand(commands.LockCmd())
	rootCmd.AddCommand(commands.IsLockedCmd())
	rootCmd.AddCommand(commands.ListCmd())
	rootCmd.AddCommand(commands.InfoCmd())
	rootCmd.AddCommand(commands.CloseCmd())
	rootCmd.AddCommand(commands.AccountCmd())
	rootCmd.AddCommand(commands.GenKeyPairCmd())
}

func init() {
	addCommands()
	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		runShell()
	}
}

func main() {
	localWallet := wallet.NewBaseWallet("default", DefaultDataDir())
	localWallet.Start()
	rootCmd.SetContext("wallet", localWallet)
	defer localWallet.Close()

	conn, err := rpc.Dial("localhost:8888")
	defer conn.Close()
	if err != nil {
		common.Fatalf("Chain should have been run first")
	} else {
		rootCmd.SetContext("rpcclient", grpcpb.NewApiServiceClient(conn))
	}

	inheritContext(rootCmd)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
