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

var plist string

var NodeNeighboursCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nodeNeighbours",
		Short: "nodeNeighbours",
		Run:   nodeNeighbours,
	}

	cmd.Flags().StringVarP(&plist, "rpc", "r", "localhost:8888", `nodeNeighbours --rpc xxx:xxx,xxx:xxx`)

	return cmd
}

func nodeNeighbours(cmd *cobra.Command, args []string) {
	defer func() {
		plist = "localhost:8888"
	}()

	nodeslist := strings.Split(plist, ",")
	var wg sync.WaitGroup

	for i:=0; i<len(nodeslist); i++ {
		wg.Add(1)
		go func (idx int) {
			defer wg.Done()

			var conn *grpc.ClientConn
			conn, err := rpc.Dial(nodeslist[idx])
			if err == nil && conn != nil {
				api := grpcpb.NewApiServiceClient(conn)
				resp, err := api.GetNodeNeighbours(context.Background(), &grpcpb.NonParamsRequest{})

				if err == nil {
					fmt.Printf("Success peer:%v, neighbour list: %s\n\n", nodeslist[idx],
						resp.Peerlist)
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