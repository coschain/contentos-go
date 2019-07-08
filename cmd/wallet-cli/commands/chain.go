package commands

import (
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/prototype"
)

var ChainCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "chain",
		Short:   "chain <main|test|dev>",
		Example: "chain test",
		Args:    cobra.ExactArgs(1),
		Run:     chainSwitch,
	}
	return cmd
}

func chainSwitch(cmd *cobra.Command, args []string) {
	chainName := args[0]
	if len(chainName) == 0 {
		chainName = "main"
	}
	chainId := prototype.ChainId{ Value:common.GetChainIdByName(chainName) }
	cmd.SetContext("chain_name", chainName)
	cmd.SetContext("chain_id", chainId)
}
