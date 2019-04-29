package main

import (
	"context"
	"fmt"
	"github.com/coschain/contentos-go/rpc"
	"github.com/coschain/contentos-go/rpc/pb"
)

func main() {
	conn, _ := rpc.Dial("localhost:8888")
	defer conn.Close()
	client := grpcpb.NewApiServiceClient(conn)
	//req := &grpcpb.NonParamsRequest{}
	//resp, err := rpc.GetChainState(context.Background(), req)
	req := &grpcpb.GetDAUStatsRequest{Days: 30}
	resp, err := client.GetDAUStats(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(resp)
		for _, r := range resp.Stat {
			fmt.Println(r.Pg)
			fmt.Println(r.Ct)
			fmt.Println(r.G2)
			fmt.Println(r.Ec)
		}
	}
}
