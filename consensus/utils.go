package consensus

import (
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/gobft/message"
)

func ExtractBlockID(commit *message.Commit) common.BlockID {
	return common.BlockID{
		Data: commit.ProposedData,
	}
}

func ConvertToBlockID(data message.ProposedData) common.BlockID {
	return common.BlockID{
		Data: data,
	}
}
