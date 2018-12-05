package rpc

import (
	"google.golang.org/grpc"
)

func Dial(target string) (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(target, grpc.WithInsecure())
	if err != nil {
		//logging.VLog().Error("rpc.Dial() failed: ", err)
	}
	return conn, err
}
