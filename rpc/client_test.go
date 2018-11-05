package rpc

import (
	"context"
	"fmt"
	"github.com/coschain/contentos-go/common/prototype"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/p2p"
	"github.com/coschain/contentos-go/rpc/pb"
	"os"
	"testing"
)

var asc grpcpb.ApiServiceClient

func TestMain(m *testing.M) {
	gs := NewGRPCServer(&node.ServiceContext{})
	err := gs.Start(&p2p.Server{})
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
	req := &grpcpb.GetAccountByNameRequest{AccountName: &prototype.AccountName{Value: "Jack"}}
	resp := &grpcpb.AccountResponse{}
	resp, err := asc.GetAccountByName(context.Background(), req)
	if err != nil {
		t.Errorf("GetAccountByName failed: %x", err)
	} else {
		t.Logf("GetAccountByName detail: %s", resp.AccountName)
	}
}

//func TestGRPCApi_GetAccountByName_Http(t *testing.T) {
//	v := url.Values{}
//	v.Set("account_name.value", "Jack")
//	body := ioutil.NopCloser(strings.NewReader(v.Encode()))
//	client := &http.Client{}
//	req, _ := http.NewRequest("POST", "http://127.0.0.1:8888/v1/user/get_account_by_name", body)
//
//	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
//	fmt.Printf("%+v\n", req)
//
//	resp, err := client.Do(req)
//	defer resp.Body.Close()
//	data, _ := ioutil.ReadAll(resp.Body)
//	fmt.Println(string(data), err)
//}

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
