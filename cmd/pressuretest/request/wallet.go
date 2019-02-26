package request

import (
	"errors"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/rpc"
	"github.com/coschain/contentos-go/rpc/pb"
	"math/rand"
	"time"
)

var IPList []string = []string{
	"localhost:8888",
}
var CmdRawStrlist []string = []string{
	"create initminer test%s",
	"transfer initminer initminer1 %d",
	"post initminer %s %s %s",
	"follow initminer initminer1",
}

func InitEnv() {
	rootCmd := MakeRootCmd()

	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		InitShell(rootCmd)
	}

	MakeWallet(0, rootCmd)
}


func MakeWallet(index int, rootCmd *cobra.Command) {
	localWallet := wallet.NewBaseWallet("default", fmt.Sprintf("./default_%d", index) )
	preader := MyMockPasswordReader{}
	localWallet.LoadAll()
	localWallet.Start()
	rootCmd.SetContext("wallet", localWallet)
	rootCmd.SetContext("preader", preader)
	//defer localWallet.Close()

	conn, err := rpc.Dial( IPList[ index%len(IPList) ] )
	defer conn.Close()
	if err != nil {
		common.Fatalf("Chain should have been run first")
	} else {
		rootCmd.SetContext("rpcclient", grpcpb.NewApiServiceClient(conn))
	}

	inheritContext(rootCmd)
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}

func MakeRootCmd() *cobra.Command {
	var rootCmd = &cobra.Command{
		Short: "pressuretest is a pressure test client",
	}
	addCommands(rootCmd)
	return rootCmd
}

func addCommands(rootCmd *cobra.Command) {
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
	rootCmd.AddCommand(commands.VmTableCmd())
}

func inheritContext(c *cobra.Command) {
	for _, child := range c.Commands() {
		child.Context = c.Context
		inheritContext(child)
	}
}

func InitShell(rootCmd *cobra.Command) {
	var cmdStr string
	for i:=0;i<2;i++ {
		if i == 0 {
			cmdStr = "import -f initminer " + constants.INITMINER_PRIKEY
		} else {
			cmdStr = "create initminer initminer1"
		}

		//fmt.Println("command: ", cmdStr)
		parseAndRun(cmdStr, rootCmd)
	}
}

func RunShell(rootCmd *cobra.Command) {
	var cmdStr string
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var index = r.Intn( len(CmdRawStrlist) )

	for i:=0;;i++{
		if i == 0 {
			cmdStr = "import -f initminer " + constants.INITMINER_PRIKEY
		} else {
			//r := rand.New(rand.NewSource(time.Now().UnixNano()))
			//idx := r.Intn(len(CmdRawStrlist))
			//rawStr := CmdRawStrlist[idx]
			rawStr := CmdRawStrlist[index]

			cmdStr = makeRequest(rawStr)
			if len(cmdStr) == 0 {
				panic(errors.New("CmdRawStrlist error: " + rawStr))
			}
			index += 1
			if index >= len(CmdRawStrlist) {
				index = 0
			}
		}

		//fmt.Println("command: ", cmdStr)
		parseAndRun(cmdStr, rootCmd)
	}
}

func parseAndRun(cmdStr string, rootCmd *cobra.Command) {
	argv, err := ToArgv(cmdStr)
	if err != nil {
		fmt.Println(err)
		return
	}
	cmd, flags, err := rootCmd.Find(argv)
	if err != nil {
		fmt.Println("can't find match command:", err)
		return
	}
	cmd.InitDefaultHelpFlag()
	cmd.InitDefaultVersionFlag()
	err = cmd.ParseFlags(flags)
	if err != nil {
		fmt.Println("parse flags error")
		return
	}

	// If help is called, regardless of other flags, return we want help.
	// Also say we need help if the command isn't runnable.
	helpVal, err := cmd.Flags().GetBool("help")
	if err != nil {
		fmt.Println("\"help\" flag declared as non-bool. Please correct your code")
		return
	}

	if helpVal {
		cmd.UsageFunc()(cmd)
		cmd.Flags().Lookup("help").Value.Set("false")
		return
	}
	argWoFlags := cmd.Flags().Args()
	if err := cmd.ValidateArgs(argWoFlags); err != nil {
		fmt.Println(err)
		return
	}
	cmd.Run(cmd, argWoFlags)
}