package itype

import "github.com/coschain/contentos-go/prototype"

type Row struct {
	Date string
	Dapp string
	Dau  uint32
	Dnu  uint32
	TrxCount uint32
	Amount uint64
	TotalUserCount uint32
}

type PostInfo struct {
	Id uint64 `json:"id"`
	Created *prototype.TimePointSec `json:"created"`
	Author string `json:"author"`
	Content string `json:"content"`
	Title string `json:"title"`
	Tags []string `json:"tags"`
}

type ReplyInfo struct {
	Id uint64 `json:"id"`
	Created *prototype.TimePointSec `json:"created"`
	Author string `json:"author"`
	ParentId uint64 `json:"parentid"`
	Content string `json:"content"`
}

type VoteInfo struct {
	Voter string `json:"voter"`
	PostId uint64 `json:"postid"`
	Created *prototype.TimePointSec `json:"created"`
	VotePower string `json:"votepower"`
}

type RewardInfo struct {
	Reward uint64 `json:"reward"`
	PostId uint64 `json:"postid"`
}