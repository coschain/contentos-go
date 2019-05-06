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
	//req := &grpcpb.GetDAUStatsRequest{Days: 30, Dapp: "test"}
	req := &grpcpb.GetDNUStatsRequest{Days: 30, Dapp: "test"}
	resp, err := client.GetDNUStats(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(resp)
		for _, r := range resp.Stat {
			fmt.Println(r.Date)
			fmt.Println(r.Dapp)
			fmt.Println(r.Count)
		}
	}
}
