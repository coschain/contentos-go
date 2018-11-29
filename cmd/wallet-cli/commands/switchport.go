package commands

import (
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/rpc"
	"github.com/coschain/contentos-go/rpc/pb"
	"google.golang.org/grpc"
	"strconv"
)

var SwitchPortcmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "switchport",
		Short:   "switchport port",
		Example: "switchport port",
		Args:    cobra.ExactArgs(1),
		Run:     switchport,
	}
	return cmd
}

func switchport(cmd *cobra.Command, args []string) {

	port, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fmt.Println(err)
		return
	}

	conn, err := rpc.Dial(fmt.Sprintf("localhost:%d", port))
	if err != nil {
		common.Fatalf("Chain should have been run first")
	} else {

		c := cmd.Context["rpcclient_raw"]
		client := c.(*grpc.ClientConn)
		client.Close()
		cmd.SetContext("rpcclient_raw", conn )
		cmd.SetContext("rpcclient", grpcpb.NewApiServiceClient(conn))
	}

}
