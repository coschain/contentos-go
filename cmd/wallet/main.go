package main

import (
	"fmt"
	"github.com/chzyer/readline"
	"github.com/coschain/contentos-go/cmd/wallet/commands"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

var rootCmd = &cobra.Command{
	Use:   "wallet",
	Short: "wallet is a key-pair storage",
	//Run: func(cmd *cobra.Command, args []string) {
	//	runShell()
	//},
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
		cmd.Run(cmd, flags)
	}

}

func addCommands() {
	rootCmd.AddCommand(commands.InitCmd())
}

func init() {
	addCommands()
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
