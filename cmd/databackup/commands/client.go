package commands

import (
	"github.com/coschain/cobra"
)

var serverPort int16
var serverIP string

var ClientCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "client",
		Short: "start backup client",
		Run:   startBackUpClient,
	}
	cmd.Flags().Int16VarP(&serverPort, "server_port", "p", 9722, "")
	cmd.Flags().StringVarP(&serverIP, "server_ip", "i", "", "")
	return cmd
}

func startBackUpClient(cmd *cobra.Command, args []string) {}
