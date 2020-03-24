package app

import (
	"fmt"
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/app/annual_mint"
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"math"
	"math/big"
)

const CashoutCompleted uint64 = math.MaxUint64

type IItem interface {
	GetWvp() *big.Int
}

type Item struct {
	beneficiary string
	wvp *big.Int
}

func (i *Item) GetWvp() *big.Int {
	return i.wvp
}

type PostItem struct {
	Item
	postId uint64
}

type VoteItem struct {
	Item
	postId uint64
}

type DappItem struct {
	Item
	postId uint64
}

func Min(x, y uint64) uint64 {
	if x < y {
		return x
	} else {
		return y
	}
}

func GreaterThanZero(number *big.Int) bool {
	return number.Cmp(new(big.Int).SetUint64(0)) > 0
}

func EqualZero(number *big.Int) bool {
	return number.Cmp(new(big.Int).SetUint64(0)) == 0
}

func ProportionAlgorithm(numerator *big.Int, denominator *big.Int, total *big.Int) *big.Int {
	if denominator.Cmp(new(big.Int).SetUint64(0)) == 0 {
		return new(big.Int).SetUint64(0)
	} else {
		numeratorMul := new(big.Int).Mul(numerator, total)
		result := new(big.Int).Div(numeratorMul, denominator)
		return result
	}
}

func StringToBigInt(n string) *big.Int {
	bigInt := new(big.Int)
	if value, success := bigInt.SetString(n, 10); !success {
		panic(fmt.Sprintf("StringToBigInt cannot convert %s to big.Int", n))
	} else {
		return value
	}
}

func Decay(rawValue *big.Int) *big.Int {
	decayValue := ProportionAlgorithm(new(big.Int).SetUint64(constants.BlockInterval), new(big.Int).SetUint64(constants.VpDecayTime), rawValue)
	rawValue.Sub(rawValue, decayValue)
	return rawValue
}

func SumItemsWvp(items []IItem) *big.Int {
	sum := new(big.Int).SetUint64(0)
	for _, item := range items {
		sum = sum.Add(sum, item.GetWvp())
	}
	return sum
}

type Economist struct {
	db       iservices.IDatabaseService
	noticer  EventBus.Bus
	log *logrus.Logger
	dgp *DynamicGlobalPropsRW
	stateChange *blocklog.StateChangeContext
	hardFork func()uint64
}

func NewEconomist(db iservices.IDatabaseService, noticer EventBus.Bus, log *logrus.Logger, hardForkFunc func()uint64) *Economist {
	return &Economist{db: db, noticer:noticer, log: log, dgp: &DynamicGlobalPropsRW{db: db}, hardFork:hardForkFunc}
}

func (e *Economist) getAccount(account *prototype.AccountName) (*table.SoAccountWrap, error) {
	accountWrap := table.NewSoAccountWrap(e.db, account)
	if !accountWrap.CheckExist() {
		return nil, errors.New(fmt.Sprintf("cannot find account %s", account.Value))
	}
	return accountWrap, nil
}

func (e *Economist) Mint() {
	e.stateChange.PushCause("mint")

	//t0 := time.Now()
	globalProps := e.dgp.GetProps()
	if !globalProps.GetBlockProducerBootCompleted() {
		return
	}
	ith := globalProps.GetIthYear()
	annualBudget := annual_mint.CalculateBudget(ith)
	// new year arrived
	if globalProps.GetAnnualBudget().Value != annualBudget {
		e.dgp.ModifyProps(func(props *prototype.DynamicProperties) {
			props.AnnualBudget.Value = annualBudget
			props.AnnualMinted.Value = 0
		})
		// reload props
		globalProps = e.dgp.GetProps()
	}
	blockCurrent := annual_mint.CalculatePerBlockBudget(annualBudget)
	// prevent deficit
	if globalProps.GetAnnualBudget().Value > globalProps.GetAnnualMinted().Value &&
		globalProps.GetAnnualBudget().Value <= (globalProps.GetAnnualMinted().Value + blockCurrent) {
		blockCurrent = globalProps.GetAnnualBudget().Value - globalProps.GetAnnualMinted().Value
		// time to update year
		e.dgp.ModifyProps(func(props *prototype.DynamicProperties) {
			props.IthYear = props.IthYear + 1
		})
	}

	if globalProps.GetAnnualBudget().Value <= globalProps.GetAnnualMinted().Value {
		blockCurrent = 0
		e.dgp.ModifyProps(func(props *prototype.DynamicProperties) {
			props.IthYear = props.IthYear + 1
		})
	}

	creatorReward := blockCurrent * constants.RewardRateCreator / constants.PERCENT
	dappReward := blockCurrent * constants.RewardRateDapp / constants.PERCENT
	bpReward := blockCurrent - creatorReward - dappReward

	// merge author rewards and reply rewards
	postReward := creatorReward * constants.RewardRateAuthor / constants.PERCENT
	replyReward := creatorReward * constants.RewardRateReply / constants.PERCENT
	voteReward := creatorReward - postReward - replyReward

	e.stateChange.PopCause()
	e.stateChange.PushCause("reward")
	e.stateChange.PushCause("bp")
	bpWrap, err := e.getAccount(globalProps.CurrentBlockProducer)
	if err != nil {
		panic("Mint failed when get bp wrap")
	}
	// add rewards to bp
	bpRewardVest := &prototype.Vest{Value: bpReward}
	// add ticket fee to the bp
	oldVest := bpWrap.GetVest()

	//bpWrap.SetVest(&prototype.Vest{Value: bpWrap.GetVest().Value + bpReward})
	bpRewardVest.Add(bpWrap.GetVest())
	bpWrap.SetVest(bpRewardVest)

	bpProducerWrap := table.NewSoBlockProducerWrap(e.db, globalProps.CurrentBlockProducer)
	bpProducerWrap.Modify(func(tInfo *table.SoBlockProducer) {
		tInfo.GenBlockCount ++
	})

	updateBpVoteValue(e.db, globalProps.CurrentBlockProducer, oldVest, bpRewardVest)
	e.stateChange.PopCause()
	e.stateChange.PopCause()

	e.stateChange.PushCause("mint")
	e.dgp.ModifyProps(func(props *prototype.DynamicProperties) {
		props.PoolPostRewards.Add(&prototype.Vest{Value: postReward})
		props.PoolReplyRewards.Add(&prototype.Vest{Value: replyReward})
		props.PoolVoteRewards.Add(&prototype.Vest{Value: voteReward})
		props.PoolDappRewards.Add(&prototype.Vest{Value: dappReward})
		props.AnnualMinted.Add(&prototype.Vest{Value: blockCurrent})
		props.TotalVest.Add(&prototype.Vest{Value: blockCurrent})
	})
	e.stateChange.PopCause()
}

// maybe slow
func (e *Economist) Distribute() {
	e.stateChange.PushCause("free_ticket")
	defer e.stateChange.PopCause()

	globalProps := e.dgp.GetProps()
	if globalProps.GetCurrentEpochStartBlock() == uint64(0) {
		return
	}
	current := globalProps.HeadBlockNumber
	if globalProps.GetCurrentEpochStartBlock() + globalProps.GetEpochDuration() > current {
		return
	}
	iterator := table.NewAccountVestWrap(e.db)
	var accounts  []*prototype.AccountName
	var count uint32 = 0
	topN := globalProps.GetTopNAcquireFreeToken()
	err := iterator.ForEachByRevOrder(nil, nil, nil, nil, func(account *prototype.AccountName, sVal *prototype.Vest, idx uint32) bool {
		if count > topN {
			return false
		}
		accounts = append(accounts, account)
		count += 1
		return true
	})
	if err != nil {
		panic("economist distribute failed when iterator")
	}
	e.log.Info("economist epoch start block:", globalProps.GetCurrentEpochStartBlock())
	for _, account := range accounts {
		// type 0 free ticket
		key := &prototype.GiftTicketKeyType{Type: 0, From: constants.COSSysAccount, To: account.Value,
			CreateBlock: current}
		wrap := table.NewSoGiftTicketWrap(e.db, key)
		// impossible
		if wrap.CheckExist() {
			wrap.SetExpireBlock(current + globalProps.GetEpochDuration())
		} else {
			wrap.Create(func(tInfo *table.SoGiftTicket) {
				tInfo.Ticket = key
				tInfo.Denom = globalProps.PerTicketWeight
				tInfo.Count = 1
				tInfo.ExpireBlock = current + globalProps.GetEpochDuration()
			})
		}
	}
	e.dgp.ModifyProps(func(props *prototype.DynamicProperties) {
		props.CurrentEpochStartBlock = current
	})
}

func (e *Economist) decayGlobalWvp() {
	e.dgp.ModifyProps(func(props *prototype.DynamicProperties) {
		postWeightedVps := StringToBigInt(props.GetWeightedVpsPost())
		replyWeightedVps := StringToBigInt(props.GetWeightedVpsReply())
		voteWeightedVps := StringToBigInt(props.GetWeightedVpsVote())
		dappWeightedVps := StringToBigInt(props.GetWeightedVpsDapp())
		Decay(postWeightedVps)
		Decay(replyWeightedVps)
		Decay(voteWeightedVps)
		Decay(dappWeightedVps)
		props.WeightedVpsPost = postWeightedVps.String()
		props.WeightedVpsReply = replyWeightedVps.String()
		props.WeightedVpsVote = voteWeightedVps.String()
		props.WeightedVpsDapp = dappWeightedVps.String()
	})
}

func (e *Economist) Do() {
	e.stateChange.PushCause("reward")
	defer e.stateChange.PopCause()

	globalProps := e.dgp.GetProps()
	if !globalProps.GetBlockProducerBootCompleted() {
		return
	}
	e.decayGlobalWvp()
	iterator := table.NewPostCashoutBlockNumWrap(e.db)
	//end := globalProps.HeadBlockNumber
	// in iterator, the right is open.
	end := globalProps.HeadBlockNumber + 1
	// post and reply
	var pids []*uint64
	err := iterator.ForEachByOrder(nil, &end, nil, nil, func(mVal *uint64, sVal *uint64, idx uint32) bool {
		pids = append(pids, mVal)
		return true
	})
	if err != nil {
		panic("economist do failed when iterator")
	}
	var posts []*PostItem
	var replies []*PostItem
	var dappsRoutes []*DappItem

	for _, pid := range pids {
		post := table.NewSoPostWrap(e.db, pid)
		if !post.CheckExist() {
			e.log.Warnf("post %d ignored to cashout, pid not found", pid)
			continue
		}
		giftNum := new(big.Int).SetUint64(uint64(post.GetTicket()))
		giftVp := new(big.Int).Mul(giftNum, new(big.Int).SetUint64(globalProps.GetPerTicketWeight()))
		weightedVp := new(big.Int).Add(StringToBigInt(post.GetWeightedVp()), giftVp)
		authorName := post.GetAuthor()
		// set wvp to zero
		if author, err := e.getAccount(authorName); err != nil {
			weightedVp = new(big.Int).SetUint64(0)
			e.log.Warnf("author of post %d not found, name %s", *pid, authorName.Value)
		} else if author.GetReputation() == constants.MinReputation {
			weightedVp = new(big.Int).SetUint64(0)
			e.log.Warnf("ignored post %d due to bad reputation of author %s", *pid, authorName.Value)
		}
		if post.GetCopyright() == constants.CopyrightInfringement {
			weightedVp = new(big.Int).SetUint64(0)
			e.log.Warnf("ignored post %d due to invalid copyright,author %s", *pid, authorName.Value)
		}
		//postItem := &PostItem{postId: post.GetPostId(), Item{beneficiary: post.GetAuthor().Value, wvp: weightedVp}}
		postItem := &PostItem{Item{beneficiary: post.GetAuthor().Value, wvp: weightedVp}, post.GetPostId()}
		if post.GetParentId() == constants.PostInvalidId {
			posts = append(posts, postItem)
		} else {
			replies = append(replies, postItem)
		}

		beneficiaryRoutes := post.GetBeneficiaries()
		for _, beneficiaryRoute := range beneficiaryRoutes {
			name := beneficiaryRoute.Name.Value
			weight := beneficiaryRoute.Weight
			routeWvp := ProportionAlgorithm(new(big.Int).SetUint64(uint64(weight)), new(big.Int).SetUint64(uint64(constants.PERCENT)), weightedVp)
			if post.GetParentId() == constants.PostInvalidId {
				dappRoute := &DappItem{Item{beneficiary: name, wvp: routeWvp}, post.GetPostId()}
				dappsRoutes = append(dappsRoutes, dappRoute)
			} else {
				// 15 / 75 * routeWvp
				equalRouteWvp := ProportionAlgorithm(new(big.Int).SetUint64(uint64(constants.RewardRateReply)), new(big.Int).SetUint64(uint64(constants.RewardRateAuthor)), routeWvp)
				dappRoute := &DappItem{Item{beneficiary: name, wvp: equalRouteWvp}, post.GetPostId()}
				dappsRoutes = append(dappsRoutes, dappRoute)
			}
		}
	}
	e.cashoutPosts(posts)
	e.cashoutReplies(replies)
	e.cashoutDapps(dappsRoutes)

	voteCashoutWrap := table.NewSoVoteCashoutWrap(e.db, &globalProps.HeadBlockNumber)
	if !voteCashoutWrap.CheckExist() {
		return
	}
	var voteItems []*VoteItem
	voteIds := voteCashoutWrap.GetVoterIds()
	for _, voteId := range voteIds {
		voteWrap := table.NewSoVoteWrap(e.db, voteId)
		voteWvp := StringToBigInt(voteWrap.GetWeightedVp())
		voter := voteWrap.GetVoter().Voter.Value
		postId := voteId.PostId
		postWrap := table.NewSoPostWrap(e.db, &postId)
		// if the voted post is illegal, voter does not receive reward
		authorName := postWrap.GetAuthor()
		if author, err := e.getAccount(authorName); err != nil {
			voteWvp.SetUint64(0)
			e.log.Warnf("author of post %d not found, name %s", postId, authorName.Value)
		} else if author.GetReputation() == constants.MinReputation {
			voteWvp.SetUint64(0)
			e.log.Warnf("ignored post %d due to bad reputation of author %s", postId, authorName.Value)
		}
		if postWrap.GetCopyright() == constants.CopyrightInfringement {
			voteWvp.SetUint64(0)
			e.log.Warnf("ignored post %d due to invalid copyright,author %s", postId, authorName.Value)
		}
		voteItem := &VoteItem{Item{wvp: voteWvp, beneficiary: voter}, postId}
		voteItems = append(voteItems, voteItem)
	}
	e.cashoutVotes(voteItems)
}

func (e *Economist) cashoutPosts(postsItems []*PostItem) {
	e.stateChange.PushCause("post_author")
	defer e.stateChange.PopCause()

	if len(postsItems) == 0 {
		return
	}
	globalProps := e.dgp.GetProps()
	var items []IItem
	for _, postItem := range postsItems {
		items = append(items, postItem)
	}
	currentBlockPostsWvp := SumItemsWvp(items)
	globalPostsWvps := StringToBigInt(globalProps.GetWeightedVpsPost())
	globalPostRewards := globalProps.GetPoolPostRewards()

	currentGlobalPostsWvps := new(big.Int).Add(globalPostsWvps, currentBlockPostsWvp)

	// add global post wvp
	e.dgp.ModifyProps(func(props *prototype.DynamicProperties) {
		props.WeightedVpsPost = currentGlobalPostsWvps.String()
	})
	claimedPostsReward := new(big.Int).SetUint64(0)
	for _, postItem := range postsItems {
		wvp := postItem.wvp
		postReward := ProportionAlgorithm(wvp, currentGlobalPostsWvps, new(big.Int).SetUint64(globalPostRewards.Value))
		e.stateChange.PutCauseExtra("post", postItem.postId)
		e.stateChange.PutCauseExtra("wvps", wvp.String())
		e.stateChange.PutCauseExtra("pool", globalPostRewards.Value)
		e.stateChange.PutCauseExtra("total_wvps", currentGlobalPostsWvps.String())
		post := table.NewSoPostWrap(e.db, &postItem.postId)
		e.stateChange.PutCauseExtra("rootid", post.GetRootId())
		e.stateChange.PutCauseExtra("reward", postReward.Uint64())
		e.stateChange.PutCauseExtra("owner", postItem.beneficiary)

		// result false: author banned
		result := e.processRewardForAccount(postItem.beneficiary, postReward)
		if !result {
			postReward = new(big.Int).SetUint64(0)
		}
		e.finalizePostCashout(postItem.postId, postReward)
		claimedPostsReward = claimedPostsReward.Add(claimedPostsReward, postReward)
		e.notifyPostCashoutResult(postItem.beneficiary, postItem.postId, wvp, postReward, globalProps)
	}

	//subtract reward from global post reward pool and add it to claimed reward pool
	e.dgp.ModifyProps(func(props *prototype.DynamicProperties) {
		// the claimed post reward does not beyond max uint64
		currentBlockClaimedReward := &prototype.Vest{Value: claimedPostsReward.Uint64()}
		props.PoolPostRewards.Sub(currentBlockClaimedReward)
		props.ClaimedPostRewards.Add(currentBlockClaimedReward)
	})
}

func (e *Economist) cashoutReplies(repliesItems []*PostItem) {
	e.stateChange.PushCause("reply_author")
	defer e.stateChange.PopCause()

	if len(repliesItems) == 0 {
		return
	}
	globalProps := e.dgp.GetProps()
	var items []IItem
	for _, replyItem := range repliesItems {
		items = append(items, replyItem)
	}
	currentBlockRepliesWvp := SumItemsWvp(items)
	globalRepliesWvps := StringToBigInt(globalProps.GetWeightedVpsReply())
	globalRepliesRewards := globalProps.GetPoolReplyRewards()

	currentGlobalRepliesWvps := new(big.Int).Add(globalRepliesWvps, currentBlockRepliesWvp)

	e.dgp.ModifyProps(func(props *prototype.DynamicProperties) {
		props.WeightedVpsReply = currentGlobalRepliesWvps.String()
	})

	claimedRepliesReward := new(big.Int).SetUint64(0)
	for _, replyItem := range repliesItems {
		wvp := replyItem.wvp
		replyReward := ProportionAlgorithm(wvp, currentGlobalRepliesWvps, new(big.Int).SetUint64(globalRepliesRewards.Value))
		e.stateChange.PutCauseExtra("post", replyItem.postId)
		e.stateChange.PutCauseExtra("wvps", wvp.String())
		e.stateChange.PutCauseExtra("pool", globalRepliesRewards.Value)
		e.stateChange.PutCauseExtra("total_wvps", currentGlobalRepliesWvps.String())
		reply := table.NewSoPostWrap(e.db, &replyItem.postId)
		e.stateChange.PutCauseExtra("rootid", reply.GetRootId())
		e.stateChange.PutCauseExtra("reward", replyReward.Uint64())
		e.stateChange.PutCauseExtra("owner", replyItem.beneficiary)

		result := e.processRewardForAccount(replyItem.beneficiary, replyReward)
		if !result {
			replyReward = new(big.Int).SetUint64(0)
		}
		e.finalizePostCashout(replyItem.postId, replyReward)
		claimedRepliesReward.Add(claimedRepliesReward, replyReward)
		e.notifyReplyCashoutResult(replyItem.beneficiary, replyItem.postId, wvp, replyReward, globalProps)
	}

	e.dgp.ModifyProps(func(props *prototype.DynamicProperties) {
		currentBlockClaimedRepliesReward := &prototype.Vest{Value: claimedRepliesReward.Uint64()}
		props.ClaimedReplyRewards.Add(currentBlockClaimedRepliesReward)
		props.PoolReplyRewards.Sub(currentBlockClaimedRepliesReward)
	})
}

func (e *Economist) cashoutDapps(dappsItems []*DappItem) {
	e.stateChange.PushCause("dapp")
	defer e.stateChange.PopCause()

	if len(dappsItems) == 0 {
		return
	}
	globalProps := e.dgp.GetProps()
	var items []IItem
	for _, dappItem := range dappsItems {
		items = append(items, dappItem)
	}
	currentBlockDappsWvp := SumItemsWvp(items)
	globalDappsWvps := StringToBigInt(globalProps.GetWeightedVpsDapp())
	globalDappsRewards := globalProps.GetPoolDappRewards()

	currentGlobalDappsWvps := new(big.Int).Add(globalDappsWvps, currentBlockDappsWvp)

	e.dgp.ModifyProps(func(prop *prototype.DynamicProperties) {
		prop.WeightedVpsDapp = currentGlobalDappsWvps.String()
	})

	claimedDappReward := new(big.Int).SetUint64(0)
	for _, dappItem := range dappsItems {
		wvp := dappItem.wvp
		dappReward := ProportionAlgorithm(wvp, currentGlobalDappsWvps, new(big.Int).SetUint64(globalDappsRewards.Value))
		e.stateChange.PutCauseExtra("post", dappItem.postId)
		e.stateChange.PutCauseExtra("wvps", wvp.String())
		e.stateChange.PutCauseExtra("pool", globalDappsRewards.Value)
		e.stateChange.PutCauseExtra("total_wvps", currentGlobalDappsWvps.String())

		result := e.processRewardForAccount(dappItem.beneficiary, dappReward)
		if !result {
			dappReward = new(big.Int).SetUint64(0)
		}
		e.finalizePostDappCashout(dappItem.postId, dappReward)
		claimedDappReward.Add(claimedDappReward, dappReward)
		e.notifyDappCashoutResult(dappItem.beneficiary, dappItem.postId, wvp, dappReward, globalProps)
	}

	e.dgp.ModifyProps(func(prop *prototype.DynamicProperties) {
		currentBlockClaimedDappReward := &prototype.Vest{Value: claimedDappReward.Uint64()}
		prop.ClaimedDappRewards.Add(currentBlockClaimedDappReward)
		prop.PoolDappRewards.Sub(currentBlockClaimedDappReward)
	})
}

func (e *Economist) cashoutVotes(votesItems []*VoteItem) {
	e.stateChange.PushCause("voter")
	defer e.stateChange.PopCause()

	if len(votesItems) == 0 {
		return
	}
	globalProps := e.dgp.GetProps()
	var items []IItem
	for _, voteItem := range votesItems {
		items = append(items, voteItem)
	}
	currentBlockVotesWvp := SumItemsWvp(items)
	globalVotesWvps := StringToBigInt(globalProps.GetWeightedVpsVote())
	globalVotesRewards := globalProps.GetPoolVoteRewards()

	currentGlobalVotesWvp := new(big.Int).Add(globalVotesWvps, currentBlockVotesWvp)

	e.dgp.ModifyProps(func(props *prototype.DynamicProperties) {
		props.WeightedVpsVote = currentGlobalVotesWvp.String()
	})

	claimedVoteReward := new(big.Int).SetUint64(0)
	for _, voteItem := range votesItems {
		wvp := voteItem.wvp
		voteReward := ProportionAlgorithm(wvp, currentGlobalVotesWvp, new(big.Int).SetUint64(globalVotesRewards.Value))
		e.stateChange.PutCauseExtra("post", voteItem.postId)
		e.stateChange.PutCauseExtra("wvps", wvp.String())
		e.stateChange.PutCauseExtra("pool", globalVotesRewards.Value)
		e.stateChange.PutCauseExtra("total_wvps", currentGlobalVotesWvp.String())
		post := table.NewSoPostWrap(e.db, &voteItem.postId)
		e.stateChange.PutCauseExtra("rootid", post.GetRootId())
		e.stateChange.PutCauseExtra("reward", voteReward.Uint64())
		e.stateChange.PutCauseExtra("voter", voteItem.beneficiary)

		e.stateChange.AddChange("VoteCashout","add", nil)

		result := e.processRewardForAccount(voteItem.beneficiary, voteReward)
		if !result {
			voteReward = new(big.Int).SetUint64(0)
		}
		claimedVoteReward.Add(claimedVoteReward, voteReward)
		e.notifyVoteCashoutResult(voteItem.beneficiary, voteItem.postId, wvp, voteReward, globalProps)
	}

	e.dgp.ModifyProps(func(props *prototype.DynamicProperties) {
		currentBlockClaimedVoteReward := &prototype.Vest{Value: claimedVoteReward.Uint64()}
		props.ClaimedVoteRewards.Add(currentBlockClaimedVoteReward)
		props.PoolVoteRewards.Sub(currentBlockClaimedVoteReward)
	})
}

func (e *Economist) processRewardForAccount(accountName string, reward *big.Int) bool {
	if EqualZero(reward) {
		return true
	}
	account, err := e.getAccount(&prototype.AccountName{Value: accountName})
	if err != nil {
		e.log.Warnf("account not found, name %s", accountName)
		return false
	}
	if account.GetReputation() == constants.MinReputation {
		e.log.Warnf("ignore cashout to account %s because of bad reputation", accountName)
		return false
	}
	rewardVest := &prototype.Vest{Value: reward.Uint64()}
	oldVest := account.GetVest()
	newVest := rewardVest.Add(oldVest)
	account.SetVest(newVest)
	updateBpVoteValue(e.db, &prototype.AccountName{Value: accountName}, oldVest, newVest)
	return true
}

func (e *Economist) finalizePostCashout(postId uint64, reward *big.Int) {
	rewardVest := &prototype.Vest{Value: reward.Uint64()}
	post := table.NewSoPostWrap(e.db, &postId)
	// for multi fields assigning
	//post.SetRewards(rewardVest)
	//post.SetCashoutBlockNum(CashoutCompleted)
	post.Modify(func(tInfo *table.SoPost) {
		tInfo.Rewards = rewardVest
		tInfo.CashoutBlockNum = CashoutCompleted
	})
}

func (e *Economist) finalizePostDappCashout(postId uint64, reward *big.Int) {
	rewardVest := &prototype.Vest{Value: reward.Uint64()}
	post := table.NewSoPostWrap(e.db, &postId)
	dappReward := post.GetDappRewards()
	newDappReward := rewardVest.Add(dappReward)
	post.SetDappRewards(newDappReward)
}

func (e *Economist) notifyPostCashoutResult(beneficiary string, postId uint64, weightedVp *big.Int, reward *big.Int, prop *prototype.DynamicProperties) {
	if GreaterThanZero(reward) {
		//rInfo := &itype.RewardInfo{
		//	Beneficiary:beneficiary,
		//	Reward: reward.Uint64(),
		//	PostId: postId,
		//}
		//e.noticer.Publish(constants.NoticeCashout, beneficiary, postId, reward.Uint64(), prop.GetHeadBlockNumber())
	}
}

func (e *Economist) notifyReplyCashoutResult(beneficiary string, postId uint64, weightedVp *big.Int, reward *big.Int, prop *prototype.DynamicProperties) {
	if GreaterThanZero(reward) {
		//rInfo := &itype.RewardInfo{
		//	Beneficiary:beneficiary,
		//	Reward: reward.Uint64(),
		//	PostId: postId,
		//}
		//e.noticer.Publish(constants.NoticeCashout, beneficiary, postId, reward.Uint64(), prop.GetHeadBlockNumber())
		//e.observer.AddOpState(iservices.Update, "replyReward", beneficiary, rInfo)
	}
}

func (e *Economist) notifyVoteCashoutResult(beneficiary string, postId uint64, weightedVp *big.Int, reward *big.Int, prop *prototype.DynamicProperties) {
	if GreaterThanZero(reward) {
		//b := &itype.VoteRewardInfo{
		//	Beneficiary:beneficiary,
		//	Reward:reward.Uint64(),
		//	VotePostId:postId,
		//}
		//e.noticer.Publish(constants.NoticeCashout, beneficiary, postId, reward.Uint64(), prop.GetHeadBlockNumber())
		//e.observer.AddOpState(iservices.Add, "voteReward", beneficiary, b)
	}
}

func (e *Economist) notifyDappCashoutResult(beneficiary string, postId uint64, weightedVp *big.Int, reward *big.Int, prop *prototype.DynamicProperties) {
	if GreaterThanZero(reward) {
		//rInfo := &itype.DappRewardInfo{
		//	Beneficiary:beneficiary,
		//	Reward:reward.Uint64(),
		//	RelatedPostId:postId,
		//}
		//e.noticer.Publish(constants.NoticeCashout, beneficiary, postId, reward.Uint64(), prop.GetHeadBlockNumber())
		//e.observer.AddOpState(iservices.Add, "dappcashout", beneficiary, rInfo)
	}
}

func (e *Economist) PowerDown() {
	e.stateChange.PushCause("power_down")
	defer e.stateChange.PopCause()

	globalProps := e.dgp.GetProps()
	if !globalProps.GetBlockProducerBootCompleted() {
		return
	}
	iterator := table.NewAccountNextPowerdownBlockNumWrap(e.db)
	var accountNames []*prototype.AccountName

	timing := common.NewTiming()
	timing.Begin()

	current := globalProps.HeadBlockNumber
	err := iterator.ForEachByOrder(nil, &current, nil, nil, func(mVal *prototype.AccountName, sVal *uint64, idx uint32) bool {
		accountNames = append(accountNames, mVal)
		return true
	})
	if err != nil {
		panic("economist powerdown failed when iterator")
	}
	timing.Mark()

	var powerdownQuota uint64 = 0
	accountFee := globalProps.GetAccountCreateFee().ToVest().GetValue()
	for _, accountName := range accountNames {
		accountWrap := table.NewSoAccountWrap(e.db, accountName)
		max := accountWrap.GetVest().Sub(accountWrap.GetBorrowedVest()).GetValue()
		if max < accountFee {
			max = 0
		} else {
			max -= accountFee
		}
		remaining := accountWrap.GetToPowerdown().Sub(accountWrap.GetHasPowerdown()).GetValue()
		planned := accountWrap.GetEachPowerdownRate().GetValue()
		powerdownQuota = Min(max, Min(planned, remaining))

		oldVest := accountWrap.GetVest()
		powerDownQuotaVest := &prototype.Vest{Value: powerdownQuota}
		powerDownQuotaCoin := powerDownQuotaVest.ToCoin()
		newVest := accountWrap.GetVest()
		newVest.Sub(powerDownQuotaVest)
		newBalance := accountWrap.GetBalance()
		newBalance.Add(powerDownQuotaCoin)
		newHasPowerdown := accountWrap.GetHasPowerdown()
		newHasPowerdown.Add(powerDownQuotaVest)

		accountWrap.Modify(func(acc *table.SoAccount) {
			acc.Vest = newVest
			acc.Balance = newBalance
			acc.HasPowerdown = newHasPowerdown
		})
		updateBpVoteValue(e.db, accountName, oldVest, newVest)
		// update total cos and total vest shares
		e.dgp.ModifyProps(func(props *prototype.DynamicProperties) {
			props.TotalCos.Add(powerDownQuotaCoin)
			props.TotalVest.Sub(powerDownQuotaVest)
		})
		if accountWrap.GetHasPowerdown().Value >= accountWrap.GetToPowerdown().Value || accountWrap.GetVest().Value <= accountFee {
			accountWrap.Modify(func(acc *table.SoAccount) {
				acc.EachPowerdownRate = &prototype.Vest{Value: 0}
				acc.StartPowerdownBlockNum = 0
				acc.NextPowerdownBlockNum = math.MaxUint64
			})
		} else {
			accountWrap.SetNextPowerdownBlockNum(current + constants.PowerDownBlockInterval)
		}
	}
	timing.End()
	e.log.Debugf("powerdown: %s", timing.String())
}

func (e *Economist) SetStateChangeContext(ctx *blocklog.StateChangeContext) {
	e.stateChange = ctx
}

func (e *Economist) DeliverDelegatedVests() {
	e.stateChange.PushCause("deliver_vest")
	defer e.stateChange.PopCause()

	globalProps := e.dgp.GetProps()
	if !globalProps.GetBlockProducerBootCompleted() {
		return
	}

	timing := common.NewTiming()
	timing.Begin()

	// fetch matured delivering delegation orders
	blockNumber := globalProps.GetHeadBlockNumber() + 1
	var orders []uint64
	err := table.NewVestDelegationDeliveryBlockWrap(e.db).
		ForEachByOrder(nil, &blockNumber, nil, nil, func(mVal *uint64, sVal *uint64, idx uint32) bool {
			orders = append(orders, *mVal)
			return true
	})
	if err != nil {
		panic("economist failed fetching vest delegation orders")
	}
	e.log.Debugf("deliver_vest: %d orders", len(orders))
	timing.Mark()

	// for each order, delete it after paying the lender
	for _, orderId := range orders {
		rec := table.NewSoVestDelegationWrap(e.db, &orderId)
		accountName := rec.GetFromAccount()
		account := table.NewSoAccountWrap(e.db, accountName)
		oldVest := account.GetVest()
		amount := rec.GetAmount()
		account.Modify(func(r *table.SoAccount) {
			r.DeliveringVest.Sub(amount)
			r.Vest.Add(amount)
		})
		updateBpVoteValue(e.db, accountName, oldVest, account.GetVest())
		rec.RemoveVestDelegation()
	}
	timing.End()
	e.log.Debugf("deliver_vest: %s", timing.String())
}
