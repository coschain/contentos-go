package main

import (
	"fmt"
	"github.com/chzyer/readline"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands"
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands/utils"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/rpc"
	"github.com/coschain/contentos-go/rpc/pb"
	"os"
	"os/user"
	"path/filepath"
)

var rootCmd = &cobra.Command{
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
		HistoryFile:  filepath.Join(DefaultDataDir(), "cmd_input.history"),
	})
	if err != nil {
		panic(err)
	}
	defer shell.Close()

	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
			runShell()
		}
	}()

shell_loop:
	for {
		l, err := shell.Readline()
		if err != nil {
			break shell_loop
		}
		argv, err := ToArgv(l)
		if err != nil {
			fmt.Println(err)
			continue
		}
		cmd, flags, err := rootCmd.Find(argv)
		if err != nil {
			shell.Terminal.Write([]byte(err.Error()))
		}
		cmd.InitDefaultHelpFlag()
		cmd.InitDefaultVersionFlag()
		err = cmd.ParseFlags(flags)
		if err != nil {
			fmt.Println("parse flags error")
			continue
		}

		// If help is called, regardless of other flags, return we want help.
		// Also say we need help if the command isn't runnable.
		helpVal, err := cmd.Flags().GetBool("help")
		if err != nil {
			fmt.Println("\"help\" flag declared as non-bool. Please correct your code")
			continue
		}

		if helpVal {
			cmd.UsageFunc()(cmd)
			cmd.Flags().Lookup("help").Value.Set("false")
			continue
		}
		argWoFlags := cmd.Flags().Args()
		if err := cmd.ValidateArgs(argWoFlags); err != nil {
			fmt.Println(err)
			continue
		}
		cmd.Run(cmd, argWoFlags)
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
	rootCmd.AddCommand(commands.LoadAllCmd())
	rootCmd.AddCommand(commands.IsLockedCmd())
	rootCmd.AddCommand(commands.ListCmd())
	rootCmd.AddCommand(commands.InfoCmd())
	rootCmd.AddCommand(commands.CloseCmd())
	rootCmd.AddCommand(commands.AccountCmd())
	rootCmd.AddCommand(commands.GenKeyPairCmd())
	rootCmd.AddCommand(commands.TransferCmd())
	rootCmd.AddCommand(commands.TransferVestingCmd())
	rootCmd.AddCommand(commands.VoteCmd())
	rootCmd.AddCommand(commands.ImportCmd())
	rootCmd.AddCommand(commands.BpCmd())
	rootCmd.AddCommand(commands.PostCmd())
	rootCmd.AddCommand(commands.ReplyCmd())
	rootCmd.AddCommand(commands.FollowCmd())
	rootCmd.AddCommand(commands.FollowCntCmd())
	rootCmd.AddCommand(commands.MultinodetesterCmd())
	rootCmd.AddCommand(commands.SwitchPortcmd())
	rootCmd.AddCommand(commands.ChainStateCmd())
	rootCmd.AddCommand(commands.StressCmd())
	rootCmd.AddCommand(commands.StressCreAccountCmd())
	rootCmd.AddCommand(commands.StressVMCmd())
	rootCmd.AddCommand(commands.ClaimAllCmd())
	rootCmd.AddCommand(commands.ClaimCmd())
	rootCmd.AddCommand(commands.DeployCmd())
	rootCmd.AddCommand(commands.CallCmd())
	rootCmd.AddCommand(commands.EstimateCmd())
}

func init() {
	addCommands()
	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		runShell()
	}
}

func main() {
	localWallet := wallet.NewBaseWallet("default", DefaultDataDir())
	preader := utils.MyPasswordReader{}
	localWallet.LoadAll()
	localWallet.Start()
	rootCmd.SetContext("wallet", localWallet)
	rootCmd.SetContext("preader", preader)
	defer localWallet.Close()

	conn, err := rpc.Dial("localhost:8888")
	defer conn.Close()
	if err != nil {
		common.Fatalf("Chain should have been run first")
	} else {
		rootCmd.SetContext("rpcclient", grpcpb.NewApiServiceClient(conn))
		// for switch port
		rootCmd.SetContext("rpcclient_raw", conn)
	}

	inheritContext(rootCmd)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
