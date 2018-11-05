package rpc

import (
	"google.golang.org/grpc"
	log "github.com/inconshreveable/log15"
)

func Dial(target string) (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(target, grpc.WithInsecure())
	if err != nil {
		log.Error("rpc.Dial() failed: ", err)
	}
	return conn, err
}
