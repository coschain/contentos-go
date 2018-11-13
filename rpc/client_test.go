package rpc

import (
	"bytes"
	"context"
	"fmt"
	"github.com/coschain/contentos-go/common/logging"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/mock_grpcpb"
	"github.com/coschain/contentos-go/rpc/pb"
	"github.com/golang/mock/gomock"
	"io/ioutil"
	"net/http"
	"testing"
)

func TestMockGRPCApi_GetAccountByName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock_grpcpb.NewMockApiServiceClient(ctrl)

	{
		req := &grpcpb.GetAccountByNameRequest{}
		resp := &grpcpb.AccountResponse{}
		expected := &grpcpb.AccountResponse{AccountName: &prototype.AccountName{Value: "Jack"}}
		client.EXPECT().GetAccountByName(gomock.Any(), gomock.Any()).Return(expected, nil)

		resp, err := client.GetAccountByName(context.Background(), req)
		if err != nil {
			t.Errorf("GetAccountByName failed: %x", err)
		} else {
			t.Logf("GetAccountByName detail: %s", resp.AccountName)
		}
	}
}

func TestMockGPRCApi_GetFollowerListByName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock_grpcpb.NewMockApiServiceClient(ctrl)

	{
		req := &grpcpb.GetFollowerListByNameRequest{}
		resp := &grpcpb.GetFollowerListByNameResponse{}

		expected := &grpcpb.GetFollowerListByNameResponse{}
		client.EXPECT().GetFollowerListByName(gomock.Any(), gomock.Any()).Return(expected, nil)

		resp, err := client.GetFollowerListByName(context.Background(), req)
		if err != nil {
			t.Errorf("GetFollowerListByName failed: %x", err)
		} else {
			t.Logf("GetFollowerListByName detail: %s", resp.FollowerList)
		}
	}
}

func TestGRPCApi_GetAccountByName(t *testing.T) {
	conn, err := Dial("127.0.0.1:8888")
	if err != nil {
		fmt.Print(err)
	}
	defer conn.Close()

	asc := grpcpb.NewApiServiceClient(conn)

	req := &grpcpb.GetAccountByNameRequest{AccountName: &prototype.AccountName{Value: "Jack"}}
	resp := &grpcpb.AccountResponse{}
	resp, err = asc.GetAccountByName(context.Background(), req)

	if err != nil {
		t.Errorf("GetAccountByName failed: %x", err)
	} else {
		t.Logf("GetAccountByName detail: %s", resp.AccountName)
	}
}

func TestGPRCApi_GetFollowerListByName(t *testing.T) {
	conn, err := Dial("127.0.0.1:8888")
	if err != nil {
		fmt.Print(err)
	}
	defer conn.Close()

	asc := grpcpb.NewApiServiceClient(conn)

	req := &grpcpb.GetFollowerListByNameRequest{}
	resp := &grpcpb.GetFollowerListByNameResponse{}
	resp, err = asc.GetFollowerListByName(context.Background(), req)
	if err != nil {
		t.Errorf("GetFollowerListByName failed: %x", err)
	} else {
		t.Logf("GetFollowerListByName detail: %s", resp.FollowerList)
	}
}

func TestHTTPApi_GetAccountByName(t *testing.T) {
	postValue := "{\"account_name\": {\"value\":\"jack's test info\"}}"
	http_client("POST", "http://127.0.0.1:8080/v1/user/get_account_by_name", postValue)
}

func TestHTTPApi_GetWitnessList(t *testing.T) {
	http_client("GET", "http://127.0.0.1:8080/v1/user/get_witness_list?page=1&size=5", "")
}

func http_client(rtype, url, reqJson string) error {
	req, err := http.NewRequest(rtype, url, bytes.NewBuffer([]byte(reqJson)))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	logging.CLog().Println("response Body:", string(body))

	return nil
}
