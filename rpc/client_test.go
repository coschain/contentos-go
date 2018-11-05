package rpc

import (
	"context"
	"fmt"
	type_proto "github.com/coschain/contentos-go/common/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
	"os"

	"testing"
)

var asc grpcpb.ApiServiceClient

func TestMain(m *testing.M) {
	gs := NewGRPCServer()
	err := gs.Start()
	if err != nil {
		fmt.Print(err)
	}
	defer gs.Stop()

	addr := fmt.Sprintf("127.0.0.1:%d", uint32(8888))
	conn, err := Dial(addr)
	if err != nil {
		fmt.Print(err)
	}
	defer conn.Close()

	asc = grpcpb.NewApiServiceClient(conn)

	exitCode := m.Run()
	asc = nil
	os.Exit(exitCode)
}

func TestGRPCApi_GetAccountByName(t *testing.T) {
	req := &grpcpb.GetAccountByNameRequest{AccountName: &type_proto.AccountName{Value: "Jack"}}
	resp := &grpcpb.AccountResponse{}
	resp, err := asc.GetAccountByName(context.Background(), req)
	if err != nil {
		t.Errorf("GetAccountByName failed: %x", err)
	} else {
		t.Logf("GetAccountByName detail: %s", resp.AccountName)
	}
}

func TestGPRCApi_GetFollowerListByName(t *testing.T) {
	req := &grpcpb.GetFollowerListByNameRequest{}
	resp := &grpcpb.GetFollowerListByNameResponse{}
	resp, err := asc.GetFollowerListByName(context.Background(), req)
	if err != nil {
		t.Errorf("GetFollowerListByName failed: %x", err)
	} else {
		t.Logf("GetFollowerListByName detail: %s", resp.FollowerList)
	}
}
