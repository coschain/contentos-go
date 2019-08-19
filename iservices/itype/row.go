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
	Id uint64
	Created *prototype.TimePointSec
	Author string
	Content string
	Title string
	Tags []string
}

type ReplyInfo struct {
	Id uint64
	Created *prototype.TimePointSec
	Author string
	ParentId uint64
	Content string
}

type VoteInfo struct {
	Voter string
	PostId uint64
	Created *prototype.TimePointSec
	VotePower string
}

type RewardInfo struct {
	Reward uint64
	PostId uint64
}