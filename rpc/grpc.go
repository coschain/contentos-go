package rpc

import (
	"context"
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/common/eventloop"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
)

type APIService struct {
	ctrl     iservices.IController
	mainLoop *eventloop.EventLoop
	db       iservices.IDatabaseService
}

func (as *APIService) GetAccountByName(ctx context.Context, req *grpcpb.GetAccountByNameRequest) (*grpcpb.AccountResponse, error) {

	accWrap := table.NewSoAccountWrap(as.db, req.AccountName)
	acct := &grpcpb.AccountResponse{AccountName: &prototype.AccountName{Value: req.AccountName.Value}}

	if accWrap.CheckExist() {
		acct.AccountName = &prototype.AccountName{Value: accWrap.GetName().Value}
		acct.Coin = accWrap.GetBalance()
		acct.Vest = accWrap.GetVestingShares()
		//acct.PublicKeys = accWrap.GetPubKey()
		acct.CreatedTime = accWrap.GetCreatedTime()
	}

	return acct, nil
}

func (as *APIService) GetFollowerListByName(ctx context.Context, req *grpcpb.GetFollowerListByNameRequest) (*grpcpb.GetFollowerListByNameResponse, error) {

	var (
		ferIter iservices.IDatabaseIterator
		ferList []*prototype.AccountName
		i       uint32
		limit   uint32
	)

	ferOrderWrap := &table.SFollowerFollowerInfoWrap{Dba: as.db}

	if req.Start == nil {
		ferIter = ferOrderWrap.QueryListByRevOrder(nil, nil)
	} else {
		ferIter = ferOrderWrap.QueryListByRevOrder(req.Start, nil)
	}

	if req.Limit <= constants.RPC_PAGE_SIZE_LIMIT {
		limit = req.Limit
	} else {
		limit = constants.RPC_PAGE_SIZE_LIMIT
	}

	for ferIter.Next() {
		ferOrder := ferOrderWrap.GetSubVal(ferIter)
		if ferOrder != nil {
			ferList = append(ferList, ferOrder.Follower)
		} else {
			ferList = append(ferList, &prototype.AccountName{})
		}

		i++

		if i < limit {
			break
		}
	}

	return &grpcpb.GetFollowerListByNameResponse{FollowerList: ferList}, nil
}

func (as *APIService) GetFollowingListByName(ctx context.Context, req *grpcpb.GetFollowingListByNameRequest) (*grpcpb.GetFollowingListByNameResponse, error) {

	var (
		fingIter iservices.IDatabaseIterator
		fingList []*prototype.AccountName
		i        uint32
		limit    uint32
	)

	fingOrderWrap := &table.SFollowingFollowingInfoWrap{Dba: as.db}

	if req.Start == nil {
		fingIter = fingOrderWrap.QueryListByRevOrder(nil, nil)
	} else {
		fingIter = fingOrderWrap.QueryListByRevOrder(req.Start, nil)
	}

	if req.Limit <= constants.RPC_PAGE_SIZE_LIMIT {
		limit = req.Limit
	} else {
		limit = constants.RPC_PAGE_SIZE_LIMIT
	}

	for fingIter.Next() {
		ferOrder := fingOrderWrap.GetSubVal(fingIter)
		if ferOrder != nil {
			fingList = append(fingList, ferOrder.Following)
		} else {
			fingList = append(fingList, &prototype.AccountName{})
		}

		i++

		if i < limit {
			break
		}
	}

	return &grpcpb.GetFollowingListByNameResponse{}, nil
}

func (as *APIService) GetFollowCountByName(ctx context.Context, req *grpcpb.GetFollowCountByNameRequest) (*grpcpb.GetFollowCountByNameResponse, error) {

	var (
		ferCnt, fingCnt uint32
	)

	afc := table.NewSoFollowCountWrap(as.db, req.AccountName)

	if afc.CheckExist() {
		ferCnt = afc.GetFollowerCnt()
		fingCnt = afc.GetFollowingCnt()
	}

	return &grpcpb.GetFollowCountByNameResponse{FerCnt: ferCnt, FingCnt: fingCnt}, nil
}

func (as *APIService) GetWitnessList(ctx context.Context, req *grpcpb.GetWitnessListRequest) (*grpcpb.GetWitnessListResponse, error) {
	var (
		witIter iservices.IDatabaseIterator
		witList []*grpcpb.WitnessResponse
		i        uint32
		limit    uint32
	)

	witOrderWrap := &table.SWitnessOwnerWrap{as.db}

	if req.Start == nil {
		witIter = witOrderWrap.QueryListByOrder(nil, nil)
	} else {
		witIter = witOrderWrap.QueryListByOrder(req.Start, nil)
	}

	if req.Limit <= constants.RPC_PAGE_SIZE_LIMIT {
		limit = req.Limit
	} else {
		limit = constants.RPC_PAGE_SIZE_LIMIT
	}

	for witIter.Next() {
		witWrap := table.NewSoWitnessWrap(as.db, witOrderWrap.GetMainVal(witIter))
		if witWrap.CheckExist() {
			witList = append(witList, &grpcpb.WitnessResponse{
				Owner:witWrap.GetOwner(),
				WitnessScheduleType:witWrap.GetWitnessScheduleType(),
				CreatedTime:witWrap.GetCreatedTime(),
				Url:witWrap.GetUrl(),
				LastConfirmedBlockNum:witWrap.GetLastConfirmedBlockNum(),
				TotalMissed:witWrap.GetLastConfirmedBlockNum(),
				PowWorker:witWrap.GetPowWorker(),
				SigningKey:witWrap.GetSigningKey(),
				LastWork:witWrap.GetLastWork(),
				RunningVersion:witWrap.GetRunningVersion(),
			})
		}

		i++

		if i < limit {
			break
		}
	}

	return &grpcpb.GetWitnessListResponse{WitnessList: witList}, nil
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

	var result *prototype.TransactionInvoice = nil
	as.mainLoop.Send(func() {
		result = as.ctrl.PushTrx(req.GetTransaction())
		fmt.Println("BroadcastTrx Result:", result)
	})

	return &grpcpb.BroadcastTrxResponse{}, nil
}
