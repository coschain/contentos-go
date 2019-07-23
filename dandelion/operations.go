package dandelion

import (
	"github.com/coschain/contentos-go/prototype"
)

func AccountCreate(creator, account string, owner *prototype.PublicKeyType, fee uint64, jsonMeta string) *prototype.Operation {
	return prototype.GetPbOperation(&prototype.AccountCreateOperation{
		Creator: prototype.NewAccountName(creator),
		NewAccountName: prototype.NewAccountName(account),
		Owner: owner,
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
		Pubkey: pubkey,
	})
}

func TransferToVesting(from, to string, amount uint64) *prototype.Operation {
	return prototype.GetPbOperation(&prototype.TransferToVestingOperation{
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
