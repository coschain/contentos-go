package rpc

import (
	"github.com/coschain/contentos-go/proto/type-proto"
	"github.com/coschain/contentos-go/rpc/pb"
	context "golang.org/x/net/context"
)

type APIService struct {
	server *GRPCServer
}

func (as *APIService) GetAccountByName(ctx context.Context, req *grpctype.GetAccountByNameRequest) (*grpctype.AccountResponse, error) {
	account := &grpctype.AccountResponse{AccountName: &prototype.AccountName{Value: "Jack"}}
	return account, nil
}

func (as *APIService) GetFollowerListByName(ctx context.Context, req *grpctype.GetFollowerListByNameRequest) (*grpctype.GetFollowerListByNameResponse, error) {
	return nil, nil
}

func (as *APIService) GetFollowingListByName(ctx context.Context, req *grpctype.GetFollowingListByNameRequest) (*grpctype.GetFollowingListByNameResponse, error) {
	return nil, nil
}

func (as *APIService) GetWitnessList(ctx context.Context, req *grpctype.GetWitnessListRequest) (*grpctype.GetWitnessListResponse, error) {
	return nil, nil
}

func (as *APIService) GetPostListByCreated(ctx context.Context, req *grpctype.GetPostListByCreatedRequest) (*grpctype.GetPostListByCreatedResponse, error) {
	return nil, nil
}

func (as *APIService) GetReplayListByPostId(ctx context.Context, req *grpctype.GetReplayListByPostIdRequest) (*grpctype.GetReplayListByPostIdResponse, error) {
	return nil, nil
}

func (as *APIService) GetBlockTransactionsByNum(ctx context.Context, req *grpctype.GetBlockTransactionsByNumRequest) (*grpctype.GetBlockTransactionsByNumResponse, error) {
	return nil, nil
}

func (as *APIService) GetTrxById(ctx context.Context, req *grpctype.GetTrxByIdRequest) (*grpctype.GetTrxByIdResponse, error) {
	return nil, nil
}

func (as *APIService) BroadcastTrx(ctx context.Context, req *grpctype.BroadcastTrxRequest) (*grpctype.BroadcastTrxResponse, error) {
	return nil, nil
}

