package itype

type Row struct {
	Timestamp uint64
	Dapp string
	Dau  uint32
	Dnu  uint32
	TrxCount uint32
	Amount uint64
	TotalUserCount uint32
}

type PostInfo struct {
	Id uint64 `json:"id"`
	Created uint32 `json:"created"`
	Author string `json:"author"`
	Content string `json:"content"`
	Title string `json:"title"`
	Tags []string `json:"tags"`
}

type ReplyInfo struct {
	Id uint64 `json:"id"`
	Created uint32 `json:"created"`
	Author string `json:"author"`
	ParentId uint64 `json:"parentid"`
	Content string `json:"content"`
}

type VoteInfo struct {
	Voter string `json:"voter"`
	PostId uint64 `json:"postid"`
	Created uint32 `json:"created"`
	VotePower string `json:"votepower"`
}

type RewardInfo struct {
	Beneficiary string `json:"beneficiary"`
	Reward uint64 `json:"reward"`
	PostId uint64 `json:"postid"`
}

type VoteRewardInfo struct {
	Beneficiary string `json:"beneficiary"`
	Reward uint64 `json:"reward"`
	VotePostId uint64 `json:"postid"`
}

type DappRewardInfo struct {
	Beneficiary string `json:"beneficiary"`
	Reward uint64 `json:"reward"`
	RelatedPostId uint64 `json:"postid"`
}