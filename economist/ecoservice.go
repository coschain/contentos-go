package economist

import (
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/prototype"
	"github.com/pkg/errors"
)

var (
	SINGLE_ID int32 = 1
)

type Economist struct {
	ctx               *node.ServiceContext
	db                iservices.IDatabaseService
	rewardAccumulator uint64 // reward accumulator
	vpAccumulator     uint64 // vote power accumulator
	globalProps       *prototype.DynamicProperties
}

func (e *Economist) getDb() (iservices.IDatabaseService, error) {
	s, err := e.ctx.Service("db")
	if err != nil {
		return nil, err
	}
	db := s.(iservices.IDatabaseService)
	return db, nil
}

func New(ctx *node.ServiceContext) (*Economist, error) {

	return &Economist{ctx: ctx}, nil
}

func (e *Economist) Start(node *node.Node) error {
	db, err := e.getDb()
	if err != nil {
		return errors.New("Economist fetch db service error")
	}
	e.db = db
	dgpWrap := table.NewSoGlobalWrap(e.db, &SINGLE_ID)
	if !dgpWrap.CheckExist() {
		return errors.New("the mainkey is already exist")
	}
	e.globalProps = dgpWrap.GetProps()
	return nil
}

func (e *Economist) Stop() error {
	return nil
}

func (e *Economist) getBucket() uint32 {
	return (e.globalProps.Time.UtcSeconds - uint32(constants.GenesisTime)) / uint32(constants.BLOCK_INTERVAL)
}

func (e *Economist) pastVoteId(voterName *prototype.AccountName, idx uint64) *prototype.VoterId {
	vote_wrap := table.NewVotePostIdWrap(e.db)
	vote_iter := vote_wrap.QueryListByOrder(&idx, nil)
	for vote_iter.Valid() {
		voterId := vote_wrap.GetMainVal(vote_iter)
		if voterId.Voter.Value == voterName.Value {
			return voterId
		}
		if ok := vote_iter.Next(); !ok {
			break
		}
	}
	return nil
}

// the interactive operations between user and economic
// upvote is true: upvote otherwise downvote
// no downvote has been supplied by command, so I ignore it
func (e *Economist) DoVote(voterName *prototype.AccountName, idx uint64) error {
	voter := table.NewSoAccountWrap(e.db, voterName)
	elapsedSeconds := e.globalProps.Time.UtcSeconds - voter.GetLastVoteTime().UtcSeconds
	if elapsedSeconds < constants.MIN_VOTE_INTERVAL {
		return errors.New("voting too frequent")
	}
	// until now, No Unvote command has been supplied, so I just deal it
	// repeat vote is thought illegal.
	pastVoteId := e.pastVoteId(voterName, idx)
	if pastVoteId != nil {
		//pastVote := table.NewSoVoteWrap(e.db, pastVoteId)
		return errors.New("vote to a same post")
	}

	regeneratedPower := constants.PERCENT * elapsedSeconds / constants.VOTE_REGENERATE_TIME
	var currentVp uint32
	votePower := voter.GetVotePower() + regeneratedPower
	if votePower > constants.PERCENT {
		currentVp = constants.PERCENT
	} else {
		currentVp = votePower
	}
	usedVp := (currentVp + constants.VOTE_LIMITE_DURING_REGENERATE - 1) / constants.VOTE_LIMITE_DURING_REGENERATE

	voter.MdVotePower(currentVp - usedVp)
	vesting := voter.GetVestingShares().Value
	weighted_vp := vesting * uint64(usedVp)
	// even to vote a expired post, vote power will be discounted but do not have any benefit
	post := table.NewSoPostWrap(e.db, &idx)
	if post.GetCashoutTime().UtcSeconds < e.globalProps.Time.UtcSeconds {
		last_vp := post.GetWeightedVp()
		votePower := last_vp + weighted_vp
		e.globalProps.WeightedVps += weighted_vp
		//var votePower uint64
		//if like {
		//	votePower = last_vp + weighted_vp
		//	e.globalProps.WeightedVps += weighted_vp
		//} else {
		//	if last_vp < weighted_vp {
		//		votePower = 0
		//	} else {
		//		votePower = last_vp - weighted_vp
		//		e.globalProps.WeightedVps -= weighted_vp
		//	}
		//}
		post.MdWeightedVp(votePower)

		_ = table.NewSoVoteWrap(e.db, &prototype.VoterId{Voter: voterName, PostId: idx}).Create(func(tInfo *table.SoVote) {
			tInfo.PostId = idx
			tInfo.Voter = &prototype.VoterId{Voter: voterName, PostId: idx}
			tInfo.Upvote = true
			tInfo.WeightedVp = weighted_vp
			tInfo.VoteTime = e.globalProps.Time
		})
	}

	return nil
}

// do not consider edit
func (e *Economist) DoPost(authorName *prototype.AccountName, idx uint64, title, content string, tags []string,
	beneficiaries []*prototype.BeneficiaryRouteType) error {
	author := table.NewSoAccountWrap(e.db, authorName)
	elapsedSeconds := e.globalProps.Time.UtcSeconds - author.GetLastPostTime().UtcSeconds
	if elapsedSeconds < constants.MIN_POST_INTERVAL {
		return errors.New("posting too frequent")
	}
	_ = table.NewSoPostWrap(e.db, &idx).Create(func(tInfo *table.SoPost) {
		tInfo.PostId = idx
		tInfo.Author = authorName
		tInfo.WeightedVp = 0
		tInfo.CashoutTime = &prototype.TimePointSec{UtcSeconds: e.globalProps.Time.UtcSeconds + uint32(constants.POST_CASHPUT_DELAY_TIME)}
		tInfo.Title = title
		tInfo.Body = content
		tInfo.Tags = tags
		tInfo.Depth = 0
		tInfo.ParentId = 0
		tInfo.RootId = 0
		tInfo.Beneficiaries = beneficiaries
	})
	return nil
}

func (e *Economist) DoReply(authorName *prototype.AccountName, idx, pidx uint64, content string,
	beneficiaries []*prototype.BeneficiaryRouteType) error {
	//[author] [content] [postId]
	author := table.NewSoAccountWrap(e.db, authorName)
	elapsedSeconds := e.globalProps.Time.UtcSeconds - author.GetLastPostTime().UtcSeconds
	if elapsedSeconds < constants.MIN_POST_INTERVAL {
		return errors.New("posting too frequent")
	}
	post := table.NewSoPostWrap(e.db, &pidx)
	var rootId uint64
	if post.GetRootId() == 0 {
		rootId = post.GetPostId()
	} else {
		rootId = post.GetRootId()
	}
	_ = table.NewSoPostWrap(e.db, &idx).Create(func(tInfo *table.SoPost) {
		tInfo.PostId = idx
		tInfo.ParentId = post.GetPostId()
		tInfo.RootId = rootId
		tInfo.Author = authorName
		tInfo.WeightedVp = 0
		tInfo.CashoutTime = &prototype.TimePointSec{UtcSeconds: e.globalProps.Time.UtcSeconds + uint32(constants.POST_CASHPUT_DELAY_TIME)}
		tInfo.Title = ""
		tInfo.Body = content
		tInfo.Depth = post.GetDepth() + 1
		tInfo.Beneficiaries = beneficiaries
	})
	return nil
}

//
func (e *Economist) Do() error {

}
