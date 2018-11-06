package rpc

import (
	"github.com/coschain/contentos-go/common/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
	context "golang.org/x/net/context"
)

type APIService struct {
	server RPCServer
}

func (as *APIService) GetAccountByName(ctx context.Context, req *grpcpb.GetAccountByNameRequest) (*grpcpb.AccountResponse, error) {
	account := &grpcpb.AccountResponse{AccountName: &prototype.AccountName{Value: req.AccountName.Value}}
	return account, nil
}

func (as *APIService) GetFollowerListByName(ctx context.Context, req *grpcpb.GetFollowerListByNameRequest) (*grpcpb.GetFollowerListByNameResponse, error) {
	return &grpcpb.GetFollowerListByNameResponse{}, nil
}

func (as *APIService) GetFollowingListByName(ctx context.Context, req *grpcpb.GetFollowingListByNameRequest) (*grpcpb.GetFollowingListByNameResponse, error) {
	return &grpcpb.GetFollowingListByNameResponse{}, nil
}

func (as *APIService) GetWitnessList(ctx context.Context, req *grpcpb.GetWitnessListRequest) (*grpcpb.GetWitnessListResponse, error) {

	return &grpcpb.GetWitnessListResponse{WitnessList: []*grpcpb.WitnessResponse{&grpcpb.WitnessResponse{Url: "test url", ScheduleType: req.Page}}}, nil
}

func (as *APIService) GetPostListByCreated(ctx context.Context, req *grpcpb.GetPostListByCreatedRequest) (*grpcpb.GetPostListByCreatedResponse, error) {
	return &grpcpb.GetPostListByCreatedResponse{}, nil
}

func (as *APIService) GetReplayListByPostId(ctx context.Context, req *grpcpb.GetReplayListByPostIdRequest) (*grpcpb.GetReplayListByPostIdResponse, error) {
	return &grpcpb.GetReplayListByPostIdResponse{}, nil
}

func (as *APIService) GetBlockTransactionsByNum(ctx context.Context, req *grpcpb.GetBlockTransactionsByNumRequest) (*grpcpb.GetBlockTransactionsByNumResponse, error) {
	return &grpcpb.GetBlockTransactionsByNumResponse{}, nil
}

func (as *APIService) GetTrxById(ctx context.Context, req *grpcpb.GetTrxByIdRequest) (*grpcpb.GetTrxByIdResponse, error) {
	return &grpcpb.GetTrxByIdResponse{}, nil
}

func (as *APIService) BroadcastTrx(ctx context.Context, req *grpcpb.BroadcastTrxRequest) (*grpcpb.BroadcastTrxResponse, error) {
	return &grpcpb.BroadcastTrxResponse{}, nil
}
