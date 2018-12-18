package rpc

import (
	"context"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/common/eventloop"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	ErrPanicResp = errors.New("rpc panic")
)

type APIService struct {
	consensus iservices.IConsensus
	mainLoop  *eventloop.EventLoop
	db        iservices.IDatabaseService
	log       *logrus.Logger
}

func (as *APIService) GetAccountByName(ctx context.Context, req *grpcpb.GetAccountByNameRequest) (*grpcpb.AccountResponse, error) {

	accWrap := table.NewSoAccountWrap(as.db, req.GetAccountName())
	acct := &grpcpb.AccountResponse{}

	if accWrap != nil && accWrap.CheckExist() {
		acct.AccountName = &prototype.AccountName{Value: accWrap.GetName().Value}
		acct.Coin = accWrap.GetBalance()
		acct.Vest = accWrap.GetVestingShares()
		//acct.PublicKeys = accWrap.GetPubKey()
		acct.CreatedTime = accWrap.GetCreatedTime()

		witWrap := table.NewSoWitnessWrap(as.db, accWrap.GetName())
		if witWrap != nil && witWrap.CheckExist() {
			acct.Witness = &grpcpb.WitnessResponse{
				Owner:                 witWrap.GetOwner(),
				WitnessScheduleType:   witWrap.GetWitnessScheduleType(),
				CreatedTime:           witWrap.GetCreatedTime(),
				Url:                   witWrap.GetUrl(),
				LastConfirmedBlockNum: witWrap.GetLastConfirmedBlockNum(),
				TotalMissed:           witWrap.GetTotalMissed(),
				VoteCount:             witWrap.GetVoteCount(),
				SigningKey:            witWrap.GetSigningKey(),
				LastWork:              witWrap.GetLastWork(),
				RunningVersion:        witWrap.GetRunningVersion(),
			}
		}
		var (
			i int32 = 1
		)
		acct.Dgpo = table.NewSoGlobalWrap(as.db, &i).GetProps()
	}

	return acct, nil

}

func (as *APIService) GetAccountRewardByName(ctx context.Context, req *grpcpb.GetAccountRewardByNameRequest) (*grpcpb.AccountRewardResponse, error) {

	var i int32 = 1

	rewardKeeperWrap := table.NewSoRewardsKeeperWrap(as.db, &i)

	if rewardKeeperWrap != nil && rewardKeeperWrap.CheckExist() {
		keeper := rewardKeeperWrap.GetKeeper()
		if val, ok := keeper.Rewards[req.AccountName.Value]; ok {
			return &grpcpb.AccountRewardResponse{AccountName: req.AccountName, Reward: val}, nil
		}
	}
	return &grpcpb.AccountRewardResponse{AccountName: req.AccountName, Reward: &prototype.Vest{Value: 0}}, nil
}

func (as *APIService) GetFollowerListByName(ctx context.Context, req *grpcpb.GetFollowerListByNameRequest) (*grpcpb.GetFollowerListByNameResponse, error) {

	var (
		ferList []*prototype.AccountName
		limit   uint32
	)

	ferOrderWrap := table.NewExtFollowerFollowerCreatedOrderWrap(as.db)

	start := req.GetStart()
	end := req.GetEnd()
	if start == nil || end == nil {
		start = nil
		end = nil
	}
	limit = checkLimit(req.GetLimit())
	ferOrderWrap.QueryListByOrder(start, end, limit,
		func(mVal *prototype.FollowerRelation, sVal *prototype.FollowerCreatedOrder) {
		    if mVal != nil {
				ferList = append(ferList, mVal.Follower)
			}
	})
	return &grpcpb.GetFollowerListByNameResponse{FollowerList: ferList}, nil

}

func (as *APIService) GetFollowingListByName(ctx context.Context, req *grpcpb.GetFollowingListByNameRequest) (*grpcpb.GetFollowingListByNameResponse, error) {

	var (
		fingList []*prototype.AccountName
		limit    uint32
	)

	fingOrderWrap := table.NewExtFollowingFollowingCreatedOrderWrap(as.db)

	start := req.GetStart()
	end := req.GetEnd()
	if start == nil || end == nil {
		start = nil
		end = nil
	}
	limit = checkLimit(req.GetLimit())
	fingOrderWrap.QueryListByOrder(start, end, limit ,
		func(mVal *prototype.FollowingRelation, sVal *prototype.FollowingCreatedOrder) {
			if mVal != nil {
				fingList = append(fingList, mVal.Following)
			}
		})
	return &grpcpb.GetFollowingListByNameResponse{FollowingList: fingList}, nil

}

func (as *APIService) GetFollowCountByName(ctx context.Context, req *grpcpb.GetFollowCountByNameRequest) (*grpcpb.GetFollowCountByNameResponse, error) {

	var (
		ferCnt, fingCnt uint32
	)

	afc := table.NewSoExtFollowCountWrap(as.db, req.GetAccountName())

	if afc != nil && afc.CheckExist() {
		ferCnt = afc.GetFollowerCnt()
		fingCnt = afc.GetFollowingCnt()

	}

	return &grpcpb.GetFollowCountByNameResponse{FerCnt: ferCnt, FingCnt: fingCnt}, nil

}
func (as *APIService) GetChainState(ctx context.Context, req *grpcpb.NonParamsRequest) (*grpcpb.GetChainStateResponse, error) {
	var (
		i int32 = 1
	)

	globalVar := table.NewSoGlobalWrap(as.db, &i)

	ret := &grpcpb.GetChainStateResponse{}
	blks, err := as.consensus.FetchBlocksSince(common.EmptyBlockID)
	if err == nil {
		for _, v := range blks {

			res := &prototype.EmptySignedBlock{ SignedHeader:v.(*prototype.SignedBlock).SignedHeader, TrxCount:uint32(len(v.(*prototype.SignedBlock).Transactions)) }
			ret.Blocks = append(ret.Blocks, res )
		}
	}
	ret.Props = globalVar.GetProps()

	return ret, nil
}

func (as *APIService) GetWitnessList(ctx context.Context, req *grpcpb.GetWitnessListRequest) (*grpcpb.GetWitnessListResponse, error) {
	var (
		witList []*grpcpb.WitnessResponse
		limit   uint32
	)

	witOrderWrap := &table.SWitnessOwnerWrap{as.db}
	limit = checkLimit(req.GetLimit())
	witOrderWrap.QueryListByOrder(req.GetStart(),nil, limit,
		func(mVal *prototype.AccountName, sVal *prototype.AccountName) {
			witWrap := table.NewSoWitnessWrap(as.db, mVal)
			if witWrap != nil && witWrap.CheckExist() {
				witList = append(witList, &grpcpb.WitnessResponse{
					Owner:                 witWrap.GetOwner(),
					WitnessScheduleType:   witWrap.GetWitnessScheduleType(),
					CreatedTime:           witWrap.GetCreatedTime(),
					Url:                   witWrap.GetUrl(),
					LastConfirmedBlockNum: witWrap.GetLastConfirmedBlockNum(),
					TotalMissed:           witWrap.GetTotalMissed(),
					VoteCount:             witWrap.GetVoteCount(),
					SigningKey:            witWrap.GetSigningKey(),
					LastWork:              witWrap.GetLastWork(),
					RunningVersion:        witWrap.GetRunningVersion(),
				})
			}
		})

	return &grpcpb.GetWitnessListResponse{WitnessList: witList}, nil

}

func (as *APIService) GetPostListByCreated(ctx context.Context, req *grpcpb.GetPostListByCreatedRequest) (*grpcpb.GetPostListByCreatedResponse, error) {
	var (
		postList []*grpcpb.PostResponse
		limit    uint32
	)

	postOrderWrap := table.NewExtPostCreatedCreatedOrderWrap(as.db)

	start := req.GetStart()
	end := req.GetEnd()
	if start == nil || end == nil {
		start = nil
		end = nil
	}

	limit = checkLimit(req.GetLimit())
    postOrderWrap.QueryListByRevOrder(start, end, limit ,
		func(mVal *uint64, sVal *prototype.PostCreatedOrder) {
			postWrap := table.NewSoPostWrap(as.db, mVal)
			if postWrap != nil && postWrap.CheckExist() {
				postList = append(postList, &grpcpb.PostResponse{
					PostId:        postWrap.GetPostId(),
					Category:      postWrap.GetCategory(),
					ParentAuthor:  postWrap.GetAuthor(),
					Author:        postWrap.GetAuthor(),
					Title:         postWrap.GetTitle(),
					Body:          postWrap.GetBody(),
					Created:       postWrap.GetCreated(),
					LastPayout:    postWrap.GetLastPayout(),
					Depth:         postWrap.GetDepth(),
					Children:      postWrap.GetChildren(),
					RootId:        postWrap.GetRootId(),
					ParentId:      postWrap.GetParentId(),
					Tags:          postWrap.GetTags(),
					Beneficiaries: postWrap.GetBeneficiaries(),
				})
			}
		})

	return &grpcpb.GetPostListByCreatedResponse{PostList: postList}, nil

}

func (as *APIService) GetReplyListByPostId(ctx context.Context, req *grpcpb.GetReplyListByPostIdRequest) (*grpcpb.GetReplyListByPostIdResponse, error) {
	var (
		replyList []*grpcpb.PostResponse
		limit     uint32
	)

	replyOrderWrap := table.NewExtReplyCreatedCreatedOrderWrap(as.db)

	start := req.GetStart()
	end := req.GetEnd()
    if start == nil || end == nil {
    	start = nil
    	end = nil
	}
	limit = checkLimit(req.GetLimit())
    replyOrderWrap.QueryListByRevOrder(start, end, limit , func(mVal *uint64, sVal *prototype.ReplyCreatedOrder) {
		postWrap := table.NewSoPostWrap(as.db, mVal)
		if postWrap != nil && postWrap.CheckExist() {
			replyList = append(replyList, &grpcpb.PostResponse{
				PostId:       postWrap.GetPostId(),
				Category:     postWrap.GetCategory(),
				ParentAuthor: postWrap.GetAuthor(),
				Author:       postWrap.GetAuthor(),
				Title:        postWrap.GetTitle(),
				Body:         postWrap.GetBody(),
				Created:      postWrap.GetCreated(),
				LastPayout:   postWrap.GetLastPayout(),
				Depth:        postWrap.GetDepth(),
				Children:     postWrap.GetChildren(),
				RootId:       postWrap.GetRootId(),
				ParentId:     postWrap.GetParentId(),
			})
		}
	})

	return &grpcpb.GetReplyListByPostIdResponse{ReplyList: replyList}, nil

}

func (as *APIService) GetBlockTransactionsByNum(ctx context.Context, req *grpcpb.GetBlockTransactionsByNumRequest) (*grpcpb.GetBlockTransactionsByNumResponse, error) {
	return &grpcpb.GetBlockTransactionsByNumResponse{}, nil
}

func (as *APIService) GetTrxById(ctx context.Context, req *grpcpb.GetTrxByIdRequest) (*grpcpb.GetTrxByIdResponse, error) {

	trxWrap := table.NewSoTransactionObjectWrap(as.db, req.GetTrxId())
	resp := &grpcpb.GetTrxByIdResponse{}

	if trxWrap != nil && trxWrap.CheckExist() {
		//resp.Trx. = trxWrap.GetTrxId()

		//TODO wait trx definition
	}

	return resp, nil
}

func (as *APIService) BroadcastTrx(ctx context.Context, req *grpcpb.BroadcastTrxRequest) (*grpcpb.BroadcastTrxResponse, error) {

	var result *prototype.TransactionInvoice = nil
	as.mainLoop.Send(func() {
		r := as.consensus.PushTransaction(req.GetTransaction(), true, true)
		as.log.Infof("BroadcastTrx Result: %s", result)

		if r != nil {
			result = r.(*prototype.TransactionInvoice)
		}
	})

	return &grpcpb.BroadcastTrxResponse{Invoice: result}, nil
}

func checkLimit(limit uint32) uint32 {
	if limit <= constants.RPC_PAGE_SIZE_LIMIT {
		return limit
	} else {
		return constants.RPC_PAGE_SIZE_LIMIT
	}
}
