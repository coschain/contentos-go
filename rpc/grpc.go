package rpc

import (
	"context"
	"fmt"
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/common/eventloop"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
	"github.com/coschain/contentos-go/utils"
	"github.com/coschain/contentos-go/vm/contract/abi"
	contractTable "github.com/coschain/contentos-go/vm/contract/table"
	"github.com/coschain/gobft/message"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"time"
)

var (
	ErrPanicResp         = errors.New("rpc panic")
	defaultPageSizeLimit = 30
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
		acct.PostCount = accWrap.GetPostCount()

		witWrap := table.NewSoWitnessWrap(as.db, accWrap.GetName())
		if witWrap != nil && witWrap.CheckExist() {
			acct.Witness = &grpcpb.WitnessResponse{
				Owner:                 witWrap.GetOwner(),
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

		keyWrap := table.NewSoAccountAuthorityObjectWrap(as.db, req.GetAccountName())

		if keyWrap.CheckExist() {
			acct.PublicKey = keyWrap.GetOwner().GetKey()
		}
	}
	acct.State = as.getState()

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
	ferOrderWrap.ForEachByOrder(start, end, nil, nil,
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
	fingOrderWrap.ForEachByOrder(start, end, nil, nil,
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

	ret := &grpcpb.GetChainStateResponse{}
	ret.State = as.getState()

	return ret, nil
}

func (as *APIService) GetStatisticsInfo(ctx context.Context, req *grpcpb.NonParamsRequest) (*grpcpb.GetStatResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()

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
	ret.State = as.getState()

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
	witOrderWrap.ForEachByOrder(req.GetStart(), nil, nil, nil,
		func(mVal *prototype.AccountName, sVal *prototype.AccountName, idx uint32) bool {
			witWrap := table.NewSoWitnessWrap(as.db, mVal)
			if witWrap != nil && witWrap.CheckExist() {
				witList = append(witList, &grpcpb.WitnessResponse{
					Owner:                 witWrap.GetOwner(),
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
	postOrderWrap.ForEachByRevOrder(start, end, nil, nil,
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
					VoteCnt:       postWrap.GetVoteCnt(),
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
	replyOrderWrap.ForEachByRevOrder(start, end, nil, nil,
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
					VoteCnt:      postWrap.GetVoteCnt(),
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
	as.db.RUnlock()
	result.LastIrreversibleBlockNumber = as.consensus.GetLIB().BlockNum()
	lastCommit := as.consensus.GetLastBFTCommit()
	as.db.RLock()

	result.Dgpo = table.NewSoGlobalWrap(as.db, &i).GetProps()

	if lastCommit != nil {
		result.LastIrreversibleBlockTime = uint64(lastCommit.(*message.Commit).CommitTime.Unix())
	}
	return result
}

func (as *APIService) GetBlockList(ctx context.Context, req *grpcpb.GetBlockListRequest) (*grpcpb.GetBlockListResponse, error) {
	from := req.Start
	to := req.End
	limit := req.Limit
	//isFetchOne := false
	//if from == to && from != 0 {
	//	isFetchOne = true
	//	to = from + 1
	//}
	headNum := as.consensus.GetHeadBlockId().BlockNum()
	//if from == 0 && to == 0 {
	//	if headNum >= uint64(limit) {
	//		from = headNum - uint64(limit) + 1
	//	}
	//	to = headNum
	//} else if from >= 0 && to == 0 {
	//	to = headNum
	//}
	if to == 0 {
		to = headNum
	}
	//if from == 0 {
	//	from = headNum
	//}
	if from == to {
		from = to - 1
	}
	if to-from > uint64(limit) {
		from = to - uint64(limit) + 1
	}
	if headNum < from {
		return nil, errors.New("The start block number in range exceed the head block")
	}
	list, err := as.consensus.FetchBlocks(from, to)
	if err != nil {
		return &grpcpb.GetBlockListResponse{Blocks: make([]*grpcpb.BlockInfo, 0)}, err
	}
	var blkList []*grpcpb.BlockInfo
	for _, blk := range list {
		b := blk.(*prototype.SignedBlock)
		blkInfo := &grpcpb.BlockInfo{}
		blkInfo.Timestamp = b.SignedHeader.Header.Timestamp
		blkInfo.BlockHeight = b.Id().BlockNum()
		blkInfo.Witness = b.SignedHeader.Header.Witness
		blkInfo.TrxCount = uint32(len(b.Transactions))
		blkInfo.BlockId = &prototype.Sha256{}
		blkInfo.BlockId.FromBlockID(b.Id())
		blkInfo.PreId = b.SignedHeader.Header.Previous
		//if isFetchOne && b.Id().BlockNum() == from {
		//	blkList = append(blkList, blkInfo)
		//	break
		//}
		blkList = append(blkList, blkInfo)

	}
	if blkList == nil {
		blkList = make([]*grpcpb.BlockInfo, 0)
	}
	return &grpcpb.GetBlockListResponse{Blocks: blkList}, nil
}

func (as *APIService) GetSignedBlock(ctx context.Context, req *grpcpb.GetSignedBlockRequest) (*grpcpb.GetSignedBlockResponse, error) {
	headNum := as.consensus.GetHeadBlockId().BlockNum()
	if req.Start > headNum {
		return &grpcpb.GetSignedBlockResponse{Block: nil}, errors.New("the block not exist")
	}
	from := req.Start
	var block *prototype.SignedBlock
	list, err := as.consensus.FetchBlocks(from, from+1)
	if err != nil {
		return &grpcpb.GetSignedBlockResponse{Block: nil}, err
	}
	for _, blk := range list {
		b := blk.(*prototype.SignedBlock)
		if b.Id().BlockNum() == from {
			block = b
		}
	}
	return &grpcpb.GetSignedBlockResponse{Block: block}, nil
}

func (as *APIService) GetAccountListByBalance(ctx context.Context, req *grpcpb.GetAccountListByBalanceRequest) (*grpcpb.GetAccountListResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()

	sortWrap := table.NewAccountBalanceWrap(as.db)
	var list []*grpcpb.AccountResponse
	res := &grpcpb.GetAccountListResponse{}
	var err error
	var lastAcctNam *prototype.AccountName
	var lastAcctCoin *prototype.Coin
	limit := checkLimit(req.Limit)
	if limit == 0 {
		limit = uint32(defaultPageSizeLimit)
	}
	if req.LastAccount != nil {
		account := req.LastAccount
		if account.AccountName != nil && account.Coin != nil {
			lastAcctNam = account.AccountName
			lastAcctCoin = account.Coin
		}
	}
	if sortWrap != nil {
		err = sortWrap.ForEachByRevOrder(req.Start, req.End, lastAcctNam, lastAcctCoin, func(mVal *prototype.AccountName, sVal *prototype.Coin, idx uint32) bool {
			acct := &grpcpb.AccountResponse{}
			accWrap := table.NewSoAccountWrap(as.db, mVal)
			if accWrap != nil {
				acct.AccountName = &prototype.AccountName{Value: mVal.Value}
				acct.Coin = accWrap.GetBalance()
				acct.Vest = accWrap.GetVestingShares()
				acct.CreatedTime = accWrap.GetCreatedTime()
				acct.PostCount = accWrap.GetPostCount()
				witWrap := table.NewSoWitnessWrap(as.db, mVal)
				if witWrap != nil && witWrap.CheckExist() {
					acct.Witness = &grpcpb.WitnessResponse{
						Owner:                 witWrap.GetOwner(),
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
			if uint32(len(list)) >= limit {
				return false
			}
			return true
		})
	}
	res.List = list
	return res, err
}

func checkLimit(limit uint32) uint32 {
	if limit <= constants.RpcPageSizeLimit {
		return limit
	} else {
		return constants.RpcPageSizeLimit
	}
}

func (as *APIService) GetDailyTotalTrxInfo(ctx context.Context, req *grpcpb.GetDailyTotalTrxRequest) (*grpcpb.GetDailyTotalTrxResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()
	var (
		list     []*grpcpb.DailyTotalTrx
		err      error
		lastTime *prototype.TimePointSec
		lastVal  *prototype.TimePointSec
	)
	list = make([]*grpcpb.DailyTotalTrx, 0)
	res := &grpcpb.GetDailyTotalTrxResponse{}
	wrap := table.NewExtDailyTrxDateWrap(as.db)
	if req.LastInfo != nil {
		info := req.LastInfo
		if info.Date != nil {
			lastTime = info.Date
			lastVal = info.Date
		}
	}
	if wrap != nil {
		limit := checkLimit(req.Limit)
		if limit == 0 {
			limit = uint32(defaultPageSizeLimit)
		}
		s := req.Start
		e := req.End
		//convert the unix timestamp to day index
		if req.Start != nil {
			s = &prototype.TimePointSec{UtcSeconds: req.Start.UtcSeconds / 86400}
		}
		if req.End != nil {
			e = &prototype.TimePointSec{UtcSeconds: req.End.UtcSeconds / 86400}
		}
		err = wrap.ForEachByOrder(s, e, lastTime, lastVal, func(mVal *prototype.TimePointSec, sVal *prototype.TimePointSec,
			idx uint32) bool {
			if mVal != nil && sVal != nil {
				info := &grpcpb.DailyTotalTrx{}
				//return the normal timestamp not the index
				info.Date = &prototype.TimePointSec{UtcSeconds: mVal.UtcSeconds * 86400}
				dWrap := table.NewSoExtDailyTrxWrap(as.db, mVal)
				if dWrap != nil {
					info.Count = dWrap.GetCount()
				}
				list = append(list, info)
			}
			if uint32(len(list)) >= limit {
				return false
			}
			return true
		})
	}
	res.List = list
	return res, err
}

func (as *APIService) TrxStatByHour(ctx context.Context, req *grpcpb.TrxStatByHourRequest) (*grpcpb.TrxStatByHourResponse, error) {
	var lastMainKey *prototype.Sha256
	var lastSubVal *prototype.TimePointSec
	var infoList []*grpcpb.TrxInfo
	var err error
	res := &grpcpb.TrxStatByHourResponse{}
	sWrap := table.NewExtTrxBlockTimeWrap(as.db)
	// reverse order
	now := time.Now()
	end := &prototype.TimePointSec{UtcSeconds: uint32(now.Unix() - int64(3600*req.Hours))}
	if sWrap != nil {
		err = sWrap.ForEachByRevOrder(nil, end, lastMainKey, lastSubVal, func(mVal *prototype.Sha256, sVal *prototype.TimePointSec, idx uint32) bool {
			wrap := table.NewSoExtTrxWrap(as.db, mVal)
			info := &grpcpb.TrxInfo{}
			if wrap != nil {
				info.TrxId = mVal
				info.BlockHeight = wrap.GetBlockHeight()
				info.BlockTime = wrap.GetBlockTime()
				info.TrxWrap = wrap.GetTrxWrap()
				infoList = append(infoList, info)
			}
			return true
		})
	}
	h, _ := time.ParseDuration("-1h")
	hourData := make(map[int]int, req.Hours)
	var hours []int
	for i := 0; i < int(req.Hours); i++ {
		then := now.Add(time.Duration(i) * h)
		hours = append(hours, then.Hour())
	}
	for _, hour := range hours {
		fmt.Println(hour)
	}
	for _, trx := range infoList {
		timestamp := trx.GetBlockTime().UtcSeconds
		hour := time.Unix(int64(timestamp), 0).Hour()
		hourData[hour] += 1
	}
	var hourStat []*grpcpb.StatByHour
	// make it sequence
	for _, hour := range hours {
		h := &grpcpb.StatByHour{Hour: uint32(hour), Count: uint32(hourData[hour])}
		hourStat = append(hourStat, h)
	}
	res.Stat = hourStat
	return res, err
}

func (as *APIService) GetTrxInfoById(ctx context.Context, req *grpcpb.GetTrxInfoByIdRequest) (*grpcpb.GetTrxInfoByIdResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()
	res := &grpcpb.GetTrxInfoByIdResponse{}
	var err error
	wrap := table.NewSoExtTrxWrap(as.db, req.TrxId)
	if wrap != nil && wrap.CheckExist() {
		info := &grpcpb.TrxInfo{}
		info.TrxId = req.TrxId
		info.BlockHeight = wrap.GetBlockHeight()
		info.BlockTime = wrap.GetBlockTime()
		info.TrxWrap = wrap.GetTrxWrap()
		res.Info = info
	}
	return res, err
}

func (as *APIService) GetTrxListByTime(ctx context.Context, req *grpcpb.GetTrxListByTimeRequest) (*grpcpb.GetTrxListByTimeResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()
	var (
		infoList    []*grpcpb.TrxInfo
		err         error
		lastMainKey *prototype.Sha256
		lastSubVal  *prototype.TimePointSec
	)
	limit := req.Limit
	res := &grpcpb.GetTrxListByTimeResponse{}
	if req.LastInfo != nil && req.LastInfo.TrxId != nil && req.LastInfo.BlockTime != nil {
		lastMainKey = req.LastInfo.TrxId
		lastSubVal = req.LastInfo.BlockTime
	}
	sWrap := table.NewExtTrxBlockTimeWrap(as.db)
	if sWrap != nil {
		err = sWrap.ForEachByRevOrder(req.Start, req.End, lastMainKey, lastSubVal, func(mVal *prototype.Sha256, sVal *prototype.TimePointSec, idx uint32) bool {
			wrap := table.NewSoExtTrxWrap(as.db, mVal)
			info := &grpcpb.TrxInfo{}
			if wrap != nil {
				info.TrxId = mVal
				info.BlockHeight = wrap.GetBlockHeight()
				info.BlockTime = wrap.GetBlockTime()
				info.TrxWrap = wrap.GetTrxWrap()
				infoList = append(infoList, info)
			}
			//if len(infoList) >= (maxPageSizeLimit) {
			//	return false
			//}
			if limit != 0 && len(infoList) >= int(limit) {
				return false
			}
			return true
		})
	}
	res.List = infoList
	return res, err
}

func (as *APIService) GetPostListByCreateTime(ctx context.Context, req *grpcpb.GetPostListByCreateTimeRequest) (*grpcpb.GetPostListByCreateTimeResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()
	var (
		postList     []*grpcpb.PostResponse
		lastPost     *grpcpb.PostResponse
		lastPostId   *uint64
		lastPostTime *prototype.TimePointSec
		err          error
	)

	res := &grpcpb.GetPostListByCreateTimeResponse{}
	if req.LastPost != nil {
		lastPost = req.LastPost
		if lastPost.Created != nil {
			lastPostId = &lastPost.PostId
			lastPostTime = lastPost.Created
		}
	}
	sWrap := table.NewPostCreatedWrap(as.db)
	if sWrap != nil {
		limit := checkLimit(req.Limit)
		if limit == 0 {
			limit = uint32(defaultPageSizeLimit)
		}
		err = sWrap.ForEachByRevOrder(req.Start, req.End, lastPostId, lastPostTime,
			func(mVal *uint64, sVal *prototype.TimePointSec, idx uint32) bool {
				if mVal != nil {
					postWrap := table.NewSoPostWrap(as.db, mVal)
					if postWrap != nil && postWrap.CheckExist() {
						postInfo := &grpcpb.PostResponse{
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
						}
						postList = append(postList, postInfo)
					}
				}
				if uint32(len(postList)) >= limit {
					return false
				}
				return true
			})
	}

	res.PostedList = postList
	return res, err
}

func (as *APIService) GetPostListByName(ctx context.Context, req *grpcpb.GetPostListByNameRequest) (*grpcpb.GetPostListByCreateTimeResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()
	var (
		postList      []*grpcpb.PostResponse
		lastPostId    *uint64
		lastPostOrder *prototype.UserPostCreateOrder
		err           error
	)
	if req.LastPost != nil {
		post := req.LastPost
		lastPostId = &post.PostId
		lastPostOrder = &prototype.UserPostCreateOrder{Author: post.Author, Create: post.Created}
	}

	wrap := table.NewExtUserPostPostCreatedOrderWrap(as.db)
	res := &grpcpb.GetPostListByCreateTimeResponse{}
	if wrap != nil {
		limit := checkLimit(req.Limit)
		if limit == 0 {
			limit = uint32(defaultPageSizeLimit)
		}
		err = wrap.ForEachByRevOrder(req.Start, req.End, lastPostId, lastPostOrder, func(mVal *uint64, sVal *prototype.UserPostCreateOrder, idx uint32) bool {
			if mVal != nil {
				postWrap := table.NewSoPostWrap(as.db, mVal)
				if postWrap != nil && postWrap.CheckExist() {
					postInfo := &grpcpb.PostResponse{
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
						VoteCnt:       postWrap.GetVoteCnt(),
					}
					postList = append(postList, postInfo)
				}
			}
			if uint32(len(postList)) >= limit {
				return false
			}
			return true
		})
	}
	if postList == nil {
		postList = make([]*grpcpb.PostResponse, 0)
	}
	res.PostedList = postList
	return res, err
}
