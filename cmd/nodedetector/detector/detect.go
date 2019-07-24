package detector

import (
	"fmt"
	"context"
	"github.com/coschain/contentos-go/rpc"
	"github.com/coschain/contentos-go/rpc/pb"
	"google.golang.org/grpc"
)

func RequireNodeInfo(endPoint string) {
	var conn *grpc.ClientConn
	conn, err := rpc.Dial(endPoint)

	if err == nil && conn != nil {
		api := grpcpb.NewApiServiceClient(conn)
		resp, err := api.GetNodeRunningVersion(context.Background(), &grpcpb.NonParamsRequest{})

		if err == nil {
			fmt.Printf("Endpoint: %s, Node version: %s\n", endPoint, resp.NodeVersion)
		}
	}
}