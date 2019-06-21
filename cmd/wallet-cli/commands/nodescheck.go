package commands

import (
	"context"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/rpc"
	"github.com/coschain/contentos-go/rpc/pb"
	"google.golang.org/grpc"
	"strings"
	"sync"
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
	var wg sync.WaitGroup

	for i:=0; i<len(nodeslist); i++ {
		wg.Add(1)
		go func (idx int) {
			defer wg.Done()

			var conn *grpc.ClientConn
			conn, err := rpc.Dial(nodeslist[idx])
			if err == nil && conn != nil {
				api := grpcpb.NewApiServiceClient(conn)
				resp, err := api.GetChainState(context.Background(), &grpcpb.NonParamsRequest{})

				if err == nil {
					fmt.Printf("Success peer:%v, Irreversible: %v, HeadBlockId: %v, HeadHash: %v\n", nodeslist[idx],
						resp.State.GetLastIrreversibleBlockNumber(),
						resp.State.Dgpo.HeadBlockNumber,
						resp.State.Dgpo.HeadBlockId.ToString())
				} else {
					fmt.Printf("Failed peer: %v, Get response error %v\n", nodeslist[idx], err)
				}
				conn.Close()
			} else {
				fmt.Printf("Failed peer: %v, Connect peer error %v\n", nodeslist[idx], err)
			}
		}(i)
	}
	wg.Wait()
}
