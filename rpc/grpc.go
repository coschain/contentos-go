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
		ferIter = ferOrderWrap.QueryListByOrder(nil, nil)
	} else {
		ferIter = ferOrderWrap.QueryListByOrder(req.Start, nil)
	}

	if req.Limit <= constants.RPC_PAGE_SIZE_LIMIT {
		limit = req.Limit
	} else {
		limit = constants.RPC_PAGE_SIZE_LIMIT
	}

	for ferIter.Next() {
		ferOrder := ferOrderWrap.GetMainVal(ferIter)
		if ferOrder != nil {
			ferList = append(ferList, ferOrder.Follower)
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
		fingIter = fingOrderWrap.QueryListByOrder(nil, nil)
	} else {
		fingIter = fingOrderWrap.QueryListByOrder(req.Start, nil)
	}

	if req.Limit <= constants.RPC_PAGE_SIZE_LIMIT {
		limit = req.Limit
	} else {
		limit = constants.RPC_PAGE_SIZE_LIMIT
	}

	for fingIter.Next() {
		fingOrder := fingOrderWrap.GetMainVal(fingIter)
		if fingOrder != nil {
			fingList = append(fingList, fingOrder.Following)
		} 

		i++

		if i < limit {
			break
		}
	}

	return &grpcpb.GetFollowingListByNameResponse{FollowingList:fingList}, nil
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
	var (
		postIter iservices.IDatabaseIterator
		postList []*grpcpb.PostResponse
		i        uint32
		limit    uint32
	)

	postOrderWrap := &table.SPostCreatedOrderWrap{Dba:as.db}

	if req.Start == nil {
		postIter = postOrderWrap.QueryListByRevOrder(nil, nil)
	} else {
		postIter = postOrderWrap.QueryListByRevOrder(req.Start, nil)
	}

	if req.Limit <= constants.RPC_PAGE_SIZE_LIMIT {
		limit = req.Limit
	} else {
		limit = constants.RPC_PAGE_SIZE_LIMIT
	}

	for postIter.Next() {
		postWrap := table.NewSoPostWrap(as.db, postOrderWrap.GetMainVal(postIter))
		if postWrap.CheckExist() {
			postList = append(postList, &grpcpb.PostResponse{
				PostId:postWrap.GetPostId(),
				Category:postWrap.GetCategory(),
				ParentAuthor:postWrap.GetAuthor(),
				ParentPermlink:postWrap.GetPermlink(),
				Author:postWrap.GetAuthor(),
				Permlink:postWrap.GetPermlink(),
				Title:postWrap.GetTitle(),
				Body:postWrap.GetBody(),
				JsonMetadata:postWrap.GetJsonMetadata(),
				LastUpdate:postWrap.GetLastUpdate(),
				Created:postWrap.GetCreated(),
				Active:postWrap.GetActive(),
				LastPayout:postWrap.GetLastPayout(),
				Depth:postWrap.GetDepth(),
				Children:postWrap.GetChildren(),
				RootId:postWrap.GetRootId(),
				ParentId:postWrap.GetParentId(),
				AllowReplies:postWrap.GetAllowReplies(),
				AllowVotes:postWrap.GetAllowVotes(),
			})
		}

		i++

		if i < limit {
			break
		}
	}

	return &grpcpb.GetPostListByCreatedResponse{PostList:postList}, nil
}

func (as *APIService) GetReplyListByPostId(ctx context.Context, req *grpcpb.GetReplyListByPostIdRequest) (*grpcpb.GetReplyListByPostIdResponse, error) {
	var (
		replyIter iservices.IDatabaseIterator
		replyList []*grpcpb.PostResponse
		i        uint32
		limit    uint32
	)

	replyOrderWrap := &table.SPostReplyOrderWrap{Dba:as.db}

	if req.Start == nil {
		replyIter = replyOrderWrap.QueryListByRevOrder(nil, nil)
	} else {
		replyIter = replyOrderWrap.QueryListByRevOrder(req.Start, nil)
	}

	if req.Limit <= constants.RPC_PAGE_SIZE_LIMIT {
		limit = req.Limit
	} else {
		limit = constants.RPC_PAGE_SIZE_LIMIT
	}

	for replyIter.Next() {
		postWrap := table.NewSoPostWrap(as.db, replyOrderWrap.GetMainVal(replyIter))
		if postWrap.CheckExist() {
			replyList = append(replyList, &grpcpb.PostResponse{
				PostId:postWrap.GetPostId(),
				Category:postWrap.GetCategory(),
				ParentAuthor:postWrap.GetAuthor(),
				ParentPermlink:postWrap.GetPermlink(),
				Author:postWrap.GetAuthor(),
				Permlink:postWrap.GetPermlink(),
				Title:postWrap.GetTitle(),
				Body:postWrap.GetBody(),
				JsonMetadata:postWrap.GetJsonMetadata(),
				LastUpdate:postWrap.GetLastUpdate(),
				Created:postWrap.GetCreated(),
				Active:postWrap.GetActive(),
				LastPayout:postWrap.GetLastPayout(),
				Depth:postWrap.GetDepth(),
				Children:postWrap.GetChildren(),
				RootId:postWrap.GetRootId(),
				ParentId:postWrap.GetParentId(),
				AllowReplies:postWrap.GetAllowReplies(),
				AllowVotes:postWrap.GetAllowVotes(),
			})
		}

		i++

		if i < limit {
			break
		}
	}

	return &grpcpb.GetReplyListByPostIdResponse{ReplyList:replyList}, nil
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
