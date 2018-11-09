package main

import (
	"fmt"
	"github.com/chzyer/readline"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet/commands"
	"github.com/coschain/contentos-go/cmd/wallet/wallet"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

var rootCmd = &cobra.Command{
	Use:   "wallet",
	Short: "wallet is a key-pair storage",
}

func pcFromCommands(parent readline.PrefixCompleterInterface, c *cobra.Command) {
	pc := readline.PcItem(c.Use)
	parent.SetChildren(append(parent.GetChildren(), pc))
	for _, child := range c.Commands() {
		pcFromCommands(pc, child)
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
		//cmd.Context = rootCmd.Context
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
}

func init() {

	addCommands()

	localWallet := wallet.NewBaseWallet("default", DefaultDataDir())

	rootCmd.SetContext("wallet", localWallet)
	for _, cmd := range rootCmd.Commands() {
		cmd.Context = rootCmd.Context
	}

	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		runShell()
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
