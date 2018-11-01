package rpc

import "google.golang.org/grpc"

type Server struct {
	rpcServer *grpc.Server
}
