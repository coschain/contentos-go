package rpc

import (
	"context"
	"fmt"
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/common/eventloop"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
	"github.com/coschain/contentos-go/vm/contract/abi"
	contractTable "github.com/coschain/contentos-go/vm/contract/table"
	"github.com/coschain/gobft/message"
	"github.com/coschain/contentos-go/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/golang/protobuf/proto"
	"math"
	"time"
)

var (
	ErrPanicResp         = errors.New("rpc panic")
	defaultPageSizeLimit = 30
	defaultHourStatLimit = 24
)

type APIService struct {
	consensus iservices.IConsensus
	pool iservices.ITrxPool
	p2p       iservices.IP2P
	mainLoop  *eventloop.EventLoop
	db        iservices.IDatabaseService
	log       *logrus.Logger
	eBus      EventBus.Bus
	ctx       *node.ServiceContext
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

	cid := prototype.ContractId{Owner: &prototype.AccountName{Value: req.Owner}, Cname: req.Contract}
	scid := table.NewSoContractWrap(as.db, &cid)

	abiString := scid.GetAbi()
	abiInterface, err := abi.UnmarshalABI([]byte(abiString))
	if err != nil {
		return nil, err
	}

	limit := checkLimit(req.Count)

	tables := contractTable.NewContractTables(req.Owner, req.Contract, abiInterface, as.db)
	aimTable := tables.Table(req.Table)

	startKey := req.Begin
	limitKey := ""

	if req.Reverse{
		limitKey = req.Begin
		startKey = ""
	}

	jsonStr, err := aimTable.QueryRecordsJson(req.Field, startKey , limitKey, req.Reverse, int(limit) )
	if err != nil {
		return nil, err
	}
	res.TableContent = jsonStr
	return res, nil
}

func (as *APIService) GetAccountByName(ctx context.Context, req *grpcpb.GetAccountByNameRequest) (*grpcpb.AccountResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()

	acct := as.getAccountResponseByName(req.GetAccountName(),false)

	return acct, nil

}

func (as *APIService) GetMyStakes(ctx context.Context, req *grpcpb.GetMyStakeListByNameRequest) (*grpcpb.GetMyStakeListByNameResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()

	var (
		stakeList []*grpcpb.StakeInfo
		limit   uint32
	)

	toWrap := table.NewStakeRecordRecordWrap(as.db)
	start := req.GetStart()
	end := req.GetEnd()
	limit = checkLimit(req.GetLimit())
	if limit == 0 {
		limit = uint32(defaultPageSizeLimit)
	}
	err := toWrap.ForEachByOrder(start, end, nil, nil,
		func(mVal *prototype.StakeRecord, sVal *prototype.StakeRecord, idx uint32) bool {
			if mVal != nil {
				info := &grpcpb.StakeInfo{}
				info.Account = sVal.To
				recordWrap := table.NewSoStakeRecordWrap(as.db, &prototype.StakeRecord{
					From:   mVal.From,
					To: mVal.To,
				})
				info.StakeAmount = recordWrap.GetStakeAmount()
				info.StakeTime = recordWrap.GetLastStakeTime()
				stakeList = append(stakeList,info)
			}
			if uint32(len(stakeList)) < limit {
				return true
			}
			return false
		})
	return &grpcpb.GetMyStakeListByNameResponse{MyStakeList: stakeList}, err
}

func (as *APIService) GetMyStakers(ctx context.Context, req *grpcpb.GetMyStakerListByNameRequest) (*grpcpb.GetMyStakerListByNameResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()

	var (
		stakerList []*grpcpb.StakeInfo
		limit   uint32
	)

	toWrap := table.NewStakeRecordRecordReverseWrap(as.db)
	start := req.GetStart()
	end := req.GetEnd()
	limit = checkLimit(req.GetLimit())
	if limit == 0 {
		limit = uint32(defaultPageSizeLimit)
	}
	err := toWrap.ForEachByOrder(start, end, nil, nil,
		func(mVal *prototype.StakeRecord, sVal *prototype.StakeRecordReverse, idx uint32) bool {
			if mVal != nil {
				info := &grpcpb.StakeInfo{}
				info.Account = sVal.From
				recordWrap := table.NewSoStakeRecordWrap(as.db, &prototype.StakeRecord{
					From:   mVal.From,
					To: mVal.To,
				})
				info.StakeAmount = recordWrap.GetStakeAmount()
				info.StakeTime = recordWrap.GetLastStakeTime()
				stakerList = append(stakerList,info)
			}
			if uint32(len(stakerList)) < limit {
				return true
			}
			return false
		})
	return &grpcpb.GetMyStakerListByNameResponse{MyStakerList: stakerList}, err
}

func (as *APIService) GetFollowerListByName(ctx context.Context, req *grpcpb.GetFollowerListByNameRequest) (*grpcpb.GetFollowerListByNameResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()

	var (
		ferList []*grpcpb.FollowerListInfo
		limit   uint32
		lastRelation  *prototype.FollowerRelation
		lastCreOrder  *prototype.FollowerCreatedOrder
	)

	ferOrderWrap := table.NewExtFollowerFollowerCreatedOrderWrap(as.db)
	start := req.GetStart()
	end := req.GetEnd()
	if req.LastOrder != nil {
		lastOrder := req.LastOrder
		if lastOrder.Account != nil && lastOrder.Follower != nil {
			lastRelation = &prototype.FollowerRelation{Account:lastOrder.Account,Follower:lastOrder.Follower}
			lastCreOrder = lastOrder
		}
	}
	limit = checkLimit(req.GetLimit())
	if limit == 0 {
		limit = uint32(defaultPageSizeLimit)
	}
	err := ferOrderWrap.ForEachByOrder(start, end, lastRelation, lastCreOrder,
		func(mVal *prototype.FollowerRelation, sVal *prototype.FollowerCreatedOrder, idx uint32) bool {
			if mVal != nil {
				acctInfo := &grpcpb.FollowerListInfo{}
				acct := as.getAccountResponseByName(mVal.Follower,false)
				if acct != nil {
					acctInfo.Account = acct
					acctInfo.CreateOrder = sVal
					ferList = append(ferList,acctInfo)
				}
			}
			if uint32(len(ferList)) < limit {
				return true
			}
			return false
		})
	return &grpcpb.GetFollowerListByNameResponse{FollowerList: ferList}, err

}

func (as *APIService) GetFollowingListByName(ctx context.Context, req *grpcpb.GetFollowingListByNameRequest) (*grpcpb.GetFollowingListByNameResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()

	var (
		fingList []*grpcpb.FollowingListInfo
		limit    uint32
		lastRelation  *prototype.FollowingRelation
		lastCreOrder  *prototype.FollowingCreatedOrder
	)

	fingOrderWrap := table.NewExtFollowingFollowingCreatedOrderWrap(as.db)
	start := req.GetStart()
	end := req.GetEnd()
	if req.LastOrder != nil {
		lastOrder := req.LastOrder
		if lastOrder.Account != nil && lastOrder.Following != nil {
			lastRelation = &prototype.FollowingRelation{Account:lastOrder.Account,Following:lastOrder.Following}
			lastCreOrder = lastOrder
		}
	}
	limit = checkLimit(req.GetLimit())
	if limit == 0 {
		limit = uint32(defaultPageSizeLimit)
	}

	err := fingOrderWrap.ForEachByOrder(start, end, lastRelation, lastCreOrder,
		func(mVal *prototype.FollowingRelation, sVal *prototype.FollowingCreatedOrder, idx uint32) bool {
			if mVal != nil {
				acctInfo := &grpcpb.FollowingListInfo{}
				acct := as.getAccountResponseByName(mVal.Following,false)
				if acct != nil {
					acctInfo.Account = acct
					acctInfo.CreateOrder = sVal
					fingList = append(fingList,acctInfo)
				}
			}
			if uint32(len(fingList)) < limit {
				return true
			}
			return false
		})
	return &grpcpb.GetFollowingListByNameResponse{FollowingList: fingList}, err

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

func (as *APIService) GetBlockProducerList(ctx context.Context, req *grpcpb.GetBlockProducerListRequest) (*grpcpb.GetBlockProducerListResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()

	var (
		witList []*grpcpb.BlockProducerResponse
		limit   uint32
	)

	witOrderWrap := &table.SBlockProducerOwnerWrap{as.db}
	limit = checkLimit(req.GetLimit())
	witOrderWrap.ForEachByOrder(req.GetStart(), nil, nil, nil,
		func(mVal *prototype.AccountName, sVal *prototype.AccountName, idx uint32) bool {
			bp := as.getBlockProducerResponseByAccountName(mVal)
			if bp != nil {
				witList = append(witList,bp)
			}
			if idx < limit {
				return true
			}
			return false
		})
	return &grpcpb.GetBlockProducerListResponse{BlockProducerList: witList}, nil

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
				post := as.fetchPostInfoResponseById(*mVal,false)
				if post != nil {
					postList = append(postList,post)
				}
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
			post := as.fetchPostInfoResponseById(*mVal,false)
			if post != nil {
				replyList = append(replyList, post)
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

func (as *APIService) EstimateStamina(ctx context.Context, req *grpcpb.EsimateRequest) (*grpcpb.EsimateResponse, error) {
	receipt := as.pool.EstimateStamina(req.Transaction)
	estimateResponse := &grpcpb.EsimateResponse{Invoice:receipt}
	return estimateResponse,nil
}

func (as *APIService) BroadcastTrx(ctx context.Context, req *grpcpb.BroadcastTrxRequest) (*grpcpb.BroadcastTrxResponse, error) {

	//var result chan *prototype.TransactionReceiptWithInfo
	//result := make(chan *prototype.TransactionReceiptWithInfo)
	trx := req.GetTransaction()

	as.db.RLock()
	keyMaps := trx.GetOpCreatorsMap()
	for name := range keyMaps {
		ok,have,need := as.pool.CheckNetForRPC(name,as.db,uint64(proto.Size(trx)))
		if !ok {
			as.db.RUnlock()
			err := errors.New(fmt.Sprintf("rpc check net resource not enough, user:%v, have:%v, need:%v",name,have,need))
			return &grpcpb.BroadcastTrxResponse{Invoice: nil, Status: prototype.StatusError}, err
		}
	}
	as.db.RUnlock()

	err := as.consensus.PushTransactionToPending(trx)
	if err != nil {
		return &grpcpb.BroadcastTrxResponse{Invoice: nil, Status: prototype.StatusError}, err
	}

	if !req.OnlyDeliver {
		if !req.Finality {
			return &grpcpb.BroadcastTrxResponse{Invoice: prototype.FetchTrxApplyResult(as.eBus, constants.RpcTimeoutSeconds * time.Second, trx)}, nil
		} else {
			receipt, finality := prototype.FetchTrxFinalResult(as.eBus, constants.RpcTimeoutSeconds * time.Second, trx)
			return &grpcpb.BroadcastTrxResponse{Invoice: receipt, Finality: finality}, nil
		}
	} else {
		return &grpcpb.BroadcastTrxResponse{Invoice: nil, Status: prototype.StatusSuccess}, nil
	}
}

func (as *APIService) getState() *grpcpb.ChainState {
	result := &grpcpb.ChainState{}

	var (
		i int32 = 1
	)

	result.LastIrreversibleBlockNumber = as.consensus.GetLIB().BlockNum()
	lastCommit := as.consensus.GetLastBFTCommit()

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
		blkInfo.BlockProducer = b.SignedHeader.Header.BlockProducer
		blkInfo.TrxCount = uint32(len(b.Transactions))
		blkInfo.BlockId = &prototype.Sha256{}
		blkInfo.BlockId.FromBlockID(b.Id())
		blkInfo.PreId = b.SignedHeader.Header.Previous
		blkInfo.BlockSize = uint32(b.GetBlockSize())
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
			acct := as.getAccountResponseByName(mVal,false)
			if acct != nil {
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

func (as *APIService) GetAccountListByCreTime (ctx context.Context, req *grpcpb.GetAccountListByCreTimeRequest) (*grpcpb.GetAccountListResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()

	sortWrap := table.NewAccountCreatedTimeWrap(as.db)
	res := &grpcpb.GetAccountListResponse{}
	var (
		err error
	    list []*grpcpb.AccountResponse
		lastAcctName *prototype.AccountName
		lastAcctTime *prototype.TimePointSec
	)
	if req.LastAccount != nil {
		lastAcctName = req.LastAccount.AccountName
		lastAcctTime = req.LastAccount.CreatedTime
	}
	limit := checkLimit(req.Limit)
	if limit == 0 {
		limit = uint32(defaultPageSizeLimit)
	}
	if sortWrap != nil {
		err = sortWrap.ForEachByRevOrder(req.Start, req.End, lastAcctName,lastAcctTime, func(mVal *prototype.AccountName, sVal *prototype.TimePointSec, idx uint32) bool {
			acct := as.getAccountResponseByName(mVal,false)
			if acct != nil {
				list = append(list, acct)
			}
			if uint32(len(list)) >= limit {
				return false
			}
			return true
		})
	}

	res.List = list
	
	return res,err
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
	var lastMainKey *prototype.TimePointSec
	var lastSubVal *prototype.TimePointSec
	var hourStat []*grpcpb.StatByHour
	var err error
	res := &grpcpb.TrxStatByHourResponse{}
	wrap := table.NewExtHourTrxHourWrap(as.db)
	hours := int(req.Hours)
	if hours > defaultHourStatLimit {
		hours = defaultHourStatLimit
	}
	//convert the unix timestamp to day index
	now := time.Now().UTC()
	end := now.Unix()/3600 + 1
	start := end - int64(hours)
	s := &prototype.TimePointSec{UtcSeconds: uint32(start)}
	e := &prototype.TimePointSec{UtcSeconds: uint32(end)}
	h, _ := time.ParseDuration("-1h")
	// init from s to e map, hour as key count as value
	// default set value to zero
	var hoursList []uint32
	hourData := make(map[uint32]uint32, hours)
	for i := 0; i < hours; i++ {
		then := now.Add(time.Duration(i) * h)
		hour := uint32(then.Hour())
		hoursList = append(hoursList, hour)
		hourData[hour] = 0
	}
	if wrap != nil {
		err = wrap.ForEachByOrder(s, e, lastMainKey, lastSubVal, func(mVal *prototype.TimePointSec, sVal *prototype.TimePointSec,
			idx uint32) bool {
			if mVal != nil && sVal != nil {
				//info := &grpcpb.StatByHour{}
				dWrap := table.NewSoExtHourTrxWrap(as.db, mVal)
				count := uint32(0)
				if dWrap != nil {
					count = uint32(dWrap.GetCount())
				}
				rawHour := dWrap.GetHour().GetUtcSeconds()
				delta := now.Unix()/3600 - int64(rawHour)
				then := now.Add(time.Duration(delta) * h)
				hour := uint32(then.Hour())
				hourData[hour] = count
			}
			return true
		})
	}
	for _, hour := range hoursList {
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
	info := as.getTrxInfoByTrxId(req.TrxId,nil)
	if info != nil {
		res.Info = info
	}

	return res, nil
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
	if limit > uint32(defaultPageSizeLimit) {
		limit = uint32(defaultPageSizeLimit)
	}
	res := &grpcpb.GetTrxListByTimeResponse{}
	if req.LastInfo != nil && req.LastInfo.TrxId != nil && req.LastInfo.BlockTime != nil {
		lastMainKey = req.LastInfo.TrxId
		lastSubVal = req.LastInfo.BlockTime
	}
	sWrap := table.NewExtTrxBlockTimeWrap(as.db)
	if sWrap != nil {
		var sMap map[uint64]bool
		err = sWrap.ForEachByRevOrder(req.Start, req.End, lastMainKey, lastSubVal, func(mVal *prototype.Sha256, sVal *prototype.TimePointSec, idx uint32) bool {
			info := as.getTrxInfoByTrxId(mVal,sMap)
			if info != nil {
				infoList = append(infoList, info)
				if sMap == nil {
					sMap = make(map[uint64]bool)
				}
				sMap[info.BlockHeight] = info.BlkIsIrreversible
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
						postInfo := as.fetchPostInfoResponseById(*mVal,false)
						if postInfo != nil && postInfo.ParentId <= 0 {
							//Filter reply
							postList = append(postList, postInfo)
						}
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
				postInfo := as.fetchPostInfoResponseById(*mVal,false)
				if postInfo != nil {
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

func (as *APIService) getFreeTicketCount(name *prototype.AccountName) uint32 {
	wrapper := table.NewSoGlobalWrap(as.db, &constants.GlobalId)
	props := wrapper.GetProps()
	freeTicketWrap := table.NewSoGiftTicketWrap(as.db, &prototype.GiftTicketKeyType{
		Type: 0,
		From: "contentos",
		To: name.Value,
		CreateBlock: props.GetCurrentEpochStartBlock(),
	})
	if freeTicketWrap.CheckExist() {
		return 1
	} else {
		return 0
	}
}

func (as *APIService) getAccountResponseByName(name *prototype.AccountName, isNeedLock bool) *grpcpb.AccountResponse {
	if isNeedLock {
		as.db.RLock()
		defer as.db.RUnlock()
	}
	accWrap := table.NewSoAccountWrap(as.db, name)
	acct := &grpcpb.AccountResponse{}
	acctInfo := &grpcpb.AccountInfo{}

	rc := utils.NewResourceLimiter()
	wraper := table.NewSoGlobalWrap(as.db, &constants.GlobalId)
	gp := wraper.GetProps()

	if accWrap != nil && accWrap.CheckExist() {
		acctInfo.AccountName = &prototype.AccountName{Value: accWrap.GetName().Value}
		acctInfo.Coin = accWrap.GetBalance()
		acctInfo.Vest = accWrap.GetVest()
		acctInfo.CreatedTime = accWrap.GetCreatedTime()
		acctInfo.PostCount = accWrap.GetPostCount()
		acctInfo.TrxCount = accWrap.GetCreatedTrxCount()
		acctInfo.VotePower = accWrap.GetVotePower()
		acctInfo.StakeVest = accWrap.GetStakeVest()
		acctInfo.WithdrawRemains = accWrap.GetToPowerdown()
		acctInfo.WithdrawEachTime = accWrap.GetEachPowerdownRate()
		acctInfo.BpVoteCount = accWrap.GetBpVoteCount()
		currentBlockNum := gp.GetHeadBlockNumber()
		currentTime := gp.GetTime()
		nextWithdrawBlock := accWrap.GetNextPowerdownBlockNum()
		withdrawTime := &prototype.TimePointSec{UtcSeconds: 0}
		if nextWithdrawBlock >= currentBlockNum && nextWithdrawBlock != math.MaxUint64 {
			withdrawTime = &prototype.TimePointSec{UtcSeconds: uint32(nextWithdrawBlock - currentBlockNum) + currentTime.UtcSeconds}
		}
		acctInfo.NextWithdrawTime = withdrawTime

		witWrap := table.NewSoBlockProducerWrap(as.db, accWrap.GetName())
		if witWrap != nil && witWrap.CheckExist() && witWrap.GetBpVest().Active {
			acctInfo.BlockProducer = &grpcpb.BlockProducerResponse{
				Owner:                 witWrap.GetOwner(),
				CreatedTime:           witWrap.GetCreatedTime(),
				Url:                   witWrap.GetUrl(),
				BpVest:                witWrap.GetBpVest(),
				SigningKey:            witWrap.GetSigningKey(),
				ProposedStaminaFree:   witWrap.GetProposedStaminaFree(),
				TpsExpected:           witWrap.GetTpsExpected(),
				AccountCreateFee:      witWrap.GetAccountCreateFee(),
				TopNAcquireFreeToken:  witWrap.GetTopNAcquireFreeToken(),
				TicketFlushInterval:   witWrap.GetEpochDuration(),
				PerTicketPrice:        witWrap.GetPerTicketPrice(),
				PerTicketWeight:       witWrap.GetPerTicketWeight(),
				VoterCount:            witWrap.GetVoterCount(),
			}
		}

		keyWrap := table.NewSoAccountWrap(as.db, name)

		if keyWrap.CheckExist() {
			acctInfo.PublicKey = keyWrap.GetPubKey()
		}

		followWrap := table.NewSoExtFollowCountWrap(as.db, name)
		if followWrap != nil && followWrap.CheckExist() {
			acctInfo.FollowerCount = followWrap.GetFollowerCnt()
			acctInfo.FollowingCount = followWrap.GetFollowingCnt()
		}
		freeStaminaMaxByBp := gp.GetStaminaFree()
		_,acctInfo.StaminaFreeRemain = rc.GetFreeLeft(freeStaminaMaxByBp,accWrap.GetStaminaFree(), accWrap.GetStaminaFreeUseBlock(), gp.HeadBlockNumber)
		maxStamina := as.pool.CalculateUserMaxStamina(as.db,accWrap.GetName().Value)
		_,acctInfo.StaminaStakeRemain = rc.GetStakeLeft(accWrap.GetStamina(), accWrap.GetStaminaUseBlock(), gp.HeadBlockNumber, maxStamina)
		acctInfo.StaminaMax = maxStamina + freeStaminaMaxByBp

		acctInfo.Reputation = accWrap.GetReputation()
		acctInfo.ReputationMemo = accWrap.GetReputationMemo()

		acctInfo.FreeTicket = as.getFreeTicketCount(name)
		acctInfo.ChargedTicket = accWrap.GetChargedTicket()

		acctInfo.Freeze = accWrap.GetFreeze()
		acctInfo.FreezeMemo = accWrap.GetFreezeMemo()

		acct.Info = acctInfo
		acct.State = as.getState()

	}else {
		return nil
	}

	return acct
}

func (as *APIService) getBlockProducerResponseByName(name *prototype.AccountName, isNeedLock bool) *grpcpb.BlockProducerResponse {
	if isNeedLock {
		as.db.RLock()
		defer as.db.RUnlock()
	}
	bpWrap := table.NewSoBlockProducerWrap(as.db, name)
	bp := &grpcpb.BlockProducerResponse{}


	if bpWrap != nil && bpWrap.CheckExist() {
		bp.Owner                 = bpWrap.GetOwner()
		bp.CreatedTime           = bpWrap.GetCreatedTime()
		bp.Url                   = bpWrap.GetUrl()
		bp.BpVest                = bpWrap.GetBpVest()
		bp.SigningKey            = bpWrap.GetSigningKey()
		bp.ProposedStaminaFree   = bpWrap.GetProposedStaminaFree()
		bp.TpsExpected           = bpWrap.GetTpsExpected()
		bp.AccountCreateFee      = bpWrap.GetAccountCreateFee()
		bp.TopNAcquireFreeToken  = bpWrap.GetTopNAcquireFreeToken()
		bp.TicketFlushInterval   = bpWrap.GetEpochDuration()
		bp.PerTicketPrice        = bpWrap.GetPerTicketPrice()
		bp.PerTicketWeight       = bpWrap.GetPerTicketWeight()
		bp.VoterCount            = bpWrap.GetVoterCount()
	}else {
		return nil
	}

	return bp
}

func (as *APIService) GetUserTrxListByTime(ctx context.Context, req *grpcpb.GetUserTrxListByTimeRequest) (*grpcpb.GetUserTrxListByTimeResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()
	var (
		trxList []*grpcpb.TrxInfo
		err error
		lastTrxId *prototype.Sha256
		lastCreOrder *prototype.UserTrxCreateOrder
	)
	acct := req.Name
	if acct == nil {
		return &grpcpb.GetUserTrxListByTimeResponse{},errors.New("Account name is empty")
	}
	res := &grpcpb.GetUserTrxListByTimeResponse{}
	wrap := table.NewExtTrxTrxCreateOrderWrap(as.db)
	if wrap != nil {
		limit := checkLimit(req.Limit)
		if limit == 0 {
			limit = uint32(defaultPageSizeLimit)
		}
		start := &prototype.UserTrxCreateOrder{Creator:acct}
		end   := &prototype.UserTrxCreateOrder{Creator:acct}
		if req.Start == nil {
			start.CreateTime = &prototype.TimePointSec{UtcSeconds: math.MaxUint32}
		}else {
			start.CreateTime = req.Start
		}

		if req.End == nil {
			end.CreateTime = &prototype.TimePointSec{UtcSeconds:1}
		}else {
			end.CreateTime = req.End
		}

		if req.LastTrx != nil {
			trx := req.LastTrx
			lastTrxId =  trx.TrxId
			lastCreOrder = &prototype.UserTrxCreateOrder{Creator:acct,CreateTime:trx.BlockTime}
		}
		var sMap map[uint64]bool
		err = wrap.ForEachByRevOrder(start, end, lastTrxId, lastCreOrder, func(mVal *prototype.Sha256, sVal *prototype.UserTrxCreateOrder, idx uint32) bool {
			if mVal != nil {
				info := as.getTrxInfoByTrxId(mVal,sMap)
				if info != nil {
					trxList = append(trxList, info)
					if sMap == nil {
						sMap = make(map[uint64]bool)
					}
					sMap[info.BlockHeight] = info.BlkIsIrreversible
				}
			}
			if uint32(len(trxList)) >= limit {
				return false
			}
			return true
		})

	}
	if len(trxList) < 1 {
		trxList = make([]*grpcpb.TrxInfo, 0)
	}
	res.TrxList = trxList
	return res,err
}

func (as *APIService) GetAccountCashout(ctx context.Context, req *grpcpb.GetAccountCashoutRequest) (*grpcpb.AccountCashoutResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()

	rewardWrap := table.NewSoExtRewardWrap(as.db, &prototype.RewardCashoutId{Account:req.AccountName, PostId:req.PostId})

	if rewardWrap != nil && rewardWrap.CheckExist() {
		reward := rewardWrap.GetReward()
		return &grpcpb.AccountCashoutResponse{AccountName: req.AccountName, Reward: reward}, nil
	}
	return &grpcpb.AccountCashoutResponse{AccountName: req.AccountName, Reward: &prototype.Vest{Value: 0}}, nil
}

func (as *APIService) GetBlockCashout(ctx context.Context, req *grpcpb.GetBlockCashoutRequest) (*grpcpb.BlockCashoutResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()
	blockHeight := req.BlockHeight
	cashoutWrap := table.NewExtRewardBlockHeightWrap(as.db)
	var cashouts []*grpcpb.AccountCashoutResponse
	if cashoutWrap != nil {
		start := blockHeight - 1
		end := blockHeight
		if start < 0 {
			start = 0
		}
		_ = cashoutWrap.ForEachByOrder(&start, &end, nil, nil, func(mVal *prototype.RewardCashoutId, sVal *uint64, idx uint32) bool {
			cWrap := table.NewSoExtRewardWrap(as.db, mVal)
			if cWrap != nil && cWrap.CheckExist() {
				reward := cWrap.GetReward()
				cashout := &grpcpb.AccountCashoutResponse{AccountName: mVal.Account, Reward: reward}
				cashouts = append(cashouts, cashout)
				return true
			}
			return false
		})
	}
	blockCashout := &grpcpb.BlockCashoutResponse{CashoutList: cashouts}
	return blockCashout, nil
}

func (as *APIService) fetchPostInfoResponseById(postId uint64,isNeedLock bool) *grpcpb.PostResponse {
	if isNeedLock {
		as.db.RLock()
		defer as.db.RUnlock()
	}
	pWrap := table.NewSoPostWrap(as.db, &postId)
	var res *grpcpb.PostResponse

	var (
		i int32 = 1
	)

	props := table.NewSoGlobalWrap(as.db, &i).GetProps()


	if pWrap != nil && pWrap.CheckExist() {
		var globalRewards uint64
		var globalWeightedVp string
		if pWrap.GetParentId() == 0 {
			globalRewards = props.PoolPostRewards.Value
			globalWeightedVp = props.WeightedVpsPost
		} else {
			globalRewards = props.PoolReplyRewards.Value
			globalWeightedVp = props.WeightedVpsReply
		}
		res  =	&grpcpb.PostResponse{
			PostId:        pWrap.GetPostId(),
			Category:      pWrap.GetCategory(),
			ParentAuthor:  pWrap.GetAuthor(),
			Author:        pWrap.GetAuthor(),
			Title:         pWrap.GetTitle(),
			Body:          pWrap.GetBody(),
			Created:       pWrap.GetCreated(),
			LastPayout:    pWrap.GetLastPayout(),
			Depth:         pWrap.GetDepth(),
			Children:      pWrap.GetChildren(),
			RootId:        pWrap.GetRootId(),
			ParentId:      pWrap.GetParentId(),
			Tags:          pWrap.GetTags(),
			Beneficiaries: pWrap.GetBeneficiaries(),
			VoteCnt:       pWrap.GetVoteCnt(),
			Rewards:       pWrap.GetRewards(),
			DappRewards:   pWrap.GetDappRewards(),
			WeightedVp:    pWrap.GetWeightedVp(),
			Ticket:        pWrap.GetTicket(),
			CashoutInterval:  constants.PostCashOutDelayBlock,
			GlobalRewards: &prototype.Vest{Value: globalRewards},
			GlobalWeightedVp: globalWeightedVp,
			Copyright:      pWrap.GetCopyright(),
			CopyrightMemo:  pWrap.GetCopyrightMemo(),

		}
	}
	return res
}

func (as *APIService) GetPostInfoById (ctx context.Context, req *grpcpb.GetPostInfoByIdRequest) (*grpcpb.GetPostInfoByIdResponse, error){
	as.db.RLock()
	defer as.db.RUnlock()
	res := &grpcpb.GetPostInfoByIdResponse{}

	pId := &req.PostId
	postInfo := as.fetchPostInfoResponseById(req.PostId,false)
	res.PostInfo = postInfo
	if postInfo != nil {
		voterLimit := checkLimit(req.VoterListLimit)
		if voterLimit > 0 {
			voteWrap := table.NewVotePostIdWrap(as.db)
			if voteWrap != nil {
				end := req.PostId+1
				var voterList []*grpcpb.VoterOfPost
				err := voteWrap.ForEachByOrder(pId,&end,nil, nil, func(mVal *prototype.VoterId, sVal *uint64, idx uint32) bool {
					if mVal != nil {
						voter := &grpcpb.VoterOfPost{}
						voteWrap := table.NewSoVoteWrap(as.db,mVal)
						if voteWrap != nil && voteWrap.CheckExist() {
							voter.WeightedVp =voteWrap.GetWeightedVp()
						}
						voter.AccountName = mVal.Voter
						voterList = append(voterList,voter)

					}
					if uint32(len(voterList)) >= voterLimit {
						return false
					}
					return true
				})
				if err != nil {
					return res,err
				}
				res.VoterList = voterList
			}

		}

		replyLimit := checkLimit(req.ReplyListLimit)
		if replyLimit > 0 {
			replyOrderWrap := table.NewExtReplyCreatedCreatedOrderWrap(as.db)
			var replyList []*grpcpb.PostResponse
			if replyOrderWrap != nil {
				start := &prototype.ReplyCreatedOrder{ParentId:req.PostId,Created:prototype.NewTimePointSec(math.MaxUint32)}
				end := &prototype.ReplyCreatedOrder{ParentId:req.PostId,Created:prototype.NewTimePointSec(1)}
				err := replyOrderWrap.ForEachByRevOrder(start,end,nil,nil, func(mVal *uint64, sVal *prototype.ReplyCreatedOrder, idx uint32) bool {
					if mVal != nil {
                       reply :=  as.fetchPostInfoResponseById(*mVal,false)
                       if reply != nil {
						   replyList = append(replyList,reply)
					   }
					}
					if uint32(len(replyList)) >= replyLimit {
						return false
					}
					return true
				})
				if err != nil {
					return res,err
				}
				res.ReplyList = replyList
			}
		}
	}

	return res,nil
}



func (as *APIService) GetContractInfo (ctx context.Context, req *grpcpb.GetContractInfoRequest) (*grpcpb.GetContractInfoResponse, error){
	as.db.RLock()
	defer as.db.RUnlock()
	res := &grpcpb.GetContractInfoResponse{ Exist:false }

	cid := prototype.ContractId{Owner: req.Owner, Cname: req.ContractName}
	scid := table.NewSoContractWrap(as.db, &cid)

	if scid.CheckExist() {
		res.Exist = true

		if req.FetchAbi{
			res.Abi = scid.GetAbi()
		}
		if req.FetchCode{
			res.Code = scid.GetCode()
		}
		res.Url = scid.GetUrl()
		res.Describe = scid.GetDescribe()
	}

	return res, nil
}

func (as *APIService) GetBlkIsIrreversibleByTxId (ctx context.Context,
	req *grpcpb.GetBlkIsIrreversibleByTxIdRequest) (*grpcpb.GetBlkIsIrreversibleByTxIdResponse,error){

	as.db.RLock()
	defer as.db.RUnlock()

	res := &grpcpb.GetBlkIsIrreversibleByTxIdResponse{Result:false}

	if req.TrxId == nil {
		return res,errors.New("trx id is empty")
	}

	res.Result = as.judgeBlkIsIrreversibleByTxId(req.TrxId)

    return res,nil
}

func (as *APIService) judgeBlkIsIrreversibleByTxId(trxId *prototype.Sha256) bool {
	if trxId != nil {
		trxWrap := table.NewSoExtTrxWrap(as.db,trxId)
		if trxWrap != nil && trxWrap.CheckExist() {
			blkHash := trxWrap.GetBlockId().Hash
			return as.judgeBlkIsIrreversibleByHash(blkHash)
		}
	}
	return false
}

func (as *APIService) judgeBlkIsIrreversibleByHash(blkHash []byte) bool {
	res := false
	if blkHash != nil && len(blkHash) >= 32 {
		data := [32]byte{}
		copy(data[:],blkHash[:32])
		bId := common.BlockID{Data:data}
		res =  as.consensus.IsCommitted(bId)
	}
	return res
}

func (as *APIService) getTrxInfoByTrxId(trxId *prototype.Sha256, blkStateMap map[uint64]bool) *grpcpb.TrxInfo {
	var tInfo *grpcpb.TrxInfo
	if trxId != nil {
		wrap := table.NewSoExtTrxWrap(as.db, trxId)
		if wrap != nil && wrap.CheckExist() {
			info := &grpcpb.TrxInfo{}
			info.TrxId = trxId
			info.BlockHeight = wrap.GetBlockHeight()
			info.BlockTime = wrap.GetBlockTime()
			info.TrxWrap = wrap.GetTrxWrap()
			info.BlockId = wrap.GetBlockId()
			hasState := false
			if blkStateMap != nil {
				if res,ok := blkStateMap[info.BlockHeight]; ok {
					hasState = true
					info.BlkIsIrreversible = res
				}
			}
			if !hasState {
                info.BlkIsIrreversible = as.judgeBlkIsIrreversibleByHash(info.BlockId.Hash)
			}
			tInfo = info
		}
	}
	return tInfo
}

func (as *APIService) GetDailyStats(ctx context.Context, req *grpcpb.GetDailyStatsRequest) (*grpcpb.GetDailyStatsResponse, error) {
	dsservice, err := as.ctx.Service(iservices.DailyStatisticServiceName)
	if err != nil {
		return nil, errors.New("plugin daily statistic service isn't running")
	}
	ds := dsservice.(iservices.IDailyStats)
	rows := ds.DailyStatsSince(int(req.Days), req.Dapp)
	var stat []*grpcpb.DailyStat
	res := &grpcpb.GetDailyStatsResponse{}
	for _, row := range rows {
		t, _ := time.Parse("2006-01-02", row.Date)
		stat = append(stat, &grpcpb.DailyStat{Date: uint64(t.UTC().Unix()), Dapp: row.Dapp, Dau: row.Dau, Dnu: row.Dnu, Trxs: row.TrxCount, Amount: row.Amount, TotalUserCount: row.TotalUserCount})
	}
	res.Stat = stat
	return res, nil
}

func (as *APIService) GetContractListByTime(ctx context.Context, req *grpcpb.GetContractListByTimeRequest) (*grpcpb.GetContractListResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()
	var (
		err error
		list []*grpcpb.ContractInfo
		lastMainKey *prototype.ContractId
		lastSubKey *prototype.TimePointSec
	)
	limit := checkLimit(req.Limit)
	if limit <= 0 {
		limit = uint32(defaultPageSizeLimit)
	}
	res := &grpcpb.GetContractListResponse{}
	lastCon := req.LastContract
	if lastCon != nil {
		if lastCon.Owner != nil && lastCon.Name != nil {
			lastMainKey = &prototype.ContractId{Owner: lastCon.Owner, Cname: lastCon.Name.GetValue()}
		}
		lastSubKey = lastCon.CreateTime
	}
	sortWrap := table.NewContractCreatedTimeWrap(as.db)
	if sortWrap != nil {
		err = sortWrap.ForEachByRevOrder(req.Start, req.End, lastMainKey, lastSubKey, func(mVal *prototype.ContractId, sVal *prototype.TimePointSec, idx uint32) bool {
			if mVal != nil {
				scid := table.NewSoContractWrap(as.db, mVal)
				if scid.CheckExist() {
					info := &grpcpb.ContractInfo{
						Owner: mVal.Owner,
						Name:  &prototype.AccountName{Value: mVal.Cname},
						CreateTime: sVal,
						Balance: scid.GetBalance(),
						ApplyCount: scid.GetApplyCount(),
					}
					list = append(list, info)
				}
			}

			if uint32(len(list)) >= limit {
				return false
			}
			return true
		})

	}
	res.ContractList = list
    return  res,err
}

func (as *APIService) GetBlockProducerListByVoteCount(ctx context.Context, req *grpcpb.GetBlockProducerListByVoteCountRequest) (*grpcpb.GetBlockProducerListResponse,error){
	as.db.RLock()
	defer as.db.RUnlock()
	var (
		err error
		witList []*grpcpb.BlockProducerResponse
		limit   uint32
		lastMainKey *prototype.AccountName
		lastSubVal  *prototype.BpVestId
	)
	res := &grpcpb.GetBlockProducerListResponse{}
	limit = checkLimit(req.Limit)
	srtWrap := table.NewBlockProducerBpVestWrap(as.db)
	if srtWrap != nil {
		lastWit := req.LastBlockProducer
		if lastWit != nil {
			lastMainKey = &prototype.AccountName{Value:lastWit.Owner.Value}
			lastSubVal = lastWit.BpVest

		}
		startKey := &prototype.BpVestId{Active:true, VoteVest:req.Start}
		endKey := &prototype.BpVestId{Active:true, VoteVest:req.End}
		err = srtWrap.ForEachByRevOrder(startKey, endKey, lastMainKey,  lastSubVal,
			func(mVal *prototype.AccountName, sVal *prototype.BpVestId, idx uint32) bool {
				if mVal != nil {
					bp := as.getBlockProducerResponseByAccountName(mVal)
					if bp != nil {
						witList = append(witList, bp)
					}
				}
				if uint32(len(witList)) >= limit {
					return false
				}
				return true
			})
	}
    res.BlockProducerList = witList
	return res,err
}

func (as *APIService) getBlockProducerResponseByAccountName(acct *prototype.AccountName) *grpcpb.BlockProducerResponse {
	if acct != nil {
		witWrap := table.NewSoBlockProducerWrap(as.db, acct)
		if witWrap != nil && witWrap.CheckExist() {
			bp := &grpcpb.BlockProducerResponse{
				Owner:                 witWrap.GetOwner(),
				CreatedTime:           witWrap.GetCreatedTime(),
				Url:                   witWrap.GetUrl(),
				BpVest:                witWrap.GetBpVest(),
				SigningKey:            witWrap.GetSigningKey(),
				ProposedStaminaFree:   witWrap.GetProposedStaminaFree(),
				TpsExpected:           witWrap.GetTpsExpected(),
				VoterCount:            witWrap.GetVoterCount(),
			}
			return bp
		}
	}
	return nil
}

func (as *APIService) GetPostListByVest (ctx context.Context,
	req *grpcpb.GetPostListByVestRequest) (*grpcpb.GetPostListByVestResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()

	var (
		postList []*grpcpb.PostResponse
		limit    uint32
		err      error
		lastPostId *uint64
		lastPostVest *prototype.Vest

	)
	res := &grpcpb.GetPostListByVestResponse{}
	srtWrap := table.NewPostRewardsWrap(as.db)
	if srtWrap != nil {
		limit = checkLimit(req.Limit)
		if req.LastPost != nil {
			lastPostId = &req.LastPost.PostId
			lastPostVest = req.LastPost.Rewards
		}

		err = srtWrap.ForEachByRevOrder(req.Start, req.End, lastPostId, lastPostVest, func(mVal *uint64, sVal *prototype.Vest, idx uint32) bool {
			if mVal != nil {
				post := as.fetchPostInfoResponseById(*mVal, false)
				if post != nil && post.ParentId <= 0 {
					//Filter reply
					postList = append(postList, post)
				}
			}
			if uint32(len(postList)) >= limit {
				return false
			}
			return true
		})
	}
	res.PostList = postList
	return res,err

}

func (as *APIService) GetNodeNeighbours(ctx context.Context, req *grpcpb.NonParamsRequest) (*grpcpb.GetNodeNeighboursResponse, error) {

	peerList := as.p2p.GetNodeNeighbours()
	if peerList == "" {
		return nil, errors.New("peer has no neighbours")
	}

	ret := &grpcpb.GetNodeNeighboursResponse{}
	ret.Peerlist = peerList

	return ret, nil
}

func (as *APIService) GetNodeRunningVersion(ctx context.Context, req *grpcpb.NonParamsRequest) (*grpcpb.GetNodeRunningVersionResponse, error) {
	ret := &grpcpb.GetNodeRunningVersionResponse{}
	ret.NodeVersion = as.ctx.Config().P2P.RunningCodeVersion

	return ret, nil
}

func (as *APIService) GetAccountListByVest(ctx context.Context, req *grpcpb.GetAccountListByVestRequest) (*grpcpb.GetAccountListResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()

	sortWrap := table.NewAccountVestWrap(as.db)
	var list []*grpcpb.AccountResponse
	res := &grpcpb.GetAccountListResponse{}
	var err error
	var lastAcctNam *prototype.AccountName
	var lastAcctVest *prototype.Vest
	limit := checkLimit(req.Limit)
	if limit == 0 {
		limit = uint32(defaultPageSizeLimit)
	}
	if req.LastAccount != nil {
		account := req.LastAccount
		if account.AccountName != nil && account.Vest != nil {
			lastAcctNam = account.AccountName
			lastAcctVest = account.Vest
		}
	}
	if sortWrap != nil {
		err = sortWrap.ForEachByRevOrder(req.Start, req.End, lastAcctNam, lastAcctVest, func(mVal *prototype.AccountName, sVal *prototype.Vest, idx uint32) bool {
			acct := as.getAccountResponseByName(mVal,false)
			if acct != nil {
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

func (as *APIService) GetBlockProducerByName(ctx context.Context, req *grpcpb.GetBlockProducerByNameRequest) (*grpcpb.BlockProducerResponse, error) {
	as.db.RLock()
	defer as.db.RUnlock()

	acct := as.getBlockProducerResponseByName(req.GetBpName(),false)
	if acct == nil {
		return nil, errors.New("no bp information")
	}

	return acct, nil

}