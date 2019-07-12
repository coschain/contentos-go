package commands

import (
	"github.com/coschain/cobra"
)

var dataDir string
var interval int32
var destAddr string

var AgentCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "start backup agent",
		Long:  "start back agent and regularly sends cosd data files to backup server",
		Run:   startBackUpAgent,
	}
	cmd.Flags().StringVarP(&dataDir, "data_dir", "d", "", "directory of cosd data")
	cmd.Flags().Int32VarP(&interval, "interval", "i", 3600, "backup data every interval seconds")
	cmd.Flags().StringVarP(&destAddr, "addr", "a", "", "the address of the backup server")
	return cmd
}

func startBackUpAgent(cmd *cobra.Command, args []string) {}