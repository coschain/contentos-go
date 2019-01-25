package commands

import (
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet/mock"
	"github.com/coschain/contentos-go/rpc/mock_grpcpb"
	"github.com/coschain/contentos-go/rpc/pb"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestVote(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_grpcpb.NewMockApiServiceClient(ctrl)
	mywallet := mock_wallet.NewMockWallet(ctrl)
	myassert := assert.New(t)
	cmd := VoteCmd()
	cmd.SetContext("wallet", mywallet)
	cmd.SetContext("rpcclient", client)
	for _, child := range cmd.Commands() {
		child.Context = cmd.Context
	}
	cmd.SetArgs([]string{"initminer", "10000000000000000"})
	priv_account := &wallet.PrivAccount{
		Account: wallet.Account{
			Name:   "initminer",
			PubKey: "COS5JVLLcTPhq4Unr194JzWPDNSYGoMcam8yxnsjgRVo3Nb7ioyFW",
		},
		PrivKey: "4DjYx2KAGh1NP3dai7MZTLUBMMhMBPmwouKE8jhVSESywccpVZ",
	}
	mywallet.EXPECT().GetUnlockedAccount("initminer").Return(priv_account, true)
	resp := &grpcpb.BroadcastTrxResponse{Status: 1, Msg: "success"}
	client.EXPECT().BroadcastTrx(gomock.Any(), gomock.Any()).Return(resp, nil).Do(func(context interface{}, req *grpcpb.BroadcastTrxRequest) {
		op := req.Transaction.Trx.Operations[0]
		vote_op := op.GetOp9()
		myassert.Equal(vote_op.Voter.Value, "initminer")
	})
	_, err := cmd.ExecuteC()
	if err != nil {
		t.Error(err)
	}
}
