package commands

import (
	"context"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/rpc"
	"github.com/coschain/contentos-go/rpc/pb"
	"google.golang.org/grpc"
)

var NodesCheckCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nodescheck",
		Short: "nodescheck",
		Run:   nodesCheck,
	}

	return cmd
}

func nodesCheck(cmd *cobra.Command, args []string) {

	for port := 8888; port < 8888 + 22 ; port++ {
		var conn *grpc.ClientConn
		conn, err := rpc.Dial(fmt.Sprintf("localhost:%d", port))
		if err == nil && conn != nil {
			api := grpcpb.NewApiServiceClient(conn)
			resp, err := api.GetChainState(context.Background(), &grpcpb.NonParamsRequest{})

			if err == nil {
				fmt.Printf("Success port:%v, Irreversible: %v, HeadBlockId: %v, HeadHash: %v\n", port,
					resp.State.GetLastIrreversibleBlockNumber(),
					resp.State.Dgpo.HeadBlockNumber,
					resp.State.Dgpo.HeadBlockId.ToString())
			}
			conn.Close()
		}

	}
}
