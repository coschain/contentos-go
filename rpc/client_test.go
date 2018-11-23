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
	"os"
	"testing"
)

var asc grpcpb.ApiServiceClient

func TestMain(m *testing.M) {
	logging.Init("logs	", "debug", 0)

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
			t.Logf("GetAccountByName failed: %x", err)
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
			t.Logf("GetFollowerListByName failed: %x", err)
		} else {
			t.Logf("GetFollowerListByName detail: %s", resp.FollowerList)
		}
	}
}

func TestMockGRPCApi_GetFollowingListByName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock_grpcpb.NewMockApiServiceClient(ctrl)

	{
		req := &grpcpb.GetFollowingListByNameRequest{}
		resp := &grpcpb.GetFollowingListByNameResponse{}

		expected := &grpcpb.GetFollowingListByNameResponse{}
		client.EXPECT().GetFollowingListByName(gomock.Any(), gomock.Any()).Return(expected, nil)

		resp, err := client.GetFollowingListByName(context.Background(), req)
		if err != nil {
			t.Logf("GetFollowingListByName failed: %x", err)
		} else {
			t.Logf("GetFollowingListByName detail: %s", resp.FollowingList)
		}
	}
}

func TestMockGPRCApi_GetWitnessList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock_grpcpb.NewMockApiServiceClient(ctrl)

	{
		req := &grpcpb.GetWitnessListRequest{}
		resp := &grpcpb.GetWitnessListResponse{}

		expected := &grpcpb.GetWitnessListResponse{}
		client.EXPECT().GetWitnessList(gomock.Any(), gomock.Any()).Return(expected, nil)

		resp, err := client.GetWitnessList(context.Background(), req)
		if err != nil {
			t.Logf("GetWitnessListByName failed: %x", err)
		} else {
			t.Logf("GetWitnessListByName detail: %s", resp.WitnessList)
		}
	}
}

func TestGRPCApi_GetAccountByName(t *testing.T) {
	req := &grpcpb.GetAccountByNameRequest{AccountName: &prototype.AccountName{Value: "Jack"}}
	resp := &grpcpb.AccountResponse{}
	resp, err := asc.GetAccountByName(context.Background(), req)

	if err != nil {
		t.Errorf("GetAccountByName failed: err:[%s], resp:[%x]", err, resp)
	} else {
		t.Logf("GetAccountByName detail: %s", resp.AccountName)
	}
}

func TestGPRCApi_GetFollowerListByName(t *testing.T) {
	req := &grpcpb.GetFollowerListByNameRequest{Limit: 100, Start: &prototype.FollowerCreatedOrder{Account: &prototype.AccountName{Value: "Jack"}}}
	resp := &grpcpb.GetFollowerListByNameResponse{}
	resp, err := asc.GetFollowerListByName(context.Background(), req)
	if err != nil {
		t.Errorf("GetFollowerListByName failed: %s", err)
	} else {
		t.Logf("GetFollowerListByName detail: %s", resp.FollowerList)
	}
}

func TestGPRCApi_GetFollowingListByName(t *testing.T) {
	req := &grpcpb.GetFollowingListByNameRequest{Limit: 100, Start: &prototype.FollowingCreatedOrder{Account: &prototype.AccountName{Value: "Jack"}}}
	resp := &grpcpb.GetFollowingListByNameResponse{}
	resp, err := asc.GetFollowingListByName(context.Background(), req)
	if err != nil {
		t.Errorf("GetFollowingListByName failed: %s", err)
	} else {
		t.Logf("GetFollowingListByName detail: %s", resp.FollowingList)
	}
}

func TestGPRCApi_GetWitnessList(t *testing.T) {
	req := &grpcpb.GetWitnessListRequest{Limit: 100}
	resp := &grpcpb.GetWitnessListResponse{}
	resp, err := asc.GetWitnessList(context.Background(), req)
	if err != nil {
		t.Errorf("GetWitnessList failed: %s", err)
	} else {
		t.Logf("GetWitnessList detail: %s", resp.WitnessList)
	}
}

func TestGRPCApi_GetPostListByCreated(t *testing.T) {
	req := &grpcpb.GetPostListByCreatedRequest{}
	resp := &grpcpb.GetPostListByCreatedResponse{}

	resp, err := asc.GetPostListByCreated(context.Background(), req)
	if err != nil {
		t.Errorf("GetPostListByCreated failed: %s", err)
	} else {
		t.Logf("GetPostListByCreated detail: %s", resp.PostList)
	}
}

func TestGRPCApi_GetReplyListByPostId(t *testing.T) {
	req := &grpcpb.GetReplyListByPostIdRequest{}
	resp := &grpcpb.GetReplyListByPostIdResponse{}

	resp, err := asc.GetReplyListByPostId(context.Background(), req)
	if err != nil {
		t.Errorf("GetReplyListByPostId failed: %s", err)
	} else {
		t.Logf("GetReplyListByPostId detail: %s", resp.ReplyList)
	}
}

func TestGRPCApi_GetChainState(t *testing.T) {
	req := &grpcpb.NonParamsRequest{}
	resp := &grpcpb.GetChainStateResponse{}

	resp, err := asc.GetChainState(context.Background(), req)
	if err != nil {
		t.Errorf("GetChainState failed: %s", err)
	} else {
		t.Logf("GetChainState detail: %s", resp.Props)
	}
}

func TestGRPCApi_GetBlockTransactionsByNum(t *testing.T) {
	req := &grpcpb.GetBlockTransactionsByNumRequest{}
	resp := &grpcpb.GetBlockTransactionsByNumResponse{}

	resp, err := asc.GetBlockTransactionsByNum(context.Background(), req)
	if err != nil {
		t.Errorf("GetChainState failed: %s", err)
	} else {
		t.Logf("GetChainState detail: %s", resp.Transactions)
	}
}

func TestGRPCApi_GetTrxById(t *testing.T) {
	req := &grpcpb.GetTrxByIdRequest{}
	resp := &grpcpb.GetTrxByIdResponse{}

	resp, err := asc.GetTrxById(context.Background(), req)
	if err != nil {
		t.Errorf("GetTrxById failed: %s", err)
	} else {
		t.Logf("GetTrxById detail: %s", resp.Trx)
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
