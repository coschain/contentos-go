package dandelion

import (
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/prototype"
)

func AccountCreate(creator, account string, owner *prototype.PublicKeyType, fee uint64, jsonMeta string) *prototype.Operation {
	return prototype.GetPbOperation(&prototype.AccountCreateOperation{
		Creator: prototype.NewAccountName(creator),
		NewAccountName: prototype.NewAccountName(account),
		PubKey: owner,
		Fee: prototype.NewCoin(fee),
		JsonMetadata: jsonMeta,
	})
}

func Transfer(from, to string, amount uint64, memo string) *prototype.Operation {
	return prototype.GetPbOperation(&prototype.TransferOperation{
		From: prototype.NewAccountName(from),
		To: prototype.NewAccountName(to),
		Amount: prototype.NewCoin(amount),
		Memo: memo,
	})
}

func AccountUpdate(name string, pubkey *prototype.PublicKeyType) *prototype.Operation {
	return prototype.GetPbOperation(&prototype.AccountUpdateOperation{
		Owner: prototype.NewAccountName(name),
		PubKey: pubkey,
	})
}

func TransferToVest(from, to string, amount uint64) *prototype.Operation {
	return prototype.GetPbOperation(&prototype.TransferToVestOperation{
		From: prototype.NewAccountName(from),
		To: prototype.NewAccountName(to),
		Amount: prototype.NewCoin(amount),
	})
}

func Vote(voter string, postId uint64) *prototype.Operation {
	return prototype.GetPbOperation(&prototype.VoteOperation{
		Voter: prototype.NewAccountName(voter),
		Idx: postId,
	})
}

func BpRegister(name, url, desc string, signingKey *prototype.PublicKeyType, props *prototype.ChainProperties) *prototype.Operation {
	return prototype.GetPbOperation(&prototype.BpRegisterOperation{
		Owner: prototype.NewAccountName(name),
		Url: url,
		Desc: desc,
		BlockSigningKey: signingKey,
		Props: props,
	})
}

func BpUpdate(name string, props *prototype.ChainProperties) *prototype.Operation {
	return prototype.GetPbOperation(&prototype.BpUpdateOperation{
		Owner: prototype.NewAccountName(name),
		Props: props,
	})
}

func BpEnable(name string) *prototype.Operation {
	return prototype.GetPbOperation(&prototype.BpEnableOperation{
		Owner:    prototype.NewAccountName(name),
	})
}

func BpDisable(name string) *prototype.Operation {
	return prototype.GetPbOperation(&prototype.BpEnableOperation{
		Owner:    prototype.NewAccountName(name),
		Cancel:   true,
	})
}

func BpVote(voter, bp string, cancel bool) *prototype.Operation {
	return prototype.GetPbOperation(&prototype.BpVoteOperation{
		Voter: prototype.NewAccountName(voter),
		BlockProducer: prototype.NewAccountName(bp),
		Cancel:cancel,
	})
}

func Follow(follower, followee string, cancel bool) *prototype.Operation {
	return prototype.GetPbOperation(&prototype.FollowOperation{
		Account: prototype.NewAccountName(follower),
		FAccount: prototype.NewAccountName(followee),
		Cancel: cancel,
	})
}

func ContractDeploy(owner, contract string, abi, code []byte, upgradable bool, url, desc string) *prototype.Operation {
	return prototype.GetPbOperation(&prototype.ContractDeployOperation{
		Owner: prototype.NewAccountName(owner),
		Contract: contract,
		Abi: abi,
		Code: code,
		Upgradeable: upgradable,
		Url: url,
		Describe: desc,
	})
}

func ContractDeployUncompressed(owner, contract string, abi, code []byte, upgradable bool, url, desc string) *prototype.Operation {
	var (
		err error
		compressedCode, compressedAbi []byte
	)
	if compressedCode, err = common.Compress(code); err != nil {
		return nil
	}
	if compressedAbi, err = common.Compress(abi); err != nil {
		return nil
	}
	return ContractDeploy(owner, contract, compressedAbi, compressedCode, upgradable, url, desc)
}

func ContractApply(caller, owner, contract, method, jsonParams string, coins uint64) *prototype.Operation {
	return prototype.GetPbOperation(&prototype.ContractApplyOperation{
		Caller: prototype.NewAccountName(caller),
		Owner: prototype.NewAccountName(owner),
		Contract: contract,
		Method: method,
		Params: jsonParams,
		Amount: prototype.NewCoin(coins),
	})
}

func Post(postId uint64, author, title, content string, tags []string, beneficiaries []map[string]int) *prototype.Operation {
	var benefits []*prototype.BeneficiaryRouteType
	if len(beneficiaries) > 0 {
		for _, e := range beneficiaries {
			var (
				name string
				weight int
			)
			for name, weight = range e {
				break
			}
			benefits = append(benefits, &prototype.BeneficiaryRouteType{
				Name: prototype.NewAccountName(name),
				Weight: uint32(weight),
			})
		}
	}
	return prototype.GetPbOperation(&prototype.PostOperation{
		Uuid: postId,
		Owner: prototype.NewAccountName(author),
		Title: title,
		Content: content,
		Tags: tags,
		Beneficiaries: benefits,
	})
}

func Reply(postId, parentId uint64, author, content string, beneficiaries []map[string]int) *prototype.Operation {
	var benefits []*prototype.BeneficiaryRouteType
	if len(beneficiaries) > 0 {
		for _, e := range beneficiaries {
			var (
				name string
				weight int
			)
			for name, weight = range e {
				break
			}
			benefits = append(benefits, &prototype.BeneficiaryRouteType{
				Name: prototype.NewAccountName(name),
				Weight: uint32(weight),
			})
		}
	}
	return prototype.GetPbOperation(&prototype.ReplyOperation{
		Uuid: postId,
		ParentUuid: parentId,
		Owner: prototype.NewAccountName(author),
		Content: content,
		Beneficiaries: benefits,
	})
}

func Report(reporter string, postId uint64, reason []prototype.ReportOperationTag, arbitration, approved bool) *prototype.Operation {
	return prototype.GetPbOperation(&prototype.ReportOperation{
		Reporter: prototype.NewAccountName(reporter),
		Reported: postId,
		ReportTag: reason,
		IsArbitration: arbitration,
		IsApproved: approved,
	})
}

func ConvertVest(name string, vests uint64) *prototype.Operation {
	return prototype.GetPbOperation(&prototype.ConvertVestOperation{
		From: prototype.NewAccountName(name),
		Amount: prototype.NewVest(vests),
	})
}

func Stake(from, to string, coins uint64) *prototype.Operation {
	return prototype.GetPbOperation(&prototype.StakeOperation{
		From: prototype.NewAccountName(from),
		To: prototype.NewAccountName(to),
		Amount: prototype.NewCoin(coins),
	})
}

func UnStake(creditor, debtor string, coins uint64) *prototype.Operation {
	return prototype.GetPbOperation(&prototype.UnStakeOperation{
		Creditor: prototype.NewAccountName(creditor),
		Debtor: prototype.NewAccountName(debtor),
		Amount: prototype.NewCoin(coins),
	})
}

func AcquireTicket(name string, count uint64) *prototype.Operation {
	return prototype.GetPbOperation(&prototype.AcquireTicketOperation{
		Account: prototype.NewAccountName(name),
		Count: count,
	})
}

func VoteByTicket(name string, idx, count uint64) *prototype.Operation {
	return prototype.GetPbOperation(&prototype.VoteByTicketOperation{
		Account: prototype.NewAccountName(name),
		Idx: idx,
		Count: count,
	})
}
