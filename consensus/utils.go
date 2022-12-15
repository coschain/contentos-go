package consensus

import (
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/db/blocklog"
	"github.com/coschain/contentos-go/db/forkdb"
	"github.com/coschain/contentos-go/prototype"
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

func fetchBlocks(from, to uint64, forkDB *forkdb.DB, blog *blocklog.BLog) ([]common.ISignedBlock, error) {
	if from > to {
		return nil, nil
	}

	if forkDB.Empty() {
		return nil, ErrEmptyForkDB
	}

	lastCommitted := forkDB.LastCommitted()
	lastCommittedNum := lastCommitted.BlockNum()
	headNum := forkDB.Head().Id().BlockNum()

	if from == 0 {
		from = 1
	}
	if to > headNum {
		to = headNum
	}

	forkDBFrom := uint64(0)
	forkDBTo := to
	if to >= lastCommittedNum {
		forkDBFrom = lastCommittedNum
		if from > forkDBFrom {
			forkDBFrom = from
		}
	}

	blogFrom := uint64(0)
	if from < lastCommittedNum {
		blogFrom = from
	}
	blogTo := to
	if blogTo >= lastCommittedNum {
		blogTo = lastCommittedNum - 1
	}

	var blocksInForkDB []common.ISignedBlock
	var err error
	if forkDBFrom > 0 {
		blocksInForkDB, err = forkDB.FetchBlocksFromMainBranch(forkDBFrom)
		if err != nil {
			// there probably is a new committed block during the execution of this process, just try again
			return nil, ErrForkDBChanged
		}
		if int(forkDBTo-forkDBFrom+1) < len(blocksInForkDB) {
			blocksInForkDB = blocksInForkDB[:forkDBTo-forkDBFrom+1]
		}
	}

	var blocksInBlog []common.ISignedBlock
	if blogFrom > 0 {
		blocksInBlog = make([]common.ISignedBlock, 0, blogTo-blogFrom+1)
		for blogFrom <= blogTo {
			b := &prototype.SignedBlock{}
			if err := blog.ReadBlock(b, int64(blogFrom-1)); err != nil {
				return nil, err
			}

			blocksInBlog = append(blocksInBlog, b)
			blogFrom++
		}
	}

	return append(blocksInBlog, blocksInForkDB...), nil
}