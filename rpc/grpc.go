package rpc

import (
	"context"
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/common/eventloop"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
	"github.com/coschain/contentos-go/utils"
	"github.com/coschain/contentos-go/vm/contract/abi"
	contractTable "github.com/coschain/contentos-go/vm/contract/table"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"time"
)

var (
	ErrPanicResp = errors.New("rpc panic")
)

type APIService struct {
	consensus iservices.IConsensus
	mainLoop  *eventloop.EventLoop
	db        iservices.IDatabaseService
	log       *logrus.Logger
	eBus      EventBus.Bus
}

func NewAPIService(con iservices.IConsensus, loop *eventloop.EventLoop, db iservices.IDatabaseService, log *logrus.Logger) *APIService {
	return &APIService{
		consensus: con,
		mainLoop:  loop,
		db:        db,
		log:       log,
	}
}

func (as *APIService) QueryTableContent(ctx context.Context, req *grpcpb.GetTableContentRequest) (*grpcpb.TableContentResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()

	res := &grpcpb.TableContentResponse{}

	cid := prototype.ContractId{Owner: &prototype.AccountName{Value: req.Owner}, Cname: req.Contranct}
	scid := table.NewSoContractWrap(as.db, &cid)

	abiString := scid.GetAbi()
	abiInterface, err := abi.UnmarshalABI([]byte(abiString))
	if err != nil {
		return nil, err
	}

	tables := contractTable.NewContractTables(req.Owner, req.Contranct, abiInterface, as.db)
	aimTable := tables.Table(req.Table)
	jsonStr, err := aimTable.QueryRecordsJson(req.Field, req.Begin, req.End, false, -1)
	if err != nil {
		return nil, err
	}
	res.TableContent = jsonStr
	return res, nil
}

func (as *APIService) GetAccountByName(ctx context.Context, req *grpcpb.GetAccountByNameRequest) (*grpcpb.AccountResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()

	accWrap := table.NewSoAccountWrap(as.db, req.GetAccountName())
	acct := &grpcpb.AccountResponse{}
	rc := utils.NewResourceLimiter(as.db)
	wraper := table.NewSoGlobalWrap(as.db, &constants.GlobalId)
	gp := wraper.GetProps()

	if accWrap != nil && accWrap.CheckExist() {
		acct.AccountName = &prototype.AccountName{Value: accWrap.GetName().Value}
		acct.Coin = accWrap.GetBalance()
		acct.Vest = accWrap.GetVestingShares()

		acct.StaminaRemain = rc.GetStakeLeft(accWrap.GetName().Value, gp.HeadBlockNumber) + rc.GetFreeLeft(accWrap.GetName().Value, gp.HeadBlockNumber)
		acct.StaminaMax = rc.GetCapacity(accWrap.GetName().Value) + rc.GetCapacityFree()
		//acct.PublicKeys =
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
	as.db.RLock()
	defer as.db.RUnlock()

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
	as.db.RLock()
	defer as.db.RUnlock()

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
	ferOrderWrap.ForEachByOrder(start, end,
		func(mVal *prototype.FollowerRelation, sVal *prototype.FollowerCreatedOrder, idx uint32) bool {
			if mVal != nil {
				ferList = append(ferList, mVal.Follower)
			}
			if idx < limit {
				return true
			}
			return false
		})
	return &grpcpb.GetFollowerListByNameResponse{FollowerList: ferList}, nil

}

func (as *APIService) GetFollowingListByName(ctx context.Context, req *grpcpb.GetFollowingListByNameRequest) (*grpcpb.GetFollowingListByNameResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()

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
	fingOrderWrap.ForEachByOrder(start, end,
		func(mVal *prototype.FollowingRelation, sVal *prototype.FollowingCreatedOrder, idx uint32) bool {
			if mVal != nil {
				fingList = append(fingList, mVal.Following)
			}
			if idx < limit {
				return true
			}
			return false
		})
	return &grpcpb.GetFollowingListByNameResponse{FollowingList: fingList}, nil

}

func (as *APIService) GetFollowCountByName(ctx context.Context, req *grpcpb.GetFollowCountByNameRequest) (*grpcpb.GetFollowCountByNameResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()

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
	as.db.RLock()
	defer as.db.RUnlock()

	var (
		i int32 = 1
	)

	globalVar := table.NewSoGlobalWrap(as.db, &i)

	ret := &grpcpb.GetChainStateResponse{}
	blks, err := as.consensus.FetchBlocksSince(common.EmptyBlockID)
	if err == nil {
		for _, v := range blks {

			res := &prototype.EmptySignedBlock{SignedHeader: v.(*prototype.SignedBlock).SignedHeader, TrxCount: uint32(len(v.(*prototype.SignedBlock).Transactions))}
			ret.Blocks = append(ret.Blocks, res)
		}
	}
	ret.Props = globalVar.GetProps()

	return ret, nil
}

func (as *APIService) GetStatInfo(ctx context.Context, req *grpcpb.NonParamsRequest) (*grpcpb.GetStatResponse, error) {
	var (
		i int32 = 1
	)

	globalVar := table.NewSoGlobalWrap(as.db, &i)

	ret := &grpcpb.GetStatResponse{}

	// TODO add daily trx count
	//blks, err := as.consensus.FetchBlocksSince(common.EmptyBlockID)
	//if err == nil {
	//	for _, v := range blks {
	//
	//		res := &prototype.EmptySignedBlock{ SignedHeader:v.(*prototype.SignedBlock).SignedHeader, TrxCount:uint32(len(v.(*prototype.SignedBlock).Transactions)) }
	//		ret.Blocks = append(ret.Blocks, res )
	//	}
	//}
	ret.Props = globalVar.GetProps()

	return ret, nil
}

func (as *APIService) GetWitnessList(ctx context.Context, req *grpcpb.GetWitnessListRequest) (*grpcpb.GetWitnessListResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()

	var (
		witList []*grpcpb.WitnessResponse
		limit   uint32
	)

	witOrderWrap := &table.SWitnessOwnerWrap{as.db}
	limit = checkLimit(req.GetLimit())
	witOrderWrap.ForEachByOrder(req.GetStart(), nil,
		func(mVal *prototype.AccountName, sVal *prototype.AccountName, idx uint32) bool {
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
			if idx < limit {
				return true
			}
			return false
		})
	return &grpcpb.GetWitnessListResponse{WitnessList: witList}, nil

}

func (as *APIService) GetPostListByCreated(ctx context.Context, req *grpcpb.GetPostListByCreatedRequest) (*grpcpb.GetPostListByCreatedResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()

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
	postOrderWrap.ForEachByRevOrder(start, end,
		func(mVal *uint64, sVal *prototype.PostCreatedOrder, idx uint32) bool {
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
			if idx < limit {
				return true
			}
			return false
		})
	return &grpcpb.GetPostListByCreatedResponse{PostList: postList}, nil

}

func (as *APIService) GetReplyListByPostId(ctx context.Context, req *grpcpb.GetReplyListByPostIdRequest) (*grpcpb.GetReplyListByPostIdResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()

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
	replyOrderWrap.ForEachByRevOrder(start, end,
		func(mVal *uint64, sVal *prototype.ReplyCreatedOrder, idx uint32) bool {
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
			if idx < limit {
				return true
			}
			return false
		})
	return &grpcpb.GetReplyListByPostIdResponse{ReplyList: replyList}, nil

}

func (as *APIService) GetBlockTransactionsByNum(ctx context.Context, req *grpcpb.GetBlockTransactionsByNumRequest) (*grpcpb.GetBlockTransactionsByNumResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()

	return &grpcpb.GetBlockTransactionsByNumResponse{}, nil
}

func (as *APIService) GetTrxById(ctx context.Context, req *grpcpb.GetTrxByIdRequest) (*grpcpb.GetTrxByIdResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()

	trxWrap := table.NewSoTransactionObjectWrap(as.db, req.GetTrxId())
	resp := &grpcpb.GetTrxByIdResponse{}

	if trxWrap != nil && trxWrap.CheckExist() {
		//resp.Trx. = trxWrap.GetTrxId()

		//TODO wait trx definition
	}

	return resp, nil
}

func (as *APIService) BroadcastTrx(ctx context.Context, req *grpcpb.BroadcastTrxRequest) (*grpcpb.BroadcastTrxResponse, error) {

	//var result chan *prototype.TransactionReceipt
	//result := make(chan *prototype.TransactionReceipt)
	trx := req.GetTransaction()

	var pErr error
	as.mainLoop.Send(func() {
		as.consensus.PushTransactionToPending(trx, func(err error) {
			pErr = err
		})
		//as.log.Infof("BroadcastTrx Result: %s", result)
	})
	//result <- prototype.FetchTrxApplyResult(as.eBus , 30*time.Second ,trx)

	if !req.OnlyDeliver {
		return &grpcpb.BroadcastTrxResponse{Invoice: prototype.FetchTrxApplyResult(as.eBus, 30*time.Second, trx)}, pErr
	} else {
		return &grpcpb.BroadcastTrxResponse{Invoice: nil, Status: prototype.StatusSuccess}, pErr
	}
}

func (as *APIService) getState() *grpcpb.ChainState {
	result := &grpcpb.ChainState{}

	var (
		i int32 = 1
	)
	result.Dgpo = table.NewSoGlobalWrap(as.db, &i).GetProps()
	result.LastIrreversibleBlockNumber = as.consensus.GetLIB().BlockNum()
	return result
}

func (as *APIService) GetBlockList(ctx context.Context, req *grpcpb.GetBlockListRequest) (*grpcpb.GetBlockListResponse, error) {
	from := req.Start
	to := req.End
	list, err := as.consensus.FetchBlocks(from, to)
	if err != nil {
		return &grpcpb.GetBlockListResponse{Blocks: make([]*prototype.SignedBlock, 0)}, err
	}
	blkList := make([]*prototype.SignedBlock, len(list))
	for i, blk := range list {
		blkList[i] = blk.(*prototype.SignedBlock)
	}

	return &grpcpb.GetBlockListResponse{Blocks: blkList}, nil
}

func (as *APIService) GetAccountListByBalance(ctx context.Context, req *grpcpb.NonParamsRequest) (*grpcpb.GetAccountListResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()

	sortWrap := table.NewAccountBalanceWrap(as.db)
	list := make([]*grpcpb.AccountResponse, 0)
	res := &grpcpb.GetAccountListResponse{}
	var err error
	if sortWrap != nil {
		err = sortWrap.ForEachByOrder(nil, nil, func(mVal *prototype.AccountName, sVal *prototype.Coin, idx uint32) bool {
			acct := &grpcpb.AccountResponse{}
			accWrap := table.NewSoAccountWrap(as.db, mVal)
			if accWrap != nil {
				acct.AccountName = &prototype.AccountName{Value: mVal.Value}
				acct.Coin = accWrap.GetBalance()
				acct.Vest = accWrap.GetVestingShares()
				acct.CreatedTime = accWrap.GetCreatedTime()
				witWrap := table.NewSoWitnessWrap(as.db, mVal)
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
				acct.State = as.getState()
				list = append(list, acct)
			}
			return true
		})
	}
	res.List = list
	return res, err
}

func checkLimit(limit uint32) uint32 {
	if limit <= constants.RPC_PAGE_SIZE_LIMIT {
		return limit
	} else {
		return constants.RPC_PAGE_SIZE_LIMIT
	}
}
