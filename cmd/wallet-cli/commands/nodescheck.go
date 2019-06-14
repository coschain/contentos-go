package commands

import (
	"context"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/rpc"
	"github.com/coschain/contentos-go/rpc/pb"
	"google.golang.org/grpc"
	"strings"
)

var rpclist string

var NodesCheckCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nodescheck",
		Short: "nodescheck",
		Run:   nodesCheck,
	}

	cmd.Flags().StringVarP(&rpclist, "rpc", "r", "localhost:8888", `nodescheck --rpc xxx:xxx,xxx:xxx`)

	return cmd
}

func nodesCheck(cmd *cobra.Command, args []string) {

	defer func() {
		rpclist = "localhost:8888"
	}()

	nodeslist := strings.Split(rpclist, ",")

	for i:=0; i<len(nodeslist); i++ {
		var conn *grpc.ClientConn
		conn, err := rpc.Dial(nodeslist[i])
		if err == nil && conn != nil {
			api := grpcpb.NewApiServiceClient(conn)
			resp, err := api.GetChainState(context.Background(), &grpcpb.NonParamsRequest{})

			if err == nil {
				fmt.Printf("Success peer:%v, Irreversible: %v, HeadBlockId: %v, HeadHash: %v\n", nodeslist[i],
					resp.State.GetLastIrreversibleBlockNumber(),
					resp.State.Dgpo.HeadBlockNumber,
					resp.State.Dgpo.HeadBlockId.ToString())
			}
			conn.Close()
		}

	}
}
